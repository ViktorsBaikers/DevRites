package main

import (
	"os"
	"strings"
)

// Hook control plane.
//
// Every wired hook has a canonical id — its `hook <name>` CLI verb (allow,
// orient, stop-gate, …). That id is the addressable handle for two orthogonal
// runtime controls, both read from the environment so an operator tunes safety
// vs speed without editing any JSON:
//
//   DEVRITES_HOOK_PROFILE=minimal|standard|strict   which hooks run at all
//   DEVRITES_DISABLED_HOOKS="stop-gate,a1-guard"     an explicit per-id kill list
//
// The profile is a floor: a hook runs when its tier is at or below the active
// profile AND it is not in the kill list. Everything else short-circuits to a
// clean fail-open no-op in cmdHook — identical to the inline `command -v … ||
// exit 0` guard, so a disabled hook is indistinguishable from an absent binary.
//
// A SEPARATE axis is enforce-vs-observe: the OBSERVE-by-default guards (a1-guard,
// stop-gate, reviewer-readonly) only ever log until their specific
// DEVRITES_*=enforce var is set. The `strict` profile is the ergonomic shortcut
// that flips ALL of them on at once (see hookEnforce), so a hardened session is
// one env var, not four.

// hookTier is the minimum profile level at which a hook is active.
type hookTier int

const (
	tierMinimal  hookTier = iota // orientation-only; never blocks or mutates flow
	tierStandard                 // the default working set — gates, guards, caches
	tierStrict                   // reserved for experimental/aggressive hooks
)

// hookRegistry maps each canonical hook id to the profile tier that first
// activates it. A name absent from the registry is treated as tierMinimal
// (always active) — fail-open: an unrecognised hook is never silently disabled
// by the profile mechanism.
var hookRegistry = map[string]hookTier{
	// minimal — orientation and auto-approval, which never interrupt work.
	"allow":           tierMinimal,
	"orient":          tierMinimal,
	"subagent-orient": tierMinimal,
	"cursor":          tierMinimal,
	"statusline":      tierMinimal,

	// standard — the default gates, guards, sentinels and caches.
	"a1-guard":          tierStandard,
	"stop-gate":         tierStandard,
	"reviewer-readonly": tierStandard,
	"wright-scope":      tierStandard,
	"redwatch":          tierStandard,
	"source-cache-pre":  tierStandard,
	"source-cache-post": tierStandard,
	"refresh-indexes":   tierStandard,
	"event":             tierStandard,
	"handoff-snapshot":  tierStandard,
}

// hookProfileTier resolves DEVRITES_HOOK_PROFILE to a tier, defaulting to
// standard on empty or unrecognised input (fail-open — a typo never silently
// strips the working set below default).
func hookProfileTier() hookTier {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DEVRITES_HOOK_PROFILE"))) {
	case "minimal":
		return tierMinimal
	case "strict":
		return tierStrict
	default:
		return tierStandard
	}
}

// hookActive reports whether a hook id may run under the current profile and
// kill list. It is the single gate consulted at the top of cmdHook.
func hookActive(name string) bool {
	if hookDisabled(name) {
		return false
	}
	tier, known := hookRegistry[name]
	if !known {
		return true // unknown id — never disabled by profile
	}
	return tier <= hookProfileTier()
}

// hookDisabled reports whether a hook id appears in the DEVRITES_DISABLED_HOOKS
// comma-separated kill list.
func hookDisabled(name string) bool {
	list := os.Getenv("DEVRITES_DISABLED_HOOKS")
	if list == "" {
		return false
	}
	for _, id := range strings.Split(list, ",") {
		if strings.TrimSpace(id) == name {
			return true
		}
	}
	return false
}

// hookEnforce reports whether an OBSERVE-default guard should block rather than
// log. It is true when the guard's own DEVRITES_*=enforce var is set, OR when
// the strict profile is active — strict enforces every guard at once.
func hookEnforce(specificVar string) bool {
	if os.Getenv(specificVar) == "enforce" {
		return true
	}
	return hookProfileTier() == tierStrict
}
