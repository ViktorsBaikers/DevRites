package lib

import "testing"

func TestTallyOpenQuestionsAcceptsCanonicalUppercaseIDs(t *testing.T) {
	b, v, a, e := tallyOpenQuestions([]byte("## Q-001\nstatus: open\ngate: blocking\n\n## q-2026-07-20-001\nstatus: open\ngate: escalating\n"))
	if b != 1 || v != 0 || a != 0 || e != 1 {
		t.Fatalf("counts=(%d,%d,%d,%d), want (1,0,0,1)", b, v, a, e)
	}
}
