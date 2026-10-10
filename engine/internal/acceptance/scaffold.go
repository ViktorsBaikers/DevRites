package acceptance

import (
	"fmt"
	"strings"

	"github.com/devrites/devrites/internal/state"
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

// specACIDs returns the unique AC-### ids in the spec's Acceptance criteria
// section, the same set the acceptance map enforces onto tasks.md/test-plan.md.
func specACIDs(spec []byte) []string {
	return state.ParseAcceptanceMap(spec, nil, nil, false, false).SpecIDs
}
