package overhaul

import (
	"strings"
	"testing"
)

func TestUsageListsRecordsPrune(t *testing.T) {
	if !strings.Contains(Usage, "overhaul records prune <run> [--keep K]") {
		t.Fatalf("Usage omits records prune:\n%s", Usage)
	}
}
