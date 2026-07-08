package lib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
)

// Extensions is the project-local extension surface: user-authored rites and
// reviewers that live under .devrites/extensions/<name>/ and are held to the same
// schema as the shipped pack, then mirrored into the harness's skill/agent dirs.
// It is DevRites' answer to "add a rite or reviewer without forking the pack" —
// deliberately project-scoped, with no channel/marketplace/registry apparatus.
//
// Layout of one extension:
//
//	.devrites/extensions/<name>/
//	  skill/SKILL.md      (optional) a user skill — must carry name + description frontmatter
//	  agent.md            (optional) a user reviewer agent — must carry name + description
//	  extension.yaml      (optional) metadata: aliases (prior names, so a rename doesn't orphan)
//
// base is the .devrites root; the project directory (where .claude lives) is its
// parent.
//
//	list       enumerate extensions and what each provides
//	validate   check each against the schema; 1 on a violation
//	sync       mirror skills/agents into .claude/ so the harness discovers them
func Extensions(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "usage: devrites-engine extensions list|validate|sync"
	sub := argAt(args, 0)
	extDir := filepath.Join(root, "extensions")
	switch sub {
	case "list":
		return extensionsList(extDir, stdout)
	case "validate":
		return extensionsValidate(extDir, stdout, stderr)
	case "sync":
		return extensionsSync(extDir, filepath.Dir(root), stdout, stderr)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
}

// extension is one discovered extension directory and what it provides.
type extension struct {
	name      string
	dir       string
	skillPath string // "" if none
	agentPath string // "" if none
	aliases   []string
}

// discoverExtensions returns the extensions under extDir, sorted by name. A
// missing extensions dir is not an error — it means the project declares none.
func discoverExtensions(extDir string) ([]extension, error) {
	entries, err := os.ReadDir(extDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []extension
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dir := filepath.Join(extDir, e.Name())
		ext := extension{name: e.Name(), dir: dir}
		if p := filepath.Join(dir, "skill", "SKILL.md"); isFile(p) {
			ext.skillPath = p
		}
		if p := filepath.Join(dir, "agent.md"); isFile(p) {
			ext.agentPath = p
		}
		if meta, ok := readFileOK(filepath.Join(dir, "extension.yaml")); ok {
			ext.aliases = parseAliases(meta)
		}
		out = append(out, ext)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func extensionsList(extDir string, stdout io.Writer) int {
	exts, err := discoverExtensions(extDir)
	if err != nil {
		fmt.Fprintf(stdout, "extensions: cannot read %s: %v\n", extDir, err)
		return 0
	}
	if len(exts) == 0 {
		fmt.Fprintln(stdout, "extensions: none (.devrites/extensions/ empty or absent)")
		return 0
	}
	for _, e := range exts {
		var provides []string
		if e.skillPath != "" {
			provides = append(provides, "skill")
		}
		if e.agentPath != "" {
			provides = append(provides, "agent")
		}
		if len(provides) == 0 {
			provides = []string{"(empty)"}
		}
		line := fmt.Sprintf("  %s — %s", e.name, strings.Join(provides, "+"))
		if len(e.aliases) > 0 {
			line += " · aliases: " + strings.Join(e.aliases, ", ")
		}
		fmt.Fprintln(stdout, line)
	}
	return 0
}

// extensionsValidate holds each extension to the same bar as the shipped pack: a
// declared skill/agent must carry name + description frontmatter, an extension
// must provide at least one of them, and no two may claim the same skill/agent
// name. Reserved pack prefixes (rite-, devrites-) are a collision warning.
//
//	0  every extension well-formed (or none)
//	1  a schema violation — an extension that would install broken
func extensionsValidate(extDir string, stdout, stderr io.Writer) int {
	exts, err := discoverExtensions(extDir)
	if err != nil {
		fmt.Fprintf(stderr, "extensions: cannot read %s: %v\n", extDir, err)
		return 1
	}
	if len(exts) == 0 {
		fmt.Fprintln(stdout, "extensions: none to validate")
		return 0
	}

	var problems []string
	seenSkill, seenAgent := map[string]string{}, map[string]string{}
	for _, e := range exts {
		if e.skillPath == "" && e.agentPath == "" {
			problems = append(problems, fmt.Sprintf("%s: provides neither skill/SKILL.md nor agent.md", e.name))
			continue
		}
		if e.skillPath != "" {
			name, missing := frontmatterName(e.skillPath)
			if missing != "" {
				problems = append(problems, fmt.Sprintf("%s: skill %s", e.name, missing))
			} else {
				if prior, dup := seenSkill[name]; dup {
					problems = append(problems, fmt.Sprintf("%s: skill name %q also declared by %s", e.name, name, prior))
				}
				seenSkill[name] = e.name
				if reservedPackName(name) {
					fmt.Fprintf(stdout, "  warning: %s skill name %q uses a reserved pack prefix (rite-/devrites-) — it will shadow or collide with the shipped pack\n", e.name, name)
				}
			}
		}
		if e.agentPath != "" {
			name, missing := frontmatterName(e.agentPath)
			if missing != "" {
				problems = append(problems, fmt.Sprintf("%s: agent %s", e.name, missing))
			} else {
				if prior, dup := seenAgent[name]; dup {
					problems = append(problems, fmt.Sprintf("%s: agent name %q also declared by %s", e.name, name, prior))
				}
				seenAgent[name] = e.name
			}
		}
	}

	if len(problems) > 0 {
		fmt.Fprintf(stderr, "extensions: %d problem(s):\n", len(problems))
		for _, p := range problems {
			fmt.Fprintf(stderr, "  - %s\n", p)
		}
		return 1
	}
	fmt.Fprintf(stdout, "extensions: OK — %d extension(s) well-formed\n", len(exts))
	return 0
}

// extensionsSync mirrors every valid extension into the project's .claude tree,
// where the Claude harness discovers skills and agents: skill/ → .claude/skills/<name>/,
// agent.md → .claude/agents/<name>.md. Idempotent (a re-sync overwrites in place).
// It validates first and refuses to sync a broken set. Codex extension mirroring
// is intentionally not implemented here so the engine stays out of the
// Claude-to-Codex generation path.
func extensionsSync(extDir, projectDir string, stdout, stderr io.Writer) int {
	if code := extensionsValidate(extDir, io.Discard, stderr); code != 0 {
		fmt.Fprintln(stderr, "extensions: not syncing — validation failed (run `devrites-engine extensions validate`)")
		return code
	}
	exts, _ := discoverExtensions(extDir)
	if len(exts) == 0 {
		fmt.Fprintln(stdout, "extensions: nothing to sync")
		return 0
	}

	skillsDst := filepath.Join(projectDir, ".claude", "skills")
	agentsDst := filepath.Join(projectDir, ".claude", "agents")
	synced := 0
	for _, e := range exts {
		if e.skillPath != "" {
			dst := filepath.Join(skillsDst, e.name)
			if err := fsutil.CopyTree(filepath.Join(e.dir, "skill"), dst); err != nil {
				fmt.Fprintf(stderr, "extensions: sync skill %s failed: %v\n", e.name, err)
				return 1
			}
			fmt.Fprintf(stdout, "  synced skill: .claude/skills/%s/\n", e.name)
			synced++
		}
		if e.agentPath != "" {
			if err := os.MkdirAll(agentsDst, 0o755); err != nil {
				fmt.Fprintf(stderr, "extensions: sync agent %s failed: %v\n", e.name, err)
				return 1
			}
			data, err := os.ReadFile(e.agentPath)
			if err == nil {
				err = writeFileAtomic(filepath.Join(agentsDst, e.name+".md"), data, 0o644)
			}
			if err != nil {
				fmt.Fprintf(stderr, "extensions: sync agent %s failed: %v\n", e.name, err)
				return 1
			}
			fmt.Fprintf(stdout, "  synced agent: .claude/agents/%s.md\n", e.name)
			synced++
		}
	}
	fmt.Fprintf(stdout, "extensions: synced %d artifact(s) from %d extension(s)\n", synced, len(exts))
	return 0
}

// frontmatterName reads the `name:` from a markdown file's YAML frontmatter and
// reports which required field is missing ("" when both name and description are
// present). Used to hold an extension skill/agent to the same contract as the pack.
func frontmatterName(path string) (name, missing string) {
	content, ok := readFileOK(path)
	if !ok {
		return "", "file unreadable"
	}
	fm, ok := frontmatterBlock(content)
	if !ok {
		return "", "missing YAML frontmatter (--- fenced block)"
	}
	hasDesc := false
	for _, line := range strings.Split(fm, "\n") {
		if v, ok := cutYAMLKey(line, "name"); ok {
			name = v
		}
		if _, ok := cutYAMLKey(line, "description"); ok {
			hasDesc = true
		}
	}
	switch {
	case name == "":
		return "", "missing `name:` in frontmatter"
	case !hasDesc:
		return name, "missing `description:` in frontmatter"
	}
	return name, ""
}

// frontmatterBlock returns the text between the leading `---` fences, and false
// when the file has no frontmatter block.
func frontmatterBlock(content string) (string, bool) {
	s := strings.TrimLeft(content, "\uFEFF\n\r ")
	if !strings.HasPrefix(s, "---") {
		return "", false
	}
	rest := s[len("---"):]
	// The opening fence must end its line.
	if i := strings.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[i+1:]
	} else {
		return "", false
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// cutYAMLKey parses a simple `key: value` line, trimming surrounding quotes. It
// only handles the flat scalar keys an extension frontmatter needs.
func cutYAMLKey(line, key string) (string, bool) {
	t := strings.TrimSpace(line)
	if !strings.HasPrefix(t, key+":") {
		return "", false
	}
	v := strings.TrimSpace(t[len(key)+1:])
	v = strings.Trim(v, `"'`)
	return v, true
}

// parseAliases reads a flat `aliases: [a, b]` or a block list from extension.yaml.
func parseAliases(meta string) []string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(meta, "\n") {
		t := strings.TrimSpace(line)
		if v, ok := cutYAMLKey(line, "aliases"); ok {
			inBlock = true
			v = strings.Trim(v, "[]")
			for _, a := range strings.Split(v, ",") {
				if a = strings.TrimSpace(strings.Trim(a, `"'`)); a != "" {
					out = append(out, a)
				}
			}
			continue
		}
		if inBlock && strings.HasPrefix(t, "- ") {
			out = append(out, strings.Trim(strings.TrimSpace(t[2:]), `"'`))
			continue
		}
		if inBlock && t != "" && !strings.HasPrefix(t, "- ") {
			inBlock = false
		}
	}
	return out
}

// reservedPackName reports whether a name collides with the shipped pack's
// namespaces, which a user extension must not claim.
func reservedPackName(name string) bool {
	return strings.HasPrefix(name, "rite-") || strings.HasPrefix(name, "devrites-") || name == "rite"
}
