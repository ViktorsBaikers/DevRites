package main_test

import "testing"

// TestParityAnalyze checks `analyze`'s coverage/consistency report against the
// golden snapshots for each spec↔tasks fixture.
func TestParityAnalyze(t *testing.T) {
	work := t.TempDir()

	// covered: every spec AC has a slice that Satisfies it → clear (exit 0).
	writeFeatureFile(t, work, "covered", "spec.md", "# spec\n- [AC1] must foo\n")
	writeFeatureFile(t, work, "covered", "tasks.md", "## Slice 1: foo\nSatisfies: AC1\n")

	// uncovered: a spec AC no slice satisfies → CRITICAL (exit 1).
	writeFeatureFile(t, work, "uncovered", "spec.md", "# spec\n- [AC1] must foo\n- [AC2] must bar\n")
	writeFeatureFile(t, work, "uncovered", "tasks.md", "## Slice 1: foo\nSatisfies: AC1\n")

	// dangling: a slice Satisfies an AC the spec never defines → CRITICAL (exit 1).
	writeFeatureFile(t, work, "dangling", "spec.md", "# spec\n- [AC1] must foo\n")
	writeFeatureFile(t, work, "dangling", "tasks.md", "## Slice 1: foo\nSatisfies: AC1, AC9\n")

	// noac: spec has no [ACn] ids → warn, nothing dangling/orphan → clear (exit 0).
	writeFeatureFile(t, work, "noac", "spec.md", "# spec\njust prose, no acceptance ids\n")
	writeFeatureFile(t, work, "noac", "tasks.md", "# tasks\njust prose, no slices\n")

	// orphan: a slice with no Satisfies line → warn, still clear (exit 0).
	writeFeatureFile(t, work, "orphan", "spec.md", "# spec\n- [AC1] must foo\n")
	writeFeatureFile(t, work, "orphan", "tasks.md",
		"## Slice 1: foo\nSatisfies: AC1\n## Slice 2: bar\n- does stuff\n")

	// order: byte-order sort must place AC10 before AC2 (LC_ALL=C), all covered.
	writeFeatureFile(t, work, "order", "spec.md", "# spec\n- [AC1] a\n- [AC2] b\n- [AC10] c\n")
	writeFeatureFile(t, work, "order", "tasks.md", "## Slice 1: all\nSatisfies: AC1, AC2, AC10\n")

	// ghost: slug with no workspace → need spec.md+tasks.md (exit 2, empty stdout).

	for _, slug := range []string{"covered", "uncovered", "dangling", "noac", "orphan", "order", "ghost"} {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  []string{"analyze", slug},
		}
		t.Run("slug="+slug, func(t *testing.T) { c.assertEqual(t) })
	}

	// no active workspace: no slug arg and no .devrites/ACTIVE → exit 2, empty stdout.
	noactive := t.TempDir()
	t.Run("noactive", func(t *testing.T) {
		(parityCase{
			workdir: noactive,
			env:     libRootEnv(noactive),
			goArgs:  []string{"analyze"},
		}).assertEqual(t)
	})
}
