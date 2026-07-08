package lib

import (
	"os"
	"regexp"
	"strings"
)

// Structured parse of a spec.md into its Requirement blocks, tagged with the
// delta kind (added/modified/removed) and capability of the "## … Requirements
// — capability: <c>" section they sit under. It is the shared reader behind two
// consumers: the spec-validate `--against` ledger cross-check (specvalidate.go)
// and the capability-ledger fold (ledger.go). The grammar linter (lintSpec) is
// deliberately left untouched — delta H2 headers are transparent to it — so this
// parser adds delta-awareness without perturbing the flat-spec lint contract.

// Delta kinds, lower-cased. A requirement outside any delta section has kind "".
const (
	DeltaAdded    = "added"
	DeltaModified = "modified"
	DeltaRemoved  = "removed"
)

var (
	// "## ADDED Requirements", "## Modified Requirements — capability: theming".
	deltaSectionRe = regexp.MustCompile(`(?i)^[[:space:]]*##[[:space:]]+(ADDED|MODIFIED|REMOVED)[[:space:]]+Requirements\b`)
	// The capability tag on a delta section header. Accepts an em dash, en dash,
	// or hyphen separator; the capability token runs to end-of-line, trimmed.
	capabilityTagRe = regexp.MustCompile(`(?i)capability:[[:space:]]*(.+?)[[:space:]]*$`)
	// Any H2/H3 header terminates the preceding requirement block.
	blockBoundaryRe = regexp.MustCompile(`^[[:space:]]*###?[[:space:]]`)
)

// Requirement is one "### Requirement: <name>" block with its delta context.
type Requirement struct {
	Name       string // header text after "Requirement:", trailing space trimmed
	Key        string // strings.ToLower(Name) — the identity used to match across specs
	Kind       string // added | modified | removed | "" (flat, no delta section)
	Capability string // from the section tag; "" when untagged
	HeaderLine int    // 1-based line of the "### Requirement:" header
	Raw        string // the full block text (header through the line before the next ##/### or EOF), one trailing newline
}

// SpecDoc is the parsed view of a spec.md.
type SpecDoc struct {
	Requirements []Requirement
	HasDelta     bool // at least one "## … Requirements" delta section was seen
}

// ParseSpec reads and structures a spec.md. It never fails on grammar problems —
// that is lintSpec's job; ParseSpec only extracts blocks and their delta context.
func ParseSpec(file string) (*SpecDoc, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read all lines first so a requirement's Raw block can span forward to its
	// boundary without a second pass.
	var lines []string
	sc := newLineScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	doc := &SpecDoc{}
	curKind := ""
	curCap := ""
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if m := deltaSectionRe.FindStringSubmatch(line); m != nil {
			doc.HasDelta = true
			curKind = strings.ToLower(m[1])
			curCap = ""
			if c := capabilityTagRe.FindStringSubmatch(line); c != nil {
				curCap = strings.TrimSpace(c[1])
			}
			continue
		}
		if reqHeaderRe.MatchString(line) {
			name := trailWSRe.ReplaceAllString(reqPrefixRe.ReplaceAllString(line, ""), "")
			// Raw block: this header line through the line before the next ##/###
			// header or EOF, so a fold stores a self-contained block.
			end := len(lines)
			for j := i + 1; j < len(lines); j++ {
				if blockBoundaryRe.MatchString(lines[j]) || deltaSectionRe.MatchString(lines[j]) {
					end = j
					break
				}
			}
			raw := strings.Join(lines[i:end], "\n")
			raw = strings.TrimRight(raw, "\n") + "\n"
			doc.Requirements = append(doc.Requirements, Requirement{
				Name:       name,
				Key:        strings.ToLower(name),
				Kind:       curKind,
				Capability: curCap,
				HeaderLine: i + 1,
				Raw:        raw,
			})
		}
	}
	return doc, nil
}

// requirementKeys returns the set of lower-cased requirement names present in a
// spec file — the identity index a cross-spec existence check consults. A missing
// or unreadable file yields an empty set (the caller decides whether that is an
// error), matching how the ledger treats an unseeded capability.
func requirementKeys(file string) map[string]bool {
	keys := map[string]bool{}
	doc, err := ParseSpec(file)
	if err != nil {
		return keys
	}
	for _, r := range doc.Requirements {
		keys[r.Key] = true
	}
	return keys
}
