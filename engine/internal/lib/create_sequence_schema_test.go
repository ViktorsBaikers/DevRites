package lib

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateSequenceSchemaIgnoresFencedExamples(t *testing.T) {
	for _, actual := range []bool{true, false} {
		t.Run(map[bool]string{true: "real-current-schema", false: "fenced-only-schema"}[actual], func(t *testing.T) {
			root, brief := sequenceCreationFixture(t, true)
			cursor := filepath.Join(root, "work", "current16", "state.md")
			text := readTestFile(t, cursor)
			if !actual {
				text = strings.Replace(text, "| schema | 4 |\n", "", 1)
			}
			writeFile(t, cursor, text+"\n```markdown\n| schema | 4 |\n```\n")
			if actual {
				sequenceCreated(t, root, "future17", brief, 2, "current16")
				sequenceCreated(t, root, "future18", brief, 3, "future17")
			} else {
				sequenceRefused(t, root, "future17", brief)
			}
		})
	}
}
