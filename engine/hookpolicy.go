package main

import (
	"os"
	"strings"
)

// Hook names are also their canonical IDs. Two environment settings control
// whether a hook runs:
//
//   DEVRITES_HOOK_PROFILE=minimal|standard|strict   which hooks run at all
//   DEVRITES_DISABLED_HOOKS="stop-gate,a1-guard"     hook IDs to disable
//
// A hook runs when its tier is at or below the selected profile and its ID is
// not disabled. Disabled hooks return success without doing anything, just as
// they would if the binary were absent.
//
// Enforcement is separate from activation. Main-thread guards observe by
// default until their DEVRITES_*=enforce variable is set. The strict profile
// enforces every legacy observe path. reviewer-readonly and wright-scope always
// enforce policy for declared DevRites leaf agents.

// hookTier is the minimum profile level at which a hook is active.
type hookTier int

const (
	tierMinimal  hookTier = iota // orientation only; never blocks or changes flow
	tierStandard                 // default gates, guards, and caches
	tierStrict                   // experimental or aggressive hooks
)

type hookDefinition struct {
	tier     hookTier
	canBlock bool
}

// hookRegistry records each hook's tier and whether it can block. The blocker
// audit derives its cases from canBlock, so a new blocker must cover reentry,
// malformed input, bounds, and the disabled path. Unknown hook names stay
// active instead of being silently disabled.
var hookRegistry = map[string]hookDefinition{
	// Minimal hooks provide orientation and automatic approval without
	// interrupting work.
	"allow":           {tier: tierMinimal},
	"orient":          {tier: tierMinimal},
	"subagent-orient": {tier: tierMinimal},
	"cursor":          {tier: tierMinimal},
	"statusline":      {tier: tierMinimal},

	// Standard hooks add the default gates, guards, sentinels, and caches.
	"a1-guard":          {tier: tierStandard, canBlock: true},
	"git-guard":         {tier: tierStandard, canBlock: true},
	"stop-gate":         {tier: tierStandard, canBlock: true},
	"reviewer-readonly": {tier: tierStandard, canBlock: true},
	"wright-scope":      {tier: tierStandard, canBlock: true},
	"redwatch":          {tier: tierStandard},
	"source-cache-pre":  {tier: tierStandard},
	"source-cache-post": {tier: tierStandard},
	"refresh-indexes":   {tier: tierStandard},
	"event":             {tier: tierStandard},
	"auq":               {tier: tierStandard},
	"handoff-snapshot":  {tier: tierStandard},
}

// hookProfileTier resolves DEVRITES_HOOK_PROFILE. Empty or unrecognized input
// uses the standard tier so a typo cannot disable the default hooks.
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

// hookActive reports whether the current profile and disabled list allow a hook
// ID to run. cmdHook checks it before dispatch.
func hookActive(name string) bool {
	if hookDisabled(name) {
		return false
	}
	def, known := hookRegistry[name]
	if !known {
		return true // Profiles do not disable unknown IDs.
	}
	return def.tier <= hookProfileTier()
}

// hookDisabled reports whether DEVRITES_DISABLED_HOOKS contains the hook ID.
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

// hookEnforce reports whether a guard that normally observes should block. Its
// own enforce variable or the strict profile enables blocking.
func hookEnforce(specificVar string) bool {
	if os.Getenv(specificVar) == "enforce" {
		return true
	}
	return hookProfileTier() == tierStrict
}
