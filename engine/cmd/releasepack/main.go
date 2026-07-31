// Command releasepack writes a deterministic tar.gz from a staged release tree.
package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type archiveEntry struct {
	archiveName string
	sourcePath  string
	info        os.FileInfo
}

func main() {
	root := flag.String("root", "", "staged release tree")
	output := flag.String("output", "", "output .tar.gz")
	prefix := flag.String("prefix", "", "archive top-level directory")
	epochText := flag.String("epoch", "0", "SOURCE_DATE_EPOCH")
	flag.Parse()

	epoch, err := strconv.ParseUint(*epochText, 10, 32)
	if err != nil {
		exitf("invalid epoch %q: %v", *epochText, err)
	}
	if *root == "" || *output == "" || *prefix == "" {
		exitf("-root, -output, and -prefix are required")
	}
	if err := writeArchive(*root, *output, *prefix, time.Unix(int64(epoch), 0).UTC()); err != nil {
		exitf("%v", err)
	}
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "releasepack: "+format+"\n", args...)
	os.Exit(1)
}

func writeArchive(root, output, prefix string, epoch time.Time) error {
	if err := validatePrefix(prefix); err != nil {
		return err
	}
	if epoch.Unix() < 0 || epoch.Unix() > math.MaxUint32 {
		return fmt.Errorf("epoch must fit the gzip timestamp field")
	}

	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	canonicalRoot, err = filepath.Abs(canonicalRoot)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}
	outputDir, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	output, err = filepath.Abs(filepath.Join(outputDir, filepath.Base(output)))
	if err != nil {
		return fmt.Errorf("resolve output: %w", err)
	}
	if within(canonicalRoot, output) {
		return errors.New("output must be outside the staged release tree")
	}

	sourceRoot, err := os.OpenRoot(canonicalRoot)
	if err != nil {
		return fmt.Errorf("open staged release tree: %w", err)
	}
	defer sourceRoot.Close()

	entries, err := collectEntries(sourceRoot, prefix)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(output), "."+filepath.Base(output)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary archive: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		return errors.Join(fmt.Errorf("set archive mode: %w", err), tmp.Close())
	}

	gz, err := gzip.NewWriterLevel(tmp, gzip.BestCompression)
	if err != nil {
		return errors.Join(fmt.Errorf("create gzip writer: %w", err), tmp.Close())
	}
	gz.Header.ModTime = epoch
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)

	writeErr := writeEntries(sourceRoot, tw, entries, epoch)
	if err := tw.Close(); writeErr == nil {
		writeErr = err
	}
	if err := gz.Close(); writeErr == nil {
		writeErr = err
	}
	if err := tmp.Close(); writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		return fmt.Errorf("write archive: %w", writeErr)
	}
	if err := os.Remove(output); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("replace output: %w", err)
	}
	if err := os.Rename(tmpName, output); err != nil {
		return fmt.Errorf("publish archive: %w", err)
	}
	return nil
}

func validatePrefix(prefix string) error {
	if prefix == "" || prefix == "." || path.IsAbs(prefix) || path.Clean(prefix) != prefix || strings.ContainsAny(prefix, `/\`) {
		return fmt.Errorf("unsafe archive prefix %q", prefix)
	}
	return nil
}

func within(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && (rel == "." || filepath.IsLocal(rel))
}

func collectEntries(root *os.Root, prefix string) ([]archiveEntry, error) {
	var entries []archiveEntry
	err := fs.WalkDir(root.FS(), ".", func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if sourcePath == "." {
			entries = append(entries, archiveEntry{archiveName: prefix + "/", sourcePath: sourcePath, info: info})
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink is not allowed in release payload: %s", sourcePath)
		}
		if !info.Mode().IsRegular() && !info.IsDir() {
			return fmt.Errorf("unsupported release payload entry: %s (%s)", sourcePath, info.Mode())
		}
		if strings.Contains(sourcePath, `\`) {
			return fmt.Errorf("non-portable release payload path: %s", sourcePath)
		}
		name := path.Join(prefix, sourcePath)
		if info.IsDir() {
			name += "/"
		}
		entries = append(entries, archiveEntry{archiveName: name, sourcePath: sourcePath, info: info})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect release payload: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].archiveName < entries[j].archiveName
	})
	return entries, nil
}

func writeEntries(root *os.Root, tw *tar.Writer, entries []archiveEntry, epoch time.Time) error {
	for _, entry := range entries {
		file, err := root.Open(entry.sourcePath)
		if err != nil {
			return fmt.Errorf("open %s: %w", entry.archiveName, err)
		}
		currentInfo, err := file.Stat()
		if err != nil {
			return errors.Join(fmt.Errorf("stat %s: %w", entry.archiveName, err), file.Close())
		}
		if !os.SameFile(entry.info, currentInfo) {
			return errors.Join(fmt.Errorf("release payload changed after collection: %s", entry.archiveName), file.Close())
		}
		if !currentInfo.Mode().IsRegular() && !currentInfo.IsDir() {
			return errors.Join(fmt.Errorf("unsupported release payload entry: %s (%s)", entry.archiveName, currentInfo.Mode()), file.Close())
		}

		mode := int64(0o644)
		typeflag := byte(tar.TypeReg)
		size := currentInfo.Size()
		if currentInfo.IsDir() {
			mode, typeflag, size = 0o755, tar.TypeDir, 0
		} else if currentInfo.Mode().Perm()&0o111 != 0 || path.Ext(entry.archiveName) == ".sh" {
			mode = 0o755
		}
		header := &tar.Header{
			Name:       entry.archiveName,
			Mode:       mode,
			Size:       size,
			ModTime:    epoch,
			AccessTime: epoch,
			ChangeTime: epoch,
			Typeflag:   typeflag,
			Format:     tar.FormatPAX,
		}
		if err := tw.WriteHeader(header); err != nil {
			return errors.Join(err, file.Close())
		}
		if typeflag != tar.TypeReg {
			if err := file.Close(); err != nil {
				return err
			}
			continue
		}
		_, copyErr := io.Copy(tw, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
