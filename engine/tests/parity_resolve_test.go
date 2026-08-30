package main_test

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// Golden snapshots for resolve include its changes to questions.md and state.md.
// Each case records stdout and the exit code. Mutating cases also record the
// resulting file bytes. The two nondeterministic timestamp fields (answered_at
// and the Log entry) are masked out before the file content is snapshotted so the
// golden is deterministic across runs.
var (
	reAnsweredAt = regexp.MustCompile(`answered_at: \S+`)
	reLogTS      = regexp.MustCompile(`(?m)^- \S+ build: resolved`)
)

func maskResolveTS(s string) string {
	s = reAnsweredAt.ReplaceAllString(s, "answered_at: TS")
	s = reLogTS.ReplaceAllString(s, "- TS build: resolved")
	return s
}

const resolveQuestions = `## q-1 blocking
status: open
answered_at:
answer:

## q-2 advisory
status: open
answered_at:
answer:
`

const resolveState = `- Status: awaiting_human
- Next step: wait for q-1
- Schema: 3

## Awaiting human
- qid: q-1

## Log
- 2024-01-01 spec: created
`

func TestParityResolve(t *testing.T) {
	// setup writes fixtures under .devrites/work/feat and returns the
	// workspace root.
	setup := func(t *testing.T, questions, state string) (gwork string) {
		t.Helper()
		gwork = t.TempDir()
		writeFile(t, gwork, filepath.Join(".devrites", "work", "feat", "questions.md"), questions)
		writeFile(t, gwork, filepath.Join(".devrites", "work", "feat", "state.md"), state)
		writeFile(t, gwork, filepath.Join(".devrites", "ACTIVE"), "feat\n")
		return gwork
	}
	// run executes resolve in gwork and snapshots stdout and the exit code under the
	// current subtest's golden.
	run := func(t *testing.T, gwork string, args ...string) {
		t.Helper()
		out, code := runArgv(t, gwork, libRootEnv(gwork), "", binPath, append([]string{"state", "resolve"}, args...)...)
		assertGolden(t, out, code)
	}
	// assertFileGolden snapshots the masked content of a mutated file under a
	// per-file golden key (t.Name()+"/"+name).
	assertFileGolden := func(t *testing.T, gwork, name string) {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(gwork, ".devrites", "work", "feat", name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		assertGoldenKey(t, t.Name()+"/"+name, maskResolveTS(string(b)))
	}

	t.Run("answer", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-1", "use JWT with 15m expiry")
		assertFileGolden(t, gwork, "questions.md")
		assertFileGolden(t, gwork, "state.md")
	})

	t.Run("drop", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "--drop", "q-2", "out of scope")
		assertFileGolden(t, gwork, "questions.md")
		assertFileGolden(t, gwork, "state.md")
	})

	t.Run("not-found", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-99", "nope") // exit 3
	})

	t.Run("not-open", func(t *testing.T) {
		answered := `## q-1 blocking
status: answered
answered_at: 2024-01-01T00:00:00Z
answer: prior
`
		gwork := setup(t, answered, resolveState)
		run(t, gwork, "q-1", "again") // exit 4
	})

	t.Run("batch", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		writeFile(t, gwork, "batch.txt", "q-1: answer one\n--drop q-2: reason two\n")
		run(t, gwork, "--batch", "batch.txt")
		assertFileGolden(t, gwork, "questions.md")
	})

	t.Run("batch-unterminated", func(t *testing.T) {
		// No trailing newline: a final line without a terminating newline is not
		// applied (q-2 stays open), so the golden captures the truncated batch.
		gwork := setup(t, resolveQuestions, resolveState)
		writeFile(t, gwork, "batch.txt", "q-1: answer one\n--drop q-2: reason two") // no trailing \n
		run(t, gwork, "--batch", "batch.txt")
		assertFileGolden(t, gwork, "questions.md")
	})

	t.Run("missing-answer", func(t *testing.T) {
		gwork := setup(t, resolveQuestions, resolveState)
		run(t, gwork, "q-1") // no answer text -> exit 5
	})

	t.Run("no-workspace", func(t *testing.T) {
		// A bare tree with no explicit root, ACTIVE, or questions.md exits 2.
		// Pointing DEVRITES_ROOT at the nonexistent .devrites child would instead
		// be an explicit unsafe-root refusal, which is a different contract.
		gwork := t.TempDir()
		out, code := runArgv(t, gwork, libEnv, "", binPath, "state", "resolve", "q-1", "x")
		assertGolden(t, out, code)
	})

}
