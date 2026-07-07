package main_test

// Cross-cutting assertions the spec calls out explicitly: the engine makes zero
// network calls, and the inline fail-open guard no-ops when the binary is absent.

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// TestFirstPartyMakesNoNetworkCalls asserts that no first-party package imports a
// network client. Transitive dependencies (e.g. the pure-Go SQLite driver) may
// pull in net, but OUR code never does — the engine is a local control plane
// that makes zero model or network calls (PRD: "zero API"). This is the
// executable form of that guarantee.
func TestFirstPartyMakesNoNetworkCalls(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain unavailable")
	}
	out, err := exec.Command("go", "list", "-f",
		"{{.ImportPath}} {{join .Imports \" \"}}",
		"github.com/devrites/devrites/...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	forbidden := map[string]bool{"net": true, "net/http": true, "net/rpc": true, "net/smtp": true}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg, imports := fields[0], fields[1:]
		for _, imp := range imports {
			if forbidden[imp] {
				t.Errorf("first-party package %s imports network package %q — the engine must make no network calls", pkg, imp)
			}
		}
	}
}

// TestFailOpenGuardNoOpsWhenBinaryMissing exercises the inline POSIX guard the
// hooks.json entries wrap every devrites hook in. With no devrites on PATH the
// guard must be a silent no-op (exit 0), so a teammate without the binary
// installed is never wedged.
func TestFailOpenGuardNoOpsWhenBinaryMissing(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh unavailable")
	}
	const guard = `command -v devrites >/dev/null 2>&1 && devrites hook orient --harness=claude || exit 0`
	cmd := exec.Command("sh", "-c", guard)
	cmd.Env = []string{"PATH="} // empty PATH → devrites is not found
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("guard did not exit 0 with the binary absent: %v\n%s", err, out)
	}
	if len(bytes.TrimSpace(out)) != 0 {
		t.Errorf("guard produced output with the binary absent: %q", out)
	}
}
