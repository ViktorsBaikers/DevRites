package lib

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsTestFileClassifiesTests(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Test files, by basename marker.
		{"internal/lib/reconcile_test.go", true},
		{"src/parser.test.ts", true},
		{"src/parser.spec.tsx", true},
		{"src/parser_spec.rb", true},
		{"lib/thing.tests.js", true},
		{"java/com/acme/OrderTest.java", true},
		{"java/com/acme/OrderTests.java", true},
		{"csharp/OrderSpec.cs", true},
		{"test_parser.py", true},
		{"python/pkg/test_parser.py", true},
		{"ruby/spec_helper.rb", true},

		// Test files, by directory segment.
		{"tests/integration/flow.go", true},
		{"src/__tests__/button.tsx", true},
		{"crate/test/harness.rs", true},
		{"app/spec/models/user_spec.rb", true},
		{"api/specs/contract.py", true},

		// Regression: ordinary source whose path merely *contains* "test" or
		// "spec" as a substring must not be classified as a test. Misclassifying
		// these trips the WEAKENED gate when they are deleted, and hides them
		// from the verification-gap advisory when they change.
		{"internal/latest/handler.go", false},
		{"internal/respect/config.go", false},
		{"internal/contest/rules.go", false},
		{"pkg/digest/sha.go", false},
		{"src/greatest_hits.ts", false},
		{"src/special.tsx", false},
		{"cmd/protest.go", false},
		{"internal/attestation/verify.go", false},

		// Ordinary source with no marker at all.
		{"internal/lib/reconcile.go", false},
		{"README.md", false},
	}

	for _, tc := range cases {
		if got := isTestFile(tc.path); got != tc.want {
			t.Errorf("isTestFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIsSourceFileRecognisesCode(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"internal/lib/reconcile.go", true},
		{"src/app.tsx", true},
		{"pkg/mod.rs", true},
		{"README.md", false},
		{"package-lock.json", false},
		{"config.yaml", false},
	}
	for _, tc := range cases {
		if got := isSourceFile(tc.path); got != tc.want {
			t.Errorf("isSourceFile(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestIntegrityDetectsDeletionFromDirtySliceBaseline(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	testPath := "tests/preexisting_test.go"
	writeFile(t, filepath.Join(gitRoot, testPath), "package tests\n\nfunc TestBaseline(t *testing.T) { t.Fatal(\"assert\") }\n")
	writeWrightAllowlist(t, root, "feat", testPath)
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(gitRoot, testPath)); err != nil {
		t.Fatal(err)
	}
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("reconcile check = %d, want 0\n%s", code, out)
	}

	var stdout, stderr bytes.Buffer
	code := TestIntegrity(root, []string{"feat"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("test-integrity = %d, want 3\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), testPath+": test file DELETED") {
		t.Fatalf("deleted dirty-baseline test missing from finding:\n%s", stderr.String())
	}
}

func TestIntegrityFailsClosedOnPartialReconcileBaseline(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(featureDir(root, "feat"), reconcileBaseName), "deadbeef\n")

	var stdout, stderr bytes.Buffer
	code := TestIntegrity(root, []string{"feat"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("test-integrity = %d, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "partial lifecycle") {
		t.Fatalf("missing fail-closed lifecycle diagnostic:\n%s", stderr.String())
	}
}

func TestIntegrityAdvisesWhenSourceChangesWithoutTests(t *testing.T) {
	gitRoot := newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(gitRoot, "src", "feature.go"), "package src\n")

	var stdout, stderr bytes.Buffer
	code := TestIntegrity(root, []string{"feat"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("test-integrity = %d, want 0\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "source file(s) changed, 0 test file(s) touched") {
		t.Fatalf("missing verification-gap advisory:\n%s", stdout.String())
	}
}
