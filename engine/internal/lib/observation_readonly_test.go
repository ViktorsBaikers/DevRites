package lib

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/state"
)

func observationFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	files := map[string]string{}
	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			files[rel+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestRejectedObservationsReadOnly(t *testing.T) {
	for _, observer := range []string{"candidate", "next"} {
		t.Run(observer, func(t *testing.T) {
			root, project := treeWorkspace(t)
			if code, out := runDispatchT(t, root, "open", "--phase", "vet", "--wave", "alpha", "--role", "a"); code != 0 {
				t.Fatalf("baseline: %d %s", code, out)
			}
			writeFile(t, filepath.Join(project, "tracked.go"), "package alpha\n")
			writeFile(t, filepath.Join(root, "work", "feat", "touched-files.md"), candidateTestManifest(candidateTestRow{state: "present", path: "tracked.go", slice: "S-1", reason: "Changed."}))
			if _, _, err := CandidateIdentity(root, "feat"); err != nil {
				t.Fatalf("identity prerequisite: %v", err)
			}
			if err := VerifyReleaseUnion(root, "feat"); err != nil {
				t.Fatalf("union prerequisite: %v", err)
			}
			if err := CheckCandidateCoverage(root, "feat"); err != nil {
				t.Fatalf("coverage prerequisite: %v", err)
			}
			before := observationFiles(t, root)
			for range 2 {
				if observer == "candidate" {
					if err := CheckTreeBaseline(root, "feat"); err == nil || !strings.Contains(err.Error(), "unattributed root edit: tracked.go") {
						t.Fatalf("tripwire: %v", err)
					}
				} else if code, out := runNextT(t, root); code != 3 || !strings.Contains(out, "unattributed root edit: tracked.go") {
					t.Fatalf("tripwire: %d %s", code, out)
				}
			}
			if after := observationFiles(t, root); !reflect.DeepEqual(before, after) {
				t.Fatal("rejected observations changed workspace file bytes or inventory")
			}
		})
	}
}

func TestNextKeepsCurrentArtifactlessWork(t *testing.T) {
	for _, phase := range []state.Phase{state.PhaseBuild, state.PhasePolish} {
		t.Run(string(phase), func(t *testing.T) {
			root, dir := newWorkspace(t, string(phase))
			policy, _ := state.PolicyFor(phase)
			for _, artifact := range policy.RequiredArtifacts {
				if artifact != state.LedgerFile {
					writeFile(t, filepath.Join(dir, string(artifact)), "x\n")
				}
			}
			writeFile(t, filepath.Join(dir, "tasks.md"), "# Tasks\n\n## SLICE-001 Alpha\nStatus: pending\nDependencies: none\nWriter allowlist: src/alpha.go\n")
			var out, err bytes.Buffer
			if code := RunNext(root, []string{"feat"}, &out, &err); code != 0 {
				t.Fatalf("next: %d %s", code, err.String())
			}
			if !strings.Contains(out.String(), "next-phase: "+string(phase)+"\n") {
				t.Fatalf("unfinished current phase skipped:\n%s", out.String())
			}
			if !strings.Contains(out.String(), "current-status: complete\n") {
				t.Fatalf("artifact-only status semantics changed:\n%s", out.String())
			}
		})
	}
}

func TestNextCommandRetainsExplicitSource(t *testing.T) {
	root, dir := newWorkspace(t, "spec")
	alpha := filepath.Join(root, "work", "alpha")
	if err := os.Rename(dir, alpha); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "ACTIVE"), "beta\n")
	writeFile(t, filepath.Join(root, "work", "beta", "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | build |\n| status | running |\n")
	var out, err bytes.Buffer
	if code := RunNext(root, []string{"alpha"}, &out, &err); code != 0 {
		t.Fatalf("next: %d %s", code, err.String())
	}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "next-command: ") {
			fields := strings.Fields(strings.TrimPrefix(line, "next-command: "))
			if len(fields) < 2 || fields[len(fields)-1] != "alpha" {
				t.Fatalf("follow-up loses selected source to ACTIVE beta: %q", line)
			}
			return
		}
	}
	t.Fatal("next command missing")
}

func TestNextBuildSourceWithoutSliceArgument(t *testing.T) {
	root, dir := newWorkspace(t, "build")
	if err := os.Rename(dir, filepath.Join(root, "work", "alpha")); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "ACTIVE"), "beta\n")
	writeFile(t, filepath.Join(root, "work", "beta", "state.md"), "# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | spec |\n| status | running |\n")
	var out, err bytes.Buffer
	if code := RunNext(root, []string{"alpha"}, &out, &err); code != 0 {
		t.Fatalf("next: %d %s", code, err.String())
	}
	if !strings.Contains(out.String(), "source: alpha\n") || !strings.Contains(out.String(), "next-command: /rite-build\n") {
		t.Fatalf("Build needs selected SOURCE, not a fake slice argument:\n%s", out.String())
	}
}
