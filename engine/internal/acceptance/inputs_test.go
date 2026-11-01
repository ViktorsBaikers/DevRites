package acceptance

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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

func TestRunGatesRejectsInputsChangedDuringExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell synchronization")
	}
	for _, tc := range []struct {
		name                  string
		shared, readBefore    bool
		removeDuringExecution bool
	}{
		{name: "single before read"},
		{name: "single after read", readBefore: true},
		{name: "shared before read", shared: true},
		{name: "shared after read", shared: true, readBefore: true},
		{name: "shared unreadable", shared: true, readBefore: true, removeDuringExecution: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			src := filepath.Join(project, "src.txt")
			initial, moved := "bad", "good"
			if tc.readBefore {
				initial, moved = "good", "bad"
			}
			if err := os.WriteFile(src, []byte(initial), 0o644); err != nil {
				t.Fatal(err)
			}
			read := `if test "$(cat src.txt)" != good; then exit 1; fi; `
			check := "echo x>>runs.txt; "
			if tc.readBefore {
				check += read
			}
			check += "echo started>started; while ! test -f proceed; do sleep 0.01; done; "
			if !tc.readBefore {
				check += read
			}
			check += "printf 'PROBE_OK\\n'"
			body := "- [ ] G1: probe\n  CHECK: " + check + "\n  EXPECT: PROBE_OK\n"
			if tc.shared {
				body += "  EVIDENCE: pending\n- [ ] G2: input-bound probe\n  CHECK: " + check + "\n  EXPECT: PROBE_OK\n"
			}
			body += "  INPUTS: src.txt\n  EVIDENCE: pending\n"
			writeLedger(t, ledgerPath, body)
			doc := ParseLedger(mustRead(t, ledgerPath))
			approved := []ApprovedCommand{{Command: check}}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan struct{})
			var results []GateResult
			go func() {
				defer close(done)
				results = RunGates(ctx, project, ledgerPath, doc, approved, false, 10*time.Second, nil)
			}()
			t.Cleanup(func() {
				cancel()
				<-done
			})
			deadline := time.NewTimer(5 * time.Second)
			defer deadline.Stop()
			tick := time.NewTicker(10 * time.Millisecond)
			defer tick.Stop()
			for {
				if _, err := os.Stat(filepath.Join(project, "started")); err == nil {
					break
				}
				select {
				case <-done:
					t.Fatalf("CHECK ended before synchronization: %+v", results)
				case <-deadline.C:
					t.Fatal("CHECK did not signal started")
				case <-tick.C:
				}
			}
			if tc.removeDuringExecution {
				if err := os.Remove(src); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(src, []byte(moved), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(project, "proceed"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-deadline.C:
				t.Fatal("CHECK did not finish after release")
			}
			result := results[len(results)-1]
			if !result.Ran || result.Passed || result.Reused || result.WriteErr != nil ||
				result.Outcome.ExitCode != 0 || !strings.Contains(result.Detail, "INPUTS") {
				t.Fatalf("successful CHECK with moved inputs must not certify: %+v", results)
			}
			if tc.shared && !results[0].Passed {
				t.Fatalf("unbound gate must retain its own result: %+v", results)
			}
			doc = ParseLedger(mustRead(t, ledgerPath))
			gate := doc.Gates[len(doc.Gates)-1]
			if gate.Checked || gate.Evidence != "pending" {
				t.Fatalf("moved inputs left reusable evidence: %+v", gate)
			}
			if err := os.WriteFile(src, []byte("bad"), 0o644); err != nil {
				t.Fatal(err)
			}
			results = RunGates(context.Background(), project, ledgerPath, doc, approved, true, 10*time.Second, nil)
			result = results[len(results)-1]
			if !result.Ran || result.Passed || result.Reused {
				t.Fatalf("untested bytes reused a false pass: %+v", results)
			}
			if runs := strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x"); runs != 2 {
				t.Fatalf("shared CHECK ran %d times, want 2", runs)
			}
		})
	}
}

func TestRunGatesRejectsInputsMovedAfterSharedExecution(t *testing.T) {
	for _, remove := range []bool{false, true} {
		name := "changed"
		if remove {
			name = "unreadable"
		}
		t.Run(name, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			src := filepath.Join(project, "src.txt")
			if err := os.WriteFile(src, []byte("good"), 0o644); err != nil {
				t.Fatal(err)
			}
			check := "echo x>>runs.txt && echo PROBE_OK"
			writeLedger(t, ledgerPath,
				"- [ ] G1: unbound\n  CHECK: "+check+"\n  EXPECT: PROBE_OK\n  EVIDENCE: pending\n"+
					"- [ ] G2: bound\n  CHECK: "+check+"\n  EXPECT: PROBE_OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
			writes := 0
			writeLock := func(fn func() error) error {
				writes++
				err := fn()
				if err == nil && writes == 1 {
					if remove {
						err = os.Remove(src)
					} else {
						err = os.WriteFile(src, []byte("bad"), 0o644)
					}
				}
				return err
			}
			approved := []ApprovedCommand{{Command: check}}
			results := RunGates(context.Background(), project, ledgerPath, ParseLedger(mustRead(t, ledgerPath)), approved, false, 10*time.Second, &RunOptions{WriteLock: writeLock})
			if len(results) != 2 || !results[0].Passed || results[1].Passed || !results[1].Ran || results[1].WriteErr != nil {
				t.Fatalf("later consumer certified moved inputs: %+v", results)
			}
			if runs := strings.Count(mustRead(t, filepath.Join(project, "runs.txt")), "x"); runs != 1 {
				t.Fatalf("shared CHECK ran %d times, want 1", runs)
			}
			doc := ParseLedger(mustRead(t, ledgerPath))
			if doc.Gates[1].Checked || doc.Gates[1].Evidence != "pending" {
				t.Fatalf("moved inputs left reusable evidence: %+v", doc.Gates[1])
			}
			if err := os.WriteFile(src, []byte("good"), 0o644); err != nil {
				t.Fatal(err)
			}
			results = RunGates(context.Background(), project, ledgerPath, doc, approved, false, 10*time.Second, nil)
			if !results[1].Ran || !results[1].Passed || results[1].Reused {
				t.Fatalf("restored inputs must execute, not reuse: %+v", results)
			}
		})
	}
}

func TestRunGatesRejectsInputsUnavailableBeforeExecution(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	check := "echo good>src.txt && echo PROBE_OK"
	writeLedger(t, ledgerPath, "- [ ] G1: generated input\n  CHECK: "+check+"\n  EXPECT: PROBE_OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
	approved := []ApprovedCommand{{Command: check}}
	results := RunGates(context.Background(), project, ledgerPath, ParseLedger(mustRead(t, ledgerPath)), approved, false, 10*time.Second, nil)
	if !results[0].Ran || results[0].Passed || results[0].WriteErr != nil || !strings.Contains(results[0].Detail, "INPUTS") {
		t.Fatalf("input created during CHECK must not certify: %+v", results)
	}
	doc := ParseLedger(mustRead(t, ledgerPath))
	if doc.Gates[0].Checked || doc.Gates[0].Evidence != "pending" {
		t.Fatalf("unavailable input left reusable evidence: %+v", doc.Gates[0])
	}
}

func TestRunGatesRejectsInputsMovedBeforeWriteback(t *testing.T) {
	project, ledgerPath := writeWorkspace(t)
	src := filepath.Join(project, "src.txt")
	if err := os.WriteFile(src, []byte("good"), 0o644); err != nil {
		t.Fatal(err)
	}
	check := "echo PROBE_OK"
	writeLedger(t, ledgerPath, "- [ ] G1: probe\n  CHECK: "+check+"\n  EXPECT: PROBE_OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n")
	writeLock := func(fn func() error) error {
		if err := os.WriteFile(src, []byte("bad"), 0o644); err != nil {
			return err
		}
		return fn()
	}
	results := RunGates(context.Background(), project, ledgerPath, ParseLedger(mustRead(t, ledgerPath)),
		[]ApprovedCommand{{Command: check}}, false, 10*time.Second, &RunOptions{WriteLock: writeLock})
	if !results[0].Ran || results[0].Passed || results[0].WriteErr != nil || !strings.Contains(results[0].Detail, "INPUTS") {
		t.Fatalf("input moved before locked writeback must not certify: %+v", results)
	}
	doc := ParseLedger(mustRead(t, ledgerPath))
	if doc.Gates[0].Checked || doc.Gates[0].Evidence != "pending" {
		t.Fatalf("writeback certified moved input: %+v", doc.Gates[0])
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

func TestRunGatesInvalidatesEarlierPassWhenLaterCheckMovesInputs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX shell conditionals")
	}
	for _, reused := range []bool{false, true} {
		name := "executed"
		if reused {
			name = "reused shared check"
		}
		t.Run(name, func(t *testing.T) {
			project, ledgerPath := writeWorkspace(t)
			src := filepath.Join(project, "src.txt")
			if err := os.WriteFile(src, []byte("original"), 0o644); err != nil {
				t.Fatal(err)
			}
			mutate := "if test -f mutate.flag; then printf changed > src.txt; fi; printf PROBE_OK"
			first := "printf PROBE_OK"
			if reused {
				first = mutate
			}
			body := "- [ ] G1: input probe\n  CHECK: " + first + "\n  EXPECT: PROBE_OK\n  INPUTS: src.txt\n  EVIDENCE: pending\n"
			writeLedger(t, ledgerPath, body)
			approved := []ApprovedCommand{{Command: first}, {Command: mutate}}
			run := func() []GateResult {
				return RunGates(context.Background(), project, ledgerPath, ParseLedger(mustRead(t, ledgerPath)), approved, false, 10*time.Second, nil)
			}
			if reused {
				if result := run()[0]; !result.Passed {
					t.Fatalf("initial pass: %+v", result)
				}
			}
			writeLedger(t, ledgerPath, mustRead(t, ledgerPath)+"\n- [ ] G2: later check\n  CHECK: "+mutate+"\n  EXPECT: PROBE_OK\n  EVIDENCE: pending\n")
			if err := os.WriteFile(filepath.Join(project, "mutate.flag"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			results := run()
			if results[0].Passed || !results[1].Passed {
				t.Fatalf("earlier pass certified moved inputs: %+v", results)
			}
			doc := ParseLedger(mustRead(t, ledgerPath))
			if GateState(doc.Gates[0], doc) == StateMet {
				t.Fatal("moved inputs retained met evidence")
			}
		})
	}
}
