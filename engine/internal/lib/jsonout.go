package lib

import (
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// The machine-readable agent contract. A `--json` run of an AFK-parsed command
// wraps its normal text output in a stable envelope so an unattended driver reads
// a structured result instead of scraping prose. The command's own logic is
// unchanged: the envelope captures its stdout (as data.text), its stderr (parsed
// into diagnostics), and the exit code. Full contract: docs/engine/agent-contract.md.

// Envelope is the top-level JSON result of a `--json` command run.
type Envelope struct {
	Command     string        `json:"command"`
	OK          bool          `json:"ok"`
	ExitCode    int           `json:"exitCode"`
	Data        *EnvelopeData `json:"data,omitempty"`
	Diagnostics []Diagnostic  `json:"diagnostics,omitempty"`
}

// EnvelopeData carries the command's human-readable stdout verbatim, so nothing is
// lost versus the text form while structured consumers key on ok/exitCode/diagnostics.
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
	env := Envelope{Command: command, OK: code == 0, ExitCode: code}
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

// diagnosticCode assigns a stable, greppable code. It is intentionally coarse in
// v1 (a `<command>_<severity>` slug) so the catalog in agent-contract.md can grow
// specific codes over time without breaking consumers that match on the prefix.
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
