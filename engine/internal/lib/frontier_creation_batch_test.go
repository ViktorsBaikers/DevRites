package lib

import (
	"bytes"
	"strings"
	"testing"
)

func TestCanonicalCreationRoutesRemainRootActions(t *testing.T) {
	root, _ := canonicalPreparationFixture(t, true)
	for name, run := range map[string]func(string, []string, *bytes.Buffer, *bytes.Buffer) int{
		"next": func(root string, args []string, out, errOut *bytes.Buffer) int {
			return RunNext(root, args, out, errOut)
		},
		"continuation": func(root string, args []string, out, errOut *bytes.Buffer) int {
			return RunCheckContinuation(root, args, out, errOut)
		},
	} {
		t.Run(name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(root, []string{"current16"}, &out, &errOut); code != 0 {
				t.Fatalf("%d: %s", code, &errOut)
			}
			if !strings.Contains(out.String(), "\ncreate: controller=current16 target=future17 parent=current16\n") || strings.Contains(out.String(), "\nlaunch: /rite-spec future17\n") {
				t.Fatalf("missing source must be root-created, not dispatched: %s", &out)
			}
		})
	}
}

func TestFrontierBatchPrioritizesWritersThenCreation(t *testing.T) {
	var out bytes.Buffer
	writeFrontierBatch(&out, []string{
		"frontier-concurrent: /rite-spec b",
		"frontier-create: controller=a target=c parent=b",
		"frontier-picks: a/SLICE-001",
	})
	if got, want := out.String(), "picks: a/SLICE-001\ncreate: controller=a target=c parent=b\nlaunch: /rite-spec b\n"; got != want {
		t.Fatalf("batch priority: %q, want %q", got, want)
	}
}
