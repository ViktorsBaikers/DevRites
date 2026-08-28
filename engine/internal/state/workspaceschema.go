package state

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ErrWorkspaceSchemaRefused identifies a workspace whose declared schema is not
// the current engine contract; callers map it to the blocked exit code.
var ErrWorkspaceSchemaRefused = errors.New("workspace schema version refused")

// workspaceSchemaKey is the state.md cursor row carrying the workspace schema
// version. Workspaces written before the v5 engine carry no row and resolve to
// schema 2, the last pre-v5 contract.
const workspaceSchemaKey = CursorSchema

// WorkspaceSchema reports the schema version declared by the feature's
// state.md cursor. A missing row means the pre-v5 (schema 2) contract.
func WorkspaceSchema(root, slug string) (int, error) {
	raw, err := os.ReadFile(featureDir(root, slug) + "/" + LedgerFile)
	if err != nil {
		return 0, fmt.Errorf("read workspace schema for %s: %w", slug, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	return workspaceSchemaRow(lines)
}

// RequireWorkspaceSchema refuses feature workspaces whose schema is not the
// current one, naming the exact recovery path for each direction.
func RequireWorkspaceSchema(root, slug string) error {
	version, err := WorkspaceSchema(root, slug)
	if err != nil {
		return err
	}
	if version < SchemaVersion {
		return fmt.Errorf("%w: workspace %s predates the v5 workspace schema (schema %d, engine requires %d); run devrites-engine migrate %s", ErrWorkspaceSchemaRefused, slug, version, SchemaVersion, slug)
	}
	if version > SchemaVersion {
		return fmt.Errorf("%w: workspace %s declares schema %d, newer than this engine's %d; upgrade devrites", ErrWorkspaceSchemaRefused, slug, version, SchemaVersion)
	}
	return nil
}

func workspaceSchemaRow(lines []string) (int, error) {
	value, ok := CursorField(lines, workspaceSchemaKey)
	if !ok {
		return 2, nil
	}
	version, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("state.md schema row is invalid (value %q); expected the workspace schema version", value)
	}
	return version, nil
}
