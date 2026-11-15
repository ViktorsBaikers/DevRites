package lib

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/fsutil"
	"github.com/devrites/devrites/internal/markdowntext"
	"github.com/devrites/devrites/internal/parallel"
	"github.com/devrites/devrites/internal/state"
)

const CreateSequenceUsage = "usage: devrites-engine state create-sequence <controller> <target> --brief <file>"
const sequenceStagingDir = ".sequence-staging"

// RunCreateSequence publishes a pending Spec cursor, never completed Spec artifacts.
func RunCreateSequence(root string, args []string, stdout, stderr io.Writer) int {
	return runCreateSequence(root, args, stdout, stderr, fsutil.PublishDirNoReplace, fsutil.SyncPublishedDir)
}

func runCreateSequence(root string, args []string, stdout, stderr io.Writer, publish func(*os.Root, string, string) (bool, error), syncPublished func(*os.Root, string) error) int {
	var positional []string
	briefPath := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--brief" && briefPath == "" && i+1 < len(args) {
			i++
			briefPath = args[i]
		} else if strings.HasPrefix(args[i], "-") {
			fmt.Fprintln(stderr, CreateSequenceUsage)
			return 2
		} else {
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 2 || briefPath == "" || briefPath == "-" || !devritespaths.ValidSlug(positional[0]) || !devritespaths.ValidSlug(positional[1]) || positional[0] == positional[1] {
		fmt.Fprintln(stderr, CreateSequenceUsage)
		return 2
	}
	controller, target := positional[0], positional[1]
	if override, err := devritespaths.WorkspaceOverrideChecked(root, controller); err != nil || override != "" {
		fmt.Fprintln(stderr, "create-sequence: unset DEVRITES_WORKSPACE before root creation")
		return 3
	}
	result := ""
	err := parallel.WithControlLock(root, func() error {
		selected, err := os.OpenRoot(root)
		if err != nil {
			return err
		}
		defer func() { _ = selected.Close() }()
		return state.WithFeatureIdentityLockIn(selected, target, func() error {
			nodes, err := frontierScan(root)
			if err != nil {
				return err
			}
			chain, err := frontierChain(nodes, controller)
			if err != nil {
				return err
			}
			if nodes[controller].location != "live" {
				return fmt.Errorf("controller %s is not live; resume the live sequence controller", controller)
			}
			ordered, _ := frontierRelated(nodes, chain)
			mapping, _, err := frontierCanonicalRoadmap(nodes, ordered)
			if err != nil {
				return err
			}
			if mapping == nil {
				return fmt.Errorf("canonical roadmap required; run native Spec/Plan authority normalization")
			}
			position := 0
			for p, slug := range mapping {
				if slug == target {
					position = p
					break
				}
			}
			if position <= 1 {
				return fmt.Errorf("target %s is not a future canonical identity; run native Spec/Plan clarification", target)
			}
			parent := mapping[position-1]
			predecessor := nodes[parent]
			if predecessor == nil {
				return fmt.Errorf("create immediate predecessor %s first, then refresh frontier", parent)
			}
			for _, n := range []*frontierWorkspace{nodes[controller], predecessor} {
				if err := sequenceCurrentCursor(n); err != nil {
					return err
				}
			}
			destination := filepath.Join("work", target)
			if _, err := selected.Lstat(destination); err == nil {
				existing := nodes[target]
				if existing == nil || existing.location != "live" || existing.parent != parent || existing.position != position {
					return fmt.Errorf("target %s conflicts with canonical identity; preserve it and run native Spec/Plan repair", target)
				}
				if err := sequenceCurrentCursor(existing); err != nil {
					return err
				}
				if _, err := readSequenceBrief(filepath.Join(existing.dir, "brief.md")); err != nil {
					return fmt.Errorf("target %s has no readable substantive brief; repair partial workspace without replacing it", target)
				}
				result = fmt.Sprintf("sequence-workspace: reused target=%s position=%d parent=%s", target, position, parent)
				if err := syncPublished(selected, destination); err != nil {
					return err
				}
				if _, err := selected.Lstat(sequenceStagingDir); err == nil {
					return syncPublished(selected, sequenceStagingDir)
				} else if !os.IsNotExist(err) {
					return err
				}
				return nil
			} else if !os.IsNotExist(err) {
				return err
			}
			pool, err := evaluatePreparationPool(root, controller, nil)
			if err != nil {
				return err
			}
			if !pool.enabled || pool.controller != controller {
				return fmt.Errorf("creation requires the canonical Autocomplete controller at effective parallel capacity greater than one; refresh frontier")
			}
			entry := pool.entry(target)
			if entry == nil || !entry.missing || !entry.selected {
				return fmt.Errorf("target %s has no free preparation reservation or opening allowance; refresh frontier and resolve its recorded constraint", target)
			}
			brief, err := readSequenceBrief(briefPath)
			if err != nil {
				return fmt.Errorf("brief: %w; supply a bounded regular approved-scope brief", err)
			}
			cursor := fmt.Sprintf("# State\n\n## Cursor\n| Key | Value |\n| --- | --- |\n| phase | spec |\n| status | running |\n| next_action | /rite-spec %s |\n| schema | %d |\n| sequence_parent | %s |\n| sequence_position | %d |\n| sequence_workspaces_remaining | %d |\n", target, state.SchemaVersion, parent, position, entry.remaining)
			published, err := publishSequence(selected, target, brief, []byte(cursor), publish, syncPublished)
			if published {
				result = fmt.Sprintf("sequence-workspace: created target=%s position=%d parent=%s", target, position, parent)
			}
			return err
		})
	})
	if result != "" {
		fmt.Fprintln(stdout, result)
	}
	if err != nil {
		if result != "" {
			fmt.Fprintf(stderr, "create-sequence: identity already published; preserve target and retry the same command to reconcile durability: %v\n", err)
		} else {
			fmt.Fprintf(stderr, "create-sequence: refused without publishing target: %v\n", err)
		}
		return 3
	}
	return 0
}

func sequenceCurrentCursor(n *frontierWorkspace) error {
	if n == nil || n.problem != nil || !n.hasPosition || !n.hasRemaining || n.parent != "" && !n.hasParent {
		return fmt.Errorf("sequence cursor incomplete or invalid; initialize its identity/budget through native Spec/Plan before creation")
	}
	raw, err := readBoundedRegularFile(filepath.Join(n.dir, "state.md"), workspaceArtifactLimit)
	if err != nil {
		return err
	}
	structural, err := markdowntext.Structural(raw)
	if err != nil {
		return err
	}
	schema, present, err := frontierField(strings.Split(string(structural), "\n"), state.CursorSchema)
	if err != nil || !present || schema != strconv.Itoa(state.SchemaVersion) {
		return fmt.Errorf("workspace %s requires current schema %d; run devrites-engine migrate %s", n.slug, state.SchemaVersion, n.slug)
	}
	return nil
}

func readSequenceBrief(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 || before.Size() > workspaceArtifactLimit {
		return nil, fmt.Errorf("brief must be a nonempty regular file within artifact size limit")
	}
	file, err := openSequenceBrief(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("brief identity changed while opening")
	}
	raw, err := io.ReadAll(io.LimitReader(file, workspaceArtifactLimit+1))
	if err != nil {
		return nil, err
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > workspaceArtifactLimit || after.Size() != int64(len(raw)) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, fmt.Errorf("brief changed or exceeded artifact size limit")
	}
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 || len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("brief must contain real UTF-8 scope text")
	}
	return raw, nil
}

func publishSequence(root *os.Root, target string, brief, cursor []byte, publish func(*os.Root, string, string) (bool, error), syncPublished func(*os.Root, string) error) (published bool, err error) {
	if err := root.Mkdir(sequenceStagingDir, 0o700); err == nil {
		if err := syncPublished(root, sequenceStagingDir); err != nil {
			return false, err
		}
	} else if !os.IsExist(err) {
		return false, err
	}
	info, err := root.Lstat(sequenceStagingDir)
	if err != nil {
		return false, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return false, fmt.Errorf("staging parent must be a real directory")
	}
	parent, err := root.OpenRoot(sequenceStagingDir)
	if err != nil {
		return false, err
	}
	defer func() { _ = parent.Close() }()
	opened, err := parent.Stat(".")
	if err != nil {
		return false, err
	}
	if !os.SameFile(info, opened) {
		return false, fmt.Errorf("staging parent identity changed")
	}
	name := rand.Text()
	stage := filepath.Join(sequenceStagingDir, name)
	if err := parent.Mkdir(name, 0o700); err != nil {
		return false, err
	}
	defer func() {
		if !published {
			err = errors.Join(err, parent.RemoveAll(name))
		}
	}()
	staged, err := parent.OpenRoot(name)
	if err != nil {
		return false, err
	}
	defer func() { _ = staged.Close() }()
	for _, artifact := range [...]struct {
		name string
		body []byte
	}{{"brief.md", brief}, {"state.md", cursor}} {
		file, err := staged.OpenFile(artifact.name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return false, err
		}
		_, writeErr := file.Write(artifact.body)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return false, err
		}
	}
	return publish(root, stage, filepath.Join("work", target))
}
