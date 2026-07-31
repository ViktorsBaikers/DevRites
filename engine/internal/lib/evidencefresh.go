package lib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/workflow"
)

// EvidenceFresh verifies that proof, review, and Seal bind the current candidate.
// The call surface is retained for the final Seal aggregate.
func EvidenceFresh(root string, args []string, stdout, stderr io.Writer) int {
	slug := ""
	if len(args) > 0 {
		slug = args[0]
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	workDir := featureDir(root, slug)
	if slug == "" || !isDir(workDir) {
		name := slug
		if name == "" {
			name = "<unset>"
		}
		fmt.Fprintf(stderr, "evidence-fresh: no active workspace (slug=%s)\n", name)
		return 5
	}

	identity, err := loadCandidateIdentity(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "evidence-fresh: BLOCKED: %v. Refresh the candidate and re-run %s.\n", err, workflow.ForVerb("prove").Both())
		return 3
	}
	artifacts := []string{"evidence.md", "review.md", "seal.md"}
	browser := filepath.Join(workDir, "browser-evidence.md")
	if _, err := os.Lstat(browser); err == nil {
		artifacts = append(artifacts, "browser-evidence.md")
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "evidence-fresh: BLOCKED: inspect browser-evidence.md: %v\n", err)
		return 3
	}
	for _, name := range artifacts {
		if err := requireCandidateDigest(filepath.Join(workDir, name), identity.digest); err != nil {
			fmt.Fprintf(stderr, "evidence-fresh: BLOCKED: %s: %v. Record exactly one current candidate digest and retry.\n", name, err)
			return 3
		}
	}
	fmt.Fprintf(stdout, "evidence-fresh: OK: candidate digest %s matches evidence, review, and seal.\n", identity.digest)
	return 0
}

func requireCandidateDigest(name, want string) error {
	content, err := readBoundedRegularFile(name, maxCandidateArtifactBytes)
	if err != nil {
		return err
	}
	var tagged []string
	for _, line := range strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n") {
		if strings.Contains(line, "Candidate SHA-256:") {
			tagged = append(tagged, line)
		}
	}
	switch len(tagged) {
	case 0:
		return errors.New("candidate digest line is missing")
	case 1:
	default:
		return errors.New("candidate digest line is duplicate")
	}
	const prefix = "Candidate SHA-256: "
	line := tagged[0]
	if !strings.HasPrefix(line, prefix) || len(line) != len(prefix)+64 || !lowerHexDigest(line[len(prefix):]) {
		return errors.New("candidate digest line is malformed")
	}
	if line[len(prefix):] != want {
		return errors.New("candidate digest line does not match current candidate digest")
	}
	return nil
}

func lowerHexDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
