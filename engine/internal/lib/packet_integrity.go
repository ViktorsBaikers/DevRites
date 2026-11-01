package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean != filepath.FromSlash(name) {
		return "", fmt.Errorf("unsafe frozen candidate diff path %q", name)
	}
	dirs := []string{projectDir(root), featureDir(root, slug)}
	// Freeze's checksum rows use this workspace-relative namespace, never a product packets/ directory.
	if strings.HasPrefix(name, "packets/") {
		dirs = []string{featureDir(root, slug)}
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

// Caller may already hold the feature lock (dispatch); never acquire it here.
func lintPacketDiffs(root, slug, packet string, required bool) error {
	raw, err := readBoundedRegularFile(packet, maxContractBytes)
	if err != nil {
		return fmt.Errorf("reviewer packet %s: %w", packet, err)
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
