package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/devritespaths"
)

// workspaceArtifactLimit bounds one artifact read. Slice extraction must keep
// working on a workspace whose tasks.md is over budget: the budget gate flags the
// file, while Build still reads its own slice instead of the whole document.
const workspaceArtifactLimit = 16 << 20

func readWorkspaceArtifact(root, slug, name string) ([]byte, error) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return nil, fmt.Errorf("workspace artifact: workspace unavailable: %w", err)
	}
	path := filepath.Join(workspace, name)
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("workspace artifact: %s unavailable: %w", name, err)
	}
	if info.Size() > workspaceArtifactLimit {
		return nil, fmt.Errorf("workspace artifact: %s exceeds %d MiB limit", name, workspaceArtifactLimit>>20)
	}
	// #nosec G304 -- workspace artifact; size cap checked above
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("workspace artifact: cannot read %s: %w", name, err)
	}
	return raw, nil
}
