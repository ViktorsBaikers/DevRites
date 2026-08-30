package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrites/devrites/internal/hostpack"
)

func (r *runner) uninstall() error {
	mf := filepath.Join(r.target, ManifestName)
	entries, err := readManifestList(mf)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("no DevRites manifest at %s - nothing to uninstall", mf)
	}
	fmt.Fprintln(r.opts.Stdout, "DevRites uninstaller")
	fmt.Fprintf(r.opts.Stdout, "  target  : %s\n", r.target)
	fmt.Fprintf(r.opts.Stdout, "  manifest: %s\n", mf)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  (dry run - no changes)")
	}
	fmt.Fprintln(r.opts.Stdout)
	if err := r.preflightUninstall(entries); err != nil {
		return err
	}

	for _, rel := range entries {
		merge, ok := hostpack.ManagedMergeForMarker(rel)
		if !ok {
			continue
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [merge-remove] %s\n", merge.DryRun)
			continue
		}
		if err := r.recheckPath(rel); err != nil {
			return err
		}
		if err := r.recheckPath(merge.TargetRel); err != nil {
			return err
		}
		if merge.MarkerRel == hostpack.LegacyCodexHooksMerge.MarkerRel {
			if err := stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))); err != nil {
				return fmt.Errorf("strip hooks from %s: %w", merge.TargetRel, err)
			}
			continue
		}
		if merge.TargetRel == hostpack.ClaudeSettingsMerge.TargetRel {
			if err := r.stripClaudeSettings(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), true); err != nil {
				return fmt.Errorf("strip DevRites settings from %s: %w", merge.TargetRel, err)
			}
			continue
		}
		if err := stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End); err != nil {
			return fmt.Errorf("strip marker block from %s: %w", merge.TargetRel, err)
		}
	}

	var dirs []string
	for _, rel := range entries {
		if !hostpack.ShouldRemoveOnUninstall(rel, entries) {
			continue
		}
		dest := filepath.Join(r.target, filepath.FromSlash(rel))
		if exists(dest) {
			action := "remove"
			if managedConflict(r.prev[rel], r.preflight[rel]) != "" && r.opts.Force {
				action = "remove(force-customized)"
			}
			if r.opts.DryRun {
				fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
			} else {
				if err := r.recheckPath(rel); err != nil {
					return err
				}
				if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("remove %s: %w", rel, err)
				}
			}
			dirs = append(dirs, filepath.Dir(dest))
			r.stats.removed++
		} else {
			r.stats.missing++
		}
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [remove] %s\n", ManifestName)
	} else {
		if err := r.recheckPath(ManifestName); err != nil {
			return err
		}
		_ = os.Remove(mf)
		dirs = append(dirs, filepath.Dir(mf))
		for _, d := range dirs {
			pruneEmptyDirs(d, r.target)
		}
	}
	if err := r.removeBinary(); err != nil {
		return fmt.Errorf("remove engine binary: %w", err)
	}
	fmt.Fprintln(r.opts.Stdout)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "DevRites uninstall plan complete (dry run)")
	} else {
		fmt.Fprintln(r.opts.Stdout, "DevRites uninstalled")
	}
	fmt.Fprintf(r.opts.Stdout, "  removed: %d   already-absent: %d\n", r.stats.removed, r.stats.missing)
	if exists(filepath.Join(r.target, ".devrites", "work")) {
		fmt.Fprintln(r.opts.Stdout, "  kept .devrites/work/ (your feature data)")
	}
	if exists(filepath.Join(r.target, ".devrites", "ACTIVE")) {
		fmt.Fprintln(r.opts.Stdout, "  kept .devrites/ACTIVE (active-feature cursor)")
	}
	return nil
}
