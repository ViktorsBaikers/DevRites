package lib

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/state"
)

// loadsManifest is the machine-readable read-set a skill declares in an HTML
// comment: <!-- loads: {...} -->. Paths under "always"/"triggers" are relative
// to the skills root (the directory holding rite-*/ and devrites-lib/); paths
// under "workspace" are relative to the feature workspace. "workspaceByRole"
// optionally narrows the workspace read-set for a named dispatch role: when the
// role key is present its list replaces "workspace" entirely (an empty list
// means the role needs no workspace artifacts). "agents" maps a --role name to
// an agents-root-relative path when it deviates from the devrites-<role>
// convention. "rootOnly" lists always/trigger paths that only the controlling
// root executes; a --role bundle names them instead of inlining them.
type loadsManifest struct {
	Always          []string            `json:"always"`
	Triggers        map[string][]string `json:"triggers"`
	Workspace       []string            `json:"workspace"`
	WorkspaceByRole map[string][]string `json:"workspaceByRole"`
	Agents          map[string]string   `json:"agents"`
	RootOnly        []string            `json:"rootOnly"`
}

const loadsMarker = "<!-- loads:"

const contextUsage = `usage: devrites-engine context [slug] (--phase <p> | --skill <name>) [--role <r>] [--slice <SLICE-###>] [--recheck <inventory>] [--trigger a,b] [--inline-contract] [--skills-root <dir>] [--out <path>]

Emits one deduplicated context bundle for a phase or standalone skill
(+ optional dispatch role) into .devrites/work/<slug>/ctx/ so an agent
reads a single file. The bundle opens with a table of contents giving
each section's line range. A --role bundle leads with the role contract; when it
resolves under .claude/agents, .omp/agents, or .codex/agents (the host
loads it as the agent's system prompt) a one-line stub names its path,
size, and SHA-256 instead, unless --inline-contract is given.
With --skill, slug may be omitted; then --out is
required and no workspace section is included. --slice scopes the
workspace section to one slice: tasks.md contributes that slice's block,
and an artifact over its line/byte budget or over 16 KiB (spec.md excepted)
becomes a by-reference index (path, SHA-256, headings, and line hits for
every ID the block cites).
Repeat --slice for a recheck that spans several slices. In phase build a
--role bundle other than slice-wright requires --slice (exit 3).
--recheck <inventory> (phase build, with --role and --slice) builds a
bounded recheck bundle: the inventory file (relative to the workspace)
and core.md are inlined; every other pack file becomes a by-reference
index plus the sections the inventory cites as "<file>.md § <heading>".
`

// coreStandard is the universal floor every bundle inlines whole.
const coreStandard = "devrites-lib/reference/standards/core.md"

// RunContext emits one deduplicated context bundle for a phase (+ optional
// dispatch role) so an agent reads a single file instead of selecting files.
//
//	context <slug> --phase <p> [--role <r>] [--slice <SLICE-###>] [--recheck <inventory>] [--trigger a,b] [--skills-root <dir>] [--out <path>]
func RunContext(root string, args []string, stdout, stderr io.Writer) int {
	opts, code := parseContextArgs(args, stderr)
	if code != 0 {
		return code
	}
	// A Build reviewer judges one slice; an unscoped bundle inlines whole
	// plan/tasks/architecture into every dispatch.
	if opts.phase == "build" && opts.role != "" && opts.role != "slice-wright" && opts.slice == "" {
		fmt.Fprintf(stderr, "context: BLOCKED: unscoped build reviewer bundle: --role %s in phase build needs --slice <SLICE-ID>\n", opts.role)
		return 3
	}
	var featureDir string
	if opts.slug != "" {
		var err error
		featureDir, err = devritespaths.ExistingFeatureDirChecked(root, opts.slug)
		if err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 2
		}
	}
	skillsRoot, err := resolveSkillsRoot(root, opts.skillsRoot)
	if err != nil {
		fmt.Fprintf(stderr, "context: %v\n", err)
		return 2
	}
	var inventory []byte
	recheckPath := opts.recheck
	if recheckPath != "" {
		if !filepath.IsAbs(recheckPath) {
			recheckPath = filepath.Join(featureDir, filepath.FromSlash(recheckPath))
		}
		inventory, err = os.ReadFile(recheckPath) // #nosec G304 -- the root names its own recheck inventory file
		if err == nil && len(bytes.TrimSpace(inventory)) == 0 {
			err = fmt.Errorf("%s is empty", opts.recheck)
		}
		if err != nil {
			fmt.Fprintf(stderr, "context: --recheck: %v\n", err)
			return 2
		}
	}
	skillDir := "rite-" + opts.phase
	if opts.skill != "" {
		skillDir = opts.skill
	}
	skillPath := filepath.Join(skillsRoot, skillDir, "SKILL.md")
	raw, err := os.ReadFile(skillPath) // #nosec G304 -- skillPath is resolved under the validated skills root
	if err != nil {
		fmt.Fprintf(stderr, "context: BLOCKED: no skill %q at %s\n", skillDir, skillPath)
		return 3
	}
	manifest, hasManifest, err := parseLoadsManifest(string(raw))
	if err != nil {
		fmt.Fprintf(stderr, "context: %v\n", err)
		return 2
	}

	var sliceBlock string
	var sliceIDs []citedID
	if opts.slice != "" {
		raw, err := os.ReadFile(filepath.Join(featureDir, "tasks.md")) // #nosec G304 -- featureDir is the validated feature workspace
		if err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 2
		}
		var blocks []string
		for _, id := range opts.slices {
			block, ok := state.ExtractTaskSlice(raw, id)
			if !ok {
				fmt.Fprintf(stdout, "context: BLOCKED: %s not found in tasks.md\n", id)
				return 3
			}
			blocks = append(blocks, block)
		}
		sliceBlock = strings.Join(blocks, "\n")
		// A recheck inventory may cite IDs outside the slice block (dependency or
		// regression closure); index them too.
		sliceIDs = citedIDs(sliceBlock + "\n" + string(inventory))
	}

	var files []string
	var missingRequired []string
	var autoTriggers []string
	if hasManifest {
		for _, name := range opts.triggers {
			if _, ok := manifest.Triggers[name]; !ok {
				fmt.Fprintf(stderr, "context: unknown trigger %q for %s (declared: %s)\n",
					name, skillDir, strings.Join(sortedKeys(manifest.Triggers), ","))
				return 2
			}
		}
		files = append(files, manifest.Always...)
		for _, name := range opts.triggers {
			files = append(files, manifest.Triggers[name]...)
		}
		// A --role call means a dispatch is happening now: the conventional
		// "agents" trigger (the dispatch contract) auto-fires so a role
		// packet can never omit it.
		if opts.role != "" && !slices.Contains(opts.triggers, "agents") {
			if extra, ok := manifest.Triggers["agents"]; ok {
				files = append(files, extra...)
				autoTriggers = append(autoTriggers, "agents")
			}
		}
	} else {
		// No manifest: bundle the universal floor only. Agents still get a
		// correct (if incomplete) packet and a loud marker.
		files = []string{filepath.FromSlash(coreStandard)}
		fmt.Fprintf(stderr, "context: WARNING: %s declares no loads: manifest; bundle is core-only\n", skillDir)
	}
	if !hasManifest {
		files = append(files, filepath.Join(skillDir, "SKILL.md"))
	}

	var bundle strings.Builder
	fmt.Fprintf(&bundle, "<!-- devrites context bundle: slug=%s phase=%s role=%s triggers=%s auto=%s -->\n",
		opts.slug, opts.phase, orDash(opts.role), orDash(strings.Join(opts.triggers, ",")),
		orDash(strings.Join(autoTriggers, ",")))
	fmt.Fprintf(&bundle, "<!-- generated by devrites-engine context; do not edit. Read this file instead of the listed sources. -->\n\n")

	seen := map[string]bool{}
	loaded := 0
	// The role contract leads the bundle so the agent reads its rules before
	// reference text.
	if opts.role != "" {
		rel := roleAgentPath(manifest, opts.role)
		abs, label := filepath.Join(skillsRoot, "..", "agents", filepath.FromSlash(rel)), "agents/"+rel
		// Claude Code, omp, and Codex load a named agent's body as its system
		// prompt, so those layouts get a stub, not a second copy. Other
		// layouts (pi, devin, unknown) are unverified and keep the body.
		native := slices.Contains([]string{".claude", ".omp"}, filepath.Base(filepath.Dir(filepath.Clean(skillsRoot))))
		// A Codex install has skills in .agents/skills but role agents at
		// <project>/.codex/agents/<name>.toml, with no Markdown sibling.
		if _, err := os.Stat(abs); err != nil {
			codex := strings.TrimSuffix(filepath.Base(filepath.FromSlash(rel)), ".md") + ".toml"
			candidate := filepath.Join(filepath.Dir(root), ".codex", "agents", codex)
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				abs, label, native = candidate, ".codex/agents/"+codex, true
			}
		}
		var wrote bool
		if native && !opts.inlineContract {
			wrote = writeContractStub(&bundle, filepath.Dir(root), abs, label, seen)
		} else {
			wrote = writeContextEntry(&bundle, abs, label, seen, true)
		}
		if wrote {
			loaded++
		} else {
			// The agent contract is a pack file, not a workspace artifact: a
			// miss means a typo'd role or a pack bug, and a dispatch without
			// its contract must fail loudly.
			missingRequired = append(missingRequired, "agents/"+rel)
		}
	}
	// The open findings lead a recheck bundle, before any reference text.
	if inventory != nil {
		writeContextEntry(&bundle, recheckPath, "recheck inventory: "+opts.recheck, seen, false)
		loaded++
	}
	var rootOnly []string
	citedSections, packByReference := 0, 0
	keepModel := opts.role == "" || keepModelSurface(opts.role, gatherTriggerFacts(root, featureDir, opts.role))
	for _, rel := range files {
		abs := filepath.Join(skillsRoot, filepath.FromSlash(rel))
		if seen[abs] {
			continue
		}
		seen[abs] = true
		if opts.role != "" && slices.Contains(manifest.RootOnly, rel) {
			rootOnly = append(rootOnly, rel)
			continue
		}
		data, err := os.ReadFile(abs) // #nosec G304 -- abs is resolved from the validated skills root
		if err != nil {
			fmt.Fprintf(&bundle, "===== %s =====\nMISSING: %s\n\n", rel, rel)
			missingRequired = append(missingRequired, rel)
			continue
		}
		body := expandIncludes(abs, data)
		if opts.role != "" {
			if body, err = stripSections(body, rel, opts.role, keepModel); err != nil {
				fmt.Fprintf(stderr, "context: %v\n", err)
				return 2
			}
		}
		// A recheck keeps every other triggered pack file one anchored Read
		// away (fix-introduced defects may break rules no finding cites) and
		// inlines only the sections the open findings cite.
		if inventory != nil && rel != coreStandard {
			writeAnchorIndex(&bundle, filepath.Dir(root), abs, rel, data, nil)
			citedSections += writeCitedSections(&bundle, rel, body, inventory)
			packByReference++
			loaded++
			continue
		}
		fmt.Fprintf(&bundle, "===== %s =====\n", rel)
		bundle.Write(body)
		if len(body) > 0 && body[len(body)-1] != '\n' {
			bundle.WriteByte('\n')
		}
		bundle.WriteByte('\n')
		loaded++
	}
	if len(rootOnly) > 0 {
		fmt.Fprintf(&bundle, "===== root-only (not inlined for role %s) =====\n%s\n\n", opts.role, strings.Join(rootOnly, "\n"))
	}

	whole, byReference := 0, 0
	workspaceList, workspaceScoped := workspaceReadSet(manifest, opts.role)
	if hasManifest && featureDir != "" {
		if workspaceScoped && len(workspaceList) == 0 {
			bundle.WriteString("===== workspace =====\nskipped: role-scoped read-set declares no workspace artifacts\n\n")
		}
		for _, rel := range workspaceList {
			abs := filepath.Join(featureDir, filepath.FromSlash(rel))
			// Role bundles without --slice (full Vet, Review, Seal) read
			// everything; root and --slice bundles take over-budget
			// artifacts by anchor.
			if opts.slice == "" && opts.role != "" {
				if writeContextEntry(&bundle, abs, "workspace/"+rel, seen, false) {
					loaded++
				}
				continue
			}
			ok, byRef := writeSliceEntry(&bundle, filepath.Dir(root), abs, "workspace/"+rel, opts.slice, sliceBlock, sliceIDs, seen)
			if ok {
				loaded++
			}
			if byRef {
				byReference++
			} else if ok {
				whole++
			}
		}
	} else if hasManifest && len(workspaceList) > 0 {
		bundle.WriteString("===== workspace =====\nskipped: no feature workspace (slug omitted)\n\n")
	}
	withTOC := insertSectionTOC(bundle.String())
	bundle.Reset()
	bundle.WriteString(withTOC)
	out := opts.out
	if out == "" {
		out = filepath.Join(featureDir, "ctx", bundleName(opts))
	} else if !filepath.IsAbs(out) && featureDir != "" {
		out = filepath.Join(featureDir, filepath.FromSlash(out))
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintf(stderr, "context: %v\n", err)
		return 2
	}
	digestPath := out + ".sha256"
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(bundle.String())))
	unchanged := false
	if prev, err := os.ReadFile(digestPath); err == nil && strings.TrimSpace(string(prev)) == digest { // #nosec G304 -- digestPath is generated under the feature workspace
		if _, err := os.Stat(out); err == nil {
			unchanged = true
		}
	}
	if !unchanged {
		if err := os.WriteFile(out, []byte(bundle.String()), 0o600); err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 2
		}
		if err := os.WriteFile(digestPath, []byte(digest+"\n"), 0o600); err != nil {
			fmt.Fprintf(stderr, "context: %v\n", err)
			return 2
		}
	}
	if opts.slug != "" {
		phaseLabel := opts.phase
		if phaseLabel == "" {
			phaseLabel = opts.skill
		}
		recordMetric(root, opts.slug, phaseLabel, "context", opts.role, int64(bundle.Len()))
	}
	if unchanged {
		fmt.Fprintf(stdout, "context: %s (unchanged — reuse previous read; %d sources, %d bytes", out, loaded, bundle.Len())
	} else {
		fmt.Fprintf(stdout, "context: %s (%d sources, %d bytes", out, loaded, bundle.Len())
	}
	if len(missingRequired) > 0 {
		fmt.Fprintf(stdout, ", MISSING: %s", strings.Join(missingRequired, ","))
	}
	fmt.Fprintln(stdout, ")")
	if opts.slice != "" {
		fmt.Fprintf(stdout, "scope: slice %s (%d whole, %d by-reference)\n", opts.slice, whole, byReference)
	} else if byReference > 0 {
		fmt.Fprintf(stdout, "scope: root (%d whole, %d by-reference over budget)\n", whole, byReference)
	}
	if inventory != nil {
		fmt.Fprintf(stdout, "recheck: %s (%d cited sections inlined, %d pack files by reference)\n", opts.recheck, citedSections, packByReference)
	}
	if workspaceScoped {
		fmt.Fprintf(stdout, "workspace: role-scoped for %s (%d of %d declared artifacts)\n",
			opts.role, len(workspaceList), len(manifest.Workspace))
	}
	// Trigger omissions are legal (semantic judgment) but must be visible.
	if hasManifest && len(manifest.Triggers) > 0 {
		var unselected []string
		for _, name := range sortedKeys(manifest.Triggers) {
			if !slices.Contains(opts.triggers, name) && !slices.Contains(autoTriggers, name) {
				unselected = append(unselected, name)
			}
		}
		fmt.Fprintf(stdout, "triggers: applied=[%s] unselected=[%s]\n",
			strings.Join(opts.triggers, ","), strings.Join(unselected, ","))
		if len(autoTriggers) > 0 {
			fmt.Fprintf(stdout, "triggers: auto=[%s] (fired by --role dispatch)\n",
				strings.Join(autoTriggers, ","))
		}
		if suggested := suggestTriggers(manifest, root, featureDir, opts); len(suggested) > 0 {
			fmt.Fprintf(stdout, "triggers: suggested=[%s] (advisory — confirm each before use)\n",
				strings.Join(suggested, ","))
		}
	}
	if len(missingRequired) > 0 {
		return 3
	}
	return 0
}

// sectionHeaderRE matches a bundle section header line.
var sectionHeaderRE = regexp.MustCompile(`^===== (.+) =====$`)

// insertSectionTOC lists each ===== section's line range right after the
// bundle's header comments, so an agent reads a section by range instead of
// grepping for headers. Section order and content are unchanged.
// shortcut: an inlined artifact line shaped like a section header is listed too; record header offsets at write time if that shows up.
func insertSectionTOC(bundle string) string {
	cut := strings.Index(bundle, "\n\n") + 2
	head, body := bundle[:cut], bundle[cut:]
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	var labels []string
	var starts []int
	for i, line := range lines {
		if m := sectionHeaderRE.FindStringSubmatch(line); m != nil {
			labels, starts = append(labels, m[1]), append(starts, i)
		}
	}
	if len(starts) == 0 {
		return bundle
	}
	// Body line i lands after the header, the TOC comment, one line per
	// section, and the TOC's blank line.
	offset := strings.Count(head, "\n") + 1 + len(starts) + 1
	var toc strings.Builder
	toc.WriteString("<!-- sections: line ranges in this file; read a section by its range instead of searching for headers -->\n")
	for i, start := range starts {
		end := len(lines)
		if i+1 < len(starts) {
			end = starts[i+1]
		}
		fmt.Fprintf(&toc, "L%d-%d %s\n", offset+start+1, offset+end, clipRunes(labels[i], 120))
	}
	toc.WriteByte('\n')
	return head + toc.String() + body
}

// workspaceReadSet returns the workspace artifact list for this call. A role
// key present in workspaceByRole fully replaces the flat workspace list for
// that dispatch (empty list = the role reads no workspace artifacts). A role
// absent from the map keeps the phase default.
func workspaceReadSet(manifest loadsManifest, role string) ([]string, bool) {
	if role == "" {
		return manifest.Workspace, false
	}
	if list, ok := manifest.WorkspaceByRole[role]; ok {
		return list, true
	}
	return manifest.Workspace, false
}

var includeMarkerRe = regexp.MustCompile(`<!--\s*include:([^\s>]+)\s*-->`)

// expandIncludes resolves <!-- include:REL --> markers in canonical source
// (paths are relative to the including file's directory), matching
// scripts/expand-includes.py. Unresolvable markers pass through literally so
// the bundle shows the gap instead of dropping it.
func expandIncludes(abs string, data []byte) []byte {
	if !bytes.Contains(data, []byte("include:")) {
		return data
	}
	dir := filepath.Dir(abs)
	return includeMarkerRe.ReplaceAllFunc(data, func(m []byte) []byte {
		target := filepath.Join(dir, filepath.FromSlash(string(includeMarkerRe.FindSubmatch(m)[1])))
		inc, err := os.ReadFile(target) // #nosec G304 -- target is resolved from the validated skill file directory
		if err != nil {
			return m
		}
		return bytes.TrimRight(inc, "\n")
	})
}

// sectionMarkerRe matches a whole-line section marker in canonical source:
// <!-- root-only --> ... <!-- /root-only --> wraps rules only the controlling
// root executes; <!-- model-surface --> ... <!-- /model-surface --> wraps rules
// that apply only to a model, agent, RAG, or tool surface.
var sectionMarkerRe = regexp.MustCompile(`^<!--\s*(/?)(root-only|model-surface)\s*-->\s*$`)

// stripSections replaces each marked block a --role bundle omits with a
// one-line stub naming the omitted heading: root-only blocks always,
// model-surface blocks unless keepModel. Headings never move in the source, so
// anchors stay valid. A nested, unbalanced, or headingless block is an error
// so a pack bug never silently drops or leaks rules.
func stripSections(data []byte, rel, role string, keepModel bool) ([]byte, error) {
	if !bytes.Contains(data, []byte("<!--")) {
		return data, nil
	}
	lines := strings.SplitAfter(string(data), "\n")
	structural := lines
	if masked, err := markdowntext.Structural(data); err == nil {
		structural = strings.SplitAfter(string(masked), "\n")
	}
	var out, block strings.Builder
	kind, heading, stubHeading, ownHeading := "", "", "", false
	for i, line := range lines {
		text := strings.TrimRight(structural[i], "\r\n")
		m := sectionMarkerRe.FindStringSubmatch(text)
		switch {
		case m == nil:
			if anchorHeadingRE.MatchString(text) {
				heading = strings.TrimSpace(strings.TrimLeft(text, "#"))
				if kind != "" && !ownHeading {
					stubHeading, ownHeading = heading, true
				}
			}
			if kind == "" {
				out.WriteString(line)
			} else {
				block.WriteString(line)
			}
		case m[1] == "" && kind == "":
			kind, stubHeading, ownHeading = m[2], heading, false
			block.Reset()
			block.WriteString(line)
		case m[1] == "/" && m[2] == kind:
			block.WriteString(line)
			switch {
			case stubHeading == "":
				return nil, fmt.Errorf("%s:%d: <!-- %s --> section has no heading to name in its stub", rel, i+1, kind)
			case kind == "model-surface" && keepModel:
				out.WriteString(block.String())
			case kind == "model-surface":
				fmt.Fprintf(&out, "[model-surface section %q omitted for role %s: spec/plan/tasks name no model, agent, RAG, or tool surface; if the change touches one, read %s § %s]\n", stubHeading, role, rel, stubHeading)
			default:
				fmt.Fprintf(&out, "[root-only section %q omitted for role %s; the controlling root executes it]\n", stubHeading, role)
			}
			kind = ""
		default:
			return nil, fmt.Errorf("%s:%d: unbalanced section marker %q", rel, i+1, text)
		}
	}
	if kind != "" {
		return nil, fmt.Errorf("%s: unclosed <!-- %s --> section", rel, kind)
	}
	return []byte(out.String()), nil
}

// sliceInlineMaxBytes caps a workspace artifact inlined whole into a --slice
// bundle. Planning budgets size a whole feature, not one slice's read; a
// larger artifact goes by reference so the slice's agent reads only the
// sections its block cites. spec.md, the acceptance source, stays whole.
const sliceInlineMaxBytes = 16 << 10

// writeSliceEntry writes one workspace artifact of a --slice or root bundle:
// with a slice, tasks.md as the slice block plus a by-reference index and any
// other artifact but spec.md over sliceInlineMaxBytes as a by-reference index;
// any canonical artifact over its line/byte budget as a by-reference index;
// everything else whole.
func writeSliceEntry(bundle *strings.Builder, project, abs, label, slice, block string, ids []citedID, seen map[string]bool) (loaded, byRef bool) {
	if seen[abs] {
		return false, false
	}
	data, err := os.ReadFile(abs) // #nosec G304 -- abs is resolved from the validated feature workspace
	name := filepath.Base(abs)
	if err == nil && name == "tasks.md" && slice != "" {
		seen[abs] = true
		fmt.Fprintf(bundle, "===== %s (%s block) =====\n%s\n", label, slice, block)
	} else if err == nil && slice != "" && name != "spec.md" && len(data) > sliceInlineMaxBytes {
		seen[abs] = true
	} else if status, ok := state.ArtifactBudget(name, data); err != nil || !ok || !status.Over {
		return writeContextEntry(bundle, abs, label, seen, false), false
	} else {
		seen[abs] = true
	}
	writeAnchorIndex(bundle, project, abs, label, data, ids)
	return true, true
}

var (
	citedIDRE       = regexp.MustCompile(`\b[A-Z]{1,6}-\d{2,}(?:-[A-Z][A-Z0-9]*)*\b`)
	anchorHeadingRE = regexp.MustCompile(`^#{1,3}\s`)
)

// maxIDHits bounds the line anchors listed per ID; the live file holds the rest.
const maxIDHits = 20

// citedID is one ID the slice block cites with its word-bounded matcher.
type citedID struct {
	id string
	re *regexp.Regexp
}

// citedIDs returns each distinct ID the slice block cites (AC-###, DEC-###,
// T-###, PF-##, SLICE-###, ...).
func citedIDs(block string) []citedID {
	var out []citedID
	seen := map[string]bool{}
	for _, id := range citedIDRE.FindAllString(block, -1) {
		if !seen[id] {
			seen[id] = true
			out = append(out, citedID{id, regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`)})
		}
	}
	return out
}

// writeAnchorIndex emits a by-reference entry: identity, heading anchors, and
// line hits for each cited ID, so the agent reads owning sections from the
// live file instead of the whole artifact.
func writeAnchorIndex(bundle *strings.Builder, project, abs, label string, data []byte, ids []citedID) {
	rel, err := filepath.Rel(project, abs)
	if err != nil {
		rel = abs
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	structural := lines
	if masked, err := markdowntext.Structural(data); err == nil {
		structural = strings.Split(strings.ReplaceAll(string(masked), "\r\n", "\n"), "\n")
	}
	fmt.Fprintf(bundle, "===== %s (by reference) =====\n", label)
	fmt.Fprintf(bundle, "path: %s sha256: %x bytes: %d lines: %d\n", filepath.ToSlash(rel), sha256.Sum256(data), len(data), len(lines))
	bundle.WriteString("Not inlined. Read owning sections from the live file by line anchor; nothing is omitted there.\n")
	bundle.WriteString("headings:\n")
	for i, line := range structural {
		if anchorHeadingRE.MatchString(line) {
			fmt.Fprintf(bundle, "- L%d %s\n", i+1, clipRunes(lines[i], 120))
		}
	}
	if len(ids) > 0 {
		bundle.WriteString("id hits (IDs cited by the slice block):\n")
	}
	for _, cited := range ids {
		var at []string
		total := 0
		for i, line := range lines {
			if cited.re.MatchString(line) {
				total++
				if len(at) < maxIDHits {
					at = append(at, fmt.Sprintf("L%d", i+1))
				}
			}
		}
		hits := "none"
		if total > 0 {
			hits = strings.Join(at, " ")
			if total > len(at) {
				hits += fmt.Sprintf(" (+%d more)", total-len(at))
			}
		}
		fmt.Fprintf(bundle, "- %s: %s\n", cited.id, hits)
	}
	bundle.WriteByte('\n')
}

// standardCiteRE matches a "<file>.md § <heading>" citation, as a finding's
// "breaks <standard §>" field writes it; the heading text ends at a quote,
// pipe, semicolon, closing link bracket, or line end.
var standardCiteRE = regexp.MustCompile(`([\w.-]+\.md)\S*\s*§\s*([^"|;\n\]]+)`)

// writeCitedSections inlines each H1-H3 section of body (heading through the
// next heading of the same or higher level) that the inventory cites by
// "<file>.md § <heading>"; a citation may abbreviate the heading or name only
// its title before the colon. It returns the number of sections written.
// shortcut: a citation naming a heading by other words is missed and stays reachable through the anchor index; match the quoted line if reviewers cite that way.
func writeCitedSections(bundle *strings.Builder, rel string, body, inventory []byte) int {
	var cites []string
	for _, m := range standardCiteRE.FindAllStringSubmatch(string(inventory), -1) {
		if m[1] == filepath.Base(rel) {
			if c := strings.ToLower(strings.Trim(m[2], " \t.,:()`*")); c != "" {
				cites = append(cites, c)
			}
		}
	}
	if len(cites) == 0 {
		return 0
	}
	lines := strings.SplitAfter(string(body), "\n")
	structural := lines
	if masked, err := markdowntext.Structural(body); err == nil {
		structural = strings.SplitAfter(string(masked), "\n")
	}
	level := func(i int) int {
		text := strings.TrimRight(structural[i], "\r\n")
		if !anchorHeadingRE.MatchString(text) {
			return 0
		}
		return len(text) - len(strings.TrimLeft(text, "#"))
	}
	written, coveredTo := 0, 0
	for i := range lines {
		lvl := level(i)
		if lvl == 0 || i < coveredTo {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(strings.TrimRight(lines[i], "\r\n"), "#"))
		lower := strings.ToLower(heading)
		title, _, _ := strings.Cut(lower, ":")
		title = strings.TrimSpace(title)
		if !slices.ContainsFunc(cites, func(c string) bool {
			// The title rule skips the H1: a one-word file title would match every citation.
			return strings.HasPrefix(lower, c) || (lvl > 1 && title != "" && (c == title || strings.HasPrefix(c, title+" ")))
		}) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if l := level(j); l > 0 && l <= lvl {
				end = j
				break
			}
		}
		fmt.Fprintf(bundle, "===== %s § %s (cited by the recheck inventory) =====\n", rel, clipRunes(heading, 120))
		bundle.WriteString(strings.TrimRight(strings.Join(lines[i:end], ""), "\r\n") + "\n\n")
		written, coveredTo = written+1, end
	}
	return written
}

func clipRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func writeContextEntry(bundle *strings.Builder, abs, label string, seen map[string]bool, expand bool) bool {
	if seen[abs] {
		return false
	}
	seen[abs] = true
	fmt.Fprintf(bundle, "===== %s =====\n", label)
	data, err := os.ReadFile(abs) // #nosec G304 -- abs is resolved from the validated context source set
	if err != nil {
		fmt.Fprintf(bundle, "MISSING: %s\n\n", label)
		return false
	}
	if expand {
		bundle.Write(expandIncludes(abs, data))
	} else {
		bundle.Write(data)
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		bundle.WriteByte('\n')
	}
	bundle.WriteByte('\n')
	return true
}

// writeContractStub names a role contract the host already loads as the
// agent's system prompt instead of inlining a second copy. A missing file
// still fails, as in writeContextEntry.
func writeContractStub(bundle *strings.Builder, project, abs, label string, seen map[string]bool) bool {
	seen[abs] = true
	data, err := os.ReadFile(abs) // #nosec G304 -- abs is resolved from the validated skills root or project
	if err != nil {
		fmt.Fprintf(bundle, "===== %s =====\nMISSING: %s\n\n", label, label)
		return false
	}
	path := label
	if r, err := filepath.Rel(project, abs); err == nil && !strings.HasPrefix(r, "..") {
		path = filepath.ToSlash(r)
	}
	fmt.Fprintf(bundle, "===== %s (not inlined: your host loads this role contract as your system prompt; %d bytes, sha256 %x; read %s only if your system prompt lacks it) =====\n\n",
		label, len(data), sha256.Sum256(data), path)
	return true
}

type contextOpts struct {
	slug           string
	phase          string
	skill          string
	role           string
	slice          string // display label: slices joined with "+"
	slices         []string
	triggers       []string
	skillsRoot     string
	out            string
	recheck        string
	inlineContract bool
}

func parseContextArgs(args []string, stderr io.Writer) (contextOpts, int) {
	var opts contextOpts
	var positional []string
	for i := 0; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--phase":
			opts.phase = value
		case "--skill":
			opts.skill = value
		case "--role":
			opts.role = value
		case "--slice":
			opts.slices = append(opts.slices, value)
		case "--trigger", "--triggers":
			for _, name := range strings.Split(value, ",") {
				if name = strings.TrimSpace(name); name != "" {
					opts.triggers = append(opts.triggers, name)
				}
			}
		case "--skills-root":
			opts.skillsRoot = value
		case "--out":
			opts.out = value
		case "--recheck":
			opts.recheck = value
		case "--inline-contract":
			opts.inlineContract = true
			continue
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "context: unknown flag %q\n", args[i])
				return opts, 2
			}
			positional = append(positional, args[i])
			continue
		}
		i++
	}
	if len(positional) > 1 || (opts.phase == "") == (opts.skill == "") {
		fmt.Fprint(stderr, contextUsage)
		return opts, 2
	}
	if opts.skill != "" && (strings.Contains(opts.skill, "/") || strings.Contains(opts.skill, "..")) {
		fmt.Fprintf(stderr, "context: --skill must be a directory name, not a path\n")
		return opts, 2
	}
	for _, id := range opts.slices {
		if !state.ValidSliceID(id) || len(positional) == 0 {
			fmt.Fprintf(stderr, "context: --slice needs a slug and a SLICE-### id, got %q\n", id)
			return opts, 2
		}
	}
	opts.slice = strings.Join(opts.slices, "+")
	if opts.recheck != "" && (opts.phase != "build" || opts.role == "" || opts.slice == "") {
		fmt.Fprintf(stderr, "context: --recheck needs --phase build, --role, and --slice\n")
		return opts, 2
	}
	if len(positional) == 1 {
		opts.slug = positional[0]
	} else if opts.out == "" {
		// No feature workspace: --out is required so the bundle has an
		// explicit destination instead of an implicit cwd write.
		fmt.Fprintf(stderr, "context: slug omitted — --out <path> is required\n")
		return opts, 2
	}
	return opts, 0
}

// resolveSkillsRoot finds the installed skill tree. Order: explicit flag, the
// canonical pack (this repo), a project-level install, then the user-global
// install.
func resolveSkillsRoot(root, explicit string) (string, error) {
	// root is the .devrites directory; its parent is the project root that
	// holds pack/ and host skill installs.
	project := filepath.Dir(root)
	if explicit != "" {
		candidates := []string{explicit}
		// Host guidance passes a project-relative root (.omp/skills); keep it
		// working when the engine runs from a subdirectory.
		if !filepath.IsAbs(explicit) {
			candidates = append(candidates, filepath.Join(project, explicit))
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				return candidate, nil
			}
		}
		return "", fmt.Errorf("skills root %q is not a directory", explicit)
	}
	candidates := []string{
		filepath.Join(project, "pack", ".claude", "skills"),
		filepath.Join(root, "pack", ".claude", "skills"),
		filepath.Join(project, ".claude", "skills"),
		filepath.Join(root, ".claude", "skills"),
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".claude", "skills"))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(filepath.Join(candidate, "devrites-lib")); err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no DevRites skills root found (looked in pack/.claude/skills, .claude/skills, ~/.claude/skills)")
}

// parseLoadsManifest extracts the <!-- loads: {...} --> block from a SKILL.md.
func parseLoadsManifest(text string) (loadsManifest, bool, error) {
	idx := strings.Index(text, loadsMarker)
	if idx < 0 {
		return loadsManifest{}, false, nil
	}
	end := strings.Index(text[idx:], "-->")
	if end < 0 {
		return loadsManifest{}, false, fmt.Errorf("loads: manifest comment is not closed")
	}
	payload := strings.TrimSpace(text[idx+len(loadsMarker) : idx+end])
	var manifest loadsManifest
	if err := json.Unmarshal([]byte(payload), &manifest); err != nil {
		return loadsManifest{}, false, fmt.Errorf("loads: manifest is not valid JSON: %w", err)
	}
	return manifest, true, nil
}

func roleAgentPath(manifest loadsManifest, role string) string {
	if manifest.Agents != nil {
		if rel, ok := manifest.Agents[role]; ok {
			return rel
		}
	}
	return "devrites-" + role + ".md"
}

func bundleName(opts contextOpts) string {
	name := opts.phase
	if name == "" {
		name = opts.skill
	}
	if opts.role != "" {
		name += "-" + opts.role
	}
	if opts.slice != "" {
		name += "." + opts.slice
	}
	if opts.recheck != "" {
		name += ".recheck"
	}
	return name + ".bundle.md"
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
