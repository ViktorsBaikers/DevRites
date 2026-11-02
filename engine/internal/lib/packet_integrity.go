package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
)

// Binary patches can exceed the candidate's raw-byte bound; stream verification.
const maxFrozenDiffBytes = 4 * maxCandidateTotalBytes

type frozenDiffBinding struct {
	name   string
	digest string
}

func frozenDiffBindings(raw []byte) ([]frozenDiffBinding, error) {
	var bindings []frozenDiffBinding
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != "diff" {
			continue
		}
		if len(fields) != 3 || len(fields[1]) != sha256.Size*2 || diffPathRE.FindString(fields[2]) != fields[2] {
			return nil, errors.New("malformed diff checksum; use diff <SHA256> <path.diff|path.patch>")
		}
		if _, err := hex.DecodeString(fields[1]); err != nil {
			return nil, errors.New("malformed diff SHA256 checksum")
		}
		bindings = append(bindings, frozenDiffBinding{name: fields[2], digest: strings.ToLower(fields[1])})
	}
	return bindings, nil
}

func verifyFrozenDiff(name, expected string) error {
	before, err := os.Lstat(name)
	if err != nil {
		return err
	}
	if !before.Mode().IsRegular() || before.Size() <= 0 {
		return errors.New("frozen candidate diff must be a non-empty regular file")
	}
	if before.Size() > maxFrozenDiffBytes {
		return errors.New("frozen candidate diff exceeds 1 GiB limit")
	}
	// #nosec G304 -- validated regular artifact, bounded streaming read below
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) {
		return errors.New("frozen candidate diff changed while opening")
	}
	digest := sha256.New()
	count, err := io.Copy(digest, io.LimitReader(file, maxFrozenDiffBytes+1))
	if err != nil {
		return err
	}
	after, err := file.Stat()
	if err != nil {
		return err
	}
	current, err := os.Lstat(name)
	if err != nil {
		return err
	}
	if count != before.Size() || !os.SameFile(opened, current) || after.Size() != count || current.Size() != count || !opened.ModTime().Equal(after.ModTime()) || !opened.ModTime().Equal(current.ModTime()) {
		return errors.New("frozen candidate diff changed while reading")
	}
	if hex.EncodeToString(digest.Sum(nil)) != expected {
		return errors.New("frozen candidate diff SHA256 mismatch; freeze the intended bytes under a new wave")
	}
	return nil
}

func resolvePacketDiff(root, slug, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return "", err
	}
	if rel, ok := strings.CutPrefix(name, workspaceRel(root, workspace)+"/"); ok && clean == filepath.FromSlash(name) {
		name = rel
		clean = filepath.Clean(filepath.FromSlash(name))
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != filepath.FromSlash(name) {
		return "", fmt.Errorf("unsafe frozen candidate diff path %q", name)
	}
	dirs := []string{projectDir(root), workspace}
	// Freeze's checksum rows use this workspace-relative namespace, never a product packets/ directory.
	if strings.HasPrefix(name, "packets/") {
		dirs = []string{workspace}
	}
	if filepath.IsAbs(name) {
		parent, err := filepath.EvalSymlinks(filepath.Dir(name))
		if err != nil {
			return "", err
		}
		dirs = []string{parent}
		name = filepath.Base(name)
	}
	for _, dir := range dirs {
		info, exists, err := candidatePathInfo(dir, filepath.ToSlash(name))
		if err != nil {
			return "", err
		}
		if exists {
			if !info.Mode().IsRegular() {
				return "", errors.New("frozen candidate diff is not a regular file")
			}
			parent, err := filepath.EvalSymlinks(dir)
			if err != nil {
				return "", err
			}
			return filepath.Join(parent, filepath.FromSlash(name)), nil
		}
	}
	return "", fmt.Errorf("frozen candidate diff %q does not exist", name)
}

var packetContextPathRE = regexp.MustCompile(`(?:[A-Za-z]:/)?[A-Za-z0-9_./@+-]+\.bundle\.md\b`)

type packetContextRef struct {
	dir  string
	name string
}

// Preserve the leaf spelling below the workspace when an ancestor has an OS alias.
func packetContextWorkspaceRelative(workspace, name string) (string, error) {
	if rel, err := filepath.Rel(workspace, name); err == nil && filepath.IsLocal(rel) {
		return rel, nil
	}
	selected, err := os.Stat(workspace)
	if err != nil {
		return "", err
	}
	for parent := filepath.Dir(name); ; parent = filepath.Dir(parent) {
		info, err := os.Stat(parent)
		if err != nil {
			return "", err
		}
		if os.SameFile(selected, info) {
			return filepath.Rel(parent, name)
		}
		if parent == filepath.Dir(parent) {
			return "", nil
		}
	}
}

func resolvePacketContext(root, slug, name string) (packetContextRef, error) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return packetContextRef{}, err
	}
	workspace, err = filepath.Abs(workspace)
	if err != nil {
		return packetContextRef{}, err
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean != filepath.FromSlash(name) {
		return packetContextRef{}, fmt.Errorf("unsafe packet context path %q", name)
	}
	dirs := []string{projectDir(root), workspace}
	// Only the selected workspace's exact namespace may contain leading "..".
	if rel, ok := strings.CutPrefix(name, workspaceRel(root, workspace)+"/"); ok {
		dirs, clean = []string{workspace}, filepath.FromSlash(rel)
	} else if filepath.IsAbs(clean) {
		rel, err := packetContextWorkspaceRelative(workspace, clean)
		if err != nil {
			return packetContextRef{}, err
		}
		if rel != "" {
			dirs, clean = []string{workspace}, rel
		} else {
			// Absolute manual bundles remain supported, but not foreign workspaces.
			physicalParent, err := filepath.EvalSymlinks(filepath.Dir(clean))
			if err != nil {
				return packetContextRef{}, err
			}
			physical := filepath.Join(physicalParent, filepath.Base(clean))
			for _, work := range []string{filepath.Dir(workspace), filepath.Join(root, "work")} {
				absoluteWork, err := filepath.Abs(work)
				if err != nil {
					return packetContextRef{}, err
				}
				absoluteWork, err = filepath.EvalSymlinks(absoluteWork)
				if err != nil {
					return packetContextRef{}, err
				}
				if rel, err := filepath.Rel(absoluteWork, physical); err == nil && filepath.IsLocal(rel) {
					return packetContextRef{}, fmt.Errorf("foreign packet context path %q", name)
				}
			}
			dirs, clean = []string{filepath.Dir(clean)}, filepath.Base(clean)
		}
	} else if strings.HasPrefix(name, "ctx/") || strings.HasPrefix(name, "packets/") {
		dirs = []string{workspace}
	} else {
		work := filepath.ToSlash(workspaceRel(root, filepath.Dir(workspace))) + "/"
		if strings.HasPrefix(name, work) {
			return packetContextRef{}, fmt.Errorf("foreign packet context path %q", name)
		}
	}
	if !filepath.IsLocal(clean) {
		return packetContextRef{}, fmt.Errorf("unsafe packet context path %q", name)
	}
	for _, dir := range dirs {
		if err := contextParents(dir, clean); err != nil {
			return packetContextRef{}, err
		}
		ws, err := os.OpenRoot(dir)
		if err != nil {
			return packetContextRef{}, err
		}
		info, err := ws.Lstat(clean)
		closeErr := ws.Close()
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return packetContextRef{}, err
		}
		if closeErr != nil {
			return packetContextRef{}, closeErr
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if !info.Mode().IsRegular() {
			return packetContextRef{}, errors.New("packet context is not a regular file")
		}
		absoluteDir, err := filepath.Abs(dir)
		if err != nil {
			return packetContextRef{}, err
		}
		return packetContextRef{dir: absoluteDir, name: clean}, nil
	}
	return packetContextRef{}, fmt.Errorf("packet context %q does not exist", name)
}

func lintPacketContexts(root, slug string, raw []byte) error {
	bound := make(map[string]string)
	var names []string
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == "context" {
			if len(fields) < 3 || len(fields[1]) != sha256.Size*2 {
				return errors.New("malformed context checksum; use context <SHA256> <path.bundle.md>")
			}
			if _, err := hex.DecodeString(fields[1]); err != nil {
				return errors.New("malformed context SHA256 checksum")
			}
			entry := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "context"))
			name := strings.TrimSpace(entry[len(fields[1]):])
			ref, err := resolvePacketContext(root, slug, name)
			if err != nil {
				return err
			}
			path := filepath.Join(ref.dir, ref.name)
			digest := strings.ToLower(fields[1])
			if prior, ok := bound[path]; ok && prior != digest {
				return fmt.Errorf("conflicting context checksums for %s", name)
			}
			body, err := readContextArtifact(ref.dir, ref.name)
			if err != nil {
				return fmt.Errorf("packet context %s: %w", name, err)
			}
			if len(body) == 0 || fmt.Sprintf("%x", sha256.Sum256(body)) != digest {
				return fmt.Errorf("packet context %s SHA256 mismatch; render the intended read-set under a new wave", name)
			}
			bound[path] = digest
			continue
		}
		if len(fields) >= 3 && fields[0] == "file" {
			if fields[1] == "absent" {
				continue
			}
			if digest, err := hex.DecodeString(fields[1]); err == nil && len(digest) == sha256.Size {
				continue
			}
		}
		if name, ok := strings.CutPrefix(strings.TrimSpace(line), "- "); ok && strings.HasSuffix(name, ".bundle.md") {
			names = append(names, name)
			continue
		}
		names = append(names, packetContextPathRE.FindAllString(line, -1)...)
	}
	for _, name := range names {
		ref, err := resolvePacketContext(root, slug, name)
		if err != nil {
			return err
		}
		path := filepath.Join(ref.dir, ref.name)
		if _, ok := bound[path]; !ok {
			return fmt.Errorf("unbound packet context %s; add context <SHA256> %s to the packet", name, name)
		}
	}
	return nil
}

// Caller may already hold the feature lock (dispatch); never acquire it here.
func lintPacketDiffs(root, slug, packet string, required bool) error {
	raw, err := readBoundedRegularFile(packet, maxContractBytes)
	if err != nil {
		return fmt.Errorf("reviewer packet %s: %w", packet, err)
	}
	if err := lintPacketContexts(root, slug, raw); err != nil {
		return err
	}
	bindings, err := frozenDiffBindings(raw)
	if err != nil {
		return err
	}
	bound := make(map[string]string, len(bindings))
	for _, binding := range bindings {
		path, err := resolvePacketDiff(root, slug, binding.name)
		if err != nil {
			return err
		}
		if prior, ok := bound[path]; ok {
			if prior != binding.digest {
				return fmt.Errorf("conflicting diff checksums for %s", binding.name)
			}
			continue
		}
		if err := verifyFrozenDiff(path, binding.digest); err != nil {
			return fmt.Errorf("%s: %w", binding.name, err)
		}
		bound[path] = binding.digest
	}
	var names []string
	for _, line := range strings.Split(string(raw), "\n") {
		if entry, candidate := strings.CutPrefix(line, "file "); candidate {
			identity, path, complete := strings.Cut(entry, " ")
			// Candidate file identities describe product paths, not reviewer inputs.
			if complete && path != "" {
				if identity == "absent" {
					continue
				}
				if digest, err := hex.DecodeString(identity); err == nil && len(digest) == sha256.Size {
					continue
				}
			}
		}
		names = append(names, diffPathRE.FindAllString(line, -1)...)
	}
	if required && len(names) == 0 {
		return errors.New("a Review or Seal reviewer packet must name the frozen candidate diff and bind its bytes with diff <SHA256> <path.diff|path.patch>")
	}
	for _, name := range names {
		path, err := resolvePacketDiff(root, slug, name)
		if err != nil {
			return err
		}
		if _, ok := bound[path]; !ok {
			return fmt.Errorf("unbound frozen candidate diff %s; add diff <SHA256> %s to the packet", name, name)
		}
	}
	return nil
}
