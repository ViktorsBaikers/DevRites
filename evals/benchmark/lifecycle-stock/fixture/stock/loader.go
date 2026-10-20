package stock

import (
	"encoding/json"
	"os"
)

// Load builds a store from a JSON object of item quantities.
func Load(path string) *Store {
	initial := map[string]int{}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &initial)
	}
	return New(initial)
}
