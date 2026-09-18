package acceptance

import (
	"fmt"
	"regexp"
	"strings"
)

// ScaffoldLedger emits a template ledger: one pending gate per canonical
// AC-### id found in spec.md, or a single placeholder when the spec names
// none. It writes nothing; the caller decides whether the file may be
// created (only when absent).
func ScaffoldLedger(slug string, spec []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Gates: %s\n\n", slug)
	b.WriteString("Acceptance ledger: one gate per required outcome. Runnable gates pair a\n")
	b.WriteString("CHECK (an exact command declared in test-plan.md) with an EXPECT oracle;\n")
	b.WriteString("manual gates carry EVIDENCE only. See devrites-lib standards/gates.md.\n\n")
	ids := specACIDs(spec)
	if len(ids) == 0 {
		b.WriteString("- [ ] G1: <observable outcome>\n")
		b.WriteString("  CHECK: <exact approved command>\n")
		b.WriteString("  EXPECT: <success-only marker>\n")
		b.WriteString("  EVIDENCE: pending\n")
		return b.String()
	}
	for i, id := range ids {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "- [ ] %s: <observable outcome for %s>\n", id, id)
		b.WriteString("  CHECK: <exact approved command>\n")
		b.WriteString("  EXPECT: <success-only marker>\n")
		b.WriteString("  EVIDENCE: pending\n")
	}
	return b.String()
}

var scaffoldACRE = regexp.MustCompile(`\bAC-\d{3}\b`)

// specACIDs returns the unique AC-### ids declared anywhere in the spec; the
// scaffold does not restrict to one section because the gate contract is the
// same set the acceptance map already enforces onto tasks.md/test-plan.md.
func specACIDs(spec []byte) []string {
	found := scaffoldACRE.FindAllString(string(spec), -1)
	seen := map[string]bool{}
	var ids []string
	for _, id := range found {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}
