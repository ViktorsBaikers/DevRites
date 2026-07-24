package lib

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/toolpolicy"
)

func TestGitAuthorityOpensAndReusesOneQuestion(t *testing.T) {
	root, slug, work := gitAuthorityWorkspace(t)
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")

	first := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
	second := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
	if first.Allowed || !first.Opened || first.ReasonID != reason.GitAuthorityPending {
		t.Fatalf("first decision = %#v", first)
	}
	if second.Allowed || second.Opened || second.QuestionID != first.QuestionID ||
		second.ReasonID != reason.GitAuthorityPending {
		t.Fatalf("second decision = %#v", second)
	}

	questions := gitAuthorityRead(t, filepath.Join(work, "questions.md"))
	if strings.Count(questions, "schema: "+GitAuthoritySchemaV1) != 1 {
		t.Fatalf("question was not idempotent:\n%s", questions)
	}
	for _, forbidden := range []string{"git reset", "--hard", "HEAD"} {
		if strings.Contains(questions, forbidden) {
			t.Fatalf("questions.md leaked %q:\n%s", forbidden, questions)
		}
	}
	stateText := gitAuthorityRead(t, filepath.Join(work, "state.md"))
	for _, want := range []string{"awaiting_human", "## Awaiting human", "- qid: " + first.QuestionID} {
		if !strings.Contains(stateText, want) {
			t.Fatalf("state missing %q:\n%s", want, stateText)
		}
	}
}

func TestGitAuthorityResolvedExactGrantIsOneShot(t *testing.T) {
	root, slug, work := gitAuthorityWorkspace(t)
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	classified := toolpolicy.ClassifyGitCommand("git clean -fdx")
	reasons := destructiveGitReasons(classified)
	pending := AuthorizeGitOperation(root, slug, classified.Digest, reasons)

	t.Setenv("DEVRITES_NOW", "2026-07-23T10:01:00Z")
	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{pending.QuestionID, GitAuthorityAnswer}, &stdout, &stderr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, stderr.String())
	}
	granted := AuthorizeGitOperation(root, slug, classified.Digest, reasons)
	if !granted.Allowed || granted.ReasonID != reason.GitAuthorityGranted ||
		granted.QuestionID != pending.QuestionID {
		t.Fatalf("grant = %#v", granted)
	}

	ledgerPath := filepath.Join(work, GitAuthorityLedgerFile)
	info, err := os.Stat(ledgerPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("ledger mode = %o, want 600", info.Mode().Perm())
	}
	ledgerText := gitAuthorityRead(t, ledgerPath)
	if strings.Count(ledgerText, GitAuthorityConsumptionSchemaV1) != 1 {
		t.Fatalf("ledger = %q", ledgerText)
	}
	for _, forbidden := range []string{"git clean", "-fdx"} {
		if strings.Contains(ledgerText, forbidden) {
			t.Fatalf("ledger leaked %q: %s", forbidden, ledgerText)
		}
	}

	replay := AuthorizeGitOperation(root, slug, classified.Digest, reasons)
	if replay.Allowed || !replay.Opened || replay.ReasonID != reason.GitAuthorityReplayed ||
		replay.QuestionID == pending.QuestionID {
		t.Fatalf("replay = %#v", replay)
	}
	if got := strings.Count(gitAuthorityRead(t, ledgerPath), GitAuthorityConsumptionSchemaV1); got != 1 {
		t.Fatalf("replay changed ledger entries: %d", got)
	}
}

func TestGitAuthorityWrongDigestNeverGrants(t *testing.T) {
	root, slug, _ := gitAuthorityWorkspace(t)
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	first := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
	second := toolpolicy.ClassifyGitCommand("git clean -fd")
	pending := AuthorizeGitOperation(root, slug, first.Digest, destructiveGitReasons(first))

	t.Setenv("DEVRITES_NOW", "2026-07-23T10:01:00Z")
	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{pending.QuestionID, GitAuthorityAnswer}, &stdout, &stderr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, stderr.String())
	}
	mismatch := AuthorizeGitOperation(root, slug, second.Digest, destructiveGitReasons(second))
	if mismatch.Allowed || !mismatch.Opened || mismatch.QuestionID == pending.QuestionID {
		t.Fatalf("wrong digest decision = %#v", mismatch)
	}
}

func TestGitAuthorityResolveEnforcesExactAnswerContract(t *testing.T) {
	root, slug, work := gitAuthorityWorkspace(t)
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
	pending := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
	before := gitAuthorityRead(t, filepath.Join(work, "questions.md"))

	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{pending.QuestionID, "Approve everything"}, &stdout, &stderr); code != 5 {
		t.Fatalf("wrong answer code=%d stderr=%q", code, stderr.String())
	}
	if after := gitAuthorityRead(t, filepath.Join(work, "questions.md")); after != before {
		t.Fatalf("wrong answer mutated authority question:\n%s", after)
	}

	stderr.Reset()
	if code := Resolve(root, []string{pending.QuestionID, "  " + GitAuthorityAnswer + "  "}, &stdout, &stderr); code != 0 {
		t.Fatalf("trimmed exact answer code=%d stderr=%q", code, stderr.String())
	}
	questions := gitAuthorityRead(t, filepath.Join(work, "questions.md"))
	if !strings.Contains(questions, "answer: "+GitAuthorityAnswer) ||
		strings.Contains(questions, "answer:   "+GitAuthorityAnswer) {
		t.Fatalf("answer was not canonicalized:\n%s", questions)
	}
	stderr.Reset()
	if code := Resolve(root, []string{pending.QuestionID, GitAuthorityAnswer}, &stdout, &stderr); code != 4 {
		t.Fatalf("re-answer code=%d, want immutable-question code 4 (stderr=%q)", code, stderr.String())
	}
}

func TestGitAuthorityExpiryAndRefusal(t *testing.T) {
	t.Run("expired answer opens fresh question", func(t *testing.T) {
		root, slug, _ := gitAuthorityWorkspace(t)
		t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
		classified := toolpolicy.ClassifyGitCommand("git branch -D old")
		pending := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))

		t.Setenv("DEVRITES_NOW", "2026-07-23T10:01:00Z")
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, []string{pending.QuestionID, GitAuthorityAnswer}, &stdout, &stderr); code != 0 {
			t.Fatalf("resolve code=%d stderr=%q", code, stderr.String())
		}
		t.Setenv("DEVRITES_NOW", "2026-07-23T10:16:00Z")
		expired := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		if expired.Allowed || !expired.Opened || expired.ReasonID != reason.GitAuthorityExpired ||
			expired.QuestionID == pending.QuestionID {
			t.Fatalf("expired = %#v", expired)
		}
	})

	t.Run("drop remains refused without question spam", func(t *testing.T) {
		root, slug, work := gitAuthorityWorkspace(t)
		t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
		classified := toolpolicy.ClassifyGitCommand("git tag -d v1")
		pending := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		var stdout, stderr bytes.Buffer
		if code := Resolve(root, []string{"--drop", pending.QuestionID, "No"}, &stdout, &stderr); code != 0 {
			t.Fatalf("drop code=%d stderr=%q", code, stderr.String())
		}
		refused := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		if refused.Allowed || refused.Opened || refused.ReasonID != reason.GitAuthorityRefused {
			t.Fatalf("refused = %#v", refused)
		}
		if got := strings.Count(gitAuthorityRead(t, filepath.Join(work, "questions.md")), GitAuthoritySchemaV1); got != 1 {
			t.Fatalf("refusal created %d questions", got)
		}
		questions := gitAuthorityRead(t, filepath.Join(work, "questions.md"))
		if !strings.Contains(questions, "answer: refused") || strings.Contains(questions, "answer: No") {
			t.Fatalf("drop reason was not privacy-normalized:\n%s", questions)
		}
	})
}

func TestGitAuthorityCorruptionAndUnavailableStateDeny(t *testing.T) {
	t.Run("corrupt ledger", func(t *testing.T) {
		root, slug, work := gitAuthorityWorkspace(t)
		t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
		if err := os.WriteFile(filepath.Join(work, GitAuthorityLedgerFile), []byte("{bad\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
		got := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		if got.Allowed || got.Opened || got.ReasonID != reason.GitAuthorityCorrupt {
			t.Fatalf("corrupt ledger decision = %#v", got)
		}
	})

	t.Run("corrupt typed question", func(t *testing.T) {
		root, slug, work := gitAuthorityWorkspace(t)
		t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
		classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
		_ = AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		qpath := filepath.Join(work, "questions.md")
		data := gitAuthorityRead(t, qpath) + "unexpected: forged\n"
		if err := os.WriteFile(qpath, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
		got := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		if got.Allowed || got.Opened || got.ReasonID != reason.GitAuthorityCorrupt {
			t.Fatalf("corrupt question decision = %#v", got)
		}
	})

	t.Run("no workspace", func(t *testing.T) {
		root := filepath.Join(t.TempDir(), ".devrites")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatal(err)
		}
		classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
		got := AuthorizeGitOperation(root, "missing", classified.Digest, destructiveGitReasons(classified))
		if got.Allowed || got.ReasonID != reason.GitWorkspaceUnavailable {
			t.Fatalf("missing workspace decision = %#v", got)
		}
	})

	t.Run("lock failure leaves question unchanged", func(t *testing.T) {
		root, slug, work := gitAuthorityWorkspace(t)
		before := gitAuthorityRead(t, filepath.Join(work, "questions.md"))
		if err := os.Mkdir(filepath.Join(work, ".lock"), 0o755); err != nil {
			t.Fatal(err)
		}
		classified := toolpolicy.ClassifyGitCommand("git reset --hard HEAD")
		got := AuthorizeGitOperation(root, slug, classified.Digest, destructiveGitReasons(classified))
		if got.Allowed || got.ReasonID != reason.GitAuthorityUnavailable {
			t.Fatalf("lock failure decision = %#v", got)
		}
		if after := gitAuthorityRead(t, filepath.Join(work, "questions.md")); after != before {
			t.Fatalf("lock failure mutated questions:\n%s", after)
		}
	})
}

func TestGitAuthorityConcurrentGrantAllowsExactlyOnce(t *testing.T) {
	root, slug, work := gitAuthorityWorkspace(t)
	t.Setenv("DEVRITES_NOW", "2026-07-23T10:00:00Z")
	classified := toolpolicy.ClassifyGitCommand("git update-ref -d refs/heads/old")
	reasons := destructiveGitReasons(classified)
	pending := AuthorizeGitOperation(root, slug, classified.Digest, reasons)
	var stdout, stderr bytes.Buffer
	if code := Resolve(root, []string{pending.QuestionID, GitAuthorityAnswer}, &stdout, &stderr); code != 0 {
		t.Fatalf("resolve code=%d stderr=%q", code, stderr.String())
	}

	const workers = 12
	start := make(chan struct{})
	results := make(chan GitAuthorityDecision, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- AuthorizeGitOperation(root, slug, classified.Digest, reasons)
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	allowed := 0
	for result := range results {
		if result.Allowed {
			allowed++
		}
	}
	if allowed != 1 {
		t.Fatalf("allowed = %d, want exactly 1", allowed)
	}
	ledgerData := gitAuthorityRead(t, filepath.Join(work, GitAuthorityLedgerFile))
	lines := strings.Fields(ledgerData)
	if len(lines) != 1 {
		t.Fatalf("ledger entries = %d, want 1: %q", len(lines), ledgerData)
	}
	var record gitAuthorityConsumption
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatal(err)
	}
	if record.QuestionID != pending.QuestionID {
		t.Fatalf("consumed qid = %q, want %q", record.QuestionID, pending.QuestionID)
	}
}

func gitAuthorityWorkspace(t *testing.T) (root, slug, work string) {
	t.Helper()
	t.Setenv("DEVRITES_WORKSPACE", "")
	project := t.TempDir()
	root = filepath.Join(project, ".devrites")
	slug = "feature"
	work = filepath.Join(root, "work", slug)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	gitAuthorityWrite(t, filepath.Join(root, "ACTIVE"), slug+"\n", 0o644)
	gitAuthorityWrite(t, filepath.Join(work, "questions.md"), "# Questions\n\nNone.\n", 0o644)
	gitAuthorityWrite(t, filepath.Join(work, "state.md"), `# State

- Phase: build
- Status: running
- Next step: /rite-build

## Log
- 2026-07-23T09:00:00Z build: running
`, 0o644)
	return root, slug, work
}

func destructiveGitReasons(result toolpolicy.Result) []toolpolicy.ReasonID {
	var out []toolpolicy.ReasonID
	for _, finding := range result.Findings {
		if finding.Verdict == toolpolicy.VerdictDestructive {
			out = append(out, finding.ReasonID)
		}
	}
	return out
}

func gitAuthorityWrite(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func gitAuthorityRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
