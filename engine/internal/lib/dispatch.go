package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/devritespaths"
)

// dispatch.go: launch-wave barrier for host-native agent dispatch. Hosts own
// the actual agent sessions; this command owns the wave state machine so a
// fan-out cannot silently degrade to serial (open -> every role started ->
// seal -> returns -> complete). start/return also auto-record metrics events.
//
//	devrites-engine dispatch <slug> <sub> [--phase p] [--wave w] [--role r] [--handle h] [--reason s]

const dispatchUsage = `usage: devrites-engine dispatch <slug> <open|start|seal|return|status|abandon> [--phase p] [--wave w] [--role r] [--handle h] [--reason s]

Wave barrier for parallel agent dispatch: seal fails until every declared
role has a distinct handle; return fails before seal; status exits nonzero
while a wave is open, sealed, or abandoned.
`

type dispatchFile struct {
	Waves map[string]*dispatchWave `json:"waves"`
}

type dispatchWave struct {
	Phase string                   `json:"phase"`
	State string                   `json:"state"` // open, sealed, abandoned, complete
	Roles map[string]*dispatchRole `json:"roles"`
}

type dispatchRole struct {
	Handle    string `json:"handle,omitempty"`
	StartedAt string `json:"started,omitempty"`
	Returned  bool   `json:"returned,omitempty"`
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
	return os.WriteFile(path, data, 0o600)
}

func nowStamp() string { return time.Now().UTC().Format(time.RFC3339) }

// RunDispatch implements the dispatch wave state machine.
func RunDispatch(root string, args []string, stdout, stderr io.Writer) int {
	var slug, sub, phase, wave, role, handle, reason string
	var roles []string
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
		w := &dispatchWave{Phase: phase, State: "open", Roles: map[string]*dispatchRole{}}
		for _, r := range roles {
			if w.Roles[r] != nil {
				return fail("duplicate role %q in wave", r)
			}
			w.Roles[r] = &dispatchRole{}
		}
		f.Waves[wave] = w
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "dispatch: wave %s open, %d roles declared\n", wave, len(roles))
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
		r.Returned = true
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
			for name, w := range f.Waves {
				fmt.Fprintf(stdout, "dispatch: wave %s state=%s roles=%d\n", name, w.State, len(w.Roles))
			}
			if len(f.Waves) == 0 {
				fmt.Fprintln(stdout, "dispatch: no waves")
			}
			return 0
		}
		w := getWave()
		if w == nil {
			return fail("no wave %q", wave)
		}
		var pending []string
		for name, r := range w.Roles {
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
		if strings.TrimSpace(reason) == "" {
			return fail("--reason is required (non-empty, surfaced in handoff)")
		}
		w.State = "abandoned"
		if err := saveDispatch(path, f); err != nil {
			return fail("%v", err)
		}
		fmt.Fprintf(stdout, "dispatch: wave %s abandoned: %s\n", wave, reason)
		return fail("wave %q is abandoned — HANDOFF REQUIRED: %s", wave, reason)
	}
	fmt.Fprint(stderr, dispatchUsage)
	return 2
}

func waveState(w *dispatchWave) string {
	if w == nil {
		return "absent"
	}
	return w.State
}
