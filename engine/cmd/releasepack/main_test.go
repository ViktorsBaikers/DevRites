package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestWriteArchiveIsDeterministic(t *testing.T) {
	t.Parallel()
	epoch := time.Unix(1_700_000_000, 0).UTC()
	firstRoot := makeTree(t, time.Unix(100, 0))
	secondRoot := makeTree(t, time.Unix(2_000_000_000, 0))
	firstArchive := filepath.Join(t.TempDir(), "release.tar.gz")
	secondArchive := filepath.Join(t.TempDir(), "release.tar.gz")

	if err := writeArchive(firstRoot, firstArchive, "devrites-v1.2.3", epoch); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(secondRoot, secondArchive, "devrites-v1.2.3", epoch); err != nil {
		t.Fatal(err)
	}
	firstBytes, err := os.ReadFile(firstArchive)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(secondArchive)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("archives differ across source roots and mtimes")
	}

	got := readArchive(t, firstArchive, epoch)
	wantNames := []string{
		"devrites-v1.2.3/",
		"devrites-v1.2.3/README.md",
		"devrites-v1.2.3/install.sh",
		"devrites-v1.2.3/nested/",
		"devrites-v1.2.3/nested/data.txt",
	}
	if !reflect.DeepEqual(got.names, wantNames) {
		t.Fatalf("archive names = %#v, want %#v", got.names, wantNames)
	}
	if got.modes["devrites-v1.2.3/"] != 0o755 ||
		got.modes["devrites-v1.2.3/install.sh"] != 0o755 ||
		got.modes["devrites-v1.2.3/README.md"] != 0o644 {
		t.Fatalf("archive modes = %#v", got.modes)
	}
	if got.contents["devrites-v1.2.3/install.sh"] != "#!/bin/sh\nexit 0\n" ||
		got.contents["devrites-v1.2.3/nested/data.txt"] != "payload\n" {
		t.Fatalf("archive contents = %#v", got.contents)
	}
}

func TestWriteArchiveRejectsUnsafePaths(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(t.TempDir(), "release.tar.gz")

	if err := writeArchive(root, filepath.Join(root, "release.tar.gz"), "devrites-v1", time.Unix(0, 0)); err == nil {
		t.Fatal("output inside the staged tree was accepted")
	}
	if err := writeArchive(root, output, "../escape", time.Unix(0, 0)); err == nil {
		t.Fatal("escaping archive prefix was accepted")
	}
	if err := writeArchive(root, output, `devrites-v1\escape`, time.Unix(0, 0)); err == nil {
		t.Fatal("non-portable archive prefix was accepted")
	}

	link := filepath.Join(root, "unsafe-link")
	if err := os.Symlink(filepath.Join(t.TempDir(), "outside"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	err := writeArchive(root, output, "devrites-v1", time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), "symlink is not allowed") {
		t.Fatalf("symlink error = %v", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("unsafe archive output exists: %v", statErr)
	}
}

func TestWriteEntriesRejectsChangedSource(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := filepath.Join(root, "payload.txt")
	if err := os.WriteFile(source, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceRoot.Close()
	entries, err := collectEntries(sourceRoot, "devrites-v1")
	if err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(root, "replacement")
	if err := os.WriteFile(replacement, []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, source); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	err = writeEntries(sourceRoot, tw, entries, time.Unix(0, 0))
	if err == nil || !strings.Contains(err.Error(), "changed after collection") {
		t.Fatalf("changed source error = %v", err)
	}
}

func TestWriteEntriesRejectsEscapingSymlinkSwap(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	source := filepath.Join(root, "payload.txt")
	if err := os.WriteFile(source, []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceRoot, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer sourceRoot.Close()
	entries, err := collectEntries(sourceRoot, "devrites-v1")
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, source); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	if err := writeEntries(sourceRoot, tw, entries, time.Unix(0, 0)); err == nil {
		t.Fatal("escaping symlink swap was accepted")
	}
}

func makeTree(t *testing.T, mtime time.Time) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"README.md":       {"read me\n", 0o644},
		"install.sh":      {"#!/bin/sh\nexit 0\n", 0o644},
		"nested/data.txt": {"payload\n", 0o644},
	}
	for name, file := range files {
		fullPath := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(file.content), file.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(fullPath, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

type archiveContents struct {
	names    []string
	modes    map[string]int64
	contents map[string]string
}

func readArchive(t *testing.T, archivePath string, epoch time.Time) archiveContents {
	t.Helper()
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	if !gz.ModTime.Equal(epoch) || gz.OS != 255 || gz.Name != "" || gz.Comment != "" || len(gz.Extra) != 0 {
		t.Fatalf("gzip header is not normalized: %#v", gz.Header)
	}

	got := archiveContents{modes: map[string]int64{}, contents: map[string]string{}}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		got.names = append(got.names, header.Name)
		got.modes[header.Name] = header.Mode
		if header.Uid != 0 || header.Gid != 0 || header.Uname != "" || header.Gname != "" {
			t.Fatalf("%s ownership is not normalized: %#v", header.Name, header)
		}
		if !header.ModTime.Equal(epoch) || !header.AccessTime.Equal(epoch) || !header.ChangeTime.Equal(epoch) {
			t.Fatalf("%s timestamps are not normalized: %#v", header.Name, header)
		}
		if header.Typeflag == tar.TypeReg {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			got.contents[header.Name] = string(content)
		}
	}
	return got
}
