// Package reason owns stable machine-readable decision identifiers.
package reason

import (
	"fmt"
	"sort"
)

// ID identifies one rule outcome independently of its human wording.
type ID string

const (
	GateReadinessPassed  ID = "DRV-GATE-READINESS-PASSED" // #nosec G101 -- stable decision ID, not a credential
	GateReadinessMissing ID = "DRV-GATE-READINESS-MISSING"
	GateReadinessStale   ID = "DRV-GATE-READINESS-STALE"
	GateSealPassed       ID = "DRV-GATE-SEAL-PASSED" // #nosec G101 -- stable decision ID, not a credential
	GateSealMissing      ID = "DRV-GATE-SEAL-MISSING"
)

var catalog = []ID{
	GateReadinessPassed,
	GateReadinessMissing,
	GateReadinessStale,
	GateSealPassed,
	GateSealMissing,
}

var known = func() map[ID]struct{} {
	out := make(map[ID]struct{}, len(catalog))
	for _, id := range catalog {
		if _, duplicate := out[id]; duplicate {
			panic("duplicate reason id " + id)
		}
		out[id] = struct{}{}
	}
	return out
}()

// Parse accepts only identifiers in the frozen catalog.
func Parse(value string) (ID, error) {
	id := ID(value)
	if _, ok := known[id]; !ok {
		return "", fmt.Errorf("unknown reason id %q", value)
	}
	return id, nil
}

// Known reports whether id belongs to the frozen catalog.
func Known(id ID) bool {
	_, ok := known[id]
	return ok
}

// All returns the catalog in lexical order for generators and tests.
func All() []ID {
	out := append([]ID(nil), catalog...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
