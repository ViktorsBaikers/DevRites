package acceptance

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunGatesReusesGateWhileInputsUnchanged(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	src := filepath.Join(project, "src.txt")
	if err := os.WriteFile(src, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	check := "echo x>>runs.txt && echo INPUT-MARKER"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: INPUT-MARKER\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
	approved := []ApprovedCommand{{Command: check}}
	run := func(reverify bool) GateResult {
		doc := ParseLedger(mustRead(t, ledgerPath))
		return RunGates(context.Background(), project, ledgerPath, doc, approved, reverify, 10*time.Second, nil)[0]
	}

	if r := run(false); !r.Ran || !r.Passed {
		t.Fatalf("first run must execute: %+v", r)
	}
	if r := run(true); r.Ran || !r.Passed || r.Detail != "reused: inputs unchanged" {
		t.Fatalf("reverify must reuse a gate whose inputs are unchanged: %+v", r)
	}
	if err := os.WriteFile(src, []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(false); !r.Ran || !r.Passed {
		t.Fatalf("a changed input must re-execute the gate even under run: %+v", r)
	}
	if runs := strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x"); runs != 2 {
		t.Fatalf("CHECK ran %d times, want 2", runs)
	}
}

func TestGateWithoutInputsKeepsItsEvidence(t *testing.T) {
	gate := &Gate{ID: "G1", Check: "echo ok", Expect: "ok", Checked: true}
	gate.Evidence = FormatEvidence(gate, OutputDigest("ok"), 2, "t")
	if GateState(gate, &Document{Abandoned: map[string]string{}}) != StateMet {
		t.Fatal("a gate without INPUTS must keep its existing evidence binding")
	}
	gate.Inputs = "src.txt"
	if GateState(gate, &Document{Abandoned: map[string]string{}}) != StateStaleUnmet {
		t.Fatal("adding INPUTS changes the definition and must stale old evidence")
	}
}

func TestInputsDigest(t *testing.T) {
	project := t.TempDir()
	for name, body := range map[string]string{"a.go": "a", "dir/b.go": "b", "dir/sub/c.txt": "c", ".git/HEAD": "x"} {
		path := filepath.Join(project, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	whole, err := InputsDigest(project, "a.go dir")
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := InputsDigest(project, "dir   a.go"); again != whole {
		t.Fatal("pattern order and spacing must not change the digest")
	}
	if glob, _ := InputsDigest(project, "*.go dir"); glob != whole {
		t.Fatal("a glob naming the same files must give the same digest")
	}
	if err := os.WriteFile(filepath.Join(project, "dir/sub/c.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if changed, _ := InputsDigest(project, "a.go dir"); changed == whole {
		t.Fatal("a nested content change must change the digest")
	}
	for _, spec := range []string{"missing.go", "/etc/passwd", "../outside", `dir\b.go`} {
		if _, err := InputsDigest(project, spec); err == nil {
			t.Fatalf("spec %q must fail closed", spec)
		}
	}
}
