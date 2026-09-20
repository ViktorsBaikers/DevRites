package lib

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/testutil"
)

func runDupGit(t *testing.T, project string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", project}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func initDupRepo(t *testing.T, project string) {
	t.Helper()
	runDupGit(t, project, "init", "-q")
	runDupGit(t, project, "config", "user.email", "tests@example.invalid")
	runDupGit(t, project, "config", "user.name", "DevRites Tests")
}

// dupBlockA/B are the same logic under different names: the renaming case a
// verbatim-line diff cannot catch but token normalization can.
const dupBlockA = `func fetchDiscount(customer *Customer) (float64, error) {
	if customer == nil {
		return 0, errors.New("nil customer")
	}
	total, err := lookupOrders(customer.ID)
	if err != nil {
		return 0, fmt.Errorf("orders: %w", err)
	}
	if total > 1000 {
		return 0.15, nil
	}
	if total > 500 {
		return 0.10, nil
	}
	return 0, nil
}
`

const dupBlockB = `func resolveRate(account *Account) (float64, error) {
	if account == nil {
		return 0, errors.New("nil account")
	}
	sum, err := queryInvoices(account.Ref)
	if err != nil {
		return 0, fmt.Errorf("invoices: %w", err)
	}
	if sum > 2000 {
		return 0.20, nil
	}
	if sum > 800 {
		return 0.05, nil
	}
	return 0, nil
}
`

func TestNormalizeDupSourceStripsCommentsAndStrings(t *testing.T) {
	src := "package x // trail\n\n// full-line comment\n/* block\n   still */\nfunc f() string { return \"http://a//b\" } // done\n"
	lines := normalizeDupSource(src, dupCStyle)
	if len(lines) != 7 {
		t.Fatalf("lines=%d", len(lines))
	}
	if lines[2].tokens != "_" || lines[3].tokens != "_" || lines[4].tokens != "_" {
		t.Fatalf("comment lines got tokens: %q %q %q", lines[2].tokens, lines[3].tokens, lines[4].tokens)
	}
	if !strings.Contains(lines[5].tokens, "s") || strings.Contains(lines[5].tokens, "http") {
		t.Fatalf("string literal not collapsed: %q", lines[5].tokens)
	}
	if strings.Contains(lines[5].tokens, "done") {
		t.Fatalf("trailing comment survived: %q", lines[5].tokens)
	}
}

func TestCanonicalizeDupLineCollapsesIdentifiers(t *testing.T) {
	a := canonicalizeDupLine("if total > 1000 {")
	b := canonicalizeDupLine("if sum > 2000 {")
	if a.tokens != b.tokens {
		t.Fatalf("renamed identifiers diverged: %q vs %q", a.tokens, b.tokens)
	}
	if !strings.HasPrefix(a.tokens, "if") {
		t.Fatalf("keyword erased: %q", a.tokens)
	}
	c := canonicalizeDupLine("while total > 1000 {")
	if a.tokens == c.tokens {
		t.Fatal("if and while collapsed to the same skeleton")
	}
	trivial := canonicalizeDupLine("}")
	if !trivial.trivial || trivial.tokens != "_" {
		t.Fatalf("trivial line: %+v", trivial)
	}
}

func TestDupDetectFindsRenamedCopy(t *testing.T) {
	files := []dupFile{
		{path: "a/one.go", lines: normalizeDupSource("package a\n\n"+dupBlockA, dupCStyle)},
		{path: "b/two.go", lines: normalizeDupSource("package b\n\n"+dupBlockB, dupCStyle)},
	}
	clusters, units := dupDetect(files, dupOptions{minLines: 5, threshold: 0.6})
	if len(clusters) != 1 {
		t.Fatalf("clusters=%d want 1", len(clusters))
	}
	if len(clusters[0].units) != 2 {
		t.Fatalf("cluster units=%v", clusters[0].units)
	}
	if len(units) != 2 || clusters[0].hash == "" {
		t.Fatalf("units=%d hash=%q", len(units), clusters[0].hash)
	}
}

func TestDupDetectIgnoresShortOverlap(t *testing.T) {
	shared := "x := 1\ny := 2\nz := 3\nw := 4\n"
	tailA := "if x > 1 {\n\treturn x + y\n}\nfor i := 0; i < 3; i++ {\n\tfmt.Println(i)\n}\nswitch z {\ncase 1:\n\tw++\n}\n"
	tailB := "m := map[string]int{}\nfor k := range m {\n\tdelete(m, k)\n}\nif len(m) == 0 && z < 0 {\n\tpanic(\"empty\")\n}\nreturn\n"
	files := []dupFile{
		{path: "a.go", lines: normalizeDupSource("package a\n"+shared+tailA, dupCStyle)},
		{path: "b.go", lines: normalizeDupSource("package b\n"+shared+tailB, dupCStyle)},
	}
	clusters, _ := dupDetect(files, dupOptions{minLines: 8, threshold: 0.6})
	if len(clusters) != 0 {
		t.Fatalf("short overlap reported: %+v", clusters)
	}
}

func TestDupDistanceBoostPrefersDistantTwins(t *testing.T) {
	seg := dupSegment{aStart: 0, aEnd: 10, bStart: 0, bEnd: 10}
	near := dupDistanceBoost("a/x.go", "a/y.go", seg)
	far := dupDistanceBoost("a/b/c/x.go", "d/e/f/y.go", seg)
	if far <= near {
		t.Fatalf("far=%f near=%f", far, near)
	}
	same := dupDistanceBoost("a/x.go", "a/x.go", dupSegment{aStart: 0, aEnd: 10, bStart: 600, bEnd: 610})
	if same <= 0 {
		t.Fatalf("same-file distant copy got no boost: %f", same)
	}
}

func TestDupClusterHashStableAcrossMoves(t *testing.T) {
	// The same pair of twins with one block shifted down in its file must
	// hash identically: a cluster hash keys on content, not position.
	run := func(aSrc string) string {
		files := []dupFile{
			{path: "a/one.go", lines: normalizeDupSource(aSrc, dupCStyle)},
			{path: "b/two.go", lines: normalizeDupSource("package b\n\n"+dupBlockB, dupCStyle)},
		}
		clusters, _ := dupDetect(files, dupOptions{minLines: 5, threshold: 0.6})
		if len(clusters) != 1 {
			t.Fatalf("clusters=%d", len(clusters))
		}
		return clusters[0].hash
	}
	base := run("package a\n\n" + dupBlockA)
	moved := run("package a\n\n\n// spacer\n\n" + dupBlockA)
	if base != moved {
		t.Fatal("cluster hash shifted when the block moved lines")
	}
	edited := run("package a\n\n" + strings.Replace(dupBlockA, "total > 1000", "total >= 1000", 1))
	if edited == base {
		t.Fatal("cluster hash unchanged after a body edit")
	}
}

func TestDupChangedRangesParsesHunks(t *testing.T) {
	diff := "diff --git a/f.go b/f.go\nindex 1..2 100644\n--- a/f.go\n+++ b/f.go\n" +
		"@@ -10,0 +11,3 @@\n+a\n+b\n+c\n@@ -30 +31 @@\n+x\n" +
		"diff --git a/gone.go b/gone.go\n--- a/gone.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-y\n"
	ranges := dupChangedRanges(diff)
	got := ranges["f.go"]
	if len(got) != 2 || got[0] != [2]int{11, 13} || got[1] != [2]int{31, 31} {
		t.Fatalf("ranges=%v", got)
	}
	if _, ok := ranges["gone.go"]; ok {
		t.Fatal("deleted file recorded a range")
	}
}

func TestDupLoadIgnoreStripsComments(t *testing.T) {
	f := filepath.Join(t.TempDir(), "dup-ignore")
	testutil.WriteFile(t, f, "abcdef123456 # coincidental helpers\n\n  \n# full comment\nbeef00\n")
	got := dupLoadIgnore(f)
	if !got["abcdef123456"] || !got["beef00"] || len(got) != 2 {
		t.Fatalf("ignored=%v", got)
	}
}

func TestRunCheckDupWorktreeMarksChanged(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a/one.go"), "package a\n\n"+dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	// New file duplicating committed code: untracked → whole file changed.
	testutil.WriteFile(t, filepath.Join(project, "b/two.go"), "package b\n\n"+dupBlockB)

	var stdout, stderr bytes.Buffer
	code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--min-lines", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "cluster 1 (") || !strings.Contains(out, "* b/two.go:") {
		t.Fatalf("missing changed-marked cluster:\n%s", out)
	}
	if strings.Contains(out, "* a/one.go:") {
		t.Fatalf("unchanged file marked:\n%s", out)
	}
}

func TestRunCheckDupAllFindsPreexistingDup(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a/one.go"), "package a\n\n"+dupBlockA)
	testutil.WriteFile(t, filepath.Join(project, "b/two.go"), "package b\n\n"+dupBlockB)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	root := filepath.Join(project, ".devrites")

	var stdout, stderr bytes.Buffer
	code := RunCheckDup(root, []string{"--all", "--min-lines", "5"}, &stdout, &stderr)
	if code != 0 || !strings.Contains(stdout.String(), "cluster 1 (") {
		t.Fatalf("code=%d out=%s err=%s", code, stdout.String(), stderr.String())
	}
	// Worktree mode with a clean tree reports nothing attributable, but the
	// message must say the diff is clean, not imply the repo has no duplicates.
	stdout.Reset()
	code = RunCheckDup(root, nil, &stdout, &stderr)
	if code != 0 || strings.Contains(stdout.String(), "cluster 1 (") ||
		!strings.Contains(stdout.String(), "no clusters touch the diff") {
		t.Fatalf("clean worktree misreported: code=%d out=%s", code, stdout.String())
	}
}

func TestRunCheckDupIgnoreFileSuppressesCluster(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a/one.go"), "package a\n\n"+dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	testutil.WriteFile(t, filepath.Join(project, "b/two.go"), "package b\n\n"+dupBlockB)
	root := filepath.Join(project, ".devrites")

	var stdout, stderr bytes.Buffer
	code := RunCheckDup(root, []string{"--all", "--min-lines", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	start := strings.Index(out, "cluster 1 (")
	if start < 0 {
		t.Fatalf("no cluster:\n%s", out)
	}
	hash := out[start+len("cluster 1 (") : start+len("cluster 1 (")+12]
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, "dup-ignore"), hash+" # coincidence\n")

	stdout.Reset()
	code = RunCheckDup(root, []string{"--all", "--min-lines", "5"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ignored") {
		t.Fatalf("ignore file not honored:\n%s", stdout.String())
	}
}

func TestDupScannableSkipsNestedVendoredTrees(t *testing.T) {
	for _, p := range []string{
		"vendor/x/lib.go",
		"services/api/vendor/x/lib.go",
		"a/node_modules/pkg/index.js",
		"pkg/testdata/fixture.go",
		"sub/.devrites/work/f.go",
		"gen/output.pb.go",
		"dist/bundle.min.js",
	} {
		if dupScannable(p, nil) {
			t.Fatalf("%s should be skipped", p)
		}
	}
	for _, p := range []string{
		"vendorless/lib.go", // name contains but is not the dir
		"pkg/vendorx/v.go",
		"engine/main.go",
	} {
		if !dupScannable(p, nil) {
			t.Fatalf("%s should be scannable", p)
		}
	}
	if dupScannable("sub/lib/util.go", []string{"lib/"}) {
		t.Fatal("--exclude lib/ should match at any depth")
	}
}

func TestDupRejectsInvalidArgs(t *testing.T) {
	for _, args := range [][]string{{"--threshold", "NaN"}, {"--base", "--output=escaped"}, {"--all", "--staged"}, {"--limit", "1", "--limit", "2"}, {"--phase"}, {"--cwd", "--all"}, {"--ignore-file"}, {"--exclude", ""}} {
		var out, errOut bytes.Buffer
		root := filepath.Join(t.TempDir(), ".devrites")
		if code := RunCheckDup(root, args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "dup:") || strings.Contains(errOut.String(), "git ") || out.Len() != 0 {
			t.Errorf("args=%v code=%d out=%s err=%s", args, code, &out, &errOut)
		}
		if _, err := os.Stat(filepath.Join(root, "metrics")); !os.IsNotExist(err) {
			t.Fatal("invalid args wrote metrics")
		}
	}
}

func TestDupStagedSnapshot(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	testutil.WriteFile(t, filepath.Join(project, "b.go"), dupBlockB)
	runDupGit(t, project, "add", "b.go")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), "package different")
	if err := os.Remove(filepath.Join(project, "b.go")); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(project, "untracked.go"), dupBlockA)
	var out, errOut bytes.Buffer
	code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--staged", "--min-lines", "5"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "* b.go:") || !strings.Contains(out.String(), "  a.go:") || strings.Contains(out.String(), "untracked.go") {
		t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
	}
}

func TestDupUnavailableAndExcluded(t *testing.T) {
	for _, kind := range []string{"missing", "oversized", "generated", "empty", "partial"} {
		t.Run(kind, func(t *testing.T) {
			project := t.TempDir()
			initDupRepo(t, project)
			if kind != "empty" {
				testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
				runDupGit(t, project, "add", ".")
			}
			switch kind {
			case "missing", "partial":
				if err := os.Remove(filepath.Join(project, "a.go")); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				testutil.WriteFile(t, filepath.Join(project, "a.go"), strings.Repeat("x", dupMaxFileBytes+1))
			case "generated":
				testutil.WriteFile(t, filepath.Join(project, "a.go"), "// Code generated; DO NOT EDIT.")
			}
			if kind == "partial" {
				testutil.WriteFile(t, filepath.Join(project, "b.go"), dupBlockB)
			}
			var out, errOut bytes.Buffer
			code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--all"}, &out, &errOut)
			if kind == "missing" || kind == "oversized" || kind == "partial" {
				if code != 2 || strings.Contains(out.String(), "dup: ok") || !strings.Contains(errOut.String(), "incomplete") {
					t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
				}
			} else if code != 0 || !strings.Contains(out.String(), map[string]string{"empty": "no eligible", "generated": "generated"}[kind]) {
				t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
			}
		})
	}
}

func TestDupBaseFlagCannotWriteFile(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockB)
	target := filepath.Join(project, "escaped")
	var out, errOut bytes.Buffer
	code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--base", "--output=" + target}, &out, &errOut)
	if code != 2 {
		t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("git output target exists: %v", err)
	}
}

func TestDupStagedRejectsUnsafeIndexEntries(t *testing.T) {
	for _, kind := range []string{"symlink", "unmerged", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			project := t.TempDir()
			initDupRepo(t, project)
			testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
			runDupGit(t, project, "add", ".")
			runDupGit(t, project, "commit", "-qm", "base")
			switch kind {
			case "symlink":
				if err := os.Symlink("a.go", filepath.Join(project, "b.go")); err != nil {
					t.Fatal(err)
				}
				runDupGit(t, project, "add", "b.go")
			case "oversized":
				testutil.WriteFile(t, filepath.Join(project, "b.go"), strings.Repeat("x", dupMaxFileBytes+1))
				runDupGit(t, project, "add", "b.go")
			case "unmerged":
				oid, err := exec.Command("git", "-C", project, "rev-parse", "HEAD:a.go").Output()
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command("git", "-C", project, "update-index", "--index-info")
				cmd.Stdin = strings.NewReader("0 0000000000000000000000000000000000000000\ta.go\n100644 " + strings.TrimSpace(string(oid)) + " 1\ta.go\n")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("index: %s %v", out, err)
				}
			}
			var out, errOut bytes.Buffer
			code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--staged"}, &out, &errOut)
			if code != 2 || strings.Contains(out.String(), "dup: ok") || !strings.Contains(errOut.String(), "incomplete") {
				t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
			}
		})
	}
}

func TestDupStagedDeletionAndUntrackedAreNotComparators(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	runDupGit(t, project, "rm", "a.go")
	testutil.WriteFile(t, filepath.Join(project, "b.go"), dupBlockB)
	runDupGit(t, project, "add", "b.go")
	testutil.WriteFile(t, filepath.Join(project, "untracked.go"), dupBlockA)
	var out, errOut bytes.Buffer
	code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--staged", "--min-lines", "5"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "scanned 1 files") || strings.Contains(out.String(), "cluster 1 (") {
		t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
	}
}

func TestDupDoesNotExecuteDiffHelpers(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
	testutil.WriteFile(t, filepath.Join(project, ".gitattributes"), "*.go diff=hostile\n")
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockB)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "config", "diff.external", "missing-external-diff-command")
	runDupGit(t, project, "config", "diff.hostile.textconv", "missing-textconv-command")
	for _, mode := range [][]string{{"--staged"}, {"--worktree"}, {"--base", "HEAD"}} {
		var out, errOut bytes.Buffer
		if code := RunCheckDup(filepath.Join(project, ".devrites"), mode, &out, &errOut); code != 0 {
			t.Fatalf("mode=%v code=%d out=%s err=%s", mode, code, &out, &errOut)
		}
	}
}

func TestDupGitOutputRejectsTruncation(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	if _, err := dupGitOutput(project, nil, 1, "rev-parse", "--show-toplevel"); err == nil {
		t.Fatal("truncated Git output accepted")
	}
}

func TestDupStagedQuotedPathAndDiffConfig(t *testing.T) {
	project := t.TempDir()
	initDupRepo(t, project)
	testutil.WriteFile(t, filepath.Join(project, "a.go"), dupBlockA)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "commit", "-qm", "base")
	name := "space name.go"
	testutil.WriteFile(t, filepath.Join(project, name), dupBlockB)
	runDupGit(t, project, "add", ".")
	runDupGit(t, project, "config", "diff.noprefix", "true")
	runDupGit(t, project, "config", "color.ui", "always")
	var out, errOut bytes.Buffer
	code := RunCheckDup(filepath.Join(project, ".devrites"), []string{"--staged", "--min-lines", "5"}, &out, &errOut)
	if code != 0 || !strings.Contains(out.String(), "* "+name+":") {
		t.Fatalf("code=%d out=%s err=%s", code, &out, &errOut)
	}
}
