// Package reason owns stable machine-readable decision identifiers.
package reason

import (
	"fmt"
	"sort"
)

// ID identifies one rule outcome independently of its human wording.
type ID string

const (
	RootSelected ID = "DRV-ROOT-SELECTED"

	GateReadinessPassed  ID = "DRV-GATE-READINESS-PASSED" // #nosec G101 -- stable decision ID, not a credential
	GateReadinessMissing ID = "DRV-GATE-READINESS-MISSING"
	GateSealPassed       ID = "DRV-GATE-SEAL-PASSED" // #nosec G101 -- stable decision ID, not a credential
	GateSealMissing      ID = "DRV-GATE-SEAL-MISSING"

	HookDisabled                 ID = "DRV-HOOK-DISABLED"
	HookAllowApproved            ID = "DRV-HOOK-ALLOW-APPROVED"
	HookStopClear                ID = "DRV-HOOK-STOP-CLEAR"
	HookStopInputInvalid         ID = "DRV-HOOK-STOP-INPUT-INVALID"
	HookStopReentry              ID = "DRV-HOOK-STOP-REENTRY"
	HookStopWorkspaceUnavailable ID = "DRV-HOOK-STOP-WORKSPACE-UNAVAILABLE" // #nosec G101 -- stable decision ID, not a credential
	HookStopRed                  ID = "DRV-HOOK-STOP-RED"
	HookStopUnsurfacedHumanGate  ID = "DRV-HOOK-STOP-UNSURFACED-HUMAN-GATE"
	HookStopMissingProof         ID = "DRV-HOOK-STOP-MISSING-PROOF"
	HookReviewerReadonlyDenied   ID = "DRV-HOOK-REVIEWER-READONLY-DENIED"
	HookReviewerReadonlyObserved ID = "DRV-HOOK-REVIEWER-READONLY-OBSERVED"
	HookA1Denied                 ID = "DRV-HOOK-A1-DENIED"
	HookA1Observed               ID = "DRV-HOOK-A1-OBSERVED"
	HookWrightScopeDenied        ID = "DRV-HOOK-WRIGHT-SCOPE-DENIED"
	HookWrightScopeObserved      ID = "DRV-HOOK-WRIGHT-SCOPE-OBSERVED"
	HookWrightForbiddenDenied    ID = "DRV-HOOK-WRIGHT-FORBIDDEN-DENIED"
	HookForgeBindingDenied       ID = "DRV-HOOK-FORGE-BINDING-DENIED"
	HookIngestWarning            ID = "DRV-HOOK-INGEST-WARNING"

	GitAuthorityPending     ID = "DRV-GIT-AUTHORITY-PENDING"
	GitAuthorityGranted     ID = "DRV-GIT-AUTHORITY-GRANTED"
	GitAuthorityExpired     ID = "DRV-GIT-AUTHORITY-EXPIRED"
	GitAuthorityRefused     ID = "DRV-GIT-AUTHORITY-REFUSED"
	GitAuthorityReplayed    ID = "DRV-GIT-AUTHORITY-REPLAYED"
	GitAuthorityCorrupt     ID = "DRV-GIT-AUTHORITY-CORRUPT"
	GitAuthorityUnavailable ID = "DRV-GIT-AUTHORITY-UNAVAILABLE"
	GitWorkspaceUnavailable ID = "DRV-GIT-WORKSPACE-UNAVAILABLE"

	GitInputTooLarge          ID = "DRV-GIT-INPUT-TOO-LARGE"
	GitAmbiguousSyntax        ID = "DRV-GIT-AMBIGUOUS-SYNTAX"
	GitAmbiguousDynamic       ID = "DRV-GIT-AMBIGUOUS-DYNAMIC"
	GitAmbiguousAlias         ID = "DRV-GIT-AMBIGUOUS-ALIAS"
	GitAmbiguousGlobalOption  ID = "DRV-GIT-AMBIGUOUS-GLOBAL-OPTION"
	GitAmbiguousStdin         ID = "DRV-GIT-AMBIGUOUS-STDIN"
	GitDestructiveReset       ID = "DRV-GIT-DESTRUCTIVE-RESET"
	GitDestructiveClean       ID = "DRV-GIT-DESTRUCTIVE-CLEAN"
	GitDestructiveCheckout    ID = "DRV-GIT-DESTRUCTIVE-CHECKOUT"
	GitDestructiveRestore     ID = "DRV-GIT-DESTRUCTIVE-RESTORE"
	GitDestructiveSwitch      ID = "DRV-GIT-DESTRUCTIVE-SWITCH"
	GitDestructiveRemove      ID = "DRV-GIT-DESTRUCTIVE-RM"
	GitDestructiveBranch      ID = "DRV-GIT-DESTRUCTIVE-BRANCH"
	GitDestructiveTag         ID = "DRV-GIT-DESTRUCTIVE-TAG"
	GitDestructiveUpdateRef   ID = "DRV-GIT-DESTRUCTIVE-UPDATE-REF"
	GitDestructiveStash       ID = "DRV-GIT-DESTRUCTIVE-STASH"
	GitDestructiveReflog      ID = "DRV-GIT-DESTRUCTIVE-REFLOG"
	GitDestructivePrune       ID = "DRV-GIT-DESTRUCTIVE-PRUNE"
	GitDestructiveHistory     ID = "DRV-GIT-DESTRUCTIVE-HISTORY"
	GitDestructivePushForce   ID = "DRV-GIT-DESTRUCTIVE-PUSH-FORCE"
	GitDestructivePushDelete  ID = "DRV-GIT-DESTRUCTIVE-PUSH-DELETE"
	GitDestructiveWorktree    ID = "DRV-GIT-DESTRUCTIVE-WORKTREE"
	GitDestructiveRefDeletion ID = "DRV-GIT-DESTRUCTIVE-REF-DELETE"

	AgentNamed                  ID = "DRV-AGENT-NAMED"
	AgentGenericFallback        ID = "DRV-AGENT-GENERIC-FALLBACK"
	AgentInlineFallback         ID = "DRV-AGENT-INLINE-FALLBACK"
	AgentUnavailable            ID = "DRV-AGENT-UNAVAILABLE"
	AgentResultAccepted         ID = "DRV-AGENT-RESULT-ACCEPTED"
	AgentResultMalformed        ID = "DRV-AGENT-RESULT-MALFORMED"
	AgentResultStale            ID = "DRV-AGENT-RESULT-STALE"
	AgentResultIdentityMismatch ID = "DRV-AGENT-RESULT-IDENTITY-MISMATCH"
	AgentResultOutOfScope       ID = "DRV-AGENT-RESULT-OUT-OF-SCOPE"
)

var catalog = []ID{
	RootSelected,
	GateReadinessPassed,
	GateReadinessMissing,
	GateSealPassed,
	GateSealMissing,
	HookDisabled,
	HookAllowApproved,
	HookStopClear,
	HookStopInputInvalid,
	HookStopReentry,
	HookStopWorkspaceUnavailable,
	HookStopRed,
	HookStopUnsurfacedHumanGate,
	HookStopMissingProof,
	HookReviewerReadonlyDenied,
	HookReviewerReadonlyObserved,
	HookA1Denied,
	HookA1Observed,
	HookWrightScopeDenied,
	HookWrightScopeObserved,
	HookWrightForbiddenDenied,
	HookForgeBindingDenied,
	HookIngestWarning,
	GitAuthorityPending,
	GitAuthorityGranted,
	GitAuthorityExpired,
	GitAuthorityRefused,
	GitAuthorityReplayed,
	GitAuthorityCorrupt,
	GitAuthorityUnavailable,
	GitWorkspaceUnavailable,
	GitInputTooLarge,
	GitAmbiguousSyntax,
	GitAmbiguousDynamic,
	GitAmbiguousAlias,
	GitAmbiguousGlobalOption,
	GitAmbiguousStdin,
	GitDestructiveReset,
	GitDestructiveClean,
	GitDestructiveCheckout,
	GitDestructiveRestore,
	GitDestructiveSwitch,
	GitDestructiveRemove,
	GitDestructiveBranch,
	GitDestructiveTag,
	GitDestructiveUpdateRef,
	GitDestructiveStash,
	GitDestructiveReflog,
	GitDestructivePrune,
	GitDestructiveHistory,
	GitDestructivePushForce,
	GitDestructivePushDelete,
	GitDestructiveWorktree,
	GitDestructiveRefDeletion,
	AgentNamed,
	AgentGenericFallback,
	AgentInlineFallback,
	AgentUnavailable,
	AgentResultAccepted,
	AgentResultMalformed,
	AgentResultStale,
	AgentResultIdentityMismatch,
	AgentResultOutOfScope,
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
