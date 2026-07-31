package main_test

// Cross-cutting assertion: first-party packages do not import forbidden network packages.

import (
	"os/exec"
	"strings"
	"testing"
)

// TestFirstPartyMakesNoNetworkCalls asserts that no first-party Go package imports
// a forbidden network package. This keeps the engine a network-free control plane
// that makes zero model/API calls (PRD: "zero API").
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
				t.Errorf("first-party package %s imports forbidden network package %q", pkg, imp)
			}
		}
	}
}
