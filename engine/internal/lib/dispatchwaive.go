package lib

import (
	"fmt"
	"io"
)

var dispatchWaiverReasons = map[string]struct{}{
	"human-gate-before-dispatch": {},
	"no-active-workspace":        {},
	"no-eligible-work":           {},
	"readiness-failed":           {},
	"wrong-phase":                {},
}

func ValidDispatchWaiver(reason string) bool {
	_, ok := dispatchWaiverReasons[reason]
	return ok
}

// DispatchWaive validates the reason token used by the agent-dispatch hook.
// The hook records the successful PostToolUse receipt; this command deliberately
// owns no separate state.
func DispatchWaive(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || !ValidDispatchWaiver(args[0]) {
		fmt.Fprintln(stderr, "usage: devrites-engine dispatch-waive <human-gate-before-dispatch|no-active-workspace|no-eligible-work|readiness-failed|wrong-phase>")
		return 2
	}
	fmt.Fprintf(stdout, "dispatch-waive: accepted %s\n", args[0])
	return 0
}
