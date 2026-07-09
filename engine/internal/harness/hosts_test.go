package harness

import (
	"encoding/json"
	"os"
	"testing"
)

func TestHostsDescriptorCoversSupportedHarnesses(t *testing.T) {
	data, err := os.ReadFile("hosts.json")
	if err != nil {
		t.Fatal(err)
	}
	var hosts map[string]struct {
		ArtifactLayout []string `json:"artifactLayout"`
		HookSurface    []string `json:"hookSurface"`
		SupportTier    string   `json:"supportTier"`
	}
	if err := json.Unmarshal(data, &hosts); err != nil {
		t.Fatal(err)
	}
	for _, h := range []Harness{Claude, Codex} {
		got := hosts[string(h)]
		if len(got.ArtifactLayout) == 0 || len(got.HookSurface) == 0 || got.SupportTier == "" {
			t.Fatalf("hosts.json missing descriptor data for %s: %+v", h, got)
		}
	}
}
