package secretpattern

import (
	"strings"
	"testing"
)

func TestRedactMasksCredentialsAndKeepsPlaceholders(t *testing.T) {
	token := "ghp_" + strings.Repeat("x", 36)
	got, n := Redact("token " + token + "\napi_key = example-placeholder\n")
	if n != 1 || strings.Contains(got, token) || !strings.Contains(got, "[REDACTED:github-token]") {
		t.Fatalf("got %q (%d masks)", got, n)
	}
	if !strings.Contains(got, "api_key = example-placeholder") {
		t.Fatal("a LOW placeholder is not a credential and must stay")
	}
}
