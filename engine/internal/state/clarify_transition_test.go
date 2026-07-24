package state

import (
	"errors"
	"strings"
	"testing"
)

func TestNextClarifyCursorCases(t *testing.T) {
	build := ClarifyCursor{Phase: PhaseBuild, Status: "running", NextAction: "/rite-build"}
	clarify := ClarifyCursor{
		Phase:            PhaseClarify,
		Status:           "running",
		NextAction:       "/rite-clarify",
		ReturnPhase:      PhaseBuild,
		ReturnNextAction: "/rite-build",
		HasReturn:        true,
	}
	cases := []struct {
		name    string
		intent  ClarifyIntent
		current ClarifyCursor
		want    ClarifyCursor
		wantErr bool
	}{
		{"enter", ClarifyEnter, build, clarify, false},
		{"duplicate enter", ClarifyEnter, clarify, clarify, false},
		{"too early enter", ClarifyEnter, ClarifyCursor{Phase: PhaseSpec}, ClarifyCursor{}, true},
		{"missing restore target", ClarifyRestore, ClarifyCursor{Phase: PhaseClarify}, ClarifyCursor{Phase: PhaseClarify}, false},
		{"invalid restore target", ClarifyRestore, ClarifyCursor{Phase: PhaseClarify, ReturnPhase: PhaseSpec, HasReturn: true}, ClarifyCursor{}, true},
		{"restore", ClarifyRestore, clarify, build, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NextClarifyCursor(tc.intent, ClarifyTransitionInput{
				Cursor:                  tc.current,
				ClarifyNextAction:       "/rite-clarify",
				DefaultReturnNextAction: "/rite-build",
			})
			if tc.wantErr {
				var transitionErr *ClarifyTransitionError
				if !errors.As(err, &transitionErr) {
					t.Fatalf("error=%v, want ClarifyTransitionError", err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("NextClarifyCursor()=(%+v, %v), want %+v", got, err, tc.want)
			}
		})
	}
}

func TestClarifyCursorSerializationIsByteIdempotent(t *testing.T) {
	original := []string{
		"# State",
		"",
		"## Cursor",
		"| Key | Value |",
		"| --- | --- |",
		"| phase | build |",
		"| status | running |",
		"| next_action | /rite-build |",
		"",
		"Curated note stays exactly here.",
	}
	current, err := ParseClarifyCursor(original)
	if err != nil {
		t.Fatal(err)
	}
	entered, err := NextClarifyCursor(ClarifyEnter, ClarifyTransitionInput{
		Cursor:                  current,
		ClarifyNextAction:       "/rite-clarify",
		DefaultReturnNextAction: "/rite-build",
	})
	if err != nil {
		t.Fatal(err)
	}
	once := ApplyClarifyCursor(original, entered)
	parsed, err := ParseClarifyCursor(once)
	if err != nil {
		t.Fatal(err)
	}
	twiceState, err := NextClarifyCursor(ClarifyEnter, ClarifyTransitionInput{
		Cursor:                  parsed,
		ClarifyNextAction:       "/rite-clarify",
		DefaultReturnNextAction: "/rite-build",
	})
	if err != nil {
		t.Fatal(err)
	}
	twice := ApplyClarifyCursor(once, twiceState)
	if got, want := strings.Join(twice, "\n"), strings.Join(once, "\n"); got != want {
		t.Fatalf("second serialization changed bytes:\n--- once\n%s\n--- twice\n%s", want, got)
	}
	if !strings.Contains(strings.Join(once, "\n"), "Curated note stays exactly here.") {
		t.Fatal("serializer removed curated Markdown")
	}

	restored, err := NextClarifyCursor(ClarifyRestore, ClarifyTransitionInput{
		Cursor:                  parsed,
		ClarifyNextAction:       "/rite-clarify",
		DefaultReturnNextAction: "/rite-build",
	})
	if err != nil {
		t.Fatal(err)
	}
	restoreOnce := ApplyClarifyCursor(once, restored)
	restoreParsed, err := ParseClarifyCursor(restoreOnce)
	if err != nil {
		t.Fatal(err)
	}
	restoreTwice, err := NextClarifyCursor(ClarifyRestore, ClarifyTransitionInput{Cursor: restoreParsed})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(ApplyClarifyCursor(restoreOnce, restoreTwice), "\n"), strings.Join(restoreOnce, "\n"); got != want {
		t.Fatalf("second restore serialization changed bytes:\n--- once\n%s\n--- twice\n%s", want, got)
	}
}

func TestParseClarifyCursorRejectsInvalidReturnPhase(t *testing.T) {
	_, err := ParseClarifyCursor([]string{
		"| phase | clarify |",
		"| return_phase | not-a-phase |",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid return phase") {
		t.Fatalf("error=%v, want invalid return phase", err)
	}
}
