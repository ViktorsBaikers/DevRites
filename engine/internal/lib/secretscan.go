package lib

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type secretFinding struct {
	Severity string
	Path     string
	Kind     string
	Offset   int
}

type stagedBlob struct {
	Path string
	OID  string
}

const (
	maxSecretScanBytes    = int64(64) << 20
	maxSecretScanEntries  = 4096
	maxSecretScanFindings = 4096
)

var errSecretScanLimit = errors.New("secret scan limit exceeded")

type secretScanBudget struct {
	bytesRemaining    int64
	findingsRemaining int
}

type boundedSecretScanBuffer struct {
	buffer   bytes.Buffer
	limit    int64
	overflow bool
}

func (w *boundedSecretScanBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	remaining := w.limit - int64(w.buffer.Len())
	if remaining <= 0 {
		w.overflow = true
		return 0, errSecretScanLimit
	}
	if int64(len(p)) > remaining {
		n, _ := w.buffer.Write(p[:int(remaining)])
		w.overflow = true
		return n, errSecretScanLimit
	}
	_, _ = w.buffer.Write(p)
	return len(p), nil
}

func (w *boundedSecretScanBuffer) Bytes() []byte {
	return w.buffer.Bytes()
}

func (w *boundedSecretScanBuffer) Len() int {
	return w.buffer.Len()
}

var secretPatterns = []struct {
	sev, kind string
	re        *regexp.Regexp
}{
	{"HIGH", "private-key", regexp.MustCompile(`-----BEGIN (RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----`)},
	{"HIGH", "aws-secret", regexp.MustCompile(`(?i)aws(.{0,20})?(secret|access).{0,20}[=:]\s*['\"]?[A-Za-z0-9/+=]{30,}`)},
	{"HIGH", "github-token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{30,}`)},
	{"HIGH", "slack-token", regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]{20,}`)},
	{"MEDIUM", "generic-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password)\s*[:=]\s*['\"][^'\"\n]{16,}['\"]`)},
	{"LOW", "placeholder-secret", regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password).{0,20}(example|changeme|placeholder|dummy)`)},
}

// SecretScan scans staged blobs, stdin, or touched files for credentials. HIGH exits 3.
func SecretScan(root string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	project := projectDir(root)
	staged := false
	readStdin := false
	slug := ""
	for _, arg := range args {
		switch arg {
		case "--staged":
			staged = true
		case "--stdin":
			readStdin = true
		case "--text":
			fmt.Fprintln(stderr, "secret-scan: review text must be supplied with --stdin")
			return 2
		default:
			if strings.HasPrefix(arg, "-") || slug != "" {
				fmt.Fprintln(stderr, "usage: devrites-engine secret-scan [--staged] [--stdin] [slug]")
				return 2
			}
			slug = arg
		}
	}
	if staged && slug != "" {
		fmt.Fprintln(stderr, "usage: devrites-engine secret-scan [--staged] [--stdin] [slug]")
		return 2
	}

	budget := &secretScanBudget{
		bytesRemaining:    maxSecretScanBytes,
		findingsRemaining: maxSecretScanFindings,
	}
	var findings []secretFinding
	if staged {
		blobs, err := gitStagedBlobs(project, budget)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect staged index")
			return 2
		}
		stagedFindings, err := scanStagedBlobs(project, blobs, budget)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect staged blobs")
			return 2
		}
		findings = append(findings, stagedFindings...)
	}

	if !staged && slug == "" && !readStdin {
		slug = activeSlug(root)
	}
	var paths []string
	if slug != "" {
		identity, err := loadCandidateIdentity(root, slug)
		if err != nil {
			fmt.Fprintf(stderr, "secret-scan: cannot inspect candidate manifest: %v\n", err)
			return 2
		}
		for _, row := range identity.rows {
			if row.state == "present" {
				paths = append(paths, row.path)
			}
		}
	}
	if !staged && slug == "" && !readStdin {
		changed, err := gitDiffNames(project)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect changed paths")
			return 2
		}
		paths = append(paths, changed...)
	}
	if len(paths) != 0 {
		worktreeFindings, err := scanWorktreePaths(project, paths, budget)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect changed source")
			return 2
		}
		findings = append(findings, worktreeFindings...)
	}
	if readStdin {
		body, err := readSecretScanInput(stdin, budget)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect stdin")
			return 2
		}
		stdinFindings, err := scanCountedSecrets("<stdin>", body, budget)
		if err != nil {
			fmt.Fprintln(stderr, "secret-scan: cannot inspect stdin")
			return 2
		}
		findings = append(findings, stdinFindings...)
	}
	sort.Slice(findings, func(i, j int) bool {
		left, right := findings[i], findings[j]
		if left.Severity != right.Severity {
			return left.Severity < right.Severity
		}
		if left.Path != right.Path {
			return left.Path < right.Path
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.Offset < right.Offset
	})
	return writeSecretScanResult(findings, stdout, stderr)
}

func projectDir(root string) string {
	if filepath.Base(root) == ".devrites" {
		return filepath.Dir(root)
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func readSecretScanInput(reader io.Reader, budget *secretScanBudget) ([]byte, error) {
	content, err := io.ReadAll(io.LimitReader(reader, budget.bytesRemaining+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > budget.bytesRemaining {
		return nil, errSecretScanLimit
	}
	budget.bytesRemaining -= int64(len(content))
	return content, nil
}

func (budget *secretScanBudget) readGitOutput(project string, input []byte, args ...string) ([]byte, error) {
	output := boundedSecretScanBuffer{limit: budget.bytesRemaining}
	_, runErr := runGitCommandIO(project, nil, input, &output, args...)
	if output.overflow {
		return nil, errSecretScanLimit
	}
	if runErr != nil {
		return nil, runErr
	}
	budget.bytesRemaining -= int64(output.Len())
	return output.Bytes(), nil
}

func scanCountedSecrets(path string, content []byte, budget *secretScanBudget) ([]secretFinding, error) {
	var out []secretFinding
	source := secretSourceLabel(path)
	for _, pattern := range secretPatterns {
		matches := pattern.re.FindAllIndex(content, budget.findingsRemaining+1)
		if len(matches) > budget.findingsRemaining {
			return nil, errSecretScanLimit
		}
		for _, match := range matches {
			out = append(out, secretFinding{Severity: pattern.sev, Path: source, Kind: pattern.kind, Offset: match[0]})
		}
		budget.findingsRemaining -= len(matches)
	}
	return out, nil
}

func secretSourceLabel(path string) string {
	for _, pattern := range secretPatterns {
		if pattern.re.FindStringIndex(path) != nil {
			return "<redacted-path>"
		}
	}
	return path
}

func gitStagedBlobs(project string, budget *secretScanBudget) ([]stagedBlob, error) {
	out, err := budget.readGitOutput(project, nil, "--no-replace-objects", "diff", "--cached", "--raw", "-z", "--no-abbrev", "--no-renames", "--no-ext-diff", "--")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	if bytes.Count(out, []byte{0}) > maxSecretScanEntries*2 {
		return nil, errSecretScanLimit
	}
	fields := bytes.Split(out, []byte{0})
	if len(fields)%2 != 1 || len(fields[len(fields)-1]) != 0 {
		return nil, errors.New("malformed staged index")
	}
	if len(fields)/2 > maxSecretScanEntries {
		return nil, errSecretScanLimit
	}
	blobs := make([]stagedBlob, 0, len(fields)/2)
	for i := 0; i < len(fields)-1; i += 2 {
		header := strings.Fields(string(fields[i]))
		path := string(fields[i+1])
		if len(header) != 5 || !strings.HasPrefix(header[0], ":") || path == "" || header[4] == "" {
			return nil, errors.New("malformed staged index")
		}
		mode, oid, status := header[1], header[3], header[4][0]
		if status == 'D' {
			if mode != "000000" || !isZeroObjectID(oid) {
				return nil, errors.New("malformed staged deletion")
			}
			continue
		}
		if status != 'A' && status != 'M' && status != 'T' {
			return nil, errors.New("unsupported staged status")
		}
		if mode != "100644" && mode != "100755" && mode != "120000" {
			return nil, errors.New("unsupported staged object mode")
		}
		if !isObjectID(oid) || isZeroObjectID(oid) {
			return nil, errors.New("malformed staged object")
		}
		blobs = append(blobs, stagedBlob{Path: path, OID: oid})
	}
	return blobs, nil
}

func scanStagedBlobs(project string, blobs []stagedBlob, budget *secretScanBudget) ([]secretFinding, error) {
	if len(blobs) == 0 {
		return nil, nil
	}
	if len(blobs) > maxSecretScanEntries {
		return nil, errSecretScanLimit
	}
	var input bytes.Buffer
	for _, blob := range blobs {
		input.WriteString(blob.OID)
		input.WriteByte('\n')
	}
	data, err := budget.readGitOutput(project, input.Bytes(), "--no-replace-objects", "cat-file", "--batch")
	if err != nil {
		return nil, err
	}

	cursor := 0
	var findings []secretFinding
	for _, blob := range blobs {
		lineEnd := bytes.IndexByte(data[cursor:], '\n')
		if lineEnd < 0 {
			return nil, errors.New("malformed blob header")
		}
		header := strings.Fields(string(data[cursor : cursor+lineEnd]))
		cursor += lineEnd + 1
		if len(header) != 3 || header[0] != blob.OID || header[1] != "blob" {
			return nil, errors.New("unreadable staged blob")
		}
		size, err := strconv.Atoi(header[2])
		if err != nil || size < 0 || size >= len(data)-cursor {
			return nil, errors.New("malformed staged blob size")
		}
		end := cursor + size
		if data[end] != '\n' {
			return nil, errors.New("malformed staged blob boundary")
		}
		blobFindings, err := scanCountedSecrets(blob.Path, data[cursor:end], budget)
		if err != nil {
			return nil, err
		}
		findings = append(findings, blobFindings...)
		cursor = end + 1
	}
	if cursor != len(data) {
		return nil, errors.New("unexpected staged blob output")
	}
	return findings, nil
}

func isObjectID(oid string) bool {
	if len(oid) != 40 && len(oid) != 64 {
		return false
	}
	_, err := hex.DecodeString(oid)
	return err == nil
}

func isZeroObjectID(oid string) bool {
	return isObjectID(oid) && strings.Trim(oid, "0") == ""
}

func scanWorktreePaths(project string, paths []string, budget *secretScanBudget) ([]secretFinding, error) {
	if len(paths) > maxSecretScanEntries {
		return nil, errSecretScanLimit
	}
	seen := map[string]bool{}
	var findings []secretFinding
	for _, path := range paths {
		clean := filepath.Clean(path)
		if path == "" || seen[clean] {
			continue
		}
		seen[clean] = true
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return nil, errors.New("changed path escapes project")
		}
		fullPath := filepath.Join(project, clean)
		info, err := os.Lstat(fullPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			return nil, errors.New("changed source is not a regular file")
		}
		if info.Size() < 0 || info.Size() > budget.bytesRemaining {
			return nil, errSecretScanLimit
		}
		file, err := os.Open(fullPath)
		if err != nil {
			return nil, err
		}
		content, readErr := readSecretScanInput(file, budget)
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		pathFindings, err := scanCountedSecrets(clean, content, budget)
		if err != nil {
			return nil, err
		}
		findings = append(findings, pathFindings...)
	}
	return findings, nil
}

func writeSecretScanResult(findings []secretFinding, stdout, stderr io.Writer) int {
	report := boundedSecretScanBuffer{limit: maxSecretScanBytes}
	high := false
	if len(findings) == 0 {
		_, _ = fmt.Fprintln(&report, "secret-scan: PASS")
	} else {
		for _, finding := range findings {
			_, _ = fmt.Fprintf(&report, "%s source=%s kind=%s offset=%d\n", finding.Severity, strconv.Quote(finding.Path), finding.Kind, finding.Offset)
			high = high || finding.Severity == "HIGH"
		}
	}
	if report.overflow {
		fmt.Fprintln(stderr, "secret-scan: cannot report result")
		return 2
	}
	if _, err := io.Copy(stdout, bytes.NewReader(report.Bytes())); err != nil {
		fmt.Fprintln(stderr, "secret-scan: cannot report result")
		return 2
	}
	if high {
		return 3
	}
	return 0
}
