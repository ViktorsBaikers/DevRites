package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/fsutil"
)

type timelineEntry struct {
	TS       string `json:"ts"`
	Event    string `json:"event"`
	Skill    string `json:"skill,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Outcome  string `json:"outcome,omitempty"`
	Decision string `json:"decision,omitempty"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Note     string `json:"note,omitempty"`
}

type healthEntry struct {
	TS    string  `json:"ts"`
	Score float64 `json:"score"`
	Label string  `json:"label"`
	Note  string  `json:"note,omitempty"`
}

type reviewFingerprint struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Text     string `json:"text"`
}

// Timeline is the append-only session trace: what happened, which gate/skill did
// it, and which durable decision or state transition resulted.
func Timeline(root string, args []string, stdout, stderr io.Writer) int {
	switch argAt(args, 0) {
	case "log":
		entry := timelineEntry{TS: nowUTC()}
		rest := args[1:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
			entry.Event = rest[0]
			rest = rest[1:]
		}
		for i := 0; i < len(rest); i++ {
			next := func(name string) (string, bool) {
				i++
				if i >= len(rest) {
					fmt.Fprintf(stderr, "timeline: %s needs a value\n", name)
					return "", false
				}
				return rest[i], true
			}
			switch rest[i] {
			case "--event":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Event = v
			case "--skill":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Skill = v
			case "--slug":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Slug = v
			case "--outcome":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Outcome = v
			case "--decision":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Decision = v
			case "--from":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.From = v
			case "--to":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.To = v
			case "--note":
				v, ok := next(rest[i])
				if !ok {
					return 2
				}
				entry.Note = v
			default:
				fmt.Fprintf(stderr, "timeline: unknown option %s\n", rest[i])
				return 2
			}
		}
		if entry.Event == "" {
			fmt.Fprintln(stderr, "usage: devrites-engine timeline log <event> [--skill S] [--slug S] [--outcome O] [--decision D] [--from A --to B] [--note N]")
			return 2
		}
		if err := appendJSONLine(filepath.Join(root, "timeline.jsonl"), entry); err != nil {
			fmt.Fprintf(stderr, "timeline: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "timeline: recorded.")
		return 0
	case "list":
		limit := 20
		if argAt(args, 1) == "--limit" {
			n, err := strconv.Atoi(argAt(args, 2))
			if err != nil || n < 0 {
				fmt.Fprintln(stderr, "timeline: --limit must be a non-negative integer")
				return 2
			}
			limit = n
		}
		printTail(filepath.Join(root, "timeline.jsonl"), limit, "timeline", stdout)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine timeline log|list [...]")
		return 2
	}
}

// Health records one compact score (legacy record/list) or runs the project code-health dashboard.
func Health(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || argAt(args, 0) == "run" || argAt(args, 0) == "check" {
		return CodeHealth(root, args, stdout, stderr)
	}
	switch argAt(args, 0) {
	case "record":
		score, ok := parseHealthScore(argAt(args, 1))
		if !ok {
			fmt.Fprintln(stderr, "usage: devrites-engine health record <score 0..10> <label> [--note N]")
			return 2
		}
		labelParts := []string{}
		note := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--note" {
				i++
				if i >= len(args) {
					fmt.Fprintln(stderr, "health: --note needs a value")
					return 2
				}
				note = args[i]
				continue
			}
			labelParts = append(labelParts, args[i])
		}
		label := strings.TrimSpace(strings.Join(labelParts, " "))
		if label == "" {
			fmt.Fprintln(stderr, "usage: devrites-engine health record <score 0..10> <label> [--note N]")
			return 2
		}
		if err := appendJSONLine(filepath.Join(root, "health-history.jsonl"), healthEntry{
			TS:    nowUTC(),
			Score: score,
			Label: label,
			Note:  note,
		}); err != nil {
			fmt.Fprintf(stderr, "health: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "health: recorded.")
		return 0
	case "list":
		limit := 20
		if argAt(args, 1) == "--limit" {
			n, err := strconv.Atoi(argAt(args, 2))
			if err != nil || n < 0 {
				fmt.Fprintln(stderr, "health: --limit must be a non-negative integer")
				return 2
			}
			limit = n
		}
		path := filepath.Join(root, "health.jsonl")
		if !isFile(path) {
			path = filepath.Join(root, "health-history.jsonl")
		}
		printTail(path, limit, "health", stdout)
		return 0
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine health [run|check] | record|list [...]")
		return 2
	}
}

func ReviewFingerprints(root string, args []string, stdout, stderr io.Writer) int {
	write := false
	slug := ""
	for _, a := range args {
		if a == "--write" {
			write = true
			continue
		}
		if slug == "" {
			slug = a
		}
	}
	if slug == "" {
		slug = activeSlug(root)
	}
	if slug == "" {
		fmt.Fprintln(stderr, "usage: devrites-engine review-fingerprints [--write] <slug>")
		return 2
	}
	dir := featureDir(root, slug)
	review, ok := readFileOK(filepath.Join(dir, "review.md"))
	if !ok {
		fmt.Fprintln(stdout, "review-fingerprints: no review.md — nothing to fingerprint.")
		return 0
	}
	records := reviewFindingFingerprints(review)
	if len(records) == 0 {
		fmt.Fprintln(stdout, "review-fingerprints: no findings.")
		return 0
	}
	var jsonl strings.Builder
	for _, r := range records {
		fmt.Fprintf(stdout, "%s %s %s\n", r.ID, r.Severity, r.Text)
		if b, err := json.Marshal(r); err == nil {
			jsonl.Write(b)
			jsonl.WriteByte('\n')
		}
	}
	if write {
		if err := fsutil.WriteFileAtomic(filepath.Join(dir, "review-fingerprints.jsonl"), []byte(jsonl.String()), 0o644); err != nil {
			fmt.Fprintf(stderr, "review-fingerprints: %v\n", err)
			return 1
		}
	}
	return 0
}

func reviewFindingFingerprints(md string) []reviewFingerprint {
	var out []reviewFingerprint
	for _, line := range markdownLines(md) {
		m := findingLabel.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := normalizeFindingLine(line)
		sum := sha256.Sum256([]byte(strings.ToLower(m[1]) + "\n" + text))
		out = append(out, reviewFingerprint{
			ID:       hex.EncodeToString(sum[:])[:12],
			Severity: m[1],
			Text:     text,
		})
	}
	return out
}

func normalizeFindingLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimSpace(line)
	return strings.Join(strings.Fields(line), " ")
}

func markdownLines(md string) []string {
	md = strings.TrimRight(md, "\n")
	if md == "" {
		return nil
	}
	return strings.Split(md, "\n")
}

func parseHealthScore(s string) (float64, bool) {
	score, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil || score < 0 || score > 10 {
		return 0, false
	}
	return score, true
}

func appendJSONLine(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

func printTail(path string, limit int, label string, stdout io.Writer) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stdout, "%s: no history at %s.\n", label, path)
		return
	}
	lines := splitLinesNoTrailing(data)
	if limit > 0 && len(lines) > limit {
		lines = lines[len(lines)-limit:]
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
}
