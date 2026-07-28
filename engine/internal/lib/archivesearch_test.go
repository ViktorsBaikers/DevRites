package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchiveSearch(t *testing.T) {
	root := t.TempDir()
	writeArchiveSpec(t, root, "search-ranking", "# Search ranking\nRank results by relevance score.\n")
	writeArchiveSpec(t, root, "auth-tokens", "# Auth token rotation\nRotate the search index credentials.\n")
	writeArchiveSpec(t, root, "billing", "# Billing exports\nExport invoices.\n")

	for _, tc := range []struct {
		name     string
		args     []string
		wantCode int
		// wantOrder lists slugs expected in stdout, in order; absent slugs must not appear.
		wantOrder []string
		absent    []string
		wantErr   string
	}{
		{
			name:      "ranks by distinct term hits",
			args:      []string{"search", "ranking"},
			wantCode:  0,
			wantOrder: []string{"search-ranking", "auth-tokens"}, // 2/2 before 1/2
			absent:    []string{"billing"},
		},
		{
			name:     "phrase is split on whitespace",
			args:     []string{"search ranking"},
			wantCode: 0,
			// same result as two args
			wantOrder: []string{"search-ranking", "auth-tokens"},
		},
		{
			name:     "no overlap is a silent success",
			args:     []string{"kubernetes"},
			wantCode: 0,
			absent:   []string{"search-ranking", "auth-tokens", "billing"},
		},
		{
			name:     "missing query is a usage error",
			args:     nil,
			wantCode: 2,
			wantErr:  "usage: devrites-engine archive-search",
		},
		{
			name:     "short noise terms are dropped",
			args:     []string{"a", "to"},
			wantCode: 2, // both under 3 chars → no terms → usage
			wantErr:  "usage: devrites-engine archive-search",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
			code := ArchiveSearch(root, tc.args, stdout, stderr)
			if code != tc.wantCode {
				t.Fatalf("code=%d, want %d; stdout=%q stderr=%q", code, tc.wantCode, stdout.String(), stderr.String())
			}
			out := stdout.String()
			last := -1
			for _, slug := range tc.wantOrder {
				idx := strings.Index(out, slug)
				if idx < 0 {
					t.Fatalf("stdout=%q missing expected slug %q", out, slug)
				}
				if idx < last {
					t.Fatalf("stdout=%q: slug %q out of expected rank order", out, slug)
				}
				last = idx
			}
			for _, slug := range tc.absent {
				if strings.Contains(out, slug) {
					t.Fatalf("stdout=%q unexpectedly contains %q", out, slug)
				}
			}
			if tc.wantErr != "" && !strings.Contains(stderr.String(), tc.wantErr) {
				t.Fatalf("stderr=%q, want substring %q", stderr.String(), tc.wantErr)
			}
		})
	}
}

// TestArchiveSearchNoArchive: absent archive dir → silent exit 0, not a crash.
func TestArchiveSearchNoArchive(t *testing.T) {
	root := t.TempDir()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := ArchiveSearch(root, []string{"anything"}, stdout, stderr); code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout=%q, want empty", stdout.String())
	}
}

func TestArchiveSearchReportsUnreadableArchive(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "archive"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := ArchiveSearch(root, []string{"anything"}, stdout, stderr); code != 3 {
		t.Fatalf("code=%d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "read archive") {
		t.Fatalf("stderr=%q, want archive read error", stderr.String())
	}
}

func TestArchiveSearchReportsUnreadableSpec(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "archive", "broken", "spec.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := ArchiveSearch(root, []string{"anything"}, stdout, stderr); code != 3 {
		t.Fatalf("code=%d, want 3; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "archive-search: read") {
		t.Fatalf("stderr=%q, want spec read error", stderr.String())
	}
}

func writeArchiveSpec(t *testing.T, root, slug, content string) {
	t.Helper()
	dir := filepath.Join(root, "archive", slug)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
