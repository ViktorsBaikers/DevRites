package lib

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGatherTriggerFactsReadsCappedPrefixOfOversizedArtifact(t *testing.T) {
	dir := t.TempDir()
	body := "this feature adds a frontend component\n" + strings.Repeat(" ", triggerEvidenceFileCap)
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	facts, err := gatherTriggerFacts("", "", &contextSources{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !uiSignalRe.MatchString(facts.corpus) {
		t.Fatalf("oversized artifact contributed no corpus (len=%d)", len(facts.corpus))
	}
	if len(facts.corpus) > triggerEvidenceFileCap+1 {
		t.Fatalf("corpus exceeds cap: %d", len(facts.corpus))
	}
}

func TestTriggerApplyCommandRoundTripsSelectedContext(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash unavailable for shell command roundtrip")
	}
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"vet plan", []string{"--phase", "vet", "--role", "plan-reviewer", "--recheck", "packets/owner's inventory.md", "--whole"}},
		{"vet slices via skill", []string{"--skill", "rite-vet", "--role", "devrites-plan-reviewer.backend", "--recheck", "packets/owner's inventory.md", "--slice", "SLICE-002", "--slice", "SLICE-001", "--slice", "SLICE-002"}},
		{"build slice", []string{"--phase", "build", "--role", "code-reviewer", "--recheck", "packets/owner's inventory.md", "--slice", "SLICE-001"}},
		{"preparation", []string{"--phase", "define", "--role", "plan-reviewer", "--prepare-for", "current16"}},
		{"research", []string{"--phase", "spec", "--role", "evidence-scout", "--sequence-research"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"feature", "--trigger", "security,agents,security", "--inline-contract", "--skills-root", "selected host's skills", "--out", "ctx/owner's context.bundle.md"}, tc.args...)
			var stderr bytes.Buffer
			opts, code := parseContextArgs(args, &stderr)
			if code != 0 {
				t.Fatalf("initial arguments: code=%d %s", code, &stderr)
			}
			command := triggerApplyCommand(opts, []string{"tdd", "security", "frontend", "tdd"})
			cmd := exec.Command(bash, "--noprofile", "--norc", "-c", "devrites-engine() { printf '%s\\0' \"$@\"; };\n"+command)
			raw, err := cmd.Output()
			if err != nil {
				t.Fatalf("shell roundtrip: %v\n%s", err, command)
			}
			replayed := strings.Split(strings.TrimSuffix(string(raw), "\x00"), "\x00")
			if len(replayed) < 2 || replayed[0] != "context" {
				t.Fatalf("retry did not invoke context: %q", replayed)
			}
			got, code := parseContextArgs(replayed[1:], &stderr)
			if code != 0 {
				t.Fatalf("retry arguments: code=%d %s\n%s", code, &stderr, command)
			}
			opts.triggers = []string{"security", "agents", "tdd", "frontend"}
			if !reflect.DeepEqual(got, opts) {
				t.Fatalf("retry changed selected context:\ngot  %#v\nwant %#v\n%s", got, opts, command)
			}
		})
	}
}
