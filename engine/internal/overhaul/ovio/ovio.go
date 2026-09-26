// Package ovio holds the small file, digest and JSON helpers shared by the
// /overhaul run-area tools. Run records are plain JSON documents read as
// generic maps so the tools enforce exactly the documented fields.
package ovio

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SHA256Bytes returns the lowercase hex SHA-256 of b.
func SHA256Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// SHA256File returns the lowercase hex SHA-256 of the file's exact bytes.
func SHA256File(path string) (string, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- callers pass operator-selected run-area paths
	if err != nil {
		return "", err
	}
	return SHA256Bytes(b), nil
}

// LoadJSON decodes a UTF-8 JSON document. Numbers stay json.Number so digests,
// weights and ratios are never rounded through float64 by accident.
func LoadJSON(path string) (any, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- callers pass operator-selected run-area paths
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON (%v)", path, err)
	}
	return v, nil
}

// LoadObject decodes a JSON document that must be an object.
func LoadObject(path string) (map[string]any, error) {
	v, err := LoadJSON(path)
	if err != nil {
		return nil, err
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected a JSON object", path)
	}
	return m, nil
}

// MarshalIndent renders v as two-space indented JSON with sorted object keys
// and a trailing newline.
func MarshalIndent(v any) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// WriteJSON writes v as indented JSON.
func WriteJSON(path string, v any, perm os.FileMode) error {
	b, err := MarshalIndent(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, perm)
}

// resolve follows symlinks for the longest existing prefix of path, like
// Python's os.path.realpath on a path that may not exist yet.
func resolve(path string) string {
	path = filepath.Clean(path)
	if r, err := filepath.EvalSymlinks(path); err == nil {
		return r
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(resolve(parent), filepath.Base(path))
}

// Resolve is the exported form of resolve for callers that compare paths.
func Resolve(path string) string { return resolve(path) }

// Get walks a dotted path through nested objects; missing keys give nil.
func Get(v any, dotted string) any {
	for _, k := range strings.Split(dotted, ".") {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// Str returns v as a string, or "" when it is not one.
func Str(v any) string {
	s, _ := v.(string)
	return s
}

// List returns v as a slice, or nil when it is not one.
func List(v any) []any {
	l, _ := v.([]any)
	return l
}

// Obj returns v as an object, or nil when it is not one.
func Obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

// Strings returns the string members of a JSON array.
func Strings(v any) []string {
	var out []string
	for _, x := range List(v) {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Truthy mirrors JSON truthiness used by the run records: nil, false, "",
// 0, empty arrays and empty objects are false.
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case json.Number:
		return x.String() != "0" && x.String() != "0.0"
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

// Domains is the fixed set of score domains in their documented order. Findings,
// receipts, the applicability matrix and the rubric all use these keys.
var Domains = []string{"correctness", "security", "reliability", "performance", "tests", "architecture", "ux", "operations"}
