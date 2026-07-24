package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/state"
	"github.com/devrites/devrites/internal/toolpolicy"
)

const (
	GitAuthoritySchemaV1            = "devrites-git-authority/v1"
	GitAuthorityConsumptionSchemaV1 = "devrites-git-authority-consumption/v1"
	GitAuthorityKind                = "destructive-git-once"
	GitAuthorityAnswer              = "Authorize once"
	GitAuthorityLedgerFile          = ".git-authority-consumption.jsonl"
	GitAuthorityTTL                 = 15 * time.Minute

	maxGitAuthorityFileBytes   = 1 << 20
	maxGitAuthorityLedgerBytes = 256 << 10
	maxGitAuthorityReasons     = 32
)

var (
	gitAuthorityDigestRE = regexp.MustCompile(`^drv-git-op-v1:sha256:[0-9a-f]{64}$`)
	gitAuthorityQIDRE    = regexp.MustCompile(`^q-[0-9]{4}-[0-9]{2}-[0-9]{2}-[0-9]{3}$`)
)

// GitAuthorityDecision is deliberately metadata-only. It never carries the
// command, normalized tokens, paths, refs, or a filesystem error.
type GitAuthorityDecision struct {
	Allowed    bool
	Opened     bool
	QuestionID string
	ReasonID   reason.ID
}

type gitAuthorityQuestion struct {
	QID                 string
	Status              string
	Digest              string
	ClassifierReasonIDs []string
	RequestedAt         time.Time
	ExpiresAt           time.Time
	AnsweredAt          time.Time
	Answer              string
}

type gitAuthorityConsumption struct {
	Schema              string   `json:"schema"`
	Kind                string   `json:"kind"`
	QuestionID          string   `json:"question_id"`
	OperationDigest     string   `json:"operation_digest"`
	ClassifierReasonIDs []string `json:"classifier_reason_ids"`
	ConsumedAt          string   `json:"consumed_at"`
}

// AuthorizeGitOperation consumes an exact answered grant or opens one
// idempotent escalating question. Callers must invoke it only for an
// unambiguous destructive classifier result.
func AuthorizeGitOperation(root, slug, digest string, classifierReasonIDs []toolpolicy.ReasonID) GitAuthorityDecision {
	reasons, ok := canonicalGitAuthorityReasons(classifierReasonIDs)
	if !ok || !gitAuthorityDigestRE.MatchString(digest) || strings.TrimSpace(slug) == "" {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}
	}

	work := featureDir(root, slug)
	if !regularDirectory(work) {
		return GitAuthorityDecision{ReasonID: reason.GitWorkspaceUnavailable}
	}
	qfile := filepath.Join(work, "questions.md")
	sfile := filepath.Join(work, "state.md")
	if !regularFileNoSymlink(qfile) || !regularFileNoSymlink(sfile) {
		return GitAuthorityDecision{ReasonID: reason.GitWorkspaceUnavailable}
	}

	decision := GitAuthorityDecision{ReasonID: reason.GitAuthorityUnavailable}
	if err := state.WithFeatureLock(root, slug, func() error {
		var err error
		decision, err = authorizeGitOperationLocked(root, slug, qfile, sfile, digest, reasons)
		return err
	}); err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityUnavailable}
	}
	return decision
}

func authorizeGitOperationLocked(root, slug, qfile, sfile, digest string, reasons []string) (GitAuthorityDecision, error) {
	now := clockNow().UTC().Truncate(time.Second)
	qdata, err := readBoundedRegularFile(qfile, maxGitAuthorityFileBytes)
	if err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}, nil
	}
	questions, err := parseGitAuthorityQuestions(qdata)
	if err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}, nil
	}

	ledgerPath := filepath.Join(featureDir(root, slug), GitAuthorityLedgerFile)
	ledger, ledgerData, err := readGitAuthorityLedger(ledgerPath)
	if err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}, nil
	}
	consumed := make(map[string]bool, len(ledger))
	for _, entry := range ledger {
		consumed[entry.QuestionID+"\x00"+entry.OperationDigest] = true
	}

	reopenReason := reason.ID("")
	for i := len(questions) - 1; i >= 0; i-- {
		q := questions[i]
		if q.Digest != digest {
			continue
		}
		if !equalStrings(q.ClassifierReasonIDs, reasons) {
			return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}, nil
		}
		if q.RequestedAt.After(now) {
			return GitAuthorityDecision{ReasonID: reason.GitAuthorityCorrupt}, nil
		}
		fresh := now.Before(q.ExpiresAt)
		switch q.Status {
		case "dropped":
			return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityRefused}, nil
		case "answered":
			if strings.TrimSpace(q.Answer) != GitAuthorityAnswer {
				return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityRefused}, nil
			}
			if q.AnsweredAt.Before(q.RequestedAt) || q.AnsweredAt.After(now) {
				return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityCorrupt}, nil
			}
			if !fresh || q.AnsweredAt.After(q.ExpiresAt) {
				reopenReason = reason.GitAuthorityExpired
				break
			}
			if consumed[q.QID+"\x00"+digest] {
				reopenReason = reason.GitAuthorityReplayed
				break
			}
			entry := gitAuthorityConsumption{
				Schema:              GitAuthorityConsumptionSchemaV1,
				Kind:                GitAuthorityKind,
				QuestionID:          q.QID,
				OperationDigest:     digest,
				ClassifierReasonIDs: append([]string(nil), reasons...),
				ConsumedAt:          formatGitAuthorityTime(now),
			}
			if err := appendGitAuthorityConsumption(ledgerPath, ledgerData, entry); err != nil {
				return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityUnavailable}, nil
			}
			return GitAuthorityDecision{Allowed: true, QuestionID: q.QID, ReasonID: reason.GitAuthorityGranted}, nil
		case "open":
			if fresh {
				if err := ensureGitAuthorityAwaiting(sfile, q.QID); err != nil {
					return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityUnavailable}, nil
				}
				return GitAuthorityDecision{QuestionID: q.QID, ReasonID: reason.GitAuthorityPending}, nil
			}
			reopenReason = reason.GitAuthorityExpired
		}
		break
	}

	qid, err := nextQuestionID(qdata, now)
	if err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityUnavailable}, nil
	}
	request := renderGitAuthorityQuestion(qdata, qid, digest, reasons, now)
	if err := state.AtomicWrite(qfile, request, 0o644); err != nil {
		return GitAuthorityDecision{ReasonID: reason.GitAuthorityUnavailable}, nil
	}
	if err := ensureGitAuthorityAwaiting(sfile, qid); err != nil {
		return GitAuthorityDecision{QuestionID: qid, ReasonID: reason.GitAuthorityUnavailable}, nil
	}
	if reopenReason == "" {
		reopenReason = reason.GitAuthorityPending
	}
	return GitAuthorityDecision{
		Opened:     true,
		QuestionID: qid,
		ReasonID:   reopenReason,
	}, nil
}

func canonicalGitAuthorityReasons(ids []toolpolicy.ReasonID) ([]string, bool) {
	if len(ids) == 0 || len(ids) > maxGitAuthorityReasons {
		return nil, false
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		value := string(raw)
		parsed, err := reason.Parse(value)
		if err != nil || !strings.HasPrefix(string(parsed), "DRV-GIT-DESTRUCTIVE-") {
			return nil, false
		}
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out, len(out) > 0
}

func parseGitAuthorityQuestions(data []byte) ([]gitAuthorityQuestion, error) {
	lines := splitLinesNoTrailing(data)
	var out []gitAuthorityQuestion
	for i := 0; i < len(lines); {
		if !qHeaderRe.MatchString(lines[i]) {
			i++
			continue
		}
		header := strings.TrimSpace(strings.TrimPrefix(lines[i], "## "))
		start := i
		i++
		for i < len(lines) && !qHeaderRe.MatchString(lines[i]) {
			i++
		}
		fields, duplicate, unknown := gitAuthorityQuestionFields(lines[start+1 : i])
		authority := fields["schema"] == GitAuthoritySchemaV1 || fields["kind"] == GitAuthorityKind
		if !authority {
			continue
		}
		if duplicate != "" || unknown != "" || fields["schema"] != GitAuthoritySchemaV1 ||
			fields["kind"] != GitAuthorityKind || !gitAuthorityQIDRE.MatchString(header) {
			return nil, fmt.Errorf("invalid Git authority question")
		}
		q, err := validateGitAuthorityQuestion(header, fields)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, nil
}

func normalizeGitAuthorityResolution(data []byte, qid, status, answer string) (string, error) {
	lines := splitLinesNoTrailing(data)
	target := regexp.MustCompile(`^## ` + regexp.QuoteMeta(qid) + `([[:space:]]|$)`)
	for i := 0; i < len(lines); i++ {
		if !target.MatchString(lines[i]) {
			continue
		}
		start := i + 1
		i++
		for i < len(lines) && !qHeaderRe.MatchString(lines[i]) {
			i++
		}
		fields, duplicate, unknown := gitAuthorityQuestionFields(lines[start:i])
		authority := fields["schema"] == GitAuthoritySchemaV1 || fields["kind"] == GitAuthorityKind
		if !authority {
			return answer, nil
		}
		if duplicate != "" || unknown != "" || fields["schema"] != GitAuthoritySchemaV1 ||
			fields["kind"] != GitAuthorityKind {
			return "", fmt.Errorf("git authority question is invalid")
		}
		q, err := validateGitAuthorityQuestion(qid, fields)
		if err != nil {
			return "", fmt.Errorf("git authority question is invalid")
		}
		if q.Status != "open" {
			return answer, nil
		}
		switch status {
		case "answered":
			if strings.TrimSpace(answer) != GitAuthorityAnswer {
				return "", fmt.Errorf(`git authority answer must exactly equal %q`, GitAuthorityAnswer)
			}
			return GitAuthorityAnswer, nil
		case "dropped":
			return "refused", nil
		}
		return "", fmt.Errorf("unsupported Git authority resolution")
	}
	return answer, nil
}

func gitAuthorityQuestionFields(lines []string) (map[string]string, string, string) {
	allowed := map[string]bool{
		"status": true, "gate": true, "schema": true, "kind": true,
		"operation_digest": true, "classifier_reason_ids": true,
		"requested_at": true, "expires_at": true, "answer_contract": true,
		"answered_at": true, "answer": true,
	}
	fields := make(map[string]string)
	var duplicate, unknown string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		key = strings.TrimSpace(key)
		if !ok || !allowed[key] {
			if unknown == "" {
				unknown = line
			}
			continue
		}
		if _, exists := fields[key]; exists {
			if duplicate == "" {
				duplicate = key
			}
			continue
		}
		fields[key] = strings.TrimSpace(value)
	}
	return fields, duplicate, unknown
}

func validateGitAuthorityQuestion(qid string, fields map[string]string) (gitAuthorityQuestion, error) {
	q := gitAuthorityQuestion{
		QID:    qid,
		Status: fields["status"],
		Digest: fields["operation_digest"],
		Answer: fields["answer"],
	}
	if fields["gate"] != "escalating" || fields["answer_contract"] != GitAuthorityAnswer ||
		!gitAuthorityDigestRE.MatchString(q.Digest) {
		return q, fmt.Errorf("invalid Git authority contract")
	}
	reasons, err := parseGitAuthorityReasonList(fields["classifier_reason_ids"])
	if err != nil {
		return q, err
	}
	q.ClassifierReasonIDs = reasons
	q.RequestedAt, err = time.Parse(time.RFC3339, fields["requested_at"])
	if err != nil {
		return q, fmt.Errorf("invalid Git authority request time")
	}
	q.ExpiresAt, err = time.Parse(time.RFC3339, fields["expires_at"])
	if err != nil || !q.ExpiresAt.Equal(q.RequestedAt.Add(GitAuthorityTTL)) {
		return q, fmt.Errorf("invalid Git authority expiry contract")
	}
	switch q.Status {
	case "open":
		if fields["answered_at"] != "" || fields["answer"] != "" {
			return q, fmt.Errorf("open Git authority contains an answer")
		}
	case "answered", "dropped":
		q.AnsweredAt, err = time.Parse(time.RFC3339, fields["answered_at"])
		if err != nil || fields["answer"] == "" {
			return q, fmt.Errorf("terminal Git authority is incomplete")
		}
	default:
		return q, fmt.Errorf("invalid Git authority status")
	}
	return q, nil
}

func parseGitAuthorityReasonList(value string) ([]string, error) {
	if value == "" {
		return nil, fmt.Errorf("missing classifier reasons")
	}
	parts := strings.Split(value, ",")
	if len(parts) > maxGitAuthorityReasons {
		return nil, fmt.Errorf("too many classifier reasons")
	}
	for i, part := range parts {
		if strings.TrimSpace(part) != part || part == "" {
			return nil, fmt.Errorf("non-canonical classifier reasons")
		}
		parsed, err := reason.Parse(part)
		if err != nil || !strings.HasPrefix(string(parsed), "DRV-GIT-DESTRUCTIVE-") {
			return nil, fmt.Errorf("unknown classifier reason")
		}
		if i > 0 && parts[i-1] >= part {
			return nil, fmt.Errorf("classifier reasons must be unique and sorted")
		}
	}
	return parts, nil
}

func renderGitAuthorityQuestion(existing []byte, qid, digest string, reasons []string, now time.Time) []byte {
	record := []string{
		"## " + qid,
		"status: open",
		"gate: escalating",
		"schema: " + GitAuthoritySchemaV1,
		"kind: " + GitAuthorityKind,
		"operation_digest: " + digest,
		"classifier_reason_ids: " + strings.Join(reasons, ","),
		"requested_at: " + formatGitAuthorityTime(now),
		"expires_at: " + formatGitAuthorityTime(now.Add(GitAuthorityTTL)),
		"answer_contract: " + GitAuthorityAnswer,
	}
	lines := splitLinesNoTrailing(existing)
	hasQuestions := false
	for _, line := range lines {
		if qHeaderRe.MatchString(line) {
			hasQuestions = true
			break
		}
	}
	if !hasQuestions {
		filtered := lines[:0]
		for _, line := range lines {
			if strings.EqualFold(strings.TrimSpace(line), "None.") {
				continue
			}
			filtered = append(filtered, line)
		}
		lines = filtered
	}
	lines = trimTrailingBlankLines(lines)
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	lines = append(lines, record...)
	return joinRecords(lines)
}

func nextQuestionID(data []byte, now time.Time) (string, error) {
	today := now.Format("2006-01-02")
	countRE := regexp.MustCompile(`(?m)^## q-` + regexp.QuoteMeta(today) + `-`)
	used := len(countRE.FindAllIndex(data, -1))
	qid := fmt.Sprintf("q-%s-%03d", today, used+1)
	collisionRE := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(qid) + `([[:space:]]|$)`)
	if collisionRE.Match(data) {
		return qid, fmt.Errorf("question id collision")
	}
	return qid, nil
}

func ensureGitAuthorityAwaiting(sfile, qid string) error {
	data, err := readBoundedRegularFile(sfile, maxGitAuthorityFileBytes)
	if err != nil {
		return err
	}
	lines := splitLinesNoTrailing(data)
	if current := awaitingQuestionID(lines); current != "" && current != qid {
		return fmt.Errorf("another question is already awaiting")
	}
	if awaitingQuestionID(lines) == "" {
		block := []string{"## Awaiting human", "- qid: " + qid, ""}
		insert := len(lines)
		for i, line := range lines {
			if strings.EqualFold(strings.TrimSpace(line), "## Log") {
				insert = i
				break
			}
		}
		prefix := append([]string(nil), lines[:insert]...)
		if len(prefix) > 0 && strings.TrimSpace(prefix[len(prefix)-1]) != "" {
			prefix = append(prefix, "")
		}
		lines = append(prefix, append(block, lines[insert:]...)...)
	}
	lines = state.UpsertCursorField(lines, state.CursorStatus, "awaiting_human")
	lines = state.UpsertCursorField(lines, state.CursorNextAction, `/rite-resolve `+qid+` "`+GitAuthorityAnswer+`"`)
	return state.AtomicWrite(sfile, joinRecords(trimTrailingBlankLines(lines)), 0o644)
}

func awaitingQuestionID(lines []string) string {
	inAwaiting := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inAwaiting {
				break
			}
			inAwaiting = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "Awaiting human")
			continue
		}
		if inAwaiting {
			if qid, ok := state.CursorField([]string{line}, state.CursorQuestionID); ok {
				return qid
			}
		}
	}
	return ""
}

func readGitAuthorityLedger(path string) ([]gitAuthorityConsumption, []byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil || !info.Mode().IsRegular() || !gitAuthorityLedgerModeOK(info.Mode()) {
		return nil, nil, fmt.Errorf("invalid Git authority ledger")
	}
	if info.Size() > maxGitAuthorityLedgerBytes {
		return nil, nil, fmt.Errorf("git authority ledger exceeds bound")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var out []gitAuthorityConsumption
	for _, line := range bytes.Split(data, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry gitAuthorityConsumption
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&entry); err != nil {
			return nil, nil, fmt.Errorf("invalid Git authority ledger entry")
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			return nil, nil, fmt.Errorf("invalid Git authority ledger trailing data")
		}
		if err := validateGitAuthorityConsumption(entry); err != nil {
			return nil, nil, err
		}
		out = append(out, entry)
	}
	return out, data, nil
}

func gitAuthorityLedgerModeOK(mode os.FileMode) bool {
	// Windows exposes inherited ACLs rather than POSIX group/other mode bits;
	// Chmod(0600) is still requested at every atomic write.
	return runtime.GOOS == "windows" || mode.Perm() == 0o600
}

func validateGitAuthorityConsumption(entry gitAuthorityConsumption) error {
	if entry.Schema != GitAuthorityConsumptionSchemaV1 || entry.Kind != GitAuthorityKind ||
		!gitAuthorityQIDRE.MatchString(entry.QuestionID) ||
		!gitAuthorityDigestRE.MatchString(entry.OperationDigest) {
		return fmt.Errorf("invalid Git authority ledger contract")
	}
	if _, err := parseGitAuthorityReasonList(strings.Join(entry.ClassifierReasonIDs, ",")); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, entry.ConsumedAt); err != nil {
		return fmt.Errorf("invalid Git authority consumption time")
	}
	return nil
}

func appendGitAuthorityConsumption(path string, current []byte, entry gitAuthorityConsumption) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	out := append([]byte(nil), bytes.TrimRight(current, "\n")...)
	if len(out) > 0 {
		out = append(out, '\n')
	}
	out = append(out, line...)
	out = append(out, '\n')
	if len(out) > maxGitAuthorityLedgerBytes {
		return fmt.Errorf("git authority ledger exceeds bound")
	}
	return state.AtomicWrite(path, out, 0o600)
}

func readBoundedRegularFile(path string, limit int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("invalid bounded regular file")
	}
	return os.ReadFile(path)
}

func regularFileNoSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

func regularDirectory(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

func formatGitAuthorityTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func trimTrailingBlankLines(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
