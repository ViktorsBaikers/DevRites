package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditSequenceUnionContinuity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		positions     [3]string
		missingParent bool
		blankPosition bool
		valid         bool
	}{
		{name: "archived_interior_missing_parent", positions: [3]string{"1", "2", "3"}, missingParent: true},
		{name: "release_ten_truncates_first_nine", positions: [3]string{"1", "9", "10"}, missingParent: true},
		{name: "skipped_position", positions: [3]string{"1", "3", "4"}},
		{name: "duplicate_position", positions: [3]string{"1", "1", "2"}},
		{name: "malformed_predecessor_position", positions: [3]string{"1", "two", "3"}},
		{name: "malformed_release_position", positions: [3]string{"1", "2", "three"}},
		{name: "zero_position", positions: [3]string{"0", "1", "2"}},
		{name: "negative_position", positions: [3]string{"-1", "2", "3"}},
		{name: "signed_position", positions: [3]string{"+1", "2", "3"}},
		{name: "leading_zero_position", positions: [3]string{"01", "2", "3"}},
		{name: "overflow_position", positions: [3]string{"1", "2", "18446744073709551616"}},
		{name: "blank_position", positions: [3]string{"1", "", "3"}, blankPosition: true},
		{name: "missing_root_position", positions: [3]string{"", "2", "3"}},
		{name: "missing_interior_position", positions: [3]string{"1", "", "3"}},
		{name: "missing_release_position", positions: [3]string{"1", "2", ""}},
		{name: "position_one_has_parent", positions: [3]string{"1", "2", "1"}},
		{name: "parentless_position_one", positions: [3]string{"1", "2", "3"}, valid: true},
		{name: "fully_unnumbered_legacy_chain", valid: true},
	} {
		for _, consumer := range []string{"merge", "explicit_merge", "verify"} {
			t.Run(tc.name+"/"+consumer, func(t *testing.T) {
				root := unionRoot(t)
				p1Row := "| present | `src/only-p1.ts` | SLICE-001 | First product path. |"
				p2Row := "| present | `src/p2.ts` | SLICE-002 | Second product path. |"
				releaseRow := "| present | `src/release.ts` | SLICE-003 | Release wiring. |"
				releaseRows := []string{releaseRow}
				if consumer == "verify" {
					releaseRows = []string{p2Row, releaseRow}
					// Reproduce the truncated union: p1 exists, but p2 lost its parent link.
					if !tc.missingParent {
						releaseRows = []string{p1Row, p2Row, releaseRow}
					}
				}
				unionWorkspace(t, root, "p1", manifestBody(p1Row), false)
				unionWorkspace(t, root, "p2", manifestBody(p2Row), true)
				unionWorkspace(t, root, "release", manifestBody(releaseRows...), false)
				p2Parent := "p1"
				if tc.missingParent {
					p2Parent = ""
				}
				for i, node := range []struct {
					location string
					slug     string
					parent   string
				}{
					{location: "work", slug: "p1"},
					{location: "archive", slug: "p2", parent: p2Parent},
					{location: "work", slug: "release", parent: "p2"},
				} {
					body := "| phase | seal |\n| schema | 4 |\n"
					if tc.positions[i] != "" || tc.blankPosition && i == 1 {
						body += "| sequence_position | " + tc.positions[i] + " |\n"
					}
					if node.parent != "" {
						body += "| sequence_parent | " + node.parent + " |\n"
					}
					if node.slug == "release" {
						body += "| sequence_role | release |\n"
					}
					if err := os.WriteFile(filepath.Join(root, node.location, node.slug, "state.md"), []byte(body), 0o644); err != nil {
						t.Fatal(err)
					}
				}

				if consumer == "verify" {
					err := VerifyReleaseUnion(root, "release")
					if tc.valid && err != nil {
						t.Fatalf("valid release union rejected: %v", err)
					}
					if !tc.valid && err == nil {
						t.Fatal("release gate accepted an invalid sequence as a complete union")
					}
					return
				}

				manifestPath := filepath.Join(root, "work", "release", "touched-files.md")
				before, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				var stdout, stderr bytes.Buffer
				args := []string{"release"}
				if consumer == "explicit_merge" {
					args = append(args, "p1", "p2")
				}
				code := RunMergeManifest(root, args, &stdout, &stderr)
				after, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				if !tc.valid {
					if code != 3 {
						t.Errorf("invalid sequence merge: code=%d, want blocked (3); stderr=%s", code, stderr.String())
					}
					if !bytes.Equal(before, after) {
						t.Error("invalid sequence merge overwrote the existing release manifest")
					}
					return
				}
				if code != 0 {
					t.Fatalf("valid sequence merge: code=%d stderr=%s", code, stderr.String())
				}
				for _, row := range []string{p1Row, p2Row, releaseRow} {
					if !strings.Contains(string(after), row) {
						t.Errorf("merged release omitted product row %q", row)
					}
				}
				if err := VerifyReleaseUnion(root, "release"); err != nil {
					t.Fatalf("merged valid release failed verification: %v", err)
				}
			})
		}
	}
}
