package state

import (
	"strings"
	"testing"
)

func FuzzWorkspaceSchemaRow(f *testing.F) {
	f.Add([]byte("| schema | 5 |"))
	f.Add([]byte("- schema: 2"))
	f.Add([]byte("| schema | abc |"))
	f.Add([]byte("| schema | 0 |"))
	f.Add([]byte("## Cursor\n\n| key | value |"))
	f.Add([]byte("```\n| schema | 7 |\n```"))
	f.Fuzz(func(t *testing.T, doc []byte) {
		version, err := workspaceSchemaRow(strings.Split(string(doc), "\n"))
		if err != nil {
			if version != 0 {
				t.Fatalf("error result carries version %d", version)
			}
			return
		}
		if version < 1 {
			t.Fatalf("success with non-positive version %d", version)
		}
	})
}
