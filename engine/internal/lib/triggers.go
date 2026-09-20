package lib

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// triggers.go: advisory trigger suggestions for `context`. Declared triggers are
// semantic — only the model may apply them — but the *facts* that make one
// relevant are deterministic: sentinel files, workspace-artifact keywords, and
// the dispatch role. Suggesting from facts removes per-invocation guesswork and
// the two failure modes it causes (over-loading wastes tokens; under-loading
// skips rules and costs a repair round). Suggestions are additive only: they
// never veto a caller-selected trigger and are never applied automatically.

// triggerFacts is the evidence base for suggestions.
type triggerFacts struct {
	role        string
	afk         bool
	afkParallel bool
	principles  bool
	corpus      string // lowercased text of spec/plan/tasks/design brief
}

// triggerEvidenceFiles bounds which workspace artifacts feed keyword matching.
var triggerEvidenceFiles = []string{"spec.md", "plan.md", "tasks.md", "design-brief.md"}

const triggerEvidenceFileCap = 256 << 10

var (
	uiSignalRe   = regexp.MustCompile(`\b(frontend|front-end|\bui\b|\bux\b|css|tailwind|stylesheet|component|responsive|layout|browser|viewport|screen reader|landing page|design[- ]brief|react|vue|svelte|next\.?js)\b`)
	secSignalRe  = regexp.MustCompile(`\b(auth|authentication|authorization|login|password|secret|token|credential|permission|encrypt|crypto|oauth|session|cookie|inject|xss|csrf|ssrf|webhook|untrusted|input validat)`)
	tddSignalRe  = regexp.MustCompile(`\b(tdd|test[- ]first|test driven)\b`)
	topoSignalRe = regexp.MustCompile(`(migrat|\bschema\b|\bsql\b|database|postgres|mysql|sqlite|mongo|\bqueue\b|kafka|rabbitmq|redis|\bcache\b|webhook|monorepo|multi[- ]root|multi[- ]service|microservice|\bgrpc\b|graphql|\brest\b|\bapi\b|message broker)`)
	wfaSignalRe  = regexp.MustCompile(`workflow[- ]artifact`)
)

// triggerSignals maps declared trigger-name shapes to facts. nameRe is matched
// case-insensitively against each declared (unselected) trigger name.
var triggerSignals = []struct {
	nameRe *regexp.Regexp
	fires  func(triggerFacts) bool
}{
	{regexp.MustCompile(`afk`), func(f triggerFacts) bool { return f.afk }},
	{regexp.MustCompile(`parallel`), func(f triggerFacts) bool { return f.afkParallel }},
	{regexp.MustCompile(`frontend|^ui$|ux|design`), func(f triggerFacts) bool { return uiSignalRe.MatchString(f.corpus) }},
	{regexp.MustCompile(`security|^sec`), func(f triggerFacts) bool { return secSignalRe.MatchString(f.corpus) }},
	{regexp.MustCompile(`tdd`), func(f triggerFacts) bool { return tddSignalRe.MatchString(f.corpus) }},
	{regexp.MustCompile(`applicability|topology|data[-_]?integrity|integration|repository`), func(f triggerFacts) bool {
		return topoSignalRe.MatchString(f.corpus)
	}},
	{regexp.MustCompile(`principles`), func(f triggerFacts) bool { return f.principles }},
	{regexp.MustCompile(`workflow[-_]?artifacts?`), func(f triggerFacts) bool {
		return wfaSignalRe.MatchString(f.corpus)
	}},
	// A writer role authors product source/tests; the craft-rule triggers apply.
	{regexp.MustCompile(`coding|errors?|patterns?|^dod$|definition|testing`), func(f triggerFacts) bool {
		return f.role == "slice-wright"
	}},
}

// suggestTriggers returns declared, unselected trigger names whose firing facts
// hold in this workspace. Sorted for stable output.
func suggestTriggers(manifest loadsManifest, root, featureDir string, opts contextOpts) []string {
	facts := gatherTriggerFacts(root, featureDir, opts.role)
	var suggested []string
	for name := range manifest.Triggers {
		if slices.Contains(opts.triggers, name) {
			continue
		}
		lower := strings.ToLower(name)
		for _, signal := range triggerSignals {
			if signal.nameRe.MatchString(lower) && signal.fires(facts) {
				suggested = append(suggested, name)
				break
			}
		}
	}
	sort.Strings(suggested)
	return suggested
}

func gatherTriggerFacts(root, featureDir, role string) triggerFacts {
	facts := triggerFacts{role: role}
	if root != "" {
		afkPath := filepath.Join(root, "AFK")
		if info, err := os.Stat(afkPath); err == nil && info.Mode().IsRegular() {
			facts.afk = true
			if data, err := os.ReadFile(afkPath); err == nil { // #nosec G304 -- .devrites/AFK sentinel
				facts.afkParallel = afkAllowsParallel(string(data))
			}
		}
		if info, err := os.Stat(filepath.Join(root, "principles.md")); err == nil && info.Mode().IsRegular() {
			facts.principles = true
		}
	}
	if featureDir != "" {
		var corpus strings.Builder
		for _, name := range triggerEvidenceFiles {
			path := filepath.Join(featureDir, name)
			info, err := os.Stat(path)
			if err != nil || !info.Mode().IsRegular() || info.Size() > triggerEvidenceFileCap {
				continue
			}
			// #nosec G304 -- workspace artifact; size cap checked above
			if data, err := os.ReadFile(path); err == nil {
				corpus.Write(data)
				corpus.WriteByte('\n')
			}
		}
		facts.corpus = strings.ToLower(corpus.String())
	}
	return facts
}

var afkParallelRe = regexp.MustCompile(`(?m)^\s*max_parallel\s*:\s*([0-9]+)\s*$`)

// afkAllowsParallel reports whether the AFK sentinel declares max_parallel > 1.
func afkAllowsParallel(content string) bool {
	m := afkParallelRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return false
	}
	n := 0
	for _, r := range m[1] {
		n = n*10 + int(r-'0')
	}
	return n > 1
}
