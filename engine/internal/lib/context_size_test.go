package lib

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

var updateBundleSizes = flag.Bool("update", false, "rewrite testdata/composed-bundle-sizes.json from the current pack")

const bundleSizesGolden = "testdata/composed-bundle-sizes.json"

// composedBundleSize is one skill's composed guidance in bytes, as RunContext
// reports it with no feature workspace. Role is the first workspaceByRole key;
// skills that declare no dispatch role record Floor (always only) instead.
type composedBundleSize struct {
	Root        int    `json:"root"`
	Floor       int    `json:"floor,omitempty"`
	Role        string `json:"role,omitempty"`
	RoleFloor   int    `json:"roleFloor,omitempty"`
	RoleCeiling int    `json:"roleCeiling,omitempty"`
}

var bundleBytesRE = regexp.MustCompile(`\d+ sources, (\d+) bytes`)

// TestComposedBundleSizes ratchets each shipped skill's composed bundle: root
// with every trigger, the role floor (always + auto agents - rootOnly), and
// the role ceiling (every trigger - rootOnly); a skill with no dispatch role
// records its always-only floor instead. The per-file size baseline
// misses manifest edits (a file moved into always, a dropped rootOnly entry);
// this catches them. Rebaseline with: go test ./internal/lib -run TestComposedBundleSizes -update
func TestComposedBundleSizes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("CRLF checkouts change byte counts; the other platforms enforce the baseline")
	}
	skillsRoot, err := filepath.Abs("../../../pack/.claude/skills")
	if err != nil {
		t.Fatal(err)
	}
	paths, err := filepath.Glob(filepath.Join(skillsRoot, "*", "SKILL.md"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no skills under %s: %v", skillsRoot, err)
	}
	root := filepath.Join(t.TempDir(), ".devrites")
	out := filepath.Join(t.TempDir(), "b.md")
	measure := func(skill, role string, triggers []string) int {
		t.Helper()
		args := []string{"--skill", skill, "--skills-root", skillsRoot, "--out", out}
		if role != "" {
			args = append(args, "--role", role)
		}
		if len(triggers) > 0 {
			args = append(args, "--trigger", strings.Join(triggers, ","))
		}
		stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
		if code := RunContext(root, args, stdout, stderr); code != 0 {
			t.Fatalf("context %v: code=%d\n%s%s", args, code, stdout, stderr)
		}
		m := bundleBytesRE.FindStringSubmatch(stdout.String())
		if m == nil {
			t.Fatalf("context %v: no byte count in %q", args, stdout)
		}
		n, _ := strconv.Atoi(m[1])
		return n
	}

	got := map[string]composedBundleSize{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		manifest, ok, err := parseLoadsManifest(string(raw))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if !ok {
			continue
		}
		skill := filepath.Base(filepath.Dir(path))
		triggers := sortedKeys(manifest.Triggers)
		size := composedBundleSize{Root: measure(skill, "", triggers)}
		if roles := sortedKeys(manifest.WorkspaceByRole); len(roles) > 0 {
			size.Role = roles[0]
			size.RoleFloor = measure(skill, size.Role, nil)
			size.RoleCeiling = measure(skill, size.Role, triggers)
		} else {
			size.Floor = measure(skill, "", nil)
		}
		got[skill] = size
	}

	if *updateBundleSizes {
		data, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(bundleSizesGolden, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(bundleSizesGolden)
	if err != nil {
		t.Fatalf("%v (create it with -update)", err)
	}
	var want map[string]composedBundleSize
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("%s: %v", bundleSizesGolden, err)
	}
	for _, skill := range sortedKeys(got) {
		g := got[skill]
		w, ok := want[skill]
		switch {
		case !ok:
			t.Errorf("%s: not in %s; add it with -update", skill, bundleSizesGolden)
		case g.Role != w.Role:
			t.Errorf("%s: measured role %q, baseline has %q; rebaseline with -update", skill, g.Role, w.Role)
		default:
			for _, c := range []struct {
				name      string
				got, want int
			}{{"root", g.Root, w.Root}, {"floor", g.Floor, w.Floor}, {"role floor", g.RoleFloor, w.RoleFloor}, {"role ceiling", g.RoleCeiling, w.RoleCeiling}} {
				if c.got > c.want {
					t.Errorf("%s: %s bundle grew by %d bytes (%d -> %d); shrink the manifest or rebaseline with -update", skill, c.name, c.got-c.want, c.want, c.got)
				}
			}
		}
	}
	for _, skill := range sortedKeys(want) {
		if _, ok := got[skill]; !ok {
			t.Errorf("%s: in %s but has no loads manifest in the pack; remove it with -update", skill, bundleSizesGolden)
		}
	}
}
