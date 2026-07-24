package lib

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageExistenceJavaScriptWorkspaces(t *testing.T) {
	tests := []struct {
		name        string
		sourcePath  string
		workingDir  string
		importLine  string
		files       map[string]string
		wantCode    int
		wantFinding string
	}{
		{
			name:       "nested workspace",
			sourcePath: "frontend/src/example.ts",
			workingDir: "frontend",
			importLine: `import { Dialog } from "@ark-ui/svelte/dialog";`,
			files: map[string]string{
				"frontend/package.json": `{"dependencies":{"@ark-ui/svelte":"^5.22.1"}}`,
			},
		},
		{
			name:       "scoped package subpath",
			sourcePath: "app/src/example.ts",
			importLine: `import value from "@scope/package/subpath";`,
			files: map[string]string{
				"app/package.json": `{"devDependencies":{"@scope/package":"1.0.0"}}`,
			},
		},
		{
			name:       "wildcard alias",
			sourcePath: "frontend/src/example.ts",
			importLine: `import Button from "$lib/components/button";`,
			files: map[string]string{
				"frontend/package.json":              `{}`,
				"frontend/tsconfig.json":             "{\n  // SvelteKit generates the inherited aliases.\n  \"extends\": \"./.svelte-kit/tsconfig.json\"\n}",
				"frontend/.svelte-kit/tsconfig.json": `{"compilerOptions":{"paths":{"$lib/*":["../src/lib/*"]}}}`,
			},
		},
		{
			name:       "exact alias",
			sourcePath: "frontend/src/example.ts",
			importLine: `import library from "$lib";`,
			files: map[string]string{
				"frontend/package.json":  `{}`,
				"frontend/tsconfig.json": `{"compilerOptions":{"paths":{"$lib":["./src/lib"]}}}`,
			},
		},
		{
			name:       "package shaped alias",
			sourcePath: "frontend/src/example.svelte",
			importLine: `import { css } from "styled-system/css";`,
			files: map[string]string{
				"frontend/package.json":  `{}`,
				"frontend/jsconfig.json": `{"compilerOptions":{"paths":{"styled-system/*":["./styled-system/*"]}}}`,
			},
		},
		{
			name:        "undeclared dependency",
			sourcePath:  "frontend/src/example.ts",
			importLine:  `import missing from "definitely-missing-package";`,
			files:       map[string]string{"frontend/package.json": `{}`},
			wantCode:    3,
			wantFinding: "definitely-missing-package",
		},
		{
			name:       "sibling isolation",
			sourcePath: "workspace-a/src/example.ts",
			importLine: `import sibling from "sibling-only-package";`,
			files: map[string]string{
				"workspace-a/package.json": `{}`,
				"workspace-b/package.json": `{"dependencies":{"sibling-only-package":"1.0.0"}}`,
			},
			wantCode:    3,
			wantFinding: "sibling-only-package",
		},
		{
			name:       "malformed config does not allow alias",
			sourcePath: "frontend/src/example.ts",
			importLine: `import { css } from "styled-system/css";`,
			files: map[string]string{
				"frontend/package.json":  `{}`,
				"frontend/tsconfig.json": `{"compilerOptions":{"paths":{"styled-system/*":["./styled-system/*"]}}`,
			},
			wantCode:    3,
			wantFinding: "styled-system",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := t.TempDir()
			writePackageExistenceFile(t, repo, "Cargo.toml", "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n")
			writePackageExistenceFile(t, repo, test.sourcePath, "// baseline\n")
			for path, content := range test.files {
				writePackageExistenceFile(t, repo, path, content+"\n")
			}
			gitPackageExistence(t, repo, "init", "-q")
			gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
			gitPackageExistence(t, repo, "config", "user.name", "Test")
			gitPackageExistence(t, repo, "add", ".")
			gitPackageExistence(t, repo, "commit", "-qm", "baseline")
			if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
				t.Fatal(err)
			}
			writePackageExistenceFile(t, repo, test.sourcePath, "// baseline\n"+test.importLine+"\n")
			t.Chdir(filepath.Join(repo, test.workingDir))

			var stdout, stderr bytes.Buffer
			gotCode := PackageExistence(repo, []string{"feat"}, &stdout, &stderr)
			if gotCode != test.wantCode {
				t.Fatalf("code = %d, want %d\nstdout: %s\nstderr: %s", gotCode, test.wantCode, stdout.String(), stderr.String())
			}
			if test.wantFinding != "" && !strings.Contains(stderr.String(), test.wantFinding) {
				t.Fatalf("stderr does not name %q: %s", test.wantFinding, stderr.String())
			}
		})
	}
}

func TestPackageExistenceCSSImports(t *testing.T) {
	tests := []struct {
		name        string
		imports     string
		manifest    string
		wantCode    int
		wantFinding string
	}{
		{
			name:     "declared multi import ignores comment prefix",
			imports:  "/* @import \"comment-only\"; */ @import \"tailwindcss\"; @import \"./theme.css\";",
			manifest: `{"dependencies":{"tailwindcss":"^4.1.0"}}`,
		},
		{
			name:        "undeclared url import",
			imports:     `@import url("missing-theme");`,
			manifest:    `{}`,
			wantCode:    3,
			wantFinding: "missing-theme",
		},
		{
			name:     "remote imports ignored",
			imports:  `@import url("https://cdn.example.invalid/theme.css"); @import "//cdn.example.invalid/other.css";`,
			manifest: `{}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := t.TempDir()
			writePackageExistenceFile(t, repo, "package.json", test.manifest+"\n")
			writePackageExistenceFile(t, repo, "styles/app.css", "/* baseline */\n")
			gitPackageExistence(t, repo, "init", "-q")
			gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
			gitPackageExistence(t, repo, "config", "user.name", "Test")
			gitPackageExistence(t, repo, "add", ".")
			gitPackageExistence(t, repo, "commit", "-qm", "baseline")
			if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
				t.Fatal(err)
			}
			writePackageExistenceFile(t, repo, "styles/app.css", "/* baseline */\n"+test.imports+"\n")
			t.Chdir(repo)

			var stdout, stderr bytes.Buffer
			gotCode := PackageExistence(repo, []string{"feat"}, &stdout, &stderr)
			if gotCode != test.wantCode {
				t.Fatalf("code = %d, want %d\nstdout: %s\nstderr: %s", gotCode, test.wantCode, stdout.String(), stderr.String())
			}
			if test.wantFinding != "" && !strings.Contains(stderr.String(), test.wantFinding) {
				t.Fatalf("stderr does not name %q: %s", test.wantFinding, stderr.String())
			}
		})
	}
}

func TestPackageExistenceCSSResolvesBareSiblingPath(t *testing.T) {
	repo := t.TempDir()
	writePackageExistenceFile(t, repo, "package.json", "{}\n")
	writePackageExistenceFile(t, repo, "styles/app.css", "/* baseline */\n")
	writePackageExistenceFile(t, repo, "styles/theme.css", ":root { color-scheme: dark; }\n")
	gitPackageExistence(t, repo, "init", "-q")
	gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
	gitPackageExistence(t, repo, "config", "user.name", "Test")
	gitPackageExistence(t, repo, "add", ".")
	gitPackageExistence(t, repo, "commit", "-qm", "baseline")
	if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePackageExistenceFile(t, repo, "styles/app.css", "/* baseline */\n@import \"theme.css\";\n")
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	if code := PackageExistence(repo, []string{"feat"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

func TestPackageExistenceCSSHonorsBaselineCommentContext(t *testing.T) {
	repo := t.TempDir()
	baseline := "/* comment opened on an unchanged line\nunchanged content\n*/\n"
	current := "/* comment opened on an unchanged line\n@import \"comment-only-package\";\nunchanged content\n*/\n"
	writePackageExistenceFile(t, repo, "package.json", "{}\n")
	writePackageExistenceFile(t, repo, "styles/app.css", baseline)
	gitPackageExistence(t, repo, "init", "-q")
	gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
	gitPackageExistence(t, repo, "config", "user.name", "Test")
	gitPackageExistence(t, repo, "add", ".")
	gitPackageExistence(t, repo, "commit", "-qm", "baseline")
	if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePackageExistenceFile(t, repo, "styles/app.css", current)
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	if code := PackageExistence(repo, []string{"feat"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
}

func TestPackageExistenceScansUntrackedSourceFiles(t *testing.T) {
	repo := t.TempDir()
	writePackageExistenceFile(t, repo, "package.json", "{}\n")
	gitPackageExistence(t, repo, "init", "-q")
	gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
	gitPackageExistence(t, repo, "config", "user.name", "Test")
	gitPackageExistence(t, repo, "add", ".")
	gitPackageExistence(t, repo, "commit", "-qm", "baseline")
	if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePackageExistenceFile(t, repo, "src/new.ts", `import missing from "untracked-missing-package";`+"\n")
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	gotCode := PackageExistence(repo, []string{"feat"}, &stdout, &stderr)
	if gotCode != 3 {
		t.Fatalf("code = %d, want 3\nstdout: %s\nstderr: %s", gotCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "untracked-missing-package") {
		t.Fatalf("stderr does not name untracked dependency: %s", stderr.String())
	}
}

func TestPackageExistenceScansUntrackedFilesAddedAfterSliceSnapshot(t *testing.T) {
	repo := newGitRepo(t)
	root := workspace(t, "feat")
	writePackageExistenceFile(t, repo, "package.json", "{}\n")
	commitAll(t, repo, "manifest")
	writeWrightAllowlist(t, root, "feat", "src/new.ts")
	if code, out := runReconcile(t, root, "snapshot", "feat"); code != 0 {
		t.Fatalf("snapshot = %d, want 0\n%s", code, out)
	}
	writePackageExistenceFile(t, repo, "src/new.ts", `import missing from "slice-untracked-package";`+"\n")
	if code, out := runReconcile(t, root, "check", "feat"); code != 0 {
		t.Fatalf("reconcile check = %d, want 0\n%s", code, out)
	}

	var stdout, stderr bytes.Buffer
	code := PackageExistence(root, []string{"feat"}, &stdout, &stderr)
	if code != 3 {
		t.Fatalf("code = %d, want 3\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "slice-untracked-package") {
		t.Fatalf("stderr does not name slice dependency: %s", stderr.String())
	}
}

func TestPackageExistenceFailsClosedOnPartialReconcileBaseline(t *testing.T) {
	newGitRepo(t)
	root := workspace(t, "feat")
	writeFile(t, filepath.Join(featureDir(root, "feat"), reconcileBaseName), "deadbeef\n")

	var stdout, stderr bytes.Buffer
	code := PackageExistence(root, []string{"feat"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code = %d, want 2\nstdout: %s\nstderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "partial lifecycle") {
		t.Fatalf("missing fail-closed lifecycle diagnostic:\n%s", stderr.String())
	}
}

func TestPackageExistenceNonJavaScriptWorkspaces(t *testing.T) {
	tests := []struct {
		name        string
		sourcePath  string
		addedSource string
		files       map[string]string
		wantCode    int
		wantFinding string
	}{
		{
			name:        "go import block rejects undeclared module",
			sourcePath:  "main.go",
			addedSource: "package main\n\nimport (\n\t\"definitely.invalid/missing\"\n)\n",
			files:       map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n"},
			wantCode:    3,
			wantFinding: "definitely.invalid/missing",
		},
		{
			name:        "go modules do not collide by hostname",
			sourcePath:  "main.go",
			addedSource: "package main\n\nimport \"github.com/evil/missing\"\n",
			files:       map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n\nrequire github.com/good/pkg v1.0.0\n"},
			wantCode:    3,
			wantFinding: "github.com/evil/missing",
		},
		{
			name:        "nested go module declaration",
			sourcePath:  "backend/main.go",
			addedSource: "package main\n\nimport \"github.com/good/pkg/sub\"\n",
			files: map[string]string{
				"Cargo.toml":     "[package]\nname = \"root\"\nversion = \"0.1.0\"\n",
				"backend/go.mod": "module example.com/backend\n\ngo 1.26\n\nrequire github.com/good/pkg v1.0.0\n",
			},
		},
		{
			name:        "go standard library subpackage",
			sourcePath:  "main.go",
			addedSource: "package main\n\nimport \"encoding/json\"\n",
			files:       map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n"},
		},
		{
			name:        "go standard library typo remains undeclared",
			sourcePath:  "main.go",
			addedSource: "package main\n\nimport \"encoding/josn\"\n",
			files:       map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n"},
			wantCode:    3,
			wantFinding: "encoding/josn",
		},
		{
			name:        "malformed go manifest does not declare import",
			sourcePath:  "main.go",
			addedSource: "package main\n\nimport \"github.com/missing/pkg\"\n",
			files:       map[string]string{"go.mod": "module example.com/app\n\ngo 1.26\n\nrequire (\n  github.com/missing/pkg v1.0.0\n"},
			wantCode:    3,
			wantFinding: "github.com/missing/pkg",
		},
		{
			name:        "rust rejects undeclared crate",
			sourcePath:  "src/main.rs",
			addedSource: "use definitely_missing_crate::Thing;\n",
			files:       map[string]string{"Cargo.toml": "[package]\nname = \"fixture\"\nversion = \"0.1.0\"\n\n[dependencies]\n"},
			wantCode:    3,
			wantFinding: "definitely_missing_crate",
		},
		{
			name:        "nested rust dependency and hyphen normalization",
			sourcePath:  "crates/app/src/main.rs",
			addedSource: "use serde_json::Value;\n",
			files: map[string]string{
				"Cargo.toml":            "[workspace]\nmembers = [\"crates/app\"]\n",
				"crates/app/Cargo.toml": "[package]\nname = \"app\"\nversion = \"0.1.0\"\n\n[dependencies.serde-json]\nversion = \"1\"\n",
			},
		},
		{
			name:        "python dotted import uses top-level module",
			sourcePath:  "app.py",
			addedSource: "from requests.sessions import Session\n",
			files:       map[string]string{"requirements.txt": "requests==2.32.0\n"},
		},
		{
			name:        "python standard library module",
			sourcePath:  "app.py",
			addedSource: "import asyncio\n",
			files:       map[string]string{"requirements.txt": "requests==2.32.0\n"},
		},
		{
			name:        "nested python manifest declaration",
			sourcePath:  "backend/app.py",
			addedSource: "import requests\n",
			files: map[string]string{
				"Cargo.toml":               "[package]\nname = \"root\"\nversion = \"0.1.0\"\n",
				"backend/requirements.txt": "requests==2.32.0\n",
			},
		},
		{
			name:        "pyproject dependency declaration",
			sourcePath:  "service/app.py",
			addedSource: "import requests\n",
			files: map[string]string{
				"service/pyproject.toml": "[project]\ndependencies = [\n  \"requests[socks]>=2.32\",\n]\n",
			},
		},
		{
			name:        "pipfile dependency declaration",
			sourcePath:  "service/app.py",
			addedSource: "import requests\n",
			files: map[string]string{
				"service/Pipfile": "[packages]\nrequests = \"*\"\n",
			},
		},
		{
			name:        "malformed pyproject does not declare import",
			sourcePath:  "service/app.py",
			addedSource: "import definitely_missing\n",
			files: map[string]string{
				"service/pyproject.toml": "[project]\ndependencies = [\n  \"definitely_missing\"\n",
			},
			wantCode:    3,
			wantFinding: "definitely_missing",
		},
		{
			name:        "python multiple imports reject undeclared module",
			sourcePath:  "app.py",
			addedSource: "import requests, definitely_missing\n",
			files:       map[string]string{"requirements.txt": "requests==2.32.0\n"},
			wantCode:    3,
			wantFinding: "definitely_missing",
		},
		{
			name:        "comments and other ecosystems do not declare python imports",
			sourcePath:  "app.py",
			addedSource: "import definitely_missing\n",
			files: map[string]string{
				"Cargo.toml":       "[package]\nname = \"definitely_missing\"\nversion = \"0.1.0\"\n",
				"requirements.txt": "# definitely_missing is intentionally absent\n",
			},
			wantCode:    3,
			wantFinding: "definitely_missing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := t.TempDir()
			for path, content := range test.files {
				writePackageExistenceFile(t, repo, path, content)
			}
			writePackageExistenceFile(t, repo, test.sourcePath, "// baseline\n")
			gitPackageExistence(t, repo, "init", "-q")
			gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
			gitPackageExistence(t, repo, "config", "user.name", "Test")
			gitPackageExistence(t, repo, "add", ".")
			gitPackageExistence(t, repo, "commit", "-qm", "baseline")
			if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
				t.Fatal(err)
			}
			writePackageExistenceFile(t, repo, test.sourcePath, test.addedSource)
			t.Chdir(repo)

			var stdout, stderr bytes.Buffer
			gotCode := PackageExistence(repo, []string{"feat"}, &stdout, &stderr)
			if gotCode != test.wantCode {
				t.Fatalf("code = %d, want %d\nstdout: %s\nstderr: %s", gotCode, test.wantCode, stdout.String(), stderr.String())
			}
			if test.wantFinding != "" && !strings.Contains(stderr.String(), test.wantFinding) {
				t.Fatalf("stderr does not name %q: %s", test.wantFinding, stderr.String())
			}
		})
	}
}

func TestPackageExistenceGenericResolverFailsClosed(t *testing.T) {
	repo := t.TempDir()
	writePackageExistenceFile(t, repo, "package.json", `{"dependencies":{"definitely-missing-package":"1.0.0"}}`)
	writePackageExistenceFile(t, repo, "src/example.custom", "// baseline\n")
	gitPackageExistence(t, repo, "init", "-q")
	gitPackageExistence(t, repo, "config", "user.email", "test@example.com")
	gitPackageExistence(t, repo, "config", "user.name", "Test")
	gitPackageExistence(t, repo, "add", ".")
	gitPackageExistence(t, repo, "commit", "-qm", "baseline")
	if err := os.MkdirAll(featureDir(repo, "feat"), 0o755); err != nil {
		t.Fatal(err)
	}
	writePackageExistenceFile(t, repo, "src/example.custom", "import \"definitely-missing-package\"\n")
	t.Chdir(repo)

	var stdout, stderr bytes.Buffer
	gotCode := PackageExistence(repo, []string{"feat"}, &stdout, &stderr)
	if gotCode != 3 {
		t.Fatalf("code = %d, want 3\nstdout: %s\nstderr: %s", gotCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "definitely-missing-package") {
		t.Fatalf("stderr does not name unresolved import: %s", stderr.String())
	}
}

func writePackageExistenceFile(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitPackageExistence(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
