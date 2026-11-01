package lib

import (
	"bytes"
	"crypto/sha256"
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
	var source contextSource
	if source.err = contextParents(s.dir, name); source.err == nil {
		source.data, source.err = readBoundedRegularFile(filepath.Join(s.dir, name), workspaceArtifactLimit)
	}
	if source.err != nil {
		source.err = fmt.Errorf("workspace artifact %s: %w", name, source.err)
	}
	if s.files == nil {
		s.files = make(map[string]contextSource)
	}
	s.files[name] = source
	return source.data, source.err
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
