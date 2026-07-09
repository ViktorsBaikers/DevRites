package lib

import "testing"

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
