package lib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// PacketUsage is the help and usage-error text for `packet`.
const PacketUsage = `usage: devrites-engine packet <slug> --wave <w> --phase <p> --role <r> [--slice <SLICE-ID>] [--recheck] [--checks <G-ID,...>] [--focus <file>]

Renders one role's by-reference dispatch packet and runs the lints dispatch
open runs on it; a refused packet is removed and the command exits 3. A
wright role (needs --slice and --checks) writes a same-worktree
packets/<w>.contract.md with the slice's Writer allowlist and each --checks
gates.md gate's CHECK, which must be an approved test-plan.md Build-entry
preflight row; a native-worktree pilot keeps its own contract. A reviewer
role (*-reviewer, *-auditor, test-analyst) writes packets/<w>.<role>.md with
its Outcome/Counts return head. Read first names the role's context bundle
(run context first; --recheck names the build --recheck bundle) and
packets/<w>.diff when candidate freeze wrote one; Candidate copies
packets/<w>.sha256 verbatim, or only its header rows when the per-file rows
would overflow the packet (required at review and seal). --focus (relative
to the workspace) holds one "- <ID> <path:line> <text>" row per line, text up
to 240 characters, and any other non-blank line is refused: open findings
for a reviewer, defects and quoted carry-forward constraints for a wright,
and the 1-3 row claim a doubt-reviewer attacks (required for that role).
`

// focusRowRE is the only open-finding shape a packet carries: an ID, an
// anchor, and a bounded defect statement, never narrative or outcomes.
var focusRowRE = regexp.MustCompile(`^- [A-Za-z][A-Za-z0-9._-]{0,63} [^\s:]+(?::[0-9]+)+(?:-[0-9]+)? \S.{0,239}$`)

// RunPacket implements `packet`: it renders the fixed skeleton so the root
// never hand-writes packet prose, hashes, or IDs.
func RunPacket(root string, args []string, stdout, stderr io.Writer) int {
	var wave, phase, role, slice, checks, focus, recheck string
	var positional []string
	for i := 0; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--recheck":
			// Non-empty selects the .recheck bundle name; the text is the context hint.
			recheck = " --recheck <inventory>"
			continue
		case "--wave":
			wave = value
		case "--phase":
			phase = value
		case "--role":
			role = value
		case "--slice":
			slice = value
		case "--checks":
			checks = value
		case "--focus":
			focus = value
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "packet: unknown flag %q\n", args[i])
				return 2
			}
			positional = append(positional, args[i])
			continue
		}
		i++
	}
	wright := strings.Contains(role, "wright")
	doubt := roleBase(role) == "doubt-reviewer"
	if len(positional) != 1 || !packetNameRE.MatchString(wave) || !packetNameRE.MatchString(phase) || !packetNameRE.MatchString(role) ||
		(slice != "" && !packetNameRE.MatchString(slice)) || (!wright && !isReviewerRole(role)) || (wright && slice == "") || (!wright && checks != "") ||
		(recheck != "" && (phase != "build" || slice == "")) || (doubt && focus == "") {
		fmt.Fprint(stderr, PacketUsage)
		return 2
	}
	slug := positional[0]
	fail := func(err error) int {
		fmt.Fprintf(stderr, "packet: %v\n", err)
		return 3
	}
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return fail(err)
	}
	rel := workspaceRel(root, workspace)
	bundle := "ctx/" + bundleName(contextOpts{phase: phase, role: role, slice: slice, recheck: recheck})
	if info, err := os.Stat(filepath.Join(workspace, filepath.FromSlash(bundle))); err != nil || !info.Mode().IsRegular() {
		scope := recheck
		if slice != "" {
			scope = " --slice " + slice + recheck
		}
		return fail(fmt.Errorf("%s/%s is missing; run devrites-engine context %s --phase %s --role %s%s first", rel, bundle, slug, phase, role, scope))
	}
	candidate, partial, err := frozenCandidate(workspace, wave)
	if err != nil {
		return fail(err)
	}
	finalPhase := strings.EqualFold(phase, "review") || strings.EqualFold(phase, "seal")
	if candidate == "" && !wright && finalPhase {
		return fail(fmt.Errorf("a %s reviewer packet needs a frozen candidate; run devrites-engine candidate freeze %s --wave %s first", phase, slug, wave))
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Packet %s: %s\n\nFeature: %s\nPhase: %s\n", wave, role, slug, phase)
	if slice != "" {
		fmt.Fprintf(&b, "Slice: %s\n", slice)
	}
	if wright {
		b.WriteString("Isolation: same-worktree\n")
		b.WriteString("Authority: sole writer for this slice. Edit only the Allowlist paths; never edit .devrites/**, never commit or push, and ask the user nothing.\n")
	} else {
		b.WriteString("Authority: read-only reviewer. Edit nothing and ask the user nothing; judge the candidate against the bundle sources.\n")
	}
	fmt.Fprintf(&b, "\nRead first:\n- %s/%s\n", rel, bundle)
	if candidate != "" {
		fmt.Fprintf(&b, "- %s/packets/%s.diff\n", rel, wave)
	}
	if wright {
		raw, err := readWorkspaceArtifact(root, slug, "tasks.md")
		if err != nil {
			return fail(err)
		}
		writes, _ := sliceWritePaths(raw, ParseTaskGraph(raw))
		if len(writes[slice]) == 0 {
			return fail(fmt.Errorf("slice %s names no Writer allowlist or Files likely touched in tasks.md", slice))
		}
		b.WriteString("\nAllowlist:\n")
		for _, p := range writes[slice] {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		if checks != "" {
			lines, err := writerSafeChecks(root, slug, slice, checks)
			if err != nil {
				return fail(err)
			}
			b.WriteString("\n" + lines)
		}
	}
	if candidate != "" {
		source := "verbatim from"
		if partial {
			source = "header rows verbatim; per-file rows in"
		}
		fmt.Fprintf(&b, "\nCandidate (%s %s/packets/%s.sha256):\n```text\n%s```\n", source, rel, wave, candidate)
	}
	if focus != "" {
		rows, err := focusRows(workspace, focus)
		if err != nil {
			return fail(err)
		}
		heading := "Open findings:"
		switch {
		case wright:
			heading = "Focus (defects to fix, carry-forward constraints to honor):"
		case doubt:
			heading = "Claim to attack:"
			if rows == "" || strings.Count(rows, "\n") > 3 {
				return fail(fmt.Errorf("--focus %s must hold the 1-3 row claim the doubt-reviewer attacks", focus))
			}
		}
		if rows != "" {
			b.WriteString("\n" + heading + "\n" + rows)
		}
	}
	if wright {
		b.WriteString("\nReturn: every field of your role's return block, with each Writer-safe check's command and exit status.\n")
	} else {
		b.WriteString("\nReturn (lines 1-2 exactly, then your role's rows):\nOutcome: <findings|no-findings|gap>\nCounts:")
		for _, key := range countKeys(role) {
			fmt.Fprintf(&b, " %s=<n>", key)
		}
		b.WriteString("\n")
	}

	name := wave + "." + role + ".md"
	if wright {
		name = wave + ".contract.md"
	}
	out := filepath.Join(workspace, "packets", name)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return fail(err)
	}
	// Lint a sibling draft so a refused re-render keeps the packet already on disk.
	draft := filepath.Join(workspace, "packets", "."+name+".draft")
	if err := state.AtomicWrite(draft, []byte(b.String()), 0o600); err != nil {
		return fail(err)
	}
	if err := lintRenderedPacket(root, slug, draft, role, wright, finalPhase); err != nil {
		_ = os.Remove(draft)
		return fail(fmt.Errorf("BLOCKED: %w", err))
	}
	if err := os.Rename(draft, out); err != nil {
		_ = os.Remove(draft)
		return fail(err)
	}
	fmt.Fprintf(stdout, "packet: %s/packets/%s (%s, %d bytes)\n", rel, name, role, b.Len())
	return 0
}

// lintRenderedPacket runs the lints dispatch open runs for the role; a wright
// contract also gets the gates.md `RUN: writer` precheck lint.
func lintRenderedPacket(root, slug, path, role string, wright, finalPhase bool) error {
	if wright {
		return lintWrightContractFor(root, slug, path)
	}
	if err := lintReviewerWavePacket(path, []string{role}); err != nil {
		return err
	}
	if finalPhase {
		return lintReviewCandidate(root, slug, path)
	}
	return nil
}

// frozenCandidate returns packets/<wave>.sha256 when candidate freeze wrote
// both it and a non-empty diff, or "" when the wave has no frozen candidate.
// partial reports that only the header rows were kept: the per-file rows of a
// large candidate would push the packet past the dispatch lints' read bound.
func frozenCandidate(workspace, wave string) (sums string, partial bool, err error) {
	raw, err := readBoundedRegularFile(filepath.Join(workspace, "packets", wave+".sha256"), workspaceArtifactLimit)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("packets/%s.sha256: %w", wave, err)
	}
	if info, err := os.Stat(filepath.Join(workspace, "packets", wave+".diff")); err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return "", false, fmt.Errorf("packets/%s.sha256 has no non-empty packets/%s.diff beside it; rerun candidate freeze under a new wave", wave, wave)
	}
	if len(raw) <= maxContractBytes/2 {
		return string(raw), false, nil
	}
	var head strings.Builder
	for _, line := range strings.SplitAfter(string(raw), "\n") {
		if !strings.HasPrefix(line, "file ") {
			head.WriteString(line)
		}
	}
	return head.String(), true, nil
}

// writerSafeChecks resolves each named gates.md gate to its CHECK, refusing an
// unknown gate or a CHECK no approved test-plan.md preflight row vets, and adds
// `gates precheck --slice <slice>` when the ledger marks a gate `RUN: writer`.
func writerSafeChecks(root, slug, slice, ids string) (string, error) {
	ledger, err := readWorkspaceArtifact(root, slug, acceptance.GatesFile)
	if err != nil {
		return "", fmt.Errorf("--checks: %w", err)
	}
	plan, err := readWorkspaceArtifact(root, slug, "test-plan.md")
	if err != nil {
		return "", fmt.Errorf("--checks: %w", err)
	}
	doc := acceptance.ParseLedger(string(ledger))
	approved := acceptance.ApprovedCommands(plan)
	var names []string
	var rows strings.Builder
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		var gate *acceptance.Gate
		for _, g := range doc.Gates {
			if g.ID == id {
				gate = g
				break
			}
		}
		switch {
		case gate == nil:
			return "", fmt.Errorf("--checks: gate %q is not in %s", id, acceptance.GatesFile)
		case gate.Check == "":
			return "", fmt.Errorf("--checks: gate %s has no CHECK", id)
		case !acceptance.Approved(gate.Check, gate.Cwd, approved):
			return "", fmt.Errorf("--checks: gate %s CHECK is not an approved test-plan.md Build-entry preflight row", id)
		}
		names = append(names, id)
		cwd := gate.Cwd
		if cwd == "" {
			cwd = "repository root"
		}
		fmt.Fprintf(&rows, "- %s (cwd %s): `%s`\n", id, cwd, gate.Check)
	}
	if len(doc.WriterGates()) > 0 {
		fmt.Fprintf(&rows, "- gates precheck (cwd repository root): `devrites-engine gates precheck %s --worktree . --slice %s`\n", slug, slice)
	}
	return "Writer-safe checks: " + strings.Join(names, ", ") + "\n" + rows.String(), nil
}

// focusRows reads the --focus file (relative to the workspace) and returns its
// open-finding rows, refusing any non-blank line of another shape.
func focusRows(workspace, focus string) (string, error) {
	path := focus
	if !filepath.IsAbs(path) {
		path = filepath.Join(workspace, filepath.FromSlash(focus))
	}
	raw, err := readBoundedRegularFile(path, maxContractBytes)
	if err != nil {
		return "", fmt.Errorf("--focus %s: %w", focus, err)
	}
	var rows strings.Builder
	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !focusRowRE.MatchString(line) {
			return "", fmt.Errorf("--focus %s:%d is not an open-finding row \"- <finding-ID> <path:line> <text up to 240 characters>\"; a packet carries open findings only, never narrative or outcomes", focus, i+1)
		}
		rows.WriteString(line + "\n")
	}
	return rows.String(), nil
}
