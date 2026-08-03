package main_test

// Cross-cutting assertion: release acquisition is the only engine network boundary.

import (
	"os/exec"
	"strings"
	"testing"
)

// TestNetworkImportsStayInReleaseBoundary keeps update networking isolated from
// workspace state, policy, proof, and installation logic.
func TestNetworkImportsStayInReleaseBoundary(t *testing.T) {
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
	allowed := map[string]bool{"github.com/devrites/devrites/internal/release": true}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkg, imports := fields[0], fields[1:]
		for _, imp := range imports {
			if forbidden[imp] && !allowed[pkg] {
				t.Errorf("first-party package %s imports forbidden network package %q", pkg, imp)
			}
		}
	}
}
