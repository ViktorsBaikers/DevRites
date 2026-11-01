package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func contextIntegrityFixture(t *testing.T) (string, string, []string, string) {
	t.Helper()
	root, workspace := newWorkspace(t, "build")
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(workspace, "spec.md"), "# spec\nA billing export.\n")
	args := []string{"feat", "--phase", "build", "--skills-root", skills}
	return root, workspace, args, filepath.Join(workspace, "ctx", "build.bundle.md")
}

func contextIntegrityRun(t *testing.T, root string, args []string, want int) string {
	t.Helper()
	var out, stderr bytes.Buffer
	if code := RunContext(root, args, &out, &stderr); code != want {
		t.Fatalf("context code=%d want %d\n%s%s", code, want, &out, &stderr)
	}
	return out.String()
}

func TestContextRepairsCorruptBytesAndPrivateModes(t *testing.T) {
	for _, damage := range []string{"body", "digest", "body mode", "digest mode"} {
		t.Run(damage, func(t *testing.T) {
			root, _, args, out := contextIntegrityFixture(t)
			contextIntegrityRun(t, root, args, 0)
			want, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "body":
				corrupt := bytes.Clone(want)
				corrupt[0] ^= 1 // Same size; leave the valid sidecar unchanged.
				if err := os.WriteFile(out, corrupt, 0o600); err != nil {
					t.Fatal(err)
				}
			case "digest":
				if err := os.WriteFile(out+".sha256", []byte(strings.Repeat("0", 64)+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "body mode":
				if err := os.Chmod(out, 0o644); err != nil {
					t.Fatal(err)
				}
			case "digest mode":
				if err := os.Chmod(out+".sha256", 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if result := contextIntegrityRun(t, root, args, 0); strings.Contains(result, "unchanged") {
				t.Fatal("damaged output certified unchanged")
			}
			got, err := os.ReadFile(out)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("bundle not repaired: %v", err)
			}
			digest, err := os.ReadFile(out + ".sha256")
			if err != nil || string(digest) != fmt.Sprintf("%x\n", sha256.Sum256(want)) {
				t.Fatalf("sidecar not repaired: %q %v", digest, err)
			}
			for _, path := range []string{out, out + ".sha256"} {
				info, err := os.Lstat(path)
				if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode() != 0o600) {
					t.Fatalf("output must be private regular file: %s %v %v", path, info, err)
				}
			}
		})
	}
}

func TestContextValidRepeatPreservesBothFiles(t *testing.T) {
	root, _, args, out := contextIntegrityFixture(t)
	contextIntegrityRun(t, root, args, 0)
	before := make([]os.FileInfo, 2)
	for i, path := range []string{out, out + ".sha256"} {
		var err error
		before[i], err = os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
	}
	contextIntegrityRun(t, root, args, 0)
	for i, path := range []string{out, out + ".sha256"} {
		after, err := os.Stat(path)
		if err != nil || !after.Mode().IsRegular() {
			t.Fatalf("repeat lost regular output %s: %v", path, err)
		}
		if runtime.GOOS == "windows" {
			if os.SameFile(before[i], after) {
				t.Fatalf("repeat reused unverifiable private mode: %s", path)
			}
		} else if !os.SameFile(before[i], after) || !before[i].ModTime().Equal(after.ModTime()) {
			t.Fatalf("valid private repeat rewrote %s", path)
		}
	}
}

func contextFIFO(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("named-pipe filesystem fixture needs Unix")
	}
	command, err := exec.LookPath("mkfifo")
	if err != nil {
		t.Skip("mkfifo unavailable")
	}
	if output, err := exec.Command(command, path).CombinedOutput(); err != nil {
		t.Fatalf("mkfifo: %v %s", err, output)
	}
}

func TestContextRefusesNonregularDestinationsWithoutTouchingTargets(t *testing.T) {
	for _, leaf := range []string{"bundle", "sidecar"} {
		for _, kind := range []string{"directory", "fifo", "symlink"} {
			t.Run(leaf+"/"+kind, func(t *testing.T) {
				root, workspace, args, out := contextIntegrityFixture(t)
				contextIntegrityRun(t, root, args, 0)
				path := out
				if leaf == "sidecar" {
					path += ".sha256"
				}
				original, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(t.TempDir(), "target")
				writeFile(t, target, string(original))
				switch kind {
				case "directory":
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
				case "fifo":
					contextFIFO(t, path)
				case "symlink":
					if err := os.Symlink(target, path); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
					writeFile(t, filepath.Join(workspace, "spec.md"), "# spec\nChanged input forces publication.\n")
				}
				before, err := os.Lstat(path)
				if err != nil {
					t.Fatal(err)
				}
				if result := contextIntegrityRun(t, root, args, 2); strings.Contains(result, "unchanged") {
					t.Fatal("nonregular destination certified unchanged")
				}
				after, err := os.Lstat(path)
				if err != nil || !os.SameFile(before, after) {
					t.Fatalf("unsafe destination changed: %v", err)
				}
				got, err := os.ReadFile(target)
				if err != nil || !bytes.Equal(got, original) {
					t.Fatalf("symlink target changed: %v", err)
				}
			})
		}
	}
}

func TestContextConfinesManagedPathsAndAllowsExplicitExternalOutput(t *testing.T) {
	for _, destination := range []string{"default", "relative", "absolute"} {
		t.Run(destination, func(t *testing.T) {
			root, workspace, args, _ := contextIntegrityFixture(t)
			external := t.TempDir()
			writeFile(t, filepath.Join(external, "bundle.md"), "untouched\n")
			if err := os.Symlink(external, filepath.Join(workspace, "ctx")); err != nil {
				t.Skipf("symlink unavailable: %v", err)
			}
			switch destination {
			case "relative":
				args = append(args, "--out", "ctx/bundle.md")
			case "absolute":
				args = append(args, "--out", filepath.Join(workspace, "ctx", "bundle.md"))
			}
			contextIntegrityRun(t, root, args, 2)
			entries, err := os.ReadDir(external)
			if err != nil || len(entries) != 1 {
				t.Fatalf("managed output escaped: %v %v", entries, err)
			}
			got, err := os.ReadFile(filepath.Join(external, "bundle.md"))
			if err != nil || string(got) != "untouched\n" {
				t.Fatalf("managed parent target changed: %q %v", got, err)
			}
		})
	}
	root, _, args, _ := contextIntegrityFixture(t)
	out := filepath.Join(t.TempDir(), "nested", "external.bundle.md")
	contextIntegrityRun(t, root, append(args, "--out", out), 0)
	if data, err := os.ReadFile(out); err != nil || !bytes.Contains(data, []byte("A billing export.")) {
		t.Fatalf("explicit external output unsupported: %v", err)
	}
	contextIntegrityRun(t, root, append(args, "--out", "../escaped.bundle.md"), 2)
}

func TestContextRejectsUnsafeWorkspaceSources(t *testing.T) {
	for _, source := range []string{"state.md", "spec.md", "tasks.md", "ai-spec.md", "nested/notes.md", "findings.md"} {
		kinds := []string{"symlink", "fifo", "oversize"}
		if source == "nested/notes.md" {
			kinds = append(kinds, "parent symlink")
		}
		for _, kind := range kinds {
			t.Run(source+"/"+kind, func(t *testing.T) {
				root, workspace, args, out := contextIntegrityFixture(t)
				skills := args[len(args)-1]
				declared := `"` + source + `"`
				if source == "spec.md" || source == "ai-spec.md" {
					declared = "" // Trigger/omission evidence must fail even when not rendered.
				}
				writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"), "<!-- loads: {\"always\":[\"devrites-lib/reference/standards/core.md\"],\"workspace\":["+declared+"]} -->\n")
				path := filepath.Join(workspace, source)
				if source == "tasks.md" {
					// Exercise active-slice validation before rendering as well.
					writeFile(t, filepath.Join(workspace, "state.md"), "## Cursor\n| phase | build |\n| active_slice | SLICE-001 |\n")
				}
				if source == "findings.md" {
					writeFile(t, filepath.Join(workspace, "tasks.md"), "## SLICE-001 Billing\n")
					args = append(args, "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck", source)
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				switch kind {
				case "symlink":
					target := filepath.Join(t.TempDir(), "source")
					writeFile(t, target, "untrusted source\n")
					if err := os.Symlink(target, path); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				case "fifo":
					contextFIFO(t, path)
				case "oversize":
					file, err := os.Create(path)
					if err != nil {
						t.Fatal(err)
					}
					err = file.Truncate(workspaceArtifactLimit + 1)
					_ = file.Close()
					if err != nil {
						t.Fatal(err)
					}
				case "parent symlink":
					if err := os.Remove(filepath.Dir(path)); err != nil {
						t.Fatal(err)
					}
					target := t.TempDir()
					writeFile(t, filepath.Join(target, "notes.md"), "untrusted source\n")
					if err := os.Symlink(target, filepath.Dir(path)); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
				}
				contextIntegrityRun(t, root, args, 2)
				if _, err := os.Lstat(out); !os.IsNotExist(err) {
					t.Fatalf("unsafe source published bundle: %v", err)
				}
			})
		}
	}
}

func TestContextRejectsEscapingWorkspaceReadSet(t *testing.T) {
	root, workspace, args, out := contextIntegrityFixture(t)
	skills := args[len(args)-1]
	target := filepath.Join(filepath.Dir(workspace), "outside.md")
	writeFile(t, target, "untouched\n")
	for _, source := range []string{"../outside.md", target} {
		writeFile(t, filepath.Join(skills, "rite-build", "SKILL.md"),
			`<!-- loads: {"always":["devrites-lib/reference/standards/core.md"],"workspace":["`+filepath.ToSlash(source)+`"]} -->`+"\n")
		contextIntegrityRun(t, root, args, 2)
		if _, err := os.Stat(out); !os.IsNotExist(err) {
			t.Fatalf("escaping read-set published output: %v", err)
		}
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "untouched\n" {
		t.Fatalf("escaping read target changed: %q %v", data, err)
	}
}

func TestContextRetainsTasksForScopeBodyAnchorsAndTriggerFacts(t *testing.T) {
	_, workspace := newWorkspace(t, "build")
	writeFile(t, filepath.Join(workspace, "state.md"), "## Cursor\n| phase | build |\n| active_slice | SLICE-001 |\n")
	original := []byte("# Tasks\n\n## SLICE-001 Billing\nGoal: frontend agent handles AC-001\n")
	path := filepath.Join(workspace, "tasks.md")
	writeFile(t, path, string(original))
	sources := contextSources{dir: workspace}
	id, err := activeSliceScope(&sources)
	if err != nil || id != "SLICE-001" {
		t.Fatalf("active slice: %q %v", id, err)
	}
	writeFile(t, path, "# Tasks\n\n## SLICE-002 Replacement\nGoal: replacement body\n")
	facts, err := gatherTriggerFacts("", "slice-wright", &sources)
	if err != nil {
		t.Fatal(err)
	}
	manifest := loadsManifest{Triggers: map[string][]string{"frontend": nil}}
	if !slices.Contains(suggestTriggers(manifest, contextOpts{}, facts), "frontend") || !keepModelSurface("slice-wright", facts) {
		t.Fatal("trigger/omission decisions used replacement tasks")
	}
	data, err := sources.read("tasks.md")
	if err != nil {
		t.Fatal(err)
	}
	block, ok := state.ExtractTaskSlice(data, id)
	if !ok {
		t.Fatal("retained active slice disappeared")
	}
	var scoped, whole strings.Builder
	loaded, byReference := writeSliceEntry(&scoped, workspace, path, "workspace/tasks.md", data, nil, id, block, citedIDs(block), map[string]bool{})
	if !loaded || !byReference || !strings.Contains(scoped.String(), "Goal: frontend agent handles AC-001") ||
		!strings.Contains(scoped.String(), fmt.Sprintf("sha256: %x bytes: %d", sha256.Sum256(original), len(original))) ||
		!strings.Contains(scoped.String(), "- AC-001: L4") || strings.Contains(scoped.String(), "Replacement") {
		t.Fatalf("slice body and anchors diverged from retained tasks:\n%s", &scoped)
	}
	if !writeContextData(&whole, path, "workspace/tasks.md", data, nil, map[string]bool{}, false) ||
		!strings.Contains(whole.String(), "Goal: frontend agent handles AC-001") || strings.Contains(whole.String(), "replacement body") {
		t.Fatalf("whole body diverged from retained tasks:\n%s", &whole)
	}
}

func TestContextPreservesWorkspaceSourceAllowance(t *testing.T) {
	workspace := t.TempDir()
	path := filepath.Join(workspace, "tasks.md")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = file.Truncate(workspaceArtifactLimit)
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	sources := contextSources{dir: workspace}
	data, err := sources.read("tasks.md")
	if err != nil || len(data) != workspaceArtifactLimit {
		t.Fatalf("16 MiB source refused: %d bytes %v", len(data), err)
	}
}
