package lib

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
)

type contextSource struct {
	data []byte
	err  error
}

// contextSources retains one invocation's bytes, including optional absence.
type contextSources struct {
	dir   string
	files map[string]contextSource
}

func (s *contextSources) read(name string) ([]byte, error) {
	if s.dir == "" {
		return nil, os.ErrNotExist
	}
	name = filepath.Clean(filepath.FromSlash(name))
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("workspace artifact %q escapes workspace", name)
	}
	if source, ok := s.files[name]; ok {
		return source.data, source.err
	}
	source := contextSource{}
	source.data, source.err = readContextArtifact(s.dir, name)
	if source.err != nil {
		source.err = fmt.Errorf("workspace artifact %s: %w", name, source.err)
	}
	if s.files == nil {
		s.files = make(map[string]contextSource)
	}
	s.files[name] = source
	return source.data, source.err
}

// Acquire workspace leaves through a confined root, not a checked absolute path.
func readContextArtifact(dir, name string) ([]byte, error) {
	if !filepath.IsLocal(name) {
		return nil, fmt.Errorf("workspace artifact %q escapes workspace", name)
	}
	if err := contextParents(dir, name); err != nil {
		return nil, err
	}
	ws, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = ws.Close() }()
	before, err := ws.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("artifact is not a regular file")
	}
	if before.Size() < 0 || before.Size() > workspaceArtifactLimit {
		return nil, fmt.Errorf("artifact exceeds %s limit", byteLimitName(workspaceArtifactLimit))
	}
	file, err := ws.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || opened.Size() != before.Size() || !os.SameFile(before, opened) {
		return nil, errors.New("artifact changed type while opening")
	}
	body, err := io.ReadAll(io.LimitReader(file, workspaceArtifactLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > workspaceArtifactLimit {
		return nil, fmt.Errorf("artifact exceeds %s limit", byteLimitName(workspaceArtifactLimit))
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	current, err := ws.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || after.Size() != opened.Size() || after.Size() != int64(len(body)) ||
		!os.SameFile(opened, after) || !current.Mode().IsRegular() || !os.SameFile(opened, current) ||
		current.Size() != after.Size() {
		return nil, errors.New("artifact changed size or type while reading")
	}
	return body, nil
}

// Reject even in-workspace parent links: managed paths name actual directories.
func contextParents(dir, rel string) error {
	parent := filepath.Dir(rel)
	if parent == "." {
		return nil
	}
	for _, part := range strings.Split(parent, string(filepath.Separator)) {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("context parent %s is not a directory (symlinks are not allowed)", dir)
		}
	}
	return nil
}

func publishContext(featureDir string, opts contextOpts, body []byte) (out string, unchanged bool, err error) {
	out = opts.out
	managed := featureDir != "" && (out == "" || !filepath.IsAbs(out))
	if out == "" {
		out = filepath.Join("ctx", bundleName(opts))
	}
	base, rel := featureDir, out
	if !managed && featureDir != "" {
		absoluteFeature, absErr := filepath.Abs(featureDir)
		if absErr != nil {
			return out, false, absErr
		}
		if within, relErr := filepath.Rel(absoluteFeature, out); relErr == nil && filepath.IsLocal(within) {
			managed, rel = true, within
		}
	}
	if managed {
		if !filepath.IsLocal(rel) {
			return out, false, fmt.Errorf("context output %q escapes workspace", out)
		}
		if err := contextParents(base, rel); err != nil {
			return out, false, err
		}
		out = filepath.Join(base, rel)
	} else {
		base, rel = filepath.Dir(out), filepath.Base(out)
		if err := os.MkdirAll(base, 0o755); err != nil {
			return out, false, err
		}
	}
	ws, err := os.OpenRoot(base)
	if err != nil {
		return out, false, err
	}
	defer func() { _ = ws.Close() }()
	if err := ws.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return out, false, err
	}
	digest := []byte(fmt.Sprintf("%x\n", sha256.Sum256(body)))
	// Check both destinations before publishing either; never follow a leaf link.
	matches := true
	for _, entry := range []struct {
		name string
		data []byte
	}{{rel, body}, {rel + ".sha256", digest}} {
		info, err := ws.Lstat(entry.name)
		if os.IsNotExist(err) {
			matches = false
			continue
		}
		if err != nil {
			return out, false, err
		}
		if !info.Mode().IsRegular() {
			return out, false, fmt.Errorf("context output %s is not a regular file", entry.name)
		}
		// shortcut: Windows FileMode cannot prove POSIX 0600; rebuild until ACL-aware reuse is needed.
		if info.Mode() != 0o600 || info.Size() != int64(len(entry.data)) {
			matches = false
			continue
		}
		// Open through the confined root; replacement still uses an atomic rename.
		file, err := ws.Open(entry.name)
		if err != nil {
			return out, false, err
		}
		opened, err := file.Stat()
		if err == nil && (opened.Mode() != 0o600 || !os.SameFile(info, opened)) {
			err = fmt.Errorf("context output %s changed while opening", entry.name)
		}
		var data []byte
		if err == nil {
			data, err = io.ReadAll(io.LimitReader(file, int64(len(entry.data))+1))
		}
		if err == nil {
			current, statErr := ws.Lstat(entry.name)
			if statErr != nil {
				err = statErr
			} else if current.Mode() != 0o600 || !os.SameFile(opened, current) {
				err = fmt.Errorf("context output %s changed while reading", entry.name)
			}
		}
		_ = file.Close()
		if err != nil {
			return out, false, err
		}
		matches = matches && bytes.Equal(data, entry.data)
	}
	if matches {
		return out, true, nil
	}
	if err := fsutil.WriteFileAtomicIn(ws, rel, body, 0o600); err != nil {
		return out, false, err
	}
	if err := fsutil.WriteFileAtomicIn(ws, rel+".sha256", digest, 0o600); err != nil {
		return out, false, err
	}
	return out, false, nil
}

// Caller holds the feature lock; a wave/role read-set is never replaced.
func retainPacketContext(workspace, name string, body []byte) error {
	rel := filepath.Join("packets", name)
	if err := contextParents(workspace, rel); err != nil {
		return err
	}
	prior, err := (&contextSources{dir: workspace}).read(rel)
	if err == nil {
		if !bytes.Equal(prior, body) {
			return fmt.Errorf("%s already retains a different context; render the new read-set under a new wave", rel)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	ws, err := os.OpenRoot(workspace)
	if err != nil {
		return err
	}
	defer func() { _ = ws.Close() }()
	if err := ws.MkdirAll("packets", 0o755); err != nil {
		return err
	}
	return fsutil.WriteFileAtomicIn(ws, rel, body, 0o600)
}
