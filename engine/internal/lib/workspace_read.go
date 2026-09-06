package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/devritespaths"
)

func readWorkspaceArtifact(root, slug, name string) ([]byte, error) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, fmt.Errorf("task-graph: workspace unavailable: %w", err)
	}
	path := filepath.Join(workspace, name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("task-graph: %s unavailable: %w", name, err)
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("task-graph: %s exceeds 1 MiB limit", name)
	}
	// #nosec G304 -- workspace task-graph file; 1 MiB cap checked above
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("task-graph: cannot read %s: %w", name, err)
	}
	return raw, nil
}
