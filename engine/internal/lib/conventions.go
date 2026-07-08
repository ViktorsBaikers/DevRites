package lib

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/conventions_ledger_header.md templates/conventions_orient_header.txt templates/conventions_drift.tmpl
var conventionsTemplateFS embed.FS

var (
	conventionsHeader       = mustReadConventionsTemplate("templates/conventions_ledger_header.md")
	conventionsOrientHeader = mustReadConventionsTemplate("templates/conventions_orient_header.txt")
	conventionsDriftTmpl    = template.Must(template.ParseFS(conventionsTemplateFS, "templates/conventions_drift.tmpl"))
)

type conventionEntry struct {
	Statement      string
	Kind           string
	Status         string
	Proofs         []string
	Corroborations int
	Contradictions int
}

func Conventions(args []string, stdout, stderr io.Writer) int {
	cmd := argAt(args, 0)
	switch cmd {
	case "band":
		c, ok := intFlag(args[1:], "--corroborations", true, stderr)
		if !ok {
			return 2
		}
		k, ok := intFlag(args[1:], "--contradictions", false, stderr)
		if !ok {
			return 2
		}
		if b, active := conventionBand(c, k); active {
			fmt.Fprintf(stdout, "%.2f\n", b)
		} else {
			fmt.Fprintln(stdout, "retired")
		}
		return 0
	case "read":
		root := stringFlag(args[1:], "--root", ".")
		data, err := os.ReadFile(conventionsPath(root))
		if err != nil {
			return 0
		}
		_, _ = stdout.Write(data)
		return 0
	case "orient":
		return conventionsOrient(args[1:], stdout, stderr)
	case "promote":
		return conventionsPromote(args[1:], stdout, stderr)
	case "contradict":
		return conventionsContradict(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "usage: devrites-engine conventions <band|read|orient|promote|contradict> [flags]")
		return 2
	}
}

func conventionBand(corroborations, contradictions int) (float64, bool) {
	if corroborations-contradictions <= 0 {
		return 0, false
	}
	raw := 0.4 + 0.1*float64(corroborations) - 0.2*float64(contradictions)
	return math.Max(0.3, math.Min(0.9, math.Round(raw*10)/10)), true
}

func conventionsOrient(args []string, stdout, stderr io.Writer) int {
	root := stringFlag(args, "--root", ".")
	minBand := floatFlag(args, "--min-band", 0)
	entries := loadConventions(root)
	type row struct {
		key string
		b   float64
		e   conventionEntry
	}
	var rows []row
	for key, e := range entries {
		b, active := conventionBand(e.Corroborations, e.Contradictions)
		if !active || b < minBand {
			continue
		}
		rows = append(rows, row{key: key, b: b, e: e})
	}
	if len(rows) == 0 {
		return 0
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].b != rows[j].b {
			return rows[i].b > rows[j].b
		}
		return rows[i].key < rows[j].key
	})
	fmt.Fprintln(stdout, conventionsOrientHeader)
	for _, r := range rows {
		kind := ""
		if r.e.Kind != "" {
			kind = " (" + r.e.Kind + ")"
		}
		fmt.Fprintf(stdout, "- [%.2f] %s%s: %s\n", r.b, r.key, kind, r.e.Statement)
	}
	return 0
}

func conventionsPromote(args []string, stdout, stderr io.Writer) int {
	root := stringFlag(args, "--root", ".")
	key, ok := requiredStringFlag(args, "--key", stderr)
	if !ok {
		return 2
	}
	statement, ok := requiredStringFlag(args, "--statement", stderr)
	if !ok {
		return 2
	}
	kind, ok := requiredStringFlag(args, "--kind", stderr)
	if !ok {
		return 2
	}
	slug, ok := requiredStringFlag(args, "--slug", stderr)
	if !ok {
		return 2
	}
	evidence, ok := requiredStringFlag(args, "--evidence", stderr)
	if !ok {
		return 2
	}
	date := stringFlag(args, "--date", "")
	entries := loadConventions(root)
	e, exists := entries[key]
	if !exists {
		e = conventionEntry{Statement: statement, Kind: kind, Status: "active"}
	} else {
		if statement != "" {
			e.Statement = statement
		}
		if kind != "" {
			e.Kind = kind
		}
	}
	already := false
	for _, proof := range e.Proofs {
		if proofSlug(proof) == slug {
			already = true
			break
		}
	}
	if !already {
		e.Corroborations++
		e.Proofs = append(e.Proofs, fmt.Sprintf("%s %s — %s", slug, conventionDate(date), evidence))
	}
	entries[key] = e
	if err := saveConventions(root, entries); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	b, active := conventionBand(e.Corroborations, e.Contradictions)
	band := "retired"
	if active {
		band = fmt.Sprintf("%.2f", b)
	}
	fmt.Fprintf(stdout, "promoted '%s' → band %s (%d corroboration(s))\n", key, band, e.Corroborations)
	return 0
}

func conventionsContradict(args []string, stdout, stderr io.Writer) int {
	root := stringFlag(args, "--root", ".")
	key, ok := requiredStringFlag(args, "--key", stderr)
	if !ok {
		return 2
	}
	slug, ok := requiredStringFlag(args, "--slug", stderr)
	if !ok {
		return 2
	}
	evidence, ok := requiredStringFlag(args, "--evidence", stderr)
	if !ok {
		return 2
	}
	date := stringFlag(args, "--date", "")
	driftFile := stringFlag(args, "--drift-file", "")
	entries := loadConventions(root)
	e, exists := entries[key]
	if !exists {
		fmt.Fprintf(stderr, "no such convention: '%s'\n", key)
		return 3
	}
	when := conventionDate(date)
	e.Contradictions++
	e.Proofs = append(e.Proofs, fmt.Sprintf("%s %s — CONTRADICTED: %s", slug, when, evidence))
	entries[key] = e
	if err := saveConventions(root, entries); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	b, active := conventionBand(e.Corroborations, e.Contradictions)
	state := "retired"
	if active {
		state = fmt.Sprintf("band %.2f", b)
	}
	if driftFile != "" {
		if err := os.MkdirAll(filepath.Dir(absPath(driftFile)), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		f, err := os.OpenFile(driftFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		entry, renderErr := renderConventionDrift(key, slug, evidence, state, when)
		if renderErr != nil {
			_ = f.Close()
			fmt.Fprintln(stderr, renderErr)
			return 1
		}
		_, writeErr := f.WriteString(entry)
		closeErr := f.Close()
		if writeErr != nil {
			fmt.Fprintln(stderr, writeErr)
			return 1
		}
		if closeErr != nil {
			fmt.Fprintln(stderr, closeErr)
			return 1
		}
	}
	fmt.Fprintf(stdout, "DRIFT: convention '%s' contradicted by %s — now %s\n", key, slug, state)
	return 0
}

func conventionsPath(root string) string {
	return filepath.Join(root, ".devrites", "conventions.md")
}

func loadConventions(root string) map[string]conventionEntry {
	data, err := os.ReadFile(conventionsPath(root))
	if err != nil {
		return map[string]conventionEntry{}
	}
	return parseConventions(string(data))
}

func parseConventions(text string) map[string]conventionEntry {
	entries := map[string]conventionEntry{}
	key := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "## ") {
			key = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			entries[key] = conventionEntry{Status: "active"}
			continue
		}
		if key == "" {
			continue
		}
		e := entries[key]
		s := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(s, "- proof:"):
			e.Proofs = append(e.Proofs, strings.TrimSpace(strings.TrimPrefix(s, "- proof:")))
		case strings.HasPrefix(s, "- ") && strings.Contains(s, ":"):
			field, value, _ := strings.Cut(strings.TrimPrefix(s, "- "), ":")
			field = strings.TrimSpace(field)
			value = strings.TrimSpace(value)
			switch field {
			case "corroborations":
				if n, err := strconv.Atoi(value); err == nil {
					e.Corroborations = n
				}
			case "contradictions":
				if n, err := strconv.Atoi(value); err == nil {
					e.Contradictions = n
				}
			case "statement":
				e.Statement = value
			case "kind":
				e.Kind = value
			case "status":
				e.Status = value
			}
		}
		entries[key] = e
	}
	return entries
}

func saveConventions(root string, entries map[string]conventionEntry) error {
	path := conventionsPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeFileAtomic(path, []byte(serializeConventions(entries)), 0o644)
}

func serializeConventions(entries map[string]conventionEntry) string {
	var keys []string
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var out []string
	out = append(out, conventionsHeader)
	for _, key := range keys {
		e := entries[key]
		b, active := conventionBand(e.Corroborations, e.Contradictions)
		status := "retired"
		band := "retired"
		if active {
			status = "active"
			band = fmt.Sprintf("%.2f", b)
		}
		out = append(out,
			"## "+key,
			"- statement: "+e.Statement,
			"- kind: "+e.Kind,
			"- band: "+band,
			fmt.Sprintf("- corroborations: %d", e.Corroborations),
			fmt.Sprintf("- contradictions: %d", e.Contradictions),
			"- status: "+status,
		)
		for _, proof := range e.Proofs {
			out = append(out, "- proof: "+proof)
		}
		out = append(out, "")
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

func mustReadConventionsTemplate(path string) string {
	data, err := conventionsTemplateFS.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return strings.TrimRight(string(data), "\n")
}

func renderConventionDrift(key, slug, evidence, state, when string) (string, error) {
	var buf bytes.Buffer
	err := conventionsDriftTmpl.Execute(&buf, struct {
		Key      string
		Slug     string
		Evidence string
		State    string
		When     string
	}{
		Key:      key,
		Slug:     slug,
		Evidence: evidence,
		State:    state,
		When:     when,
	})
	return buf.String(), err
}

func proofSlug(proof string) string {
	slug, _, _ := strings.Cut(proof, " ")
	return slug
}

func conventionDate(date string) string {
	if date != "" {
		return date
	}
	return time.Now().Format("2006-01-02")
}

func stringFlag(args []string, name, fallback string) string {
	value, ok := flagValue(args, name)
	if !ok {
		return fallback
	}
	return value
}

func flagValue(args []string, name string) (string, bool) {
	for i := 0; i < len(args); i++ {
		if args[i] == name && i+1 < len(args) {
			return args[i+1], true
		}
		if strings.HasPrefix(args[i], name+"=") {
			return strings.TrimPrefix(args[i], name+"="), true
		}
	}
	return "", false
}

func requiredStringFlag(args []string, name string, stderr io.Writer) (string, bool) {
	value, ok := flagValue(args, name)
	if !ok {
		fmt.Fprintf(stderr, "missing required flag: %s\n", name)
		return "", false
	}
	return value, true
}

func intFlag(args []string, name string, required bool, stderr io.Writer) (int, bool) {
	value, present := flagValue(args, name)
	if !present {
		if required {
			fmt.Fprintf(stderr, "missing required flag: %s\n", name)
			return 0, false
		}
		return 0, true
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintf(stderr, "invalid integer for %s: %s\n", name, value)
		return 0, false
	}
	return n, true
}

func floatFlag(args []string, name string, fallback float64) float64 {
	value := stringFlag(args, name, "")
	if value == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return f
}

func absPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
