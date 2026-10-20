package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func runDispatchT(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	code := RunDispatch(root, append([]string{"feat"}, args...), stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestDispatchRoundCapIgnoresSliceZeroPaddingAndPhaseCase(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	ids := []string{"SLICE-001", "SLICE-1", "SLICE-01", "SLICE-0001", "SLICE-001"}
	phases := []string{"build", "Build", "BUILD", "build", "build"}
	for i := range ids {
		wave := "r" + string(rune('1'+i))
		if code, out := runDispatchT(t, root, "open", "--phase", phases[i], "--wave", wave, "--role", "slice-wright", "--slice", ids[i], "--contract", wrightContractPath(t)); code != 0 || !strings.Contains(out, "wright round "+string(rune('1'+i))) {
			t.Fatalf("round %d (%s, %s): code=%d\n%s", i+1, ids[i], phases[i], code, out)
		}
	}
	if code, out := runDispatchT(t, root, "open", "--phase", "Build", "--wave", "r6", "--role", "slice-wright", "--slice", "SLICE-1", "--contract", wrightContractPath(t)); code != 3 || !strings.Contains(out, "BLOCKED") {
		t.Fatalf("round 6 must block however the slice/phase is spelled: code=%d\n%s", code, out)
	}
}

func TestDispatchAbandonRefusesCompleteWave(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	for _, args := range [][]string{
		{"open", "--phase", "review", "--wave", "w", "--role", "a"},
		{"start", "--wave", "w", "--role", "a", "--handle", "h"},
		{"seal", "--wave", "w"},
		{"return", "--wave", "w", "--role", "a"},
	} {
		if code, out := runDispatchT(t, root, args...); code != 0 {
			t.Fatalf("%v: code=%d\n%s", args, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "abandon", "--wave", "w", "--reason", "redo"); code != 3 || !strings.Contains(out, "complete") {
		t.Fatalf("abandon of a complete wave must be refused: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "status", "--wave", "w"); code != 0 || !strings.Contains(out, "wave w complete") {
		t.Fatalf("refused abandon changed the wave: code=%d\n%s", code, out)
	}
}

func TestDispatchAggregateStatusListsAbandonedWithoutFailing(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	runDispatchT(t, root, "open", "--phase", "review", "--wave", "w", "--role", "a")
	runDispatchT(t, root, "abandon", "--wave", "w", "--reason", "host limit")
	if code, out := runDispatchT(t, root, "status"); code != 0 || !strings.Contains(out, "wave w state=abandoned") {
		t.Fatalf("aggregate status after abandon: code=%d\n%s", code, out)
	}
	runDispatchT(t, root, "open", "--phase", "review", "--wave", "w2", "--role", "a")
	if code, _ := runDispatchT(t, root, "status"); code == 0 {
		t.Fatal("aggregate status must still fail while a wave is open")
	}
}

func TestDispatchAbandonedStartedWaveCountsTowardRoundCap(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	for i := 1; i <= 5; i++ {
		wave := "s" + string(rune('0'+i))
		for _, args := range [][]string{
			{"open", "--phase", "build", "--wave", wave, "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)},
			{"start", "--wave", wave, "--role", "slice-wright", "--handle", "h"},
		} {
			if code, out := runDispatchT(t, root, args...); code != 0 {
				t.Fatalf("round %d %v: code=%d\n%s", i, args, code, out)
			}
		}
		// abandon records the wave and exits 3 (handoff required).
		if code, out := runDispatchT(t, root, "abandon", "--wave", wave, "--reason", "redo"); code != 3 || !strings.Contains(out, "abandoned") {
			t.Fatalf("round %d abandon: code=%d\n%s", i, code, out)
		}
	}
	// An abandoned wave whose wright never started does not spend a round.
	runDispatchT(t, root, "open", "--phase", "build", "--wave", "u", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t))
	runDispatchT(t, root, "abandon", "--wave", "u", "--reason", "host limit")
	if code, out := runDispatchT(t, root, "open", "--phase", "build", "--wave", "s6", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", wrightContractPath(t)); code != 3 || !strings.Contains(out, "BLOCKED") {
		t.Fatalf("five started-then-abandoned rounds must hit the cap: code=%d\n%s", code, out)
	}
}

func TestParseReturnHead(t *testing.T) {
	cases := []struct {
		name, role, text, outcome, counts, err string
	}{
		{"findings", "code-reviewer", "Outcome: findings\nCounts: critical=0 important=2 gap=0\nFinding: ...\n", "findings", "critical=0,important=2,gap=0", ""},
		{"lane role", "code-reviewer.backend", "Outcome: no-findings\nCounts: critical=0 important=0\n", "no-findings", "critical=0,important=0", ""},
		{"gap", "test-analyst", "\nOutcome: gap\nCounts: important=1 gap=1\n", "gap", "important=1,gap=1", ""},
		{"plan kinds", "plan-reviewer", "Outcome: findings\nCounts: important=1 contract=1 mechanism=0\n", "findings", "important=1,contract=1,mechanism=0", ""},
		{"simplifier", "simplifier-reviewer", "Outcome: findings\nCounts: suggestion=2 nit=0 fyi=1\n", "findings", "suggestion=2,nit=0,fyi=1", ""},
		{"title line first", "code-reviewer", "Code review (feat) — independent\nOutcome: findings\nCounts: important=1\n", "", "", "line 1"},
		{"no counts line", "code-reviewer", "Outcome: findings\nAccount: admitted\n", "", "", "line 2"},
		{"head only", "code-reviewer", "Outcome: findings", "", "", "no Outcome/Counts head"},
		{"bad outcome", "code-reviewer", "Outcome: pass\nCounts: critical=0\n", "", "", "line 1"},
		{"prose counts", "code-reviewer", "Outcome: findings\nCounts: Important 2\n", "", "", "Counts entry"},
		{"n findings", "code-reviewer", "Outcome: findings\nCounts: 3 findings\n", "", "", "Counts entry"},
		{"semicolon counts", "code-reviewer", "Outcome: findings\nCounts: Critical 0; Important 1; gaps 0.\n", "", "", "Counts entry"},
		{"unknown key", "code-reviewer", "Outcome: findings\nCounts: critical=1 blocker=2\n", "", "", "blocker=2"},
		{"other role key", "simplifier-reviewer", "Outcome: findings\nCounts: critical=1\n", "", "", "critical=1"},
		{"duplicate key", "code-reviewer", "Outcome: findings\nCounts: important=1 important=1\n", "", "", "distinct"},
		{"empty counts", "code-reviewer", "Outcome: no-findings\nCounts:\n", "", "", "line 2"},
		{"findings all zero", "code-reviewer", "Outcome: findings\nCounts: critical=0 gap=2\n", "", "", "no non-gap count"},
		{"no-findings nonzero", "code-reviewer", "Outcome: no-findings\nCounts: nit=1\n", "", "", "no-findings but"},
		{"no-findings gap", "code-reviewer", "Outcome: no-findings\nCounts: gap=1\n", "", "", "no-findings but"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			outcome, counts, err := parseReturnHead(tc.role, tc.text)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("want error %q, got outcome=%q err=%v", tc.err, outcome, err)
				}
				return
			}
			if err != nil || outcome != tc.outcome || counts != tc.counts {
				t.Fatalf("got outcome=%q counts=%q err=%v", outcome, counts, err)
			}
		})
	}
}

func TestFindingKeys(t *testing.T) {
	text := "Outcome: findings\nCounts: critical=1 important=1\n" +
		"Finding: Critical conf 8 | engine/a.go:12 | x | when y → z | breaks AC-003 \"quote\" | fix\n" +
		"Finding: Important conf 7 | plan.md § Slices | x | when y → z | breaks security.md § Input checks \"q\" | fix\n" +
		"Finding: Nit | engine/b.go:3-9 | no breaks cell\n" +
		"Late: Suggestion engine/c.go:4 — unchanged text\n"
	keys, late := findingKeys("code-reviewer", "SLICE-001", text)
	want := []string{
		"code-reviewer|SLICE-001|engine/a.go|AC-003|critical",
		"code-reviewer|SLICE-001|plan.md_§_Slices|security.md_§_Input_checks|important",
		"code-reviewer|SLICE-001|engine/b.go|-|nit",
	}
	if !slices.Equal(keys, want) || late != 1 {
		t.Fatalf("keys=%q late=%d", keys, late)
	}
}

// A seal-blocking packet lint must not refuse the return template's Counts
// line: placeholders name keys, never a conclusion.
func TestLintReviewerPacketAllowsCountsTemplate(t *testing.T) {
	if err := lintReviewerPacket(writePacket(t, "Return shape:\nOutcome: <findings | no-findings | gap>\nCounts: critical=<n> important=<n> gap=<n>\n")); err != nil {
		t.Fatalf("counts template refused: %v", err)
	}
	if err := lintReviewerPacket(writePacket(t, "Counts: critical=<n> important=0\n")); err == nil {
		t.Fatal("a Counts line with a real value must still be refused")
	}
}

func TestDispatchReturnResult(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	roles := []string{"code-reviewer", "test-analyst", "security-auditor", "doubt-reviewer", "plan-drafter"}
	args := []string{"open", "--phase", "build", "--wave", "r1", "--packet", reviewerPacketPath(t)}
	for _, role := range roles {
		args = append(args, "--role", role)
	}
	if code, out := runDispatchT(t, root, args...); code != 0 {
		t.Fatalf("open: code=%d\n%s", code, out)
	}
	for i, role := range roles {
		if code, out := runDispatchT(t, root, "start", "--wave", "r1", "--role", role, "--handle", "h"+string(rune('0'+i))); code != 0 {
			t.Fatalf("start %s: code=%d\n%s", role, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "seal", "--wave", "r1"); code != 0 {
		t.Fatalf("seal: code=%d\n%s", code, out)
	}
	write := func(body string) string {
		path := filepath.Join(t.TempDir(), "return.md")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	findings := "Outcome: findings\nCounts: critical=0 important=1 suggestion=0 nit=0 fyi=0 gap=0\n" +
		"Finding: Important conf 8 | engine/a.go:12 | x | when y → z | breaks AC-003 \"q\" | fix\nLate: Nit engine/b.go:2\n"

	// An explicit --outcome that disagrees with the head is refused and stores nothing.
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "code-reviewer", "--outcome", "no-findings", "--result", write(findings)); code != 3 || !strings.Contains(out, "disagrees") {
		t.Fatalf("disagreeing --outcome: code=%d\n%s", code, out)
	}
	stored := filepath.Join(featureDir, "packets", "r1-code-reviewer.return.md")
	if _, err := os.Stat(stored); !os.IsNotExist(err) {
		t.Fatalf("refused return stored a packet: %v", err)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "code-reviewer", "--outcome", "findings", "--result", write(findings)); code != 0 || !strings.Contains(out, "stored packets/r1-code-reviewer.return.md (outcome=findings counts=") {
		t.Fatalf("return --result: code=%d\n%s", code, out)
	}
	if got, err := os.ReadFile(stored); err != nil || string(got) != findings {
		t.Fatalf("stored packet is not verbatim: %q %v", got, err)
	}

	// A packet that already exists is never overwritten, and the role stays pending.
	clash := filepath.Join(featureDir, "packets", "r1-test-analyst.return.md")
	if err := os.WriteFile(clash, []byte("root copy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "test-analyst", "--result", write("Outcome: no-findings\nCounts: critical=0\n")); code != 3 || !strings.Contains(out, "never overwritten") {
		t.Fatalf("overwrite: code=%d\n%s", code, out)
	}
	if got, _ := os.ReadFile(clash); string(got) != "root copy\n" {
		t.Fatalf("existing packet changed: %q", got)
	}
	if code, out := runDispatchT(t, root, "status", "--wave", "r1"); code != 3 || !strings.Contains(out, "test-analyst=started") {
		t.Fatalf("refused return must leave the role pending: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "test-analyst", "--outcome", "no-findings"); code != 0 {
		t.Fatalf("plain --outcome return still works: code=%d\n%s", code, out)
	}

	// A malformed head records gap, never no-findings, and an explicit
	// --outcome no-findings over it is refused.
	malformed := write("Security audit (feat) — independent\nOutcome: no-findings\nCounts: Critical 0; Important 0\n")
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "security-auditor", "--outcome", "no-findings", "--result", malformed); code != 3 || !strings.Contains(out, "malformed") {
		t.Fatalf("no-findings over a malformed head: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "security-auditor", "--result", malformed); code != 0 || !strings.Contains(out, "WARNING") || !strings.Contains(out, "outcome=gap head=malformed") {
		t.Fatalf("malformed head: code=%d\n%s", code, out)
	}

	// --result is for reviewer roles only.
	if code, out := runDispatchT(t, root, "return", "--wave", "r1", "--role", "plan-drafter", "--result", write(findings)); code != 3 || !strings.Contains(out, "reviewer") {
		t.Fatalf("non-reviewer --result: code=%d\n%s", code, out)
	}

	ledger, err := os.ReadFile(filepath.Join(featureDir, MetricsFile))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"note":"outcome=findings counts=critical=0,important=1,suggestion=0,nit=0,fyi=0,gap=0 slice=- late=1 finding=code-reviewer|-|engine/a.go|AC-003|important"`,
		`"note":"outcome=no-findings"`,
		`"note":"outcome=gap head=malformed slice=- late=0"`,
	} {
		if !strings.Contains(string(ledger), want) {
			t.Fatalf("ledger lacks %s:\n%s", want, ledger)
		}
	}
}

// A reviewer-only build wave may name its slice (uncapped), so its findings
// key on that slice; a --result that already is the stored packet is kept.
func TestDispatchReviewerWaveSlice(t *testing.T) {
	root, featureDir := newWorkspace(t, "build")
	open := func(wave string, extra ...string) (int, string) {
		return runDispatchT(t, root, append([]string{"open", "--phase", "build", "--wave", wave, "--packet", reviewerPacketPath(t), "--role", "code-reviewer"}, extra...)...)
	}
	if code, out := open("m1", "--role", "plan-drafter", "--slice", "SLICE-001"); code != 2 {
		t.Fatalf("--slice on a mixed non-wright wave: code=%d\n%s", code, out)
	}
	if code, out := open("b1", "--slice", "slice-1"); code != 2 || !strings.Contains(out, "not a SLICE-### id") {
		t.Fatalf("bad slice id: code=%d\n%s", code, out)
	}
	for i := 0; i < wrightRoundsPerReview+1; i++ {
		if code, out := open("c"+string(rune('0'+i)), "--slice", "SLICE-001"); code != 0 || strings.Contains(out, "round") {
			t.Fatalf("reviewer wave %d must open uncapped: code=%d\n%s", i, code, out)
		}
	}
	if code, out := runDispatchT(t, root, "start", "--wave", "c0", "--role", "code-reviewer", "--handle", "h0"); code != 0 {
		t.Fatalf("start: code=%d\n%s", code, out)
	}
	if code, out := runDispatchT(t, root, "seal", "--wave", "c0"); code != 0 {
		t.Fatalf("seal: code=%d\n%s", code, out)
	}
	stored := filepath.Join(featureDir, "packets", "c0-code-reviewer.return.md")
	body := "Outcome: findings\nCounts: critical=0 important=1 suggestion=0 nit=0 fyi=0 gap=0\nFinding: Important conf 8 | engine/a.go:12 | x | when y → z | breaks AC-003 \"q\" | fix\n"
	writeFile(t, stored, body)
	if code, out := runDispatchT(t, root, "return", "--wave", "c0", "--role", "code-reviewer", "--result", stored); code != 0 || !strings.Contains(out, "stored packets/c0-code-reviewer.return.md") {
		t.Fatalf("return --result of the stored packet itself: code=%d\n%s", code, out)
	}
	if got, _ := os.ReadFile(stored); string(got) != body {
		t.Fatalf("stored packet changed: %q", got)
	}
	ledger, err := os.ReadFile(filepath.Join(featureDir, MetricsFile))
	if err != nil {
		t.Fatal(err)
	}
	if want := "slice=SLICE-001 late=0 finding=code-reviewer|SLICE-001|engine/a.go|AC-003|important"; !strings.Contains(string(ledger), want) {
		t.Fatalf("ledger lacks %s:\n%s", want, ledger)
	}
}

// Every reviewer agent's Output block leads with the strict head: line 1
// "Outcome:", line 2 "Counts:" naming exactly that role's engine vocabulary.
func TestReviewerTemplatesLeadWithStrictHead(t *testing.T) {
	paths, err := filepath.Glob("../../../pack/.claude/agents/devrites-*.md")
	if err != nil || len(paths) == 0 {
		t.Fatalf("agent templates not found: %v", err)
	}
	countKey := regexp.MustCompile(`^([a-z]+)=<n>$`)
	checked := 0
	for _, path := range paths {
		role := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(path), "devrites-"), ".md")
		if !isReviewerRole(role) {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := strings.ReplaceAll(string(raw), "\r\n", "\n") // Windows checkouts may convert to CRLF
		_, output, ok := strings.Cut(text, "\n## Output\n")
		if !ok {
			t.Fatalf("%s: no ## Output section", path)
		}
		output, _, _ = strings.Cut(output, "\n## ")
		blocks := strings.Split(output, "```")
		if len(blocks) < 3 {
			t.Fatalf("%s: ## Output has no fenced template", path)
		}
		for i := 1; i < len(blocks); i += 2 {
			lines := strings.Split(blocks[i], "\n")
			if len(lines) < 3 || lines[1] != "Outcome: <findings | no-findings | gap>" || !strings.HasPrefix(lines[2], "Counts: ") {
				t.Fatalf("%s: template block %d must open with Outcome: then Counts:, got %q", path, i/2+1, lines[:min(3, len(lines))])
			}
			var keys []string
			for _, field := range strings.Fields(strings.TrimPrefix(lines[2], "Counts: ")) {
				m := countKey.FindStringSubmatch(field)
				if m == nil {
					t.Fatalf("%s: Counts entry %q is not key=<n>", path, field)
				}
				keys = append(keys, m[1])
			}
			if want := countKeys(role); !slices.Equal(keys, want) {
				t.Fatalf("%s: Counts keys %v, engine vocabulary %v", path, keys, want)
			}
		}
		checked++
	}
	if checked < 10 {
		t.Fatalf("checked %d reviewer templates; the glob or role match is broken", checked)
	}
}
