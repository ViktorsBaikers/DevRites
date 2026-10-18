// Package secretpattern holds the credential patterns shared by the secret
// scanner and the gate transcript writer, so both mask the same shapes.
package secretpattern

import "regexp"

// Pattern is one credential shape with its scanner severity.
type Pattern struct {
	Sev, Kind string
	Re        *regexp.Regexp
}

// Patterns are the credential shapes, most severe first.
var Patterns = []Pattern{
	{"HIGH", "private-key", regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----`)},
	{"HIGH", "aws-secret", regexp.MustCompile(`(?i)aws(.{0,20})?(secret|access).{0,20}[=:]\s*['\"]?[A-Za-z0-9/+=]{30,}`)},
	{"HIGH", "github-token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{30,}`)},
	{"HIGH", "slack-token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{20,}`)},
	{"MEDIUM", "generic-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*['\"][^'\"\n]{16,}['\"]`)},
	{"LOW", "placeholder-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password).{0,20}(example|changeme|placeholder|dummy)`)},
}

// Redact masks every HIGH or MEDIUM match in text and returns the masked text
// and the number of masks. LOW placeholders are not credentials and stay.
func Redact(text string) (string, int) {
	masked := 0
	for _, pattern := range Patterns {
		if pattern.Sev == "LOW" {
			continue
		}
		text = pattern.Re.ReplaceAllStringFunc(text, func(string) string {
			masked++
			return "[REDACTED:" + pattern.Kind + "]"
		})
	}
	return text, masked
}
