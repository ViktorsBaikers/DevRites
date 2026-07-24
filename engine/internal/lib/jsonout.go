package lib

import (
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/reason"
)

// A `--json` run wraps a command's normal output in the stable machine-readable
// agent contract without changing command behavior. The envelope stores stdout
// in data.text, parses stderr into diagnostics, and includes the exit code. See
// docs/engine/agent-contract.md.

// Envelope is the top-level JSON result of a `--json` command run.
type Envelope struct {
	Schema      string        `json:"schema"`
	Command     string        `json:"command"`
	OK          bool          `json:"ok"`
	ExitCode    int           `json:"exitCode"`
	ReasonID    reason.ID     `json:"reason_id,omitempty"`
	Data        *EnvelopeData `json:"data,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
}

const CommandEnvelopeSchemaV1 = "devrites-command/v1"

// EnvelopeData keeps stdout unchanged while structured consumers read ok,
// exitCode, and diagnostics from the envelope.
type EnvelopeData struct {
	Text string `json:"text"`
}

// Diagnostic is one stderr line, classified. Path and Line are populated when the
// emitting command used the `ERROR:<path>: <msg> (line N)` shape (spec-validate,
// ledger validate); otherwise they are empty.
type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"` // error | warning | info
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
}

var (
	diagLineRe   = regexp.MustCompile(`\(line[[:space:]]+(\d+)\)`)
	diagPathRe   = regexp.MustCompile(`^([^:]*(?:/[^:]*)?\.[A-Za-z0-9]+):[[:space:]]+(.*)$`)
	diagCodeWord = regexp.MustCompile(`[^a-z0-9]+`)
)

// NewEnvelope builds the result envelope from a captured command run.
func NewEnvelope(command string, code int, stdout, stderr string) Envelope {
	env := Envelope{Schema: CommandEnvelopeSchemaV1, Command: command, OK: code == 0, ExitCode: code}
	if s := strings.TrimRight(stdout, "\n"); s != "" {
		env.Data = &EnvelopeData{Text: s}
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		env.Diagnostics = append(env.Diagnostics, parseDiagnostic(command, line))
	}
	return env
}

// WithReason adds a rule-owned reason without changing the legacy envelope
// fields. Unknown IDs are ignored rather than written as false provenance.
func (env Envelope) WithReason(id reason.ID) Envelope {
	if reason.Known(id) {
		env.ReasonID = id
	}
	return env
}

// parseDiagnostic classifies one stderr line. It recognizes the ERROR:/WARN:
// prefixes and the `<path>: <message> (line N)` shape the spec/ledger validators
// emit; anything else becomes an info-level diagnostic with the raw text.
func parseDiagnostic(command, line string) Diagnostic {
	d := Diagnostic{Severity: "info", Message: line}
	rest := line
	switch {
	case strings.HasPrefix(line, "ERROR:"):
		d.Severity = "error"
		rest = strings.TrimPrefix(line, "ERROR:")
	case strings.HasPrefix(line, "WARN:"), strings.HasPrefix(line, "WARNING:"):
		d.Severity = "warning"
		rest = strings.TrimPrefix(strings.TrimPrefix(line, "WARNING:"), "WARN:")
	}
	rest = strings.TrimSpace(rest)
	d.Message = rest
	if m := diagPathRe.FindStringSubmatch(rest); m != nil {
		d.Path = m[1]
		d.Message = m[2]
	}
	if m := diagLineRe.FindStringSubmatch(rest); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			d.Line = n
		}
	}
	d.Code = diagnosticCode(command, d.Severity, d.Message)
	return d
}

// diagnosticCode returns a stable `<command>_<severity>` code. Version 1 keeps
// codes broad so the catalog can add more specific entries without breaking
// consumers that match the prefix.
func diagnosticCode(command, severity, message string) string {
	base := diagCodeWord.ReplaceAllString(strings.ToLower(command), "_")
	switch {
	case strings.Contains(message, "marked ADDED") || strings.Contains(message, "marked MODIFIED") || strings.Contains(message, "marked REMOVED"):
		return base + "_delta_mismatch"
	case strings.Contains(message, "SHALL") || strings.Contains(message, "WHEN") || strings.Contains(message, "THEN") || strings.Contains(message, "Scenario"):
		return base + "_grammar"
	}
	return base + "_" + severity
}

// WriteEnvelope marshals an envelope as indented JSON followed by a newline.
func WriteEnvelope(w io.Writer, env Envelope) {
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return
	}
	_, _ = w.Write(append(b, '\n'))
}
