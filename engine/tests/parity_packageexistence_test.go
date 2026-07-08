package main_test

import (
	"testing"
)

// TestParityPackageExistence sets up a git repo with a committed manifest, then a
// working-tree diff that adds a new require(), and checks the command's stdout +
// exit against the golden snapshot. Read-only w.r.t. git.
func TestParityPackageExistence(t *testing.T) {
	requireGit(t)

	// newRepo commits a package.json declaring lodash plus an empty source file.
	newRepo := func(t *testing.T, manifest bool) string {
		t.Helper()
		work := t.TempDir()
		initGitRepo(t, work)
		if manifest {
			writeFile(t, work, "package.json", "{\n  \"dependencies\": { \"lodash\": \"^4.0.0\" }\n}\n")
		}
		writeFile(t, work, "src/app.js", "// app\n")
		gitCommitAll(t, work, "baseline")
		makeFeatureDir(t, work, "feat")
		return work
	}
	runCase := func(t *testing.T, work string, goArgs ...string) {
		c := parityCase{
			workdir: work,
			env:     libRootEnv(work),
			goArgs:  append([]string{"package-existence"}, goArgs...),
		}
		c.assertEqual(t)
	}

	t.Run("declared", func(t *testing.T) {
		work := newRepo(t, true)
		writeFile(t, work, "src/app.js", "// app\nconst _ = require('lodash');\n")
		runCase(t, work, "feat")
	})
	t.Run("default-import-subpath", func(t *testing.T) {
		work := newRepo(t, true)
		writeFile(t, work, "src/app.js", "// app\nimport debounce from 'lodash/debounce';\n")
		runCase(t, work, "feat")
	})
	t.Run("undeclared", func(t *testing.T) {
		work := newRepo(t, true)
		writeFile(t, work, "src/app.js", "// app\nconst g = require('ghostpkg');\n")
		runCase(t, work, "feat")
	})
	t.Run("no-manifest", func(t *testing.T) {
		work := newRepo(t, false)
		writeFile(t, work, "src/app.js", "// app\nconst g = require('ghostpkg');\n")
		runCase(t, work, "feat")
	})
	t.Run("ghost-workspace", func(t *testing.T) {
		runCase(t, newRepo(t, true), "ghost")
	})
	t.Run("not-git", func(t *testing.T) {
		work := t.TempDir()
		makeFeatureDir(t, work, "feat")
		writeFile(t, work, "package.json", "{ \"dependencies\": { \"lodash\": \"^4.0.0\" } }\n")
		runCase(t, work, "feat")
	})
}
