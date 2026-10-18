package lib

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateOmissionsComparesCheckpointsWithManifest(t *testing.T) {
	listedA := candidateTestRow{state: "present", path: "a.txt", slice: "S-1", reason: "Listed."}
	project, root := writeCandidateTestWorkspace(t, candidateTestManifest(listedA))
	initDupRepo(t, project)
	writeCandidateTestFile(t, project, "pre.txt", "owner file\n")
	runDupGit(t, project, "add", "pre.txt")
	runDupGit(t, project, "commit", "-q", "-m", "chore: owner baseline")
	if got, err := CandidateOmissions(root, "feature"); err != nil || len(got) != 0 {
		t.Fatalf("no checkpoint yet must report nothing: %v %v", got, err)
	}

	writeCandidateTestFile(t, project, "a.txt", "a\n")
	writeCandidateTestFile(t, project, "b.txt", "b\n")
	runDupGit(t, project, "add", "a.txt", "b.txt")
	runDupGit(t, project, "commit", "-q", "-m", "WIP(feature): first slice")
	writeCandidateTestFile(t, project, "pre.txt", "uncommitted owner edit\n")

	got, err := CandidateOmissions(root, "feature")
	if err != nil || strings.Join(got, ",") != "b.txt" {
		t.Fatalf("want only the checkpointed, unlisted b.txt (owner files never count): %v %v", got, err)
	}
	if err := CheckCandidateCoverage(root, "feature"); err == nil || !strings.Contains(err.Error(), "omits 1 path(s)") || !strings.Contains(err.Error(), "b.txt") {
		t.Fatalf("coverage must block naming b.txt: %v", err)
	}

	listedB := candidateTestRow{state: "present", path: "b.txt", slice: "S-1", reason: "Listed."}
	if err := os.WriteFile(filepath.Join(root, "work", "feature", "touched-files.md"), []byte(candidateTestManifest(listedA, listedB)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckCandidateCoverage(root, "feature"); err != nil {
		t.Fatalf("a complete manifest must pass: %v", err)
	}
}

func TestLintReviewCandidateNeedsFrozenDiff(t *testing.T) {
	_, root := writeCandidateTestWorkspace(t, candidateTestManifest())
	packet := writePacket(t, "Scope: SLICE-001 against AC-001.\n")
	if err := lintReviewCandidate(root, "feature", packet); err == nil || !strings.Contains(err.Error(), "frozen candidate diff") {
		t.Fatalf("a packet without a diff must be refused: %v", err)
	}
	diff := filepath.Join(root, "work", "feature", "packets", "review.diff")
	if err := os.MkdirAll(filepath.Dir(diff), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(diff, []byte("diff --git a/a b/a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	packet = writePacket(t, "Scope: SLICE-001.\nDiff: packets/review.diff\n")
	if err := lintReviewCandidate(root, "feature", packet); err != nil {
		t.Fatalf("a packet naming an existing diff must pass: %v", err)
	}
}
