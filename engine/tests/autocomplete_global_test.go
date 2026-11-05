package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/gate"
	"github.com/devrites/devrites/internal/testutil"
)

func autocompleteCLIFixture(t *testing.T) (string, string, string) {
	t.Helper()
	requireGit(t)
	t.Setenv("DEVRITES_WORKSPACE", "")
	project := t.TempDir()
	root := filepath.Join(project, ".devrites")
	initGitRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, ".gitignore"), ".devrites/\n.scratch/\n")
	for _, slug := range []string{"current16", "future17", "extra"} {
		testutil.WriteFile(t, filepath.Join(project, "src", slug+".go"), "package fixture\n\n// "+slug+" remains unchanged by controller commands.\n")
	}
	gitCommitAll(t, project, "test: native autocomplete product baseline")
	dir := frontierCLIWorkspace(t, root, "current16", "| phase | build |\n| status | running |\n| sequence_position | 1 |\n| sequence_workspaces_remaining | 2 |\n| next_action | /rite-build SLICE-001 |\n", true)
	testutil.WriteFile(t, filepath.Join(dir, "decisions.md"), "# Decisions\n\n| Position | Workspace |\n| --- | --- |\n| 1 | current16 |\n| 2 | future17 |\n| 3 | future18 |\n")
	autocompleteCLIVet(t, root, "current16")
	testutil.WriteFile(t, filepath.Join(root, "ACTIVE"), "current16\n")
	testutil.WriteFile(t, filepath.Join(root, "AFK"), "mode: autocomplete\nmax_parallel: 2\n")
	contract := filepath.Join(t.TempDir(), "writer.contract.md")
	testutil.WriteFile(t, contract, "# Writer contract\n\nBuild selected slice only.\nWriter-safe checks: go test ./src/...\n")
	return project, root, contract
}

func autocompleteCLIVet(t *testing.T, root, slug string) {
	t.Helper()
	binding, err := gate.ReadinessBinding(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, "work", slug, "eng-review.md"), "# Engineering review\n\nREADY\n\n"+binding+"\n")
}

func autocompleteCLISnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	files := frontierCLISnapshot(t, root)
	for path, value := range files {
		if value == "directory" || strings.HasSuffix(path, ".lock") {
			delete(files, path)
		}
	}
	return files
}

func autocompleteCLIWriterArgs(slug, wave, contract string) []string {
	return []string{"dispatch", slug, "open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001", "--contract", contract}
}

func autocompleteCLIProductsUnchanged(t *testing.T, project string) {
	t.Helper()
	for _, slug := range []string{"current16", "future17", "extra"} {
		data, err := os.ReadFile(filepath.Join(project, "src", slug+".go"))
		want := "package fixture\n\n// " + slug + " remains unchanged by controller commands.\n"
		if err != nil || string(data) != want {
			t.Errorf("root controller changed product %s: err=%v content=%q", slug, err, data)
		}
	}
	runGit(t, project, "diff", "--exit-code", "HEAD", "--", "src")
}

func TestAutocompleteGlobalCLIMissingFutureNeedsActualNativePlanning(t *testing.T) {
	project, root, contract := autocompleteCLIFixture(t)
	before := autocompleteCLISnapshot(t, root)
	args := append(autocompleteCLIWriterArgs("current16", "writer16", contract), "--claim-session", "native-controller", "--claim", "src/current16.go")
	out, errOut, code := runDevrites(t, root, args...)
	if code != 3 || !strings.Contains(out+errOut, "future17") || !strings.Contains(out+errOut, "/rite-spec") {
		t.Fatalf("missing successor admitted or remedy missing: code=%d out=%q err=%q", code, out, errOut)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("missing-future refusal wrote claims, dispatch, history or rounds")
	}
	future := frontierCLIWorkspace(t, root, "future17", "| phase | plan |\n| status | running |\n| sequence_parent | current16 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 1 |\n| next_action | /rite-vet future17 |\n", false)
	testutil.WriteFile(t, filepath.Join(future, "spec.md"), "# Spec\n\nNative planning has not launched.\n")
	before = autocompleteCLISnapshot(t, root)
	if out, errOut, code := runDevrites(t, root, args...); code != 3 {
		t.Fatalf("placeholder counted as started preparation: code=%d out=%q err=%q", code, out, errOut)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("placeholder refusal changed workflow")
	}
	for _, command := range [][]string{
		{"dispatch", "future17", "open", "--phase", "plan", "--wave", "prepare17", "--role", "plan-drafter"},
		{"dispatch", "future17", "start", "--wave", "prepare17", "--role", "plan-drafter", "--handle", "native-planner17"},
	} {
		if out, errOut, code := runDevrites(t, root, command...); code != 0 {
			t.Fatalf("actual native planning account %v: code=%d out=%q err=%q", command, code, out, errOut)
		}
	}
	if out, errOut, code := runDevrites(t, root, args...); code != 0 || !strings.Contains(out, "wave writer16 open") || !strings.Contains(out, "wright round 1") {
		t.Fatalf("native planning failed to admit first writer: code=%d out=%q err=%q", code, out, errOut)
	}
	if out, errOut, code := runDevrites(t, root, "dispatch", "current16", "start", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-writer16"); code != 0 {
		t.Fatalf("start writer: code=%d out=%q err=%q", code, out, errOut)
	}
	before = autocompleteCLISnapshot(t, root)
	out, errOut, code = runDevrites(t, root, "check", "continuation", "current16")
	for _, want := range []string{"continuation: required\n", "invoke: /rite-autocomplete\n", "current16 location=live action=wait", "frontier-prepare: /rite-spec 'future18' source=future17"} {
		if code != 0 || !strings.Contains(out, want) {
			t.Errorf("busy source hid independent next preparation; missing %q: code=%d out=%q err=%q", want, code, out, errOut)
		}
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("continuation changed source cursor or planning accounts")
	}
	out, errOut, code = runDevrites(t, root, "dispatch", "current16", "return", "--wave", "writer16", "--role", "slice-wright", "--handle", "native-writer16")
	if code != 0 || !strings.Contains(out, "frontier-prepare: /rite-spec 'future18' source=future17") || strings.Count(out, "frontier-source: current16\n") != 1 {
		t.Fatalf("actual return omitted fresh global scheduling: code=%d out=%q err=%q", code, out, errOut)
	}
	if release, frontier := strings.Index(out, "released claim"), strings.Index(out, "frontier:"); release < 0 || frontier <= release {
		t.Fatalf("return scheduling precedes claim release: %s", out)
	}
	if _, err := os.Stat(filepath.Join(root, "work", "future18")); !os.IsNotExist(err) {
		t.Fatalf("engine created native workspace instead of requesting preparation: %v", err)
	}
	autocompleteCLIProductsUnchanged(t, project)
}

func TestAutocompleteGlobalCLITwoSourcesShareOneIsolatedController(t *testing.T) {
	project, root, contract := autocompleteCLIFixture(t)
	frontierCLIWorkspace(t, root, "future17", "| phase | vet |\n| status | ready |\n| sequence_parent | current16 |\n| sequence_position | 2 |\n| sequence_workspaces_remaining | 0 |\n| next_action | /rite-build SLICE-001 |\n", true)
	before := autocompleteCLISnapshot(t, root)
	out, errOut, code := runDevrites(t, root, autocompleteCLIWriterArgs("current16", "writer", contract)...)
	if code != 3 || !strings.Contains(out+errOut, "future17/SLICE-001") || !strings.Contains(out+errOut, "--workspaces") {
		t.Fatalf("ready foreign source omitted from admission: code=%d out=%q err=%q", code, out, errOut)
	}
	if !reflect.DeepEqual(before, autocompleteCLISnapshot(t, root)) {
		t.Fatal("mixed-cohort refusal wrote workflow accounts")
	}
	out, errOut, code = runDevrites(t, root, "check", "continuation", "current16")
	if code != 0 || !strings.Contains(out, "invoke: /rite-autocomplete\n") || strings.Contains(out, "\ninvoke: /rite-build") || !strings.Contains(out, "frontier-picks: SLICE-001,future17/SLICE-001") {
		t.Fatalf("local Build cursor overrode global picks: code=%d out=%q err=%q", code, out, errOut)
	}
	out, errOut, code = runDevrites(t, root, "parallel", "select", "--root", project, "--slug", "current16", "--from-plan", "--workspaces", "current16,future17", "--cap", "2")
	if code != 0 || out != "select: n_eff=2 mode=parallel slices=SLICE-001,future17/SLICE-001\n" {
		t.Fatalf("mixed selector picks: code=%d out=%q err=%q", code, out, errOut)
	}
	cohort := `[{"id":"SLICE-001","paths":["src/current16.go"]},{"workspace":"future17","id":"SLICE-001","paths":["src/future17.go"]}]`
	out, errOut, code = runDevritesIO(t, root, cohort, nil, "parallel", "create", "--root", project, "--slug", "current16", "--batch", "pipeline", "--base", "HEAD", "--json", "-", "--cap", "2")
	if code != 0 || !strings.Contains(out, "created: 2 worktrees") {
		t.Fatalf("real mixed lease creation: code=%d out=%q err=%q", code, out, errOut)
	}
	leaseOut, leaseErr, leaseCode := runDevrites(t, root, "parallel", "lease-read", "--root", project, "--slug", "current16", "--field", "json")
	var lease struct {
		Capacity int `json:"capacity"`
		Slices   []struct {
			ID           string `json:"id"`
			Workspace    string `json:"workspace"`
			WorktreePath string `json:"worktree_path"`
			WrightStatus string `json:"wright_status"`
		} `json:"slices"`
	}
	if err := json.Unmarshal([]byte(leaseOut), &lease); leaseCode != 0 || err != nil || lease.Capacity != 2 || len(lease.Slices) != 2 {
		t.Fatalf("one controller lease/cap: code=%d err=%v out=%q stderr=%q", leaseCode, err, leaseOut, leaseErr)
	}
	seen := map[string]bool{}
	for _, row := range lease.Slices {
		key := row.ID
		if row.Workspace != "" {
			key = row.Workspace + "/" + row.ID
		}
		seen[key] = true
		if row.WorktreePath == project || !strings.HasPrefix(row.WorktreePath, filepath.Join(project, ".scratch")+string(filepath.Separator)) || row.WrightStatus != "pending" {
			t.Fatalf("writer lacks actual isolated pending worktree: %+v", row)
		}
		if _, err := os.Stat(filepath.Join(row.WorktreePath, ".git")); err != nil {
			t.Fatalf("missing real writer worktree: %v", err)
		}
	}
	if !seen["SLICE-001"] || !seen["future17/SLICE-001"] || lease.Slices[0].WorktreePath == lease.Slices[1].WorktreePath {
		t.Fatalf("mixed source namespace collapsed: %+v", lease)
	}
	for _, slug := range []string{"current16", "future17"} {
		if out, errOut, code := runDevrites(t, root, autocompleteCLIWriterArgs(slug, "writer", contract)...); code != 0 {
			t.Fatalf("reserved cohort source %s cannot open separately: %d %q %q", slug, code, out, errOut)
		}
		if out, errOut, code := runDevrites(t, root, "dispatch", slug, "start", "--wave", "writer", "--role", "slice-wright", "--handle", "native-"+slug); code != 0 {
			t.Fatalf("source %s cannot record native handle: %d %q %q", slug, code, out, errOut)
		}
	}
	for _, tc := range []struct {
		name, payload string
		cap           []string
	}{
		{"full capacity", `[{"workspace":"future17","id":"SLICE-002","paths":["src/extra.go"]}]`, nil},
		{"overlapping path", `[{"workspace":"future17","id":"SLICE-002","paths":["src/current16.go"]}]`, []string{"--cap", "3"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"parallel", "add", "--root", project, "--slug", "current16", "--json", "-"}, tc.cap...)
			if out, errOut, code := runDevritesIO(t, root, tc.payload, nil, args...); code != 3 {
				t.Fatalf("lease guard bypassed: %d %q %q", code, out, errOut)
			}
			if after, errOut, code := runDevrites(t, root, "parallel", "lease-read", "--root", project, "--slug", "current16", "--field", "json"); code != 0 || after != leaseOut {
				t.Fatalf("refused add mutated global lease: %d %q %q", code, after, errOut)
			}
		})
	}
	if out, errOut, code := runDevritesIO(t, root, `[{"id":"SLICE-002","paths":["src/extra.go"]}]`, nil, "parallel", "create", "--root", project, "--slug", "future17", "--batch", "second-controller", "--base", "HEAD", "--json", "-", "--cap", "2"); code != 3 {
		t.Fatalf("second controller escaped repository cap: %d %q %q", code, out, errOut)
	}
	autocompleteCLIProductsUnchanged(t, project)
}
