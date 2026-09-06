package install

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/version"
)

func readManifest(path string) (map[string]managedRecord, error) {
	records, _, err := parseManifest(path)
	return records, err
}

func readManifestList(path string) ([]string, error) {
	_, entries, err := parseManifest(path)
	return entries, err
}

func parseManifest(path string) (map[string]managedRecord, []string, error) {
	// #nosec G304 -- install manifest under the operator-selected target
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]managedRecord{}, nil, nil
		}
		return nil, nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	hashes := map[string]string{}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# managed: ") {
			record := strings.TrimPrefix(line, "# managed: ")
			i := strings.LastIndex(record, " sha256:")
			if i > 0 {
				rel := strings.TrimSpace(record[:i])
				hash := "sha256:" + strings.ToLower(strings.TrimSpace(record[i+len(" sha256:"):]))
				if filepath.IsLocal(filepath.FromSlash(rel)) && validManagedHash(hash) {
					hashes[rel] = hash
				}
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// The manifest is hand-editable; an entry that escapes the target
		// (absolute or ..-traversal) must never be overwritten or removed.
		if !filepath.IsLocal(filepath.FromSlash(line)) {
			continue
		}
		out = append(out, line)
	}
	records := make(map[string]managedRecord, len(out))
	for _, rel := range out {
		records[rel] = managedRecord{Hash: hashes[rel]}
	}
	return records, out, nil
}

func validManagedHash(hash string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(hash, prefix) || len(hash) != len(prefix)+sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(hash, prefix))
	return err == nil
}

func manifestHeader(path, key string) string {
	// #nosec G304 -- install manifest under the operator-selected target
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	prefix := "# " + key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func installedVersion(source string) string {
	if source != "" {
		// #nosec G304 -- package.json inside the operator-selected install source
		data, err := os.ReadFile(filepath.Join(source, "package.json"))
		if err != nil {
			return "unknown"
		}
		var pkg struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &pkg) != nil || pkg.Version == "" {
			return "unknown"
		}
		return pkg.Version
	}
	if version.Version != "" && version.Version != "dev" {
		return strings.TrimPrefix(version.Version, "v")
	}
	return "unknown"
}
