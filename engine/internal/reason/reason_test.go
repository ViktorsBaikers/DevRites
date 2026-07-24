package reason

import (
	"strings"
	"testing"
)

func TestCatalogIsKnownUniqueAndStableShaped(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("empty reason catalog")
	}
	for i, id := range all {
		if !Known(id) {
			t.Fatalf("catalog id %q is not known", id)
		}
		if !strings.HasPrefix(string(id), "DRV-") {
			t.Fatalf("reason id %q lacks DRV- prefix", id)
		}
		if i > 0 && all[i-1] >= id {
			t.Fatalf("catalog is not strictly sorted: %q then %q", all[i-1], id)
		}
	}
}

func TestParseRejectsUnregisteredAndFreeText(t *testing.T) {
	if got, err := Parse(string(GateSealMissing)); err != nil || got != GateSealMissing {
		t.Fatalf("Parse registered = %q, %v", got, err)
	}
	for _, raw := range []string{"", "DRV-UNKNOWN", "/absolute/repo", "operator-private-value"} {
		if _, err := Parse(raw); err == nil {
			t.Fatalf("Parse(%q) should fail", raw)
		}
	}
}
