package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatherTriggerFactsReadsCappedPrefixOfOversizedArtifact(t *testing.T) {
	dir := t.TempDir()
	body := "this feature adds a frontend component\n" + strings.Repeat(" ", triggerEvidenceFileCap)
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	facts := gatherTriggerFacts("", dir, "")
	if !uiSignalRe.MatchString(facts.corpus) {
		t.Fatalf("oversized artifact contributed no corpus (len=%d)", len(facts.corpus))
	}
	if len(facts.corpus) > triggerEvidenceFileCap+1 {
		t.Fatalf("corpus exceeds cap: %d", len(facts.corpus))
	}
}
