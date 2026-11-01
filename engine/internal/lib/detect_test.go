package lib

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func runDetect(t *testing.T, root string, args ...string) (int, string) {
	t.Helper()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	full := append([]string{"commands", "--root", root}, args...)
	code := RunDetectCommands("", full, stdout, stderr)
	return code, stdout.String() + stderr.String()
}

func TestDetectMakefileWins(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/Makefile", "test:\n\tgo test ./internal/...\n\nlint:\n\tgolangci-lint run\n\nbuild:\n\tgo build ./cmd/x\n")
	writeFile(t, root+"/go.mod", "module example.com/x\n\ngo 1.25\n")
	code, out := runDetect(t, root)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	for _, want := range []string{"make test", "make lint", "make build", "go vet ./..."} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestDetectPackageJSONAndManager(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/package.json", `{"scripts":{"test":"vitest","lint":"eslint .","typecheck":"tsc -b","build":"vite build"}}`)
	writeFile(t, root+"/pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	code, out := runDetect(t, root)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	for _, want := range []string{"pnpm test", "pnpm run lint", "pnpm run typecheck", "pnpm run build", "package-manager: pnpm"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "integration") || !strings.Contains(out, "unresolved") {
		t.Fatalf("integration slot should be unresolved:\n%s", out)
	}
}

func TestDetectManifestFallbackAndJSON(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/go.mod", "module example.com/y\n\ngo 1.25\n")
	code, out := runDetect(t, root, "--json")
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	var report commandReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("bad json %v:\n%s", err, out)
	}
	found := map[string]string{}
	for _, c := range report.Commands {
		found[c.Name] = c.Command
	}
	if found["unit"] != "go test ./..." || found["vet"] != "go vet ./..." {
		t.Fatalf("unexpected commands: %v", found)
	}
}

func TestDetectManifestFallbackSkipsManifestWithoutSlot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/setup.py", "from setuptools import setup\nsetup()\n")
	writeFile(t, root+"/build.gradle", "plugins { id 'java' }\n")
	code, out := runDetect(t, root)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "gradle build") || !strings.Contains(out, "build.gradle") {
		t.Fatalf("build slot should resolve from build.gradle:\n%s", out)
	}
}

func TestDetectEmptyRepoReportsUnresolved(t *testing.T) {
	root := t.TempDir()
	code, out := runDetect(t, root)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if strings.Count(out, "unresolved") != len(commandSlots) {
		t.Fatalf("expected all slots unresolved:\n%s", out)
	}
}

func TestDetectMixedSources(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root+"/Makefile", "test:\n\tpytest -q\n")
	writeFile(t, root+"/package.json", `{"scripts":{"lint":"eslint ."}}`)
	writeFile(t, root+"/package-lock.json", "{}")
	writeFile(t, root+"/pyproject.toml", "[project]\nname='x'\n")
	code, out := runDetect(t, root)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out)
	}
	if !strings.Contains(out, "make test") || !strings.Contains(out, "npm run lint") || !strings.Contains(out, "mypy .") {
		t.Fatalf("mixed-source resolution wrong:\n%s", out)
	}
}

func TestDetectUsageError(t *testing.T) {
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	if code := RunDetectCommands("", nil, stdout, stderr); code != 2 {
		t.Fatalf("missing subcommand should be usage error, got %d", code)
	}
	if code := RunDetectCommands("", []string{"commands", "--bogus"}, stdout, stderr); code != 2 {
		t.Fatalf("bad flag should be usage error, got %d", code)
	}
}

func TestDetectIntegrationRequiresExplicitRepositoryCommand(t *testing.T) {
	for _, manifest := range []string{"go.mod", "Cargo.toml", "pyproject.toml", "setup.py", "pom.xml", "build.gradle", "composer.json", "Gemfile", "mix.exs"} {
		t.Run(manifest, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, root+"/"+manifest, "")
			report := detectCommands(root)
			for _, command := range report.Commands {
				if command.Name == "integration" {
					t.Fatalf("manifest alone invented integration suite: %+v", command)
				}
			}
			if !slices.Contains(report.Unresolved, "integration") {
				t.Fatalf("integration must remain unresolved: %+v", report)
			}
			writeFile(t, root+"/Makefile", "test-integration:\n\t./scripts/real-integration\n")
			report = detectCommands(root)
			found := false
			for _, command := range report.Commands {
				if command.Name == "integration" && command.Command == "make test-integration" {
					found = true
				}
			}
			if !found {
				t.Fatalf("explicit integration target not resolved: %+v", report)
			}
		})
	}
	root := t.TempDir()
	writeFile(t, root+"/package.json", `{"scripts":{"test:integration":"node integration.js"}}`)
	for _, command := range detectCommands(root).Commands {
		if command.Name == "integration" && command.Command == "npm run test:integration" {
			return
		}
	}
	t.Fatal("explicit package integration script not resolved")
}
