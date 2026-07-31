package markdowntext

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestStructuralMasksFencedBlocksWithoutMovingBytes(t *testing.T) {
	corpus, err := os.ReadFile("testdata/structural.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name   string `json:"name"`
		Input  string `json:"input"`
		Output string `json:"output"`
	}
	if err := json.Unmarshal(corpus, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("structural Markdown corpus is empty")
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			input := []byte(test.Input)
			got, err := Structural(input)
			if err != nil {
				t.Fatal(err)
			}
			if want := []byte(test.Output); !bytes.Equal(got, want) {
				t.Fatalf("Structural() = %q, want %q", got, want)
			}
			if len(got) != len(input) {
				t.Fatalf("len(Structural()) = %d, want %d", len(got), len(input))
			}
			for i, b := range input {
				if (b == '\n' || b == '\r') && got[i] != b {
					t.Fatalf("line break at byte %d changed from %q to %q", i, b, got[i])
				}
			}
		})
	}
}

func TestStructuralRejectsCorruptText(t *testing.T) {
	for _, input := range [][]byte{
		{'o', 'k', 0},
		{'o', 'k', 0xff},
	} {
		if _, err := Structural(input); err == nil {
			t.Fatalf("Structural(%q) error = nil", input)
		}
	}
}
