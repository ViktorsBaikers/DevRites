package lib

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// The capability ledger — DevRites' living "what the system does now" layer.
// Each capability owns `<root>/specs/<capability>/spec.md`, a flat list of proven
// Requirement blocks. A feature's spec.md carries deltas (ADDED/MODIFIED/REMOVED,
// per standards/spec-grammar.md); on ship, `ledger sync` folds those deltas in.
// The store lives outside `work/`, so close-out's archival never touches it.
//
// The fold is a plain upsert/delete keyed on requirement header identity, and so
// is idempotent: re-syncing the same feature is a no-op. ADDED vs MODIFIED is a
// classification the spec-validate `--against` check enforces at the spec gate —
// the fold itself does not need the distinction, which keeps re-runs safe.
//
// Exit codes: 0 ok · 1 grammar violation (validate) or nothing to do where a
// target was expected · 2 usage.

// ledgerSpecsDir is the ledger root under the .devrites root.
func ledgerSpecsDir(root string) string { return root + "/specs" }

func ledgerCapabilityPath(root, capability string) string {
	return ledgerSpecsDir(root) + "/" + capability + "/spec.md"
}

// Ledger dispatches the `ledger <sub>` command family.
func Ledger(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "sync":
		return ledgerSync(root, args[1:], stdout, stderr, false)
	case "diff":
		return ledgerSync(root, args[1:], stdout, stderr, true)
	case "validate":
		return ledgerValidate(root, stdout, stderr)
	case "list":
		return ledgerList(root, stdout, stderr)
	case "show":
		return ledgerShow(root, args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine ledger <sync|diff|validate|list|show> [args]")
		return 2
	}
}

// capabilityFold is the planned change to one capability's ledger spec.
type capabilityFold struct {
	capability string
	blocks     []Requirement // resolved ordered blocks after the fold
	added      []string
	modified   []string
	removed    []string
}

func (f capabilityFold) touched() bool {
	return len(f.added)+len(f.modified)+len(f.removed) > 0
}

// planFold reads a workspace spec and computes, per capability, the resolved
// ledger blocks and the change summary. dryRun only affects whether the caller
// writes; the plan is identical.
func planFold(root, workspaceDir string, stderr io.Writer) ([]capabilityFold, int) {
	spec := workspaceDir + "/spec.md"
	if !isFile(spec) {
		fmt.Fprintf(stderr, "ledger: no spec.md at %s\n", spec)
		return nil, 5
	}
	doc, err := ParseSpec(spec)
	if err != nil {
		fmt.Fprintf(stderr, "ledger: %v\n", err)
		return nil, 2
	}
	defaultCap := defaultCapability(workspaceDir)

	// Group the feature's requirements by capability, preserving encounter order.
	order := []string{}
	byCap := map[string][]Requirement{}
	for _, r := range doc.Requirements {
		capability := r.Capability
		if capability == "" {
			capability = defaultCap
		}
		if _, seen := byCap[capability]; !seen {
			order = append(order, capability)
		}
		byCap[capability] = append(byCap[capability], r)
	}

	var folds []capabilityFold
	for _, capability := range order {
		fold := foldCapability(root, capability, byCap[capability])
		folds = append(folds, fold)
	}
	return folds, 0
}

// foldCapability applies one capability's deltas onto its existing ledger spec.
// A requirement with no delta kind (a flat feature spec) folds as ADDED. ADDED and
// MODIFIED are both an upsert (replace in place if the header exists, else append);
// REMOVED deletes. Ordering: existing blocks keep their position, new blocks append.
func foldCapability(root, capability string, deltas []Requirement) capabilityFold {
	existing, _ := ParseSpec(ledgerCapabilityPath(root, capability))
	var blocks []Requirement
	idx := map[string]int{}
	if existing != nil {
		for _, r := range existing.Requirements {
			idx[r.Key] = len(blocks)
			blocks = append(blocks, r)
		}
	}

	fold := capabilityFold{capability: capability}
	for _, d := range deltas {
		switch d.Kind {
		case DeltaRemoved:
			if i, ok := idx[d.Key]; ok {
				blocks = append(blocks[:i], blocks[i+1:]...)
				delete(idx, d.Key)
				// Reindex entries after the removed slot.
				for k, v := range idx {
					if v > i {
						idx[k] = v - 1
					}
				}
				fold.removed = append(fold.removed, d.Name)
			}
		default: // added, modified, or "" (flat) → upsert
			if i, ok := idx[d.Key]; ok {
				blocks[i] = d
				fold.modified = append(fold.modified, d.Name)
			} else {
				idx[d.Key] = len(blocks)
				blocks = append(blocks, d)
				fold.added = append(fold.added, d.Name)
			}
		}
	}
	fold.blocks = blocks
	return fold
}

// renderLedgerSpec assembles a capability's ledger spec.md from its resolved blocks.
func renderLedgerSpec(capability string, blocks []Requirement) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Capability: %s\n\n", capability)
	b.WriteString("<!-- Managed by `devrites-engine ledger`. The living record of proven behavior\n")
	b.WriteString("for this capability, folded from feature specs on ship. Change it through a\n")
	b.WriteString("feature's spec deltas (ADDED/MODIFIED/REMOVED), not by hand. -->\n\n")
	for i, r := range blocks {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(strings.TrimRight(r.Raw, "\n"))
		b.WriteString("\n")
	}
	return b.String()
}

func ledgerSync(root string, args []string, stdout, stderr io.Writer, dryRun bool) int {
	workspaceDir := argAt(args, 0)
	if workspaceDir == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine ledger sync|diff <workspace-dir>")
		return 2
	}
	folds, code := planFold(root, workspaceDir, stderr)
	if code != 0 {
		return code
	}

	verb := "sync"
	if dryRun {
		verb = "diff"
	}
	any := false
	for _, f := range folds {
		if !f.touched() {
			continue
		}
		any = true
		fmt.Fprintf(stdout, "ledger %s: capability %q — +%d added, ~%d modified, -%d removed\n",
			verb, f.capability, len(f.added), len(f.modified), len(f.removed))
		for _, n := range f.added {
			fmt.Fprintf(stdout, "  + %s\n", n)
		}
		for _, n := range f.modified {
			fmt.Fprintf(stdout, "  ~ %s\n", n)
		}
		for _, n := range f.removed {
			fmt.Fprintf(stdout, "  - %s\n", n)
		}
		if dryRun {
			continue
		}
		if err := writeLedgerCapability(root, f); err != nil {
			fmt.Fprintf(stderr, "ledger: %v\n", err)
			return 1
		}
	}
	if !any {
		fmt.Fprintf(stdout, "ledger %s: no changes — the ledger already reflects this feature\n", verb)
	}
	return 0
}

func writeLedgerCapability(root string, f capabilityFold) error {
	dir := ledgerSpecsDir(root) + "/" + f.capability
	if len(f.blocks) == 0 {
		// The last requirement was removed — drop the now-empty capability spec.
		_ = os.Remove(ledgerCapabilityPath(root, f.capability))
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create capability dir: %w", err)
	}
	return os.WriteFile(dir+"/spec.md", []byte(renderLedgerSpec(f.capability, f.blocks)), 0o644)
}

func ledgerValidate(root string, stdout, stderr io.Writer) int {
	caps := ledgerCapabilities(root)
	if len(caps) == 0 {
		fmt.Fprintln(stdout, "ledger validate: no capabilities in the ledger yet — nothing to lint")
		return 0
	}
	violations := 0
	for _, capability := range caps {
		spec := ledgerCapabilityPath(root, capability)
		reqs, scens, findings, err := lintSpec(spec)
		if err != nil {
			fmt.Fprintf(stderr, "ledger validate: %s: %v\n", capability, err)
			violations++
			continue
		}
		if len(findings) > 0 {
			for _, f := range findings {
				fmt.Fprintln(stderr, f)
			}
			violations += len(findings)
			continue
		}
		fmt.Fprintf(stdout, "ledger validate: OK — %s: %d requirement(s) / %d scenario(s)\n", capability, reqs, scens)
	}
	if violations > 0 {
		fmt.Fprintf(stderr, "ledger validate: %d grammar error(s) across the ledger (see standards/spec-grammar.md)\n", violations)
		return 1
	}
	return 0
}

func ledgerList(root string, stdout, stderr io.Writer) int {
	caps := ledgerCapabilities(root)
	if len(caps) == 0 {
		fmt.Fprintln(stdout, "ledger: empty — no proven capabilities recorded yet")
		return 0
	}
	for _, capability := range caps {
		doc, err := ParseSpec(ledgerCapabilityPath(root, capability))
		n := 0
		if err == nil {
			n = len(doc.Requirements)
		}
		fmt.Fprintf(stdout, "%-40s %d requirement(s)\n", capability, n)
	}
	return 0
}

func ledgerShow(root string, args []string, stdout, stderr io.Writer) int {
	capability := argAt(args, 0)
	if capability == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine ledger show <capability>")
		return 2
	}
	data, err := os.ReadFile(ledgerCapabilityPath(root, capability))
	if err != nil {
		fmt.Fprintf(stderr, "ledger: no such capability %q in the ledger\n", capability)
		return 1
	}
	_, _ = stdout.Write(data)
	return 0
}

// ledgerCapabilities lists capability names (sorted) that have a spec.md.
func ledgerCapabilities(root string) []string {
	entries, err := os.ReadDir(ledgerSpecsDir(root))
	if err != nil {
		return nil
	}
	var caps []string
	for _, e := range entries {
		if e.IsDir() && isFile(ledgerCapabilityPath(root, e.Name())) {
			caps = append(caps, e.Name())
		}
	}
	sort.Strings(caps)
	return caps
}
