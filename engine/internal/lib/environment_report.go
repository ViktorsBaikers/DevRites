package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/version"
)

// PathProbe records whether one project-relative path exists.
type PathProbe struct {
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

// EnvironmentReport is a read-only snapshot of install and code-index presence.
type EnvironmentReport struct {
	EngineVersion string      `json:"engine_version"`
	ProjectRoot   string      `json:"project_root"`
	DevritesRoot  string      `json:"devrites_root,omitempty"`
	Manifest      PathProbe   `json:"manifest"`
	Indexes       []PathProbe `json:"indexes"`
}

func projectRootFromDevritesRoot(root string) string {
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "."
		}
		return wd
	}
	if filepath.Base(root) == devritespaths.DevritesRootName {
		return filepath.Dir(root)
	}
	return root
}

func probePath(projectRoot, rel string) PathProbe {
	abs := filepath.Join(projectRoot, filepath.FromSlash(rel))
	info, err := os.Stat(abs)
	return PathProbe{Path: rel, Present: err == nil && info != nil}
}

// EnvironmentReportFor builds a report for one project root.
func EnvironmentReportFor(devritesRoot, projectRoot string) (EnvironmentReport, error) {
	if projectRoot == "" {
		projectRoot = projectRootFromDevritesRoot(devritesRoot)
	}
	abs, err := filepath.Abs(projectRoot)
	if err != nil {
		return EnvironmentReport{}, err
	}
	report := EnvironmentReport{
		EngineVersion: version.Version,
		ProjectRoot:   abs,
		Manifest:      probePath(abs, devritespaths.ManifestName),
		Indexes: []PathProbe{
			probePath(abs, ".codegraph"),
			probePath(abs, ".code-review-graph"),
			probePath(abs, ".codebase-memory"),
		},
	}
	if devritesRoot != "" {
		report.DevritesRoot = devritesRoot
	}
	return report, nil
}

// WriteEnvironmentReportJSON prints one JSON object to stdout.
func WriteEnvironmentReportJSON(devritesRoot, projectRoot string, stdout io.Writer) error {
	report, err := EnvironmentReportFor(devritesRoot, projectRoot)
	if err != nil {
		return fmt.Errorf("build environment report: %w", err)
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode environment report: %w", err)
	}
	return nil
}

// RunEnvironmentCheck implements `devrites-engine check indexes`.
func RunEnvironmentCheck(devritesRoot string, args []string, stdout, stderr io.Writer) int {
	projectRoot := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--root" {
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "indexes: --root requires a path")
				return 2
			}
			projectRoot = args[i+1]
			i++
			continue
		}
		fmt.Fprintf(stderr, "indexes: unexpected argument %q\n", args[i])
		return 2
	}
	if err := WriteEnvironmentReportJSON(devritesRoot, projectRoot, stdout); err != nil {
		fmt.Fprintf(stderr, "indexes: %v\n", err)
		return 2
	}
	return 0
}
