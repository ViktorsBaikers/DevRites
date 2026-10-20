package lib

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// orient / observe summary print one compact JSON line; indentation only cost
// root context.
func TestWriteObserveSummaryJSONIsOneCompactLine(t *testing.T) {
	root, _ := newWorkspace(t, "build")
	var out bytes.Buffer
	if err := WriteObserveSummaryJSON(root, "feat", &out); err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSuffix(out.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Fatalf("want one compact line, got %q", out.String())
	}
	var summary ObserveSummary
	if err := json.Unmarshal(out.Bytes(), &summary); err != nil || summary.Slug != "feat" || summary.Phase != "build" {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
}
