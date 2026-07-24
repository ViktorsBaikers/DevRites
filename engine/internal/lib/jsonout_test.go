package lib

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/devrites/devrites/internal/reason"
)

func TestNewEnvelopeOK(t *testing.T) {
	env := NewEnvelope("preamble", 0, "phase: build\nstatus: running\n", "")
	if !env.OK || env.ExitCode != 0 {
		t.Fatalf("want ok/0, got ok=%v code=%d", env.OK, env.ExitCode)
	}
	if env.Schema != CommandEnvelopeSchemaV1 {
		t.Fatalf("schema=%q", env.Schema)
	}
	if env.Data == nil || env.Data.Text != "phase: build\nstatus: running" {
		t.Fatalf("data.text not captured verbatim: %+v", env.Data)
	}
	if len(env.Diagnostics) != 0 {
		t.Fatalf("clean run should have no diagnostics, got %v", env.Diagnostics)
	}
}

func TestEnvelopeReasonIsExplicitAndCatalogBound(t *testing.T) {
	env := NewEnvelope("seal", 3, "", "").WithReason(reason.GateSealMissing)
	if env.ReasonID != reason.GateSealMissing {
		t.Fatalf("reason_id=%q", env.ReasonID)
	}
	env = env.WithReason("DRV-UNREGISTERED")
	if env.ReasonID != reason.GateSealMissing {
		t.Fatalf("unknown reason replaced typed reason: %q", env.ReasonID)
	}
}

func TestNewEnvelopeParsesDiagnostics(t *testing.T) {
	stderr := `ERROR:.devrites/work/feat/spec.md: Requirement "X" (line 2) is marked ADDED but already exists in ledger capability "theming"
ERROR:.devrites/work/feat/spec.md: Requirement "Y" (line 9) has no THEN line`
	env := NewEnvelope("spec-validate", 1, "", stderr)
	if env.OK || env.ExitCode != 1 {
		t.Fatalf("want not-ok/1, got ok=%v code=%d", env.OK, env.ExitCode)
	}
	if len(env.Diagnostics) != 2 {
		t.Fatalf("want 2 diagnostics, got %d", len(env.Diagnostics))
	}
	d0 := env.Diagnostics[0]
	if d0.Severity != "error" || d0.Path != ".devrites/work/feat/spec.md" || d0.Line != 2 {
		t.Fatalf("diagnostic 0 misparsed: %+v", d0)
	}
	if d0.Code != "spec_validate_delta_mismatch" {
		t.Fatalf("want delta_mismatch code, got %q", d0.Code)
	}
	if env.Diagnostics[1].Code != "spec_validate_grammar" {
		t.Fatalf("want grammar code for THEN-line finding, got %q", env.Diagnostics[1].Code)
	}
}

func TestWriteEnvelopeIsValidJSON(t *testing.T) {
	var buf bytes.Buffer
	WriteEnvelope(&buf, NewEnvelope("doctor", 3, "skew", "WARN: pack ahead of binary"))
	var back Envelope
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatalf("envelope is not valid JSON: %v", err)
	}
	if back.ExitCode != 3 || back.OK {
		t.Fatalf("round-trip lost fields: %+v", back)
	}
	if len(back.Diagnostics) != 1 || back.Diagnostics[0].Severity != "warning" {
		t.Fatalf("warning diagnostic not classified: %+v", back.Diagnostics)
	}
}
