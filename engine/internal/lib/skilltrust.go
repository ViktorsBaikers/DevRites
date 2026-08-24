package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// SkillTrustFinding is one deterministic trust scan result.
type SkillTrustFinding struct {
	Severity string // HIGH or MEDIUM
	Rule     string
	Line     int
	Detail   string
}

// SkillTrustResult summarizes a skill-trust scan.
type SkillTrustResult struct {
	Path     string
	Findings []SkillTrustFinding
}

var skillTrustRules = []struct {
	severity string
	rule     string
	re       *regexp.Regexp
	detail   string
}{
	{
		severity: "HIGH",
		rule:     "prompt_injection_disregard",
		re:       regexp.MustCompile(`(?i)\b(disregard|ignore)\b.{0,40}\b(prior|previous|above|system|instruction|rule|constraint|safety)\b`),
		detail:   "instruction attempts to override prior constraints",
	},
	{
		severity: "HIGH",
		rule:     "credential_exfil_curl_env",
		re:       regexp.MustCompile(`(?i)\bcurl\b.{0,120}\$(?:ENV|HOME|[A-Z_][A-Z0-9_]*)`),
		detail:   "curl command references environment variables (possible exfiltration)",
	},
	{
		severity: "HIGH",
		rule:     "sensitive_path_reference",
		re:       regexp.MustCompile(`(?i)(~/.ssh|~/.aws|/etc/passwd|id_rsa|\.pem\b|api[_-]?key\s*[:=])`),
		detail:   "references sensitive filesystem or credential material",
	},
	{
		severity: "MEDIUM",
		rule:     "hidden_remote_fetch",
		re:       regexp.MustCompile(`(?i)\b(curl|wget|fetch)\b.{0,80}\b(http|https)://`),
		detail:   "network fetch instruction in skill prose",
	},
}

// ScanSkillTrust reads one Markdown skill/agent file and returns structural trust findings.
func ScanSkillTrust(path string) (SkillTrustResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return SkillTrustResult{}, err
	}
	if info.IsDir() {
		return SkillTrustResult{}, fmt.Errorf("skill-trust: %q is a directory", path)
	}
	if info.Size() > 1<<20 {
		return SkillTrustResult{}, fmt.Errorf("skill-trust: %q exceeds 1 MiB limit", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return SkillTrustResult{}, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, 1<<20+1))
	if err != nil {
		return SkillTrustResult{}, err
	}
	if len(raw) > 1<<20 {
		return SkillTrustResult{}, fmt.Errorf("skill-trust: %q exceeds 1 MiB limit", path)
	}
	return SkillTrustResult{Path: filepath.Clean(path), Findings: analyzeSkillTrustContent(string(raw))}, nil
}

func analyzeSkillTrustContent(text string) []SkillTrustFinding {
	var findings []SkillTrustFinding
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineNum := i + 1
		for _, r := range line {
			if isSuspiciousUnicode(r) {
				findings = append(findings, SkillTrustFinding{
					Severity: "HIGH",
					Rule:     "suspicious_unicode",
					Line:     lineNum,
					Detail:   fmt.Sprintf("suspicious Unicode U+%04X", r),
				})
				break
			}
		}
		for _, rule := range skillTrustRules {
			if rule.re.MatchString(line) {
				findings = append(findings, SkillTrustFinding{
					Severity: rule.severity,
					Rule:     rule.rule,
					Line:     lineNum,
					Detail:   rule.detail,
				})
			}
		}
	}
	findings = appendMissingNameFinding(text, findings)
	return dedupeFindings(findings)
}

func looksLikeSkillFile(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "user-invocable:") || strings.Contains(lower, "# /rite-") || strings.Contains(lower, "disable-model-invocation:")
}

func appendMissingNameFinding(text string, findings []SkillTrustFinding) []SkillTrustFinding {
	if strings.Contains(text, "---") && !strings.Contains(text, "\nname:") && looksLikeSkillFile(text) {
		findings = append(findings, SkillTrustFinding{
			Severity: "MEDIUM",
			Rule:     "missing_name_frontmatter",
			Line:     1,
			Detail:   "skill-like Markdown lacks name frontmatter",
		})
	}
	return findings
}

func isSuspiciousUnicode(r rune) bool {
	switch r {
	case '\u200B', '\u200C', '\u200D', '\uFEFF', '\u2060':
		return true
	}
	if r >= '\u202A' && r <= '\u202E' {
		return true
	}
	if r >= '\u2066' && r <= '\u2069' {
		return true
	}
	return false
}

func dedupeFindings(in []SkillTrustFinding) []SkillTrustFinding {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]SkillTrustFinding, 0, len(in))
	for _, finding := range in {
		key := finding.Rule + "|" + fmt.Sprint(finding.Line) + "|" + finding.Detail
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, finding)
	}
	return out
}

// SkillTrustBlocks returns true when findings contain any HIGH severity item.
func SkillTrustBlocks(findings []SkillTrustFinding) bool {
	for _, finding := range findings {
		if finding.Severity == "HIGH" {
			return true
		}
	}
	return false
}

// FormatSkillTrust writes a human-readable report.
func FormatSkillTrust(result SkillTrustResult) string {
	if len(result.Findings) == 0 {
		return fmt.Sprintf("skill-trust: PASS (%s)\n", result.Path)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "skill-trust: FINDINGS (%s)\n", result.Path)
	for _, finding := range result.Findings {
		fmt.Fprintf(&b, "%s %s line=%d %s\n", finding.Severity, finding.Rule, finding.Line, finding.Detail)
	}
	return b.String()
}

// ContainsOnlyASCII returns false when non-ASCII control or bidi characters appear.
func ContainsOnlyASCII(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII && isSuspiciousUnicode(r) {
			return false
		}
	}
	return true
}
