package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/safepath"
)

func (r *runner) preflightInstall() error {
	desired, err := r.desiredInstallPaths()
	if err != nil {
		return fmt.Errorf("preflight install paths: %w", err)
	}
	var conflicts []string
	for rel := range desired {
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		record, managed := r.prev[rel]
		if managed && !snapshot.missing {
			if kind := managedConflict(record, snapshot); kind != "" && !r.opts.Force {
				conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
			}
		}
	}
	for rel, record := range r.prev {
		if desired[rel] || hostpack.PreserveOnPrune(rel) {
			continue
		}
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		if !snapshot.missing {
			if kind := managedConflict(record, snapshot); kind != "" && !r.opts.Force {
				conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
			}
		}
		if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
			if _, err := r.rememberPath(merge.TargetRel); err != nil {
				return err
			}
		}
	}
	for _, rel := range r.installMergeTargets() {
		if _, err := r.rememberPath(rel); err != nil {
			return err
		}
	}
	if _, err := r.rememberPath(ManifestName); err != nil {
		return err
	}
	return managedConflictError(conflicts)
}

func (r *runner) preflightUninstall(entries []string) error {
	var conflicts []string
	for _, rel := range entries {
		if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
			if _, err := r.rememberPath(merge.TargetRel); err != nil {
				return err
			}
		}
		if !hostpack.ShouldRemoveOnUninstall(rel, entries) {
			continue
		}
		snapshot, err := r.rememberPath(rel)
		if err != nil {
			return err
		}
		if snapshot.missing {
			continue
		}
		if kind := managedConflict(r.prev[rel], snapshot); kind != "" && !r.opts.Force {
			conflicts = append(conflicts, fmt.Sprintf("%s: %s", kind, rel))
		}
	}
	if _, err := r.rememberPath(ManifestName); err != nil {
		return err
	}
	return managedConflictError(conflicts)
}

func (r *runner) desiredInstallPaths() (map[string]bool, error) {
	out := map[string]bool{".devrites/README.md": true}
	for _, tree := range hostpack.InstallTrees(r.opts.WithSkills, r.opts.WithAgents, r.opts.WithCodex) {
		err := fs.WalkDir(r.payloadFS, tree.PayloadPrefix, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(tree.PayloadPrefix, path)
			if err != nil {
				return err
			}
			out[filepath.ToSlash(filepath.Join(tree.TargetPrefix, rel))] = true
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if r.opts.WithSkills && r.opts.AliasMode == "all" {
		for _, alias := range hostpack.Aliases {
			for _, rel := range hostpack.AliasTargets(alias, r.opts.WithCodex) {
				out[rel] = true
			}
		}
	}
	if r.opts.WithSkills && r.opts.WithCodex {
		out[hostpack.CodexAgentsMerge.MarkerRel] = true
		out[hostpack.CodexConfigMerge.MarkerRel] = true
	}
	if r.opts.WithSkills {
		out[hostpack.ClaudeSettingsMerge.MarkerRel] = true
	}
	return out, nil
}

func (r *runner) installMergeTargets() []string {
	var out []string
	if r.opts.WithSkills {
		out = append(out, hostpack.ClaudeSettingsMerge.TargetRel)
		if r.opts.WithCodex {
			out = append(out, hostpack.CodexAgentsMerge.TargetRel, hostpack.CodexConfigMerge.TargetRel)
		}
	}
	return out
}

func (r *runner) rememberPath(rel string) (pathSnapshot, error) {
	rel = filepath.ToSlash(rel)
	if snapshot, ok := r.preflight[rel]; ok {
		return snapshot, nil
	}
	snapshot, err := inspectManagedPath(r.target, rel)
	if err != nil {
		return pathSnapshot{}, err
	}
	r.preflight[rel] = snapshot
	return snapshot, nil
}

func (r *runner) recheckPath(rel string) error {
	rel = filepath.ToSlash(rel)
	before, ok := r.preflight[rel]
	if !ok {
		return fmt.Errorf("path %s was not preflighted", rel)
	}
	now, err := inspectManagedPath(r.target, rel)
	if err != nil {
		return err
	}
	if now != before {
		return fmt.Errorf("refusing to change %s: file changed after preflight; retry the operation", rel)
	}
	return nil
}

func inspectManagedPath(target, rel string) (pathSnapshot, error) {
	native := filepath.FromSlash(rel)
	if !filepath.IsLocal(native) {
		return pathSnapshot{}, fmt.Errorf("refusing unsafe managed path %q", rel)
	}
	dest := filepath.Join(target, native)
	if !safepath.WithinResolved(dest, target) {
		return pathSnapshot{}, fmt.Errorf("refusing managed path outside target: %s", rel)
	}
	walk := target
	parts := strings.Split(native, string(filepath.Separator))
	for _, part := range parts {
		walk = filepath.Join(walk, part)
		info, err := os.Lstat(walk)
		if os.IsNotExist(err) {
			return pathSnapshot{missing: true}, nil
		}
		if err != nil {
			return pathSnapshot{}, fmt.Errorf("inspect managed path %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return pathSnapshot{}, fmt.Errorf("refusing symlink or junction in managed path: %s", rel)
		}
	}
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return pathSnapshot{missing: true}, nil
	}
	if err != nil {
		return pathSnapshot{}, fmt.Errorf("inspect managed path %s: %w", rel, err)
	}
	if !info.Mode().IsRegular() {
		return pathSnapshot{}, fmt.Errorf("refusing non-regular managed path: %s", rel)
	}
	f, err := os.Open(dest)
	if err != nil {
		return pathSnapshot{}, fmt.Errorf("read managed path %s: %w", rel, err)
	}
	defer f.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, f); err != nil {
		return pathSnapshot{}, fmt.Errorf("hash managed path %s: %w", rel, err)
	}
	return pathSnapshot{hash: "sha256:" + hex.EncodeToString(sum.Sum(nil))}, nil
}

func managedConflict(record managedRecord, snapshot pathSnapshot) string {
	if record.Hash == "" {
		return "legacy manifest entry (no hash)"
	}
	if record.Hash != snapshot.hash {
		return "customized managed file"
	}
	return ""
}

func managedConflictError(conflicts []string) error {
	if len(conflicts) == 0 {
		return nil
	}
	sort.Strings(conflicts)
	return fmt.Errorf("managed files differ from the install manifest:\n  %s\nrerun with --force to replace or remove these files", strings.Join(conflicts, "\n  "))
}
