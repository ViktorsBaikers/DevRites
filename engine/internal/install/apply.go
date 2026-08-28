package install

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/hostpack"
)

func (r *runner) install() error {
	fmt.Fprintln(r.opts.Stdout, "DevRites installer")
	fmt.Fprintf(r.opts.Stdout, "  target : %s\n", r.target)
	fmt.Fprintf(r.opts.Stdout, "  payload: %s\n", r.payload)
	fmt.Fprintf(r.opts.Stdout, "  skills : %s\n", yesno(r.opts.WithSkills))
	fmt.Fprintln(r.opts.Stdout, "  standards: ship inside the devrites-lib skill")
	fmt.Fprintf(r.opts.Stdout, "  agents : %s\n", yesno(r.opts.WithAgents))
	fmt.Fprintf(r.opts.Stdout, "  codex  : %s\n", yesno(r.opts.WithCodex))
	fmt.Fprintf(r.opts.Stdout, "  aliases: %s\n", r.opts.AliasMode)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  (dry run - no changes will be made)")
	}
	fmt.Fprintln(r.opts.Stdout)
	if r.preparedBinary != "" {
		if _, err := verifyEngineBinary(r.preparedBinary, r.binaryTag(), 30*time.Second); err != nil {
			return fmt.Errorf("verify staged engine binary %s: %w", r.preparedBinary, err)
		}
	}
	if err := r.preflightInstall(); err != nil {
		return err
	}

	for _, tree := range hostpack.InstallTrees(r.opts.WithSkills, r.opts.WithAgents, r.opts.WithCodex) {
		if err := r.installTree(tree.PayloadPrefix, tree.TargetPrefix); err != nil {
			return fmt.Errorf("install tree %s: %w", tree.TargetPrefix, err)
		}
	}
	if r.opts.WithSkills && r.opts.AliasMode == "all" {
		for _, alias := range hostpack.Aliases {
			data, err := hostpack.RenderAliasSkill(alias)
			if err != nil {
				return fmt.Errorf("render alias skill %s: %w", alias.Name, err)
			}
			for _, rel := range hostpack.AliasTargets(alias, r.opts.WithCodex) {
				if err := r.installData(data, rel); err != nil {
					return fmt.Errorf("install alias: %w", err)
				}
			}
		}
	}
	if r.opts.WithSkills && r.opts.WithCodex {
		if err := r.mergeMarkerFile(hostpack.CodexAgentsMerge); err != nil {
			return fmt.Errorf("merge %s: %w", hostpack.CodexAgentsMerge.TargetRel, err)
		}
		if err := r.mergeCodexConfig(); err != nil {
			return fmt.Errorf("merge %s: %w", hostpack.CodexConfigMerge.TargetRel, err)
		}
	}
	if r.opts.WithSkills {
		if err := r.seedClaudeSettings(); err != nil {
			return fmt.Errorf("seed claude settings: %w", err)
		}
	}
	if err := r.seedDevrites(); err != nil {
		return fmt.Errorf("seed .devrites: %w", err)
	}
	if err := r.pruneDropped(); err != nil {
		return fmt.Errorf("prune dropped files: %w", err)
	}
	if !r.opts.DryRun {
		if err := r.writeManifest(); err != nil {
			return fmt.Errorf("write manifest: %w", err)
		}
	}
	if err := r.installBinary(); err != nil {
		return fmt.Errorf("install engine binary: %w", err)
	}

	fmt.Fprintln(r.opts.Stdout)
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "DevRites install plan complete (dry run)")
	} else {
		fmt.Fprintln(r.opts.Stdout, "DevRites installed")
	}
	fmt.Fprintf(r.opts.Stdout, "  installed: %d   overwritten: %d   skipped(conflict): %d   pruned: %d\n", r.stats.installed, r.stats.overwrote, r.stats.skipped, r.stats.pruned)
	if !r.opts.DryRun && r.opts.WithSkills {
		if r.opts.WithCodex {
			fmt.Fprintln(r.opts.Stdout, "Next: reopen the project, then run /rite (Claude) or $rite (Codex).")
		} else {
			fmt.Fprintln(r.opts.Stdout, "Next: reopen the project, then run /rite.")
		}
	}
	return nil
}

func (r *runner) installTree(srcPrefix, relPrefix string) error {
	return fs.WalkDir(r.payloadFS, srcPrefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk payload %s: %w", path, err)
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcPrefix, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		return r.installFile(path, filepath.ToSlash(filepath.Join(relPrefix, rel)))
	})
}

func (r *runner) installFile(src, rel string) error {
	data, err := fs.ReadFile(r.payloadFS, src)
	if err != nil {
		return fmt.Errorf("read payload %s: %w", src, err)
	}
	return r.installData(data, rel)
}

func (r *runner) installData(data []byte, rel string) error {
	dest := filepath.Join(r.target, filepath.FromSlash(rel))
	action := "install"
	if exists(dest) {
		record, managed := r.prev[rel]
		switch {
		case managed && managedConflict(record, r.preflight[filepath.ToSlash(rel)]) != "" && r.opts.Force:
			action = "overwrite(force-customized)"
		case managed:
			action = "overwrite"
		case r.opts.Force:
			action = "overwrite(force)"
		default:
			if r.opts.DryRun {
				fmt.Fprintf(r.opts.Stdout, "  [skip] %s (exists, not DevRites-managed)\n", rel)
			} else {
				fmt.Fprintf(r.opts.Stderr, "warning: skip %s (exists, not DevRites-managed; use --force to overwrite)\n", rel)
			}
			r.stats.skipped++
			return nil
		}
	}
	if r.opts.DryRun {
		fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
	} else {
		if err := r.recheckPath(rel); err != nil {
			return err
		}
		if err := fsutil.WriteFileAtomic(dest, data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", rel, err)
		}
	}
	r.addManifest(rel)
	r.addInstallRecord(rel, data)
	if action == "install" {
		r.stats.installed++
	} else {
		r.stats.overwrote++
	}
	return nil
}

func (r *runner) addManifest(rel string) {
	r.manifest = append(r.manifest, filepath.ToSlash(rel))
}

func (r *runner) addInstallRecord(rel string, data []byte) {
	if r.records == nil {
		r.records = map[string]string{}
	}
	sum := sha256.Sum256(data)
	r.records[filepath.ToSlash(rel)] = fmt.Sprintf("sha256:%x", sum[:])
}

func (r *runner) installMarker(rel, text string) error {
	return r.installData([]byte(text+"\n"), rel)
}

func (r *runner) seedDevrites() error {
	readme, err := hostpack.RenderDevritesReadme()
	if err != nil {
		return fmt.Errorf("render readme: %w", err)
	}
	if err := r.installData(readme, ".devrites/README.md"); err != nil {
		return fmt.Errorf("install readme: %w", err)
	}
	active := filepath.Join(r.target, ".devrites", "ACTIVE")
	if exists(active) {
		return nil
	}
	if r.opts.DryRun {
		fmt.Fprintln(r.opts.Stdout, "  [seed] .devrites/ACTIVE (runtime state - preserved on uninstall)")
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(active), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(active), err)
	}
	f, err := os.OpenFile(active, os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("create %s: %w", active, err)
	}
	return f.Close()
}

func (r *runner) pruneDropped() error {
	if len(r.prev) == 0 {
		return nil
	}
	next := map[string]bool{}
	for _, rel := range r.manifest {
		next[rel] = true
	}
	keys := make([]string, 0, len(r.prev))
	for rel := range r.prev {
		keys = append(keys, rel)
	}
	sort.Strings(keys)
	for _, rel := range keys {
		if next[rel] || hostpack.PreserveOnPrune(rel) {
			continue
		}
		dead := filepath.Join(r.target, filepath.FromSlash(rel))
		if !exists(dead) {
			continue
		}
		action := "prune"
		if managedConflict(r.prev[rel], r.preflight[rel]) != "" && r.opts.Force {
			action = "prune(force-customized)"
		}
		if r.opts.DryRun {
			fmt.Fprintf(r.opts.Stdout, "  [%s] %s (dropped from pack)\n", action, rel)
		} else {
			if err := r.recheckPath(rel); err != nil {
				return err
			}
			if merge, ok := hostpack.ManagedMergeForMarker(rel); ok {
				if err := r.recheckPath(merge.TargetRel); err != nil {
					return err
				}
				if merge.MarkerRel == hostpack.LegacyCodexHooksMerge.MarkerRel {
					if err := stripHooksPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel))); err != nil {
						return fmt.Errorf("strip hooks from %s: %w", merge.TargetRel, err)
					}
				} else if merge.TargetRel == hostpack.ClaudeSettingsMerge.TargetRel {
					_ = r.stripClaudeSettings(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), true)
				} else {
					_ = stripMarkerPath(filepath.Join(r.target, filepath.FromSlash(merge.TargetRel)), merge.Begin, merge.End)
				}
			}
			if err := os.Remove(dead); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", rel, err)
			}
			pruneEmptyDirs(filepath.Dir(dead), r.target)
			fmt.Fprintf(r.opts.Stdout, "  [%s] %s\n", action, rel)
		}
		r.stats.pruned++
	}
	return nil
}

func (r *runner) writeManifest() error {
	sort.Strings(r.manifest)
	r.manifest = slices.Compact(r.manifest)
	var b strings.Builder
	b.WriteString("# DevRites install manifest - do not edit by hand.\n")
	b.WriteString("# Generated " + time.Now().UTC().Format(time.RFC3339) + ". Uninstall removes exactly these paths.\n")
	b.WriteString("# devrites-version: " + installedVersion(r.source) + "\n")
	b.WriteString("# devrites-flags: " + r.flagsString() + "\n")
	b.WriteString("# managed-records: source=npx payload=pack/generated format=rel sha256\n")
	for _, rel := range r.manifest {
		if hash := r.records[rel]; hash != "" {
			b.WriteString("# managed: " + rel + " " + hash + "\n")
		}
	}
	for _, rel := range r.manifest {
		b.WriteString(rel + "\n")
	}
	if err := r.recheckPath(ManifestName); err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(r.target, ManifestName), []byte(b.String()), 0o644)
}

func (r *runner) flagsString() string {
	var flags []string
	if !r.opts.WithSkills {
		flags = append(flags, "--no-skills")
	}
	if !r.opts.WithAgents {
		flags = append(flags, "--no-agents")
	}
	if !r.opts.WithCodex {
		flags = append(flags, "--no-codex")
	}
	if !r.opts.WithBinary {
		flags = append(flags, "--no-binary")
	}
	if r.opts.AliasMode == "off" {
		flags = append(flags, "--no-short-aliases")
	}
	if r.opts.AliasMode == "all" {
		flags = append(flags, "--short-aliases=all")
	}
	return strings.Join(flags, " ")
}
