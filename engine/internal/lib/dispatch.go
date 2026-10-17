package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/state"
)

// dispatch.go: launch-wave barrier for host-native agent dispatch. Hosts own
// the actual agent sessions; this command owns the wave state machine so a
// fan-out cannot silently degrade to serial (open -> every role started ->
// seal -> returns -> complete). start/return also auto-record metrics events.
//
//	devrites-engine dispatch <slug> <sub> [--phase p] [--wave w] [--role r] [--handle h] [--reason s] [--slice SLICE-###] [--reslice-review <DRIFT-ID>] [--contract <file>] [--packet <file>]

const dispatchUsage = `usage: devrites-engine dispatch <slug> <open|start|seal|return|status|abandon> [--phase p] [--wave w] [--role r] [--handle h] [--reason s] [--slice SLICE-###] [--reslice-review <DRIFT-ID>] [--contract <file>] [--packet <file>]

Wave barrier for parallel agent dispatch: seal fails until every declared
role has a distinct handle; return fails before seal. status --wave exits
nonzero until that wave is complete; bare status lists every wave and exits
nonzero while any wave is open or sealed (abandoned waves are listed only).
A complete wave cannot be abandoned.

A build wave with a wright role must name its slice(s) with --slice; a vet
wave may (slice-bound recheck). Every 5 rounds of one kind on one slice, the
next open is blocked until --reslice-review names a drift.md
"## <DRIFT-ID>" heading for that slice.
`

// wrightRoundsPerReview is how many build wright rounds (or slice-bound Vet
// rechecks) one slice may take before (and after each) reslice review.
// Recovery itself stays unbounded; the cap only forces the split question
// onto the record.
const wrightRoundsPerReview = 5

type dispatchFile struct {
	Waves map[string]*dispatchWave `json:"waves"`
}

type dispatchWave struct {
	Phase           string                   `json:"phase"`
	State           string                   `json:"state"` // open, sealed, abandoned, complete
	Reason          string                   `json:"reason,omitempty"`
	Roles           map[string]*dispatchRole `json:"roles"`
	Slices          []string                 `json:"slices,omitempty"`
	ResliceReview   string                   `json:"reslice_review,omitempty"`
	ResliceReviewed []string                 `json:"reslice_reviewed,omitempty"`
}

func waveStarted(w *dispatchWave) bool {
	for _, r := range w.Roles {
		if r != nil && r.StartedAt != "" {
			return true
		}
	}
	return false
}

type dispatchRole struct {
	Handle     string `json:"handle,omitempty"`
	StartedAt  string `json:"started,omitempty"`
	Returned   bool   `json:"returned,omitempty"`
	ReturnedAt string `json:"returned_at,omitempty"`
}

func dispatchPath(root, slug string) string {
	return filepath.Join(devritespaths.FeatureDir(root, slug), "dispatch.json")
}

func loadDispatch(root, slug string) (*dispatchFile, string, error) {
	path := dispatchPath(root, slug)
	f := &dispatchFile{Waves: map[string]*dispatchWave{}}
	data, err := os.ReadFile(path) // #nosec G304 -- path is a validated feature workspace artifact
	if os.IsNotExist(err) {
		return f, path, nil
	}
	if err != nil {
		return nil, path, err
	}
	if err := json.Unmarshal(data, f); err != nil {
		return nil, path, fmt.Errorf("dispatch.json is not valid JSON: %w", err)
	}
	if f.Waves == nil {
		f.Waves = map[string]*dispatchWave{}
	}
	// Fail closed on hand-edited impossible state.
	for name, w := range f.Waves {
		if w.Roles == nil || w.State == "" {
			return nil, path, fmt.Errorf("wave %q is malformed (missing state or roles)", name)
		}
	}
	return f, path, nil
}

func saveDispatch(path string, f *dispatchFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return state.AtomicWrite(path, data, 0o600)
}

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }

// RunDispatch implements the dispatch wave state machine.
func RunDispatch(root string, args []string, stdout, stderr io.Writer) int {
	var slug, sub, phase, wave, role, handle, reason, review, contract, packet string
	var roles, sliceIDs []string
	var positional []string
	for i := 0; i < len(args); i++ {
		value := argAt(args, i+1)
		switch args[i] {
		case "--phase":
			phase = value
		case "--wave":
			wave = value
		case "--role":
			roles = append(roles, value)
			role = value
		case "--handle":
			handle = value
		case "--reason":
			reason = value
		case "--slice":
			sliceIDs = append(sliceIDs, value)
		case "--reslice-review":
			review = value
		case "--contract":
			contract = value
		case "--packet":
			packet = value
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(stderr, "dispatch: unknown flag %q\n", args[i])
				return 2
			}
			positional = append(positional, args[i])
			continue
		}
		i++
	}
	if len(positional) != 2 {
		fmt.Fprint(stderr, dispatchUsage)
		return 2
	}
	slug, sub = positional[0], positional[1]
	if _, err := devritespaths.ExistingFeatureDirChecked(root, slug); err != nil {
		fmt.Fprintf(stderr, "dispatch: %v\n", err)
		return 2
	}
	// Serialize load-modify-save so concurrent start/return calls never lose
	// an update and strand the wave short of complete.
	code := 0
	if err := state.WithFeatureLock(root, slug, func() error {
		code = dispatchLocked(root, slug, sub, phase, wave, role, handle, reason, roles, sliceIDs, review, contract, packet, stdout, stderr)
		return nil
	}); err != nil {
		fmt.Fprintf(stderr, "dispatch: %v\n", err)
		return 3
	}
	return code
}

func dispatchLocked(root, slug, sub, phase, wave, role, handle, reason string, roles, sliceIDs []string, review, contract, packet string, stdout, stderr io.Writer) int {
	f, path, err := loadDispatch(root, slug)
	if err != nil {
		fmt.Fprintf(stderr, "dispatch: %v\n", err)
		return 3
	}

	fail := func(format string, a ...any) int {
		fmt.Fprintf(stderr, "dispatch: "+format+"\n", a...)
		return 3
	}
	getWave := func() *dispatchWave {
		w := f.Waves[wave]
		if w == nil {
			return nil
		}
		return w
	}

	switch sub {
	case "open":
		if phase == "" || wave == "" || len(roles) == 0 {
			fmt.Fprint(stderr, dispatchUsage)
			return 2
		}
		if w := getWave(); w != nil && w.State != "abandoned" {
			return fail("wave %q already exists (state %s)", wave, w.State)
		}
		w := &dispatchWave{Phase: phase, State: "open", Roles: map[string]*dispatchRole{}, Slices: sliceIDs}
		for _, r := range roles {
			if w.Roles[r] != nil {
				return fail("duplicate role %q in wave", r)
			}
			w.Roles[r] = &dispatchRole{}
		}
		rounds, code := admitWrightRounds(root, slug, f, w, review, stderr)
		if code != 0 {
			return code
		}
		if isBuildWrightWave(w) {
			if err := lintWrightContract(contract); err != nil {
				return fail("BLOCKED: %v", err)
			}
		}
		for _, r := range roles {
			if isReviewerRole(r) {
				if err := lintReviewerPacket(packet); err != nil {
					return fail("BLOCKED: %v", err)
				}
				break
			}
		}
		f.Waves[wave] = w
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "dispatch: wave %s open, %d roles declared\n", wave, len(roles))
		for _, line := range rounds {
			fmt.Fprintln(stdout, line)
		}
		return 0

	case "start":
		w := getWave()
		if w == nil || w.State != "open" {
			return fail("wave %q is not open (state %s)", wave, waveState(w))
		}
		r := w.Roles[role]
		if r == nil {
			return fail("role %q not declared in wave %q", role, wave)
		}
		if handle == "" {
			return fail("--handle is required (host agent id)")
		}
		for name, other := range w.Roles {
			if name != role && other.Handle == handle {
				return fail("handle %q already used by role %q — distinct handle required", handle, name)
			}
		}
		if r.Handle != "" {
			return fail("role %q already started (handle %q)", role, r.Handle)
		}
		r.Handle = handle
		r.StartedAt = nowStamp()
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		recordMetric(root, slug, w.Phase, "dispatch", role, 0)
		fmt.Fprintf(stdout, "dispatch: %s started (handle %s)\n", role, handle)
		return 0

	case "seal":
		w := getWave()
		if w == nil || w.State != "open" {
			return fail("wave %q is not open (state %s)", wave, waveState(w))
		}
		var missing []string
		for name, r := range w.Roles {
			if r.Handle == "" {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return fail("wave %q cannot seal: no start handle for %s", wave, strings.Join(missing, ","))
		}
		w.State = "sealed"
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "dispatch: wave %s sealed (%d roles)\n", wave, len(w.Roles))
		return 0

	case "return":
		w := getWave()
		if w == nil || w.State != "sealed" {
			return fail("wave %q is not sealed (state %s)", wave, waveState(w))
		}
		r := w.Roles[role]
		if r == nil {
			return fail("role %q not declared in wave %q", role, wave)
		}
		if r.Returned {
			return fail("role %q already returned", role)
		}
		r.Returned, r.ReturnedAt = true, nowStamp()
		all := true
		for _, o := range w.Roles {
			if !o.Returned {
				all = false
			}
		}
		if all {
			w.State = "complete"
		}
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		recordMetric(root, slug, w.Phase, "return", role, 0)
		fmt.Fprintf(stdout, "dispatch: %s returned (wave %s %s)\n", role, wave, w.State)
		return 0

	case "status":
		if wave == "" {
			var incomplete []string
			for _, name := range sortedKeys(f.Waves) {
				w := f.Waves[name]
				fmt.Fprintf(stdout, "dispatch: wave %s state=%s roles=%d\n", name, w.State, len(w.Roles))
				if w.State == "open" || w.State == "sealed" {
					incomplete = append(incomplete, name)
				}
			}
			if len(f.Waves) == 0 {
				fmt.Fprintln(stdout, "dispatch: no waves")
			}
			if len(incomplete) > 0 {
				return fail("waves not complete: %s", strings.Join(incomplete, ","))
			}
			return 0
		}
		w := getWave()
		if w == nil {
			return fail("no wave %q", wave)
		}
		var pending []string
		for _, name := range sortedKeys(w.Roles) {
			r := w.Roles[name]
			mark := "declared"
			if r.Handle != "" {
				mark = "started"
			}
			if r.Returned {
				mark = "returned"
			} else {
				pending = append(pending, name)
			}
			fmt.Fprintf(stdout, "dispatch: %s %s=%s\n", wave, name, mark)
		}
		if w.State != "complete" {
			return fail("wave %q state=%s pending=%s", wave, w.State, strings.Join(pending, ","))
		}
		fmt.Fprintf(stdout, "dispatch: wave %s complete\n", wave)
		return 0

	case "abandon":
		w := getWave()
		if w == nil {
			return fail("no wave %q", wave)
		}
		if w.State == "complete" {
			return fail("wave %q is complete; a complete wave cannot be abandoned", wave)
		}
		if strings.TrimSpace(reason) == "" {
			return fail("--reason is required (non-empty, recorded in dispatch.json)")
		}
		w.State = "abandoned"
		w.Reason = reason
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "dispatch: wave %s abandoned: %s\n", wave, reason)
		return fail("wave %q is abandoned — HANDOFF REQUIRED: %s", wave, reason)
	}
	fmt.Fprint(stderr, dispatchUsage)
	return 2
}

// roundKind names the capped loop a wave belongs to: build wright rounds or
// slice-bound Vet rechecks. Other waves are uncapped.
func roundKind(w *dispatchWave) string {
	switch {
	case isBuildWrightWave(w):
		return "wright"
	case strings.EqualFold(w.Phase, "vet") && len(w.Slices) > 0:
		return "vet"
	}
	return ""
}

func isBuildWrightWave(w *dispatchWave) bool {
	if !strings.EqualFold(w.Phase, "build") {
		return false
	}
	for r := range w.Roles {
		if strings.Contains(r, "wright") {
			return true
		}
	}
	return false
}

// admitWrightRounds enforces the per-slice round cap for a new build wright
// or slice-bound Vet wave and returns the round lines to print. A slice may
// take wrightRoundsPerReview rounds of each kind per reslice review (the first
// window is free); beyond that the open needs --reslice-review naming an
// unused drift.md heading for every slice over the cap.
func admitWrightRounds(root, slug string, f *dispatchFile, w *dispatchWave, review string, stderr io.Writer) ([]string, int) {
	kind := roundKind(w)
	if kind == "" {
		if review != "" || (len(w.Slices) > 0 && !strings.EqualFold(w.Phase, "vet")) {
			fmt.Fprintln(stderr, "dispatch: --slice/--reslice-review apply only to build wright waves and slice-bound vet waves")
			return nil, 2
		}
		return nil, 0
	}
	if len(w.Slices) == 0 {
		fmt.Fprintln(stderr, "dispatch: a build wright wave needs --slice SLICE-### for each slice it builds")
		return nil, 2
	}
	for _, id := range w.Slices {
		if !state.ValidSliceID(id) {
			fmt.Fprintf(stderr, "dispatch: %q is not a SLICE-### id\n", id)
			return nil, 2
		}
	}
	var lines, overCap []string
	for _, id := range w.Slices {
		prior, reviews := 0, 0
		var unreturned []string
		for name, other := range f.Waves {
			// An abandoned wave spends its round once any role started.
			if (other.State == "abandoned" && !waveStarted(other)) || roundKind(other) != kind {
				continue
			}
			if containsSlice(other.Slices, id) {
				prior++
				if other.State == "open" || other.State == "sealed" {
					unreturned = append(unreturned, name)
				}
			}
			if containsSlice(other.ResliceReviewed, id) {
				reviews++
			}
		}
		round := prior + 1
		if round > wrightRoundsPerReview*(1+reviews) {
			overCap = append(overCap, id)
		}
		lines = append(lines, fmt.Sprintf("dispatch: %s %s round %d (reslice review every %d)", id, kind, round, wrightRoundsPerReview))
		if len(unreturned) > 0 {
			sort.Strings(unreturned)
			lines = append(lines, fmt.Sprintf("dispatch: WARNING: %s earlier wave(s) %s never returned; record `dispatch return` or `dispatch abandon --reason`", id, strings.Join(unreturned, ",")))
		}
	}
	if kind == "wright" {
		lines = append(lines, "dispatch: wright contract: name the writer-safe checks the wright runs (syntax/type/lint, unit and self-check suites, a narrow local smoke) and never forbid them; only artifact-producing, consumptive, or full canonical gates stay root-owned")
	}
	if len(overCap) == 0 {
		if review != "" {
			fmt.Fprintf(stderr, "dispatch: --reslice-review given but no slice has reached its %d-round cap\n", wrightRoundsPerReview)
			return nil, 2
		}
		return lines, 0
	}
	if review == "" {
		fmt.Fprintf(stderr, "dispatch: BLOCKED: %s reached %d %s rounds; run a reslice review (acceptance-preserving split or simpler proof seam, or a drift.md entry recording why neither applies) and reopen with --reslice-review <DRIFT-ID>\n",
			strings.Join(overCap, ","), wrightRoundsPerReview, kind)
		return nil, 3
	}
	for _, other := range f.Waves {
		if other.State != "abandoned" && other.ResliceReview == review && roundKind(other) == kind {
			fmt.Fprintf(stderr, "dispatch: BLOCKED: reslice review %s already admitted another round window\n", review)
			return nil, 3
		}
	}
	drift, err := readWorkspaceArtifact(root, slug, "drift.md")
	if err != nil {
		fmt.Fprintf(stderr, "dispatch: BLOCKED: reslice review %s: %v\n", review, err)
		return nil, 3
	}
	heading := driftHeading(string(drift), review)
	for _, id := range overCap {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(id) + `\b`).MatchString(heading) {
			fmt.Fprintf(stderr, "dispatch: BLOCKED: drift.md has no \"## %s\" heading naming %s\n", review, id)
			return nil, 3
		}
	}
	w.ResliceReview, w.ResliceReviewed = review, overCap
	return lines, 0
}

// containsSlice reports whether ids names the same slice as id, ignoring
// zero padding (SLICE-1 == SLICE-001) so a respelling cannot reset the cap.
func containsSlice(ids []string, id string) bool {
	return slices.ContainsFunc(ids, func(other string) bool {
		return sliceNumber(other) == sliceNumber(id)
	})
}

func sliceNumber(id string) string {
	return strings.TrimLeft(strings.TrimPrefix(id, "SLICE-"), "0")
}

// driftHeading returns the drift.md level-2 heading line for id, or "".
func driftHeading(drift, id string) string {
	re := regexp.MustCompile(`(?m)^## ` + regexp.QuoteMeta(id) + `\b.*$`)
	return re.FindString(drift)
}

// InflightWave is a dispatch wave whose agents have not all returned.
type InflightWave struct {
	Wave    string   `json:"wave"`
	Phase   string   `json:"phase"`
	State   string   `json:"state"`
	Pending []string `json:"pending"`
	Slices  []string `json:"slices,omitempty"`
}

// inflightWaves lists open or sealed waves with the roles still awaiting a
// return, sorted by wave name.
func inflightWaves(f *dispatchFile) []InflightWave {
	var out []InflightWave
	for name, w := range f.Waves {
		if w.State != "open" && w.State != "sealed" {
			continue
		}
		var pending []string
		for role, r := range w.Roles {
			if !r.Returned {
				pending = append(pending, role)
			}
		}
		sort.Strings(pending)
		out = append(out, InflightWave{Wave: name, Phase: w.Phase, State: w.State, Pending: pending, Slices: w.Slices})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Wave < out[j].Wave })
	return out
}

func waveState(w *dispatchWave) string {
	if w == nil {
		return "absent"
	}
	return w.State
}
