package lib

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/devrites/devrites/internal/devritespaths"
)

const (
	// Candidate bounds cap every untrusted filesystem input before hashing.
	maxCandidateManifestBytes = int64(1 << 20)
	maxCandidateRows          = 4096
	maxCandidatePathBytes     = 4096
	maxCandidateFileBytes     = int64(64 << 20)
	maxCandidateTotalBytes    = int64(256 << 20)
	maxCandidateArtifactBytes = int64(1 << 20)
	candidateDigestDomain     = "devrites-candidate-sha256"
	candidateDigestVersion    = "v1"
)

type candidateRow struct {
	state      string
	path       string
	executable bool
	size       int64
	info       os.FileInfo
}

type candidateIdentity struct {
	digest string
	rows   []candidateRow
}

// CandidateIdentity validates and hashes the closed project candidate for slug.
func CandidateIdentity(root, slug string) (string, int, error) {
	identity, err := loadCandidateIdentity(root, slug)
	if err != nil {
		return "", 0, err
	}
	return identity.digest, len(identity.rows), nil
}

func loadCandidateIdentity(root, slug string) (candidateIdentity, error) {
	workspace, err := devritespaths.ExistingFeatureDirChecked(root, slug)
	if err != nil {
		return candidateIdentity{}, err
	}
	manifestPath := filepath.Join(workspace, "touched-files.md")
	raw, err := readBoundedRegularFile(manifestPath, maxCandidateManifestBytes)
	if err != nil {
		return candidateIdentity{}, fmt.Errorf("candidate manifest %s: %w; refresh touched-files.md", manifestPath, err)
	}
	rows, err := parseCandidateManifest(raw)
	if err != nil {
		return candidateIdentity{}, fmt.Errorf("candidate manifest %s: %w; refresh touched-files.md", manifestPath, err)
	}
	project := projectDir(root)
	var total int64
	for i := range rows {
		if err := validateCandidateRow(project, &rows[i]); err != nil {
			return candidateIdentity{}, fmt.Errorf("candidate %q: %w", rows[i].path, err)
		}
		if rows[i].state == "present" {
			// ponytail: the 4096-row cap bounds portable pairwise identity checks; index native file IDs if that cap grows.
			for prior := range rows[:i] {
				if rows[prior].state == "present" && os.SameFile(rows[prior].info, rows[i].info) {
					return candidateIdentity{}, fmt.Errorf("candidate %q: same filesystem object as candidate %q", rows[i].path, rows[prior].path)
				}
			}
			if rows[i].size > maxCandidateTotalBytes-total {
				return candidateIdentity{}, errors.New("candidate present files exceed 256 MiB aggregate limit")
			}
			total += rows[i].size
		}
	}
	digest, err := hashCandidate(project, rows)
	if err != nil {
		return candidateIdentity{}, err
	}
	return candidateIdentity{digest: digest, rows: rows}, nil
}

func readBoundedRegularFile(name string, limit int64) ([]byte, error) {
	before, err := os.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, errors.New("artifact is not a regular file")
	}
	if before.Size() < 0 || before.Size() > limit {
		return nil, fmt.Errorf("artifact exceeds %s limit", byteLimitName(limit))
	}
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || opened.Size() != before.Size() || !os.SameFile(before, opened) {
		return nil, errors.New("artifact changed type while opening")
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("artifact exceeds %s limit", byteLimitName(limit))
	}
	after, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !after.Mode().IsRegular() || after.Size() != opened.Size() || after.Size() != int64(len(content)) || !os.SameFile(opened, after) {
		return nil, errors.New("artifact changed size or type while reading")
	}
	return content, nil
}

func byteLimitName(limit int64) string {
	switch limit {
	case 1 << 20:
		return "1 MiB"
	case 64 << 20:
		return "64 MiB"
	case 256 << 20:
		return "256 MiB"
	default:
		return fmt.Sprintf("%d-byte", limit)
	}
}

func parseCandidateManifest(raw []byte) ([]candidateRow, error) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if countCandidateHeading(lines, "## Touched files") != 1 {
		return nil, errors.New("requires exactly one ## Touched files section")
	}
	start := -1
	count := 0
	for i, line := range lines {
		if strings.TrimSpace(line) == "## Candidate manifest" {
			start = i + 1
			count++
		}
	}
	if count != 1 {
		return nil, errors.New("requires exactly one ## Candidate manifest section")
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			end = i
			break
		}
	}
	body := trimBlankLines(lines[start:end])
	if len(body) == 1 && body[0] == "No project files." {
		return nil, nil
	}
	if len(body) < 3 {
		return nil, errors.New("requires No project files. or a strict table")
	}
	if body[0] != "| State | File | Slice | Reason |" {
		return nil, errors.New("table header must be | State | File | Slice | Reason |")
	}
	if body[1] != "| --- | --- | --- | --- |" {
		return nil, errors.New("table separator must contain four --- columns")
	}
	rows := make([]candidateRow, 0, len(body)-2)
	seen := make(map[string]struct{}, len(body)-2)
	seenFolded := make(map[string]string, len(body)-2)
	for i, line := range body[2:] {
		cells := strings.Split(line, "|")
		if len(cells) != 6 || strings.TrimSpace(cells[0]) != "" || strings.TrimSpace(cells[5]) != "" {
			return nil, fmt.Errorf("row %d must contain exactly four table columns", i+1)
		}
		state := strings.TrimSpace(cells[1])
		if state != "present" && state != "deleted" {
			return nil, fmt.Errorf("row %d state must be present or deleted", i+1)
		}
		fileCell := strings.TrimSpace(cells[2])
		if len(fileCell) < 2 || fileCell[0] != '`' || fileCell[len(fileCell)-1] != '`' || strings.Contains(fileCell[1:len(fileCell)-1], "`") {
			return nil, fmt.Errorf("row %d File must use one pair of backticks", i+1)
		}
		normalized, err := normalizeCandidatePath(fileCell[1 : len(fileCell)-1])
		if err != nil {
			return nil, fmt.Errorf("row %d File: %w", i+1, err)
		}
		if strings.TrimSpace(cells[3]) == "" {
			return nil, fmt.Errorf("row %d Slice must be nonempty", i+1)
		}
		if strings.TrimSpace(cells[4]) == "" {
			return nil, fmt.Errorf("row %d Reason must be nonempty", i+1)
		}
		if _, ok := seen[normalized]; ok {
			return nil, fmt.Errorf("duplicate normalized path %q", normalized)
		}
		folded := foldCandidatePath(normalized)
		if prior, ok := seenFolded[folded]; ok {
			return nil, fmt.Errorf("case-fold collision between %q and %q", prior, normalized)
		}
		if len(rows) > 0 && rows[len(rows)-1].path >= normalized {
			return nil, fmt.Errorf("rows must be strictly sorted by normalized File: %q precedes %q", rows[len(rows)-1].path, normalized)
		}
		seen[normalized] = struct{}{}
		seenFolded[folded] = normalized
		rows = append(rows, candidateRow{state: state, path: normalized})
		if len(rows) > maxCandidateRows {
			return nil, errors.New("manifest exceeds 4096 rows")
		}
	}
	if len(rows) == 0 {
		return nil, errors.New("table must contain at least one row")
	}
	return rows, nil
}

func foldCandidatePath(value string) string {
	var folded strings.Builder
	for _, char := range value {
		canonical := char
		for next := unicode.SimpleFold(char); next != char; next = unicode.SimpleFold(next) {
			if next < canonical {
				canonical = next
			}
		}
		folded.WriteRune(canonical)
	}
	return folded.String()
}

func countCandidateHeading(lines []string, heading string) int {
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == heading {
			count++
		}
	}
	return count
}

func trimBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func normalizeCandidatePath(value string) (string, error) {
	if value == "" {
		return "", errors.New("path is empty")
	}
	if len(value) > maxCandidatePathBytes {
		return "", errors.New("path exceeds 4096 bytes")
	}
	if !utf8.ValidString(value) {
		return "", errors.New("path is not valid UTF-8")
	}
	if strings.Contains(value, "\\") {
		return "", errors.New("path contains a backslash")
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return "", errors.New("path contains a control character")
		}
	}
	if strings.HasPrefix(value, "/") || path.IsAbs(value) {
		return "", errors.New("path is absolute")
	}
	if len(value) >= 2 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (len(value) == 2 || value[2] == '/') {
		return "", errors.New("path has a drive prefix")
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		switch part {
		case "":
			return "", errors.New("path contains an empty component")
		case ".":
			return "", errors.New("path contains a dot component")
		case "..":
			return "", errors.New("path contains traversal")
		}
		if strings.ContainsAny(part, `<>:"|?*`) {
			return "", errors.New("path component contains a Windows-reserved character")
		}
		if strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", errors.New("path component has a trailing dot or space")
		}
		base := part
		if dot := strings.IndexByte(base, '.'); dot >= 0 {
			base = base[:dot]
		}
		reserved := strings.ToUpper(base)
		if reserved == "CON" || reserved == "PRN" || reserved == "AUX" || reserved == "NUL" ||
			(len(reserved) == 4 && (strings.HasPrefix(reserved, "COM") || strings.HasPrefix(reserved, "LPT")) && reserved[3] >= '1' && reserved[3] <= '9') {
			return "", errors.New("path component uses a Windows-reserved name")
		}
	}
	normalized := path.Clean(value)
	if normalized == "." || normalized != value {
		return "", errors.New("path is ambiguous")
	}
	if forbiddenCandidateStatePath(normalized) {
		return "", errors.New("path names repository or ephemeral workflow state path")
	}
	return normalized, nil
}

func forbiddenCandidateStatePath(name string) bool {
	parts := strings.Split(name, "/")
	if strings.EqualFold(parts[0], ".git") {
		return true
	}
	if !strings.EqualFold(parts[0], ".devrites") {
		return false
	}
	if parts[0] != ".devrites" {
		return true
	}
	if name == ".devrites/principles.md" {
		return false
	}
	return len(parts) < 3 || parts[1] != "specs"
}

func validateCandidateRow(project string, row *candidateRow) error {
	info, exists, err := candidatePathInfo(project, row.path)
	if err != nil {
		return err
	}
	if row.state == "deleted" {
		if exists {
			return errors.New("deleted path still exists")
		}
		return nil
	}
	if !exists {
		return errors.New("present path is missing")
	}
	if !info.Mode().IsRegular() {
		return errors.New("present path is not a regular file")
	}
	if info.Size() < 0 || info.Size() > maxCandidateFileBytes {
		return errors.New("present file exceeds 64 MiB limit")
	}
	row.executable = info.Mode().Perm()&0o111 != 0
	row.size = info.Size()
	row.info = info
	return nil
}

func candidatePathInfo(project, name string) (os.FileInfo, bool, error) {
	current := project
	parts := strings.Split(name, "/")
	for i, part := range parts {
		current = filepath.Join(current, filepath.FromSlash(part))
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, false, errors.New("path contains a symlink")
		}
		if i < len(parts)-1 && !info.IsDir() {
			return nil, false, errors.New("path parent is not a directory")
		}
		if i == len(parts)-1 {
			return info, true, nil
		}
	}
	return nil, false, errors.New("path is empty")
}

func hashCandidate(project string, rows []candidateRow) (string, error) {
	digest := sha256.New()
	writeCandidateField(digest, candidateDigestDomain)
	writeCandidateField(digest, candidateDigestVersion)
	writeCandidateField(digest, fmt.Sprintf("%d", len(rows)))
	for _, row := range rows {
		writeCandidateField(digest, row.state)
		writeCandidateField(digest, row.path)
		if row.state == "deleted" {
			writeCandidateField(digest, "absent")
			writeCandidateField(digest, "0")
			writeCandidateLength(digest, 0)
			continue
		}
		writeCandidateField(digest, "regular")
		if row.executable {
			writeCandidateField(digest, "1")
		} else {
			writeCandidateField(digest, "0")
		}
		writeCandidateLength(digest, uint64(row.size)) // #nosec G115 -- validateCandidateRow bounds size to 0..64 MiB
		if err := hashCandidateFile(digest, project, row); err != nil {
			return "", fmt.Errorf("candidate %q: %w", row.path, err)
		}
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func writeCandidateField(writer hash.Hash, value string) {
	writeCandidateLength(writer, uint64(len(value)))
	_, _ = io.WriteString(writer, value)
}

func writeCandidateLength(writer hash.Hash, size uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], size)
	_, _ = writer.Write(encoded[:])
}

func hashCandidateFile(digest hash.Hash, project string, row candidateRow) error {
	if _, exists, err := candidatePathInfo(project, row.path); err != nil || !exists {
		if err != nil {
			return err
		}
		return errors.New("present path disappeared before reading")
	}
	name := filepath.Join(project, filepath.FromSlash(row.path))
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return err
	}
	if !opened.Mode().IsRegular() || opened.Size() != row.size || (opened.Mode().Perm()&0o111 != 0) != row.executable || !os.SameFile(row.info, opened) {
		return errors.New("file changed size or type before reading")
	}
	written, err := io.CopyN(digest, file, row.size)
	if err != nil || written != row.size {
		return errors.New("file changed size while reading")
	}
	var extra [1]byte
	if n, readErr := file.Read(extra[:]); n != 0 || (readErr != nil && !errors.Is(readErr, io.EOF)) {
		return errors.New("file changed size while reading")
	}
	after, err := file.Stat()
	if err != nil {
		return err
	}
	current, exists, err := candidatePathInfo(project, row.path)
	if err != nil {
		return err
	}
	if !exists || !after.Mode().IsRegular() || after.Size() != row.size || (after.Mode().Perm()&0o111 != 0) != row.executable || !os.SameFile(opened, after) || !os.SameFile(opened, current) {
		return errors.New("file changed size or type while reading")
	}
	return nil
}
