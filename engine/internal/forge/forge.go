package forge

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type PlanOptions struct {
	SliceID        string
	Slug           string
	Strategies     map[CandidateID]string
	AcceptanceHash string
	TestPlanHash   string
}

type PlanResult struct {
	Status   string    `json:"status"`
	Mode     string    `json:"mode,omitempty"`
	Reason   string    `json:"reason,omitempty"`
	Manifest *Manifest `json:"manifest,omitempty"`
}

const WorkerBindingManifestEnvV1 = "manifest-env-v1"

type ProcessTokenResult struct {
	Status       string `json:"status"`
	PID          int    `json:"pid"`
	ProcessStart string `json:"process_start"`
}

type RecordOptions struct {
	RunID        string
	Candidate    CandidateID
	State        CandidateState
	WorkerID     string
	PID          int
	ProcessStart string
}

type ReapResult struct {
	RunID     string            `json:"run_id"`
	Removed   []CandidateID     `json:"removed,omitempty"`
	Preserved map[string]string `json:"preserved,omitempty"`
	Error     string            `json:"error,omitempty"`
}

// Run is the Forge command entry point. It never accepts a worktree path or
// branch name; every destructive target is loaded from the unique manifest.
func Run(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	var (
		result any
		err    error
	)
	switch args[0] {
	case "plan":
		result, err = runPlan(root, args[1:])
	case "process-token":
		if len(args) != 2 {
			err = errors.New("usage: forge process-token <pid>")
			break
		}
		result, err = runProcessToken(args[1])
	case "record":
		result, err = runRecord(root, args[1:])
	case "extract":
		if len(args) != 3 {
			err = errors.New("usage: forge extract <run-id> <A|B|C>")
			break
		}
		result, err = Extract(root, args[1], CandidateID(args[2]))
	case "merge":
		if len(args) != 3 {
			err = errors.New("usage: forge merge <run-id> <A|B|C>")
			break
		}
		result, err = Merge(root, args[1], CandidateID(args[2]))
	case "cleanup":
		if len(args) != 2 {
			err = errors.New("usage: forge cleanup <run-id>")
			break
		}
		result, err = Cleanup(root, args[1])
	case "reap":
		if len(args) > 2 {
			err = errors.New("usage: forge reap [slug]")
			break
		}
		slug := ""
		if len(args) == 2 {
			slug = args[1]
		}
		result, err = Reap(root, slug)
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	raw, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(stderr, "forge: encode result: %v\n", marshalErr)
		return 1
	}
	fmt.Fprintln(stdout, string(raw))
	return 0
}

const usage = "usage: forge plan|process-token|record|extract|merge|cleanup|reap ..."

func runPlan(root string, args []string) (PlanResult, error) {
	var (
		strategies                 strategyFlags
		slug, acceptance, testPlan string
		workerBinding              string
		workerBindingSet           bool
		positional                 []string
	)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--strategy" || arg == "--slug" || arg == "--acceptance-hash" || arg == "--test-plan-hash" || arg == "--worker-binding":
			if i+1 == len(args) {
				return PlanResult{}, fmt.Errorf("forge: %s requires a value", arg)
			}
			i++
			switch arg {
			case "--strategy":
				if err := strategies.Set(args[i]); err != nil {
					return PlanResult{}, err
				}
			case "--slug":
				slug = args[i]
			case "--acceptance-hash":
				acceptance = args[i]
			case "--test-plan-hash":
				testPlan = args[i]
			case "--worker-binding":
				if workerBindingSet {
					return PlanResult{}, errors.New("forge: worker binding may be declared only once")
				}
				workerBinding, workerBindingSet = args[i], true
			}
		case strings.HasPrefix(arg, "--strategy="):
			if err := strategies.Set(strings.TrimPrefix(arg, "--strategy=")); err != nil {
				return PlanResult{}, err
			}
		case strings.HasPrefix(arg, "--slug="):
			slug = strings.TrimPrefix(arg, "--slug=")
		case strings.HasPrefix(arg, "--acceptance-hash="):
			acceptance = strings.TrimPrefix(arg, "--acceptance-hash=")
		case strings.HasPrefix(arg, "--test-plan-hash="):
			testPlan = strings.TrimPrefix(arg, "--test-plan-hash=")
		case strings.HasPrefix(arg, "--worker-binding="):
			if workerBindingSet {
				return PlanResult{}, errors.New("forge: worker binding may be declared only once")
			}
			workerBinding, workerBindingSet = strings.TrimPrefix(arg, "--worker-binding="), true
		case strings.HasPrefix(arg, "-"):
			return PlanResult{}, fmt.Errorf("forge: unknown plan flag %s", arg)
		default:
			positional = append(positional, arg)
		}
	}
	if len(positional) < 1 || len(positional) > 2 {
		return PlanResult{}, errors.New("usage: forge plan <slice-id> [slug] --strategy A=text --strategy B=text --acceptance-hash <sha256> --test-plan-hash <sha256>")
	}
	if slug == "" && len(positional) == 2 {
		slug = positional[1]
	}
	switch {
	case !workerBindingSet:
		return degraded("supported worker binding was not declared"), nil
	case workerBinding != WorkerBindingManifestEnvV1:
		return PlanResult{}, fmt.Errorf("forge: unsupported worker binding %q (want %q)", workerBinding, WorkerBindingManifestEnvV1)
	}
	return Plan(root, PlanOptions{
		SliceID:        positional[0],
		Slug:           slug,
		Strategies:     strategies.values,
		AcceptanceHash: acceptance,
		TestPlanHash:   testPlan,
	})
}

func runProcessToken(value string) (ProcessTokenResult, error) {
	pid, err := parsePID(value)
	if err != nil {
		return ProcessTokenResult{}, err
	}
	token, err := ProcessStartToken(pid)
	if err != nil {
		return ProcessTokenResult{}, err
	}
	return ProcessTokenResult{Status: "ok", PID: pid, ProcessStart: token}, nil
}

func runRecord(root string, args []string) (*Manifest, error) {
	if len(args) < 3 {
		return nil, errors.New("usage: forge record <run-id> <candidate|winner|verification> <state> [flags]")
	}
	runID, subject, state := args[0], args[1], args[2]
	fs := flag.NewFlagSet("forge record", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var workerID, start string
	var pid int
	fs.StringVar(&workerID, "worker-id", "", "worker or recorder identity")
	fs.IntVar(&pid, "pid", 0, "worker PID")
	fs.StringVar(&start, "process-start", "", "worker process start token")
	if err := fs.Parse(args[3:]); err != nil || fs.NArg() != 0 {
		return nil, errors.New("forge: invalid record flags")
	}
	switch subject {
	case "winner":
		return RecordWinner(root, runID, CandidateID(state), workerID)
	case "verification":
		switch state {
		case "verified":
			return RecordVerification(root, runID, true, workerID)
		case "failed":
			return RecordVerification(root, runID, false, workerID)
		default:
			return nil, fmt.Errorf("forge: invalid verification state %q", state)
		}
	default:
		if state == "started" {
			state = string(StateRunning)
		}
		return Record(root, RecordOptions{
			RunID:        runID,
			Candidate:    CandidateID(subject),
			State:        CandidateState(state),
			WorkerID:     workerID,
			PID:          pid,
			ProcessStart: start,
		})
	}
}

type strategyFlags struct {
	values map[CandidateID]string
}

func (s *strategyFlags) Set(value string) error {
	idText, text, ok := strings.Cut(value, "=")
	id := CandidateID(idText)
	if !ok || !validCandidateID(id) || strings.TrimSpace(text) == "" {
		return fmt.Errorf("strategy must be A=text, B=text, or C=text")
	}
	if s.values == nil {
		s.values = map[CandidateID]string{}
	}
	if _, exists := s.values[id]; exists {
		return fmt.Errorf("duplicate strategy %s", id)
	}
	s.values[id] = strings.TrimSpace(text)
	return nil
}

func Plan(root string, opts PlanOptions) (PlanResult, error) {
	if strings.TrimSpace(opts.SliceID) == "" {
		return PlanResult{}, errors.New("forge: slice ID is required")
	}
	if !safeSlug.MatchString(opts.Slug) {
		return PlanResult{}, fmt.Errorf("forge: invalid feature slug %q", opts.Slug)
	}
	if len(opts.Strategies) < 2 || len(opts.Strategies) > 3 {
		return PlanResult{}, errors.New("forge: exactly 2 or 3 candidate strategies are required")
	}
	for id, strategy := range opts.Strategies {
		if !validCandidateID(id) || strings.TrimSpace(strategy) == "" {
			return PlanResult{}, fmt.Errorf("forge: invalid strategy %q", id)
		}
	}
	if !validSHA256(opts.AcceptanceHash) || !validSHA256(opts.TestPlanHash) {
		return PlanResult{}, errors.New("forge: acceptance and test-plan SHA-256 scorecard hashes are required")
	}
	physical, err := physicalExisting(root)
	if err != nil {
		return PlanResult{}, err
	}
	id, err := inspectRepo(physical)
	if err != nil {
		return degraded(err.Error()), nil
	}
	if super, _ := gitText(physical, "rev-parse", "--show-superproject-working-tree"); super != "" {
		return degraded("submodule checkout is not safe for parallel Forge"), nil
	}
	if inProgress(id.commonDir) {
		return degraded("repository has an in-progress Git operation"), nil
	}
	existingManifests, err := validatedManifestPaths(physical)
	if err != nil {
		return PlanResult{}, err
	}
	if dirty, err := dirtyStatus(physical, false, existingManifests); err != nil {
		return PlanResult{}, err
	} else if dirty {
		return degraded("primary index or worktree is dirty"), nil
	}
	if _, err := ProcessStartToken(os.Getpid()); err != nil {
		return degraded("worker liveness cannot be proven on this host"), nil
	}
	runID, nonce, err := newIdentity()
	if err != nil {
		return PlanResult{}, err
	}
	staging := forgeRoot(physical, runID)
	if err := validateStagingParent(staging); err != nil {
		return degraded(err.Error()), nil
	}
	registrations, err := registeredWorktrees(physical)
	if err != nil {
		return PlanResult{}, err
	}
	for path := range registrations {
		if path == staging || strings.HasPrefix(path, staging+string(filepath.Separator)) {
			return degraded("derived staging path collides with a registered worktree"), nil
		}
	}
	if _, err := os.Lstat(staging); err == nil {
		return degraded("derived staging path already exists"), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return PlanResult{}, fmt.Errorf("forge: inspect staging path: %w", err)
	}
	manifestPath, err := ManifestPath(physical, opts.Slug, runID)
	if err != nil {
		return PlanResult{}, err
	}
	if err := validateManifestParentBeforeCreate(physical, manifestPath); err != nil {
		return degraded(err.Error()), nil
	}
	ids := make([]CandidateID, 0, len(opts.Strategies))
	for candidateID := range opts.Strategies {
		ids = append(ids, candidateID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	for i, candidateID := range ids {
		want := CandidateID(rune('A' + i))
		if candidateID != want {
			return PlanResult{}, errors.New("forge: candidate IDs must be contiguous from A")
		}
		if _, err := gitText(physical, "show-ref", "--verify", "--quiet", "refs/heads/"+candidateBranch(runID, candidateID)); err == nil {
			return degraded("derived candidate branch already exists"), nil
		}
	}
	createdAt := now()
	m := &Manifest{
		Schema:         Schema,
		RunID:          runID,
		CreationNonce:  nonce,
		CreatedAt:      createdAt,
		FeatureSlug:    opts.Slug,
		SliceID:        strings.TrimSpace(opts.SliceID),
		AcceptanceHash: strings.ToLower(opts.AcceptanceHash),
		TestPlanHash:   strings.ToLower(opts.TestPlanHash),
		PrimaryRoot:    physical,
		GitCommonDir:   id.commonDir,
		ForgeRoot:      staging,
		Primary: Primary{
			Branch:         id.branch,
			BaseCommit:     id.head,
			BaselineTree:   id.tree,
			BaselineSHA256: id.treeHash,
		},
	}
	for _, candidateID := range ids {
		m.Candidates = append(m.Candidates, Candidate{
			ID:             candidateID,
			Strategy:       strings.TrimSpace(opts.Strategies[candidateID]),
			Worktree:       filepath.Join(staging, "candidate-"+strings.ToLower(string(candidateID))),
			Branch:         candidateBranch(runID, candidateID),
			InitialBase:    id.head,
			State:          StatePlanned,
			LastTransition: createdAt,
		})
	}
	// The manifest owns every derived path before the first Git side effect.
	if err := save(manifestPath, m); err != nil {
		return PlanResult{}, err
	}
	for i := range m.Candidates {
		c := &m.Candidates[i]
		if _, err := gitBytes(physical, nil, "worktree", "add", "-b", c.Branch, c.Worktree, c.InitialBase); err != nil {
			c.Preservation = err.Error()
			c.LastTransition = now()
			_ = save(manifestPath, m)
			return PlanResult{}, fmt.Errorf("forge: create candidate %s: %w", c.ID, err)
		}
		c.State = StateCreated
		c.LastTransition = now()
		if err := save(manifestPath, m); err != nil {
			return PlanResult{}, err
		}
	}
	return PlanResult{Status: "planned", Manifest: m}, nil
}

func Record(root string, opts RecordOptions) (*Manifest, error) {
	m, path, err := Load(root, opts.RunID)
	if err != nil {
		return nil, err
	}
	c, err := m.Candidate(opts.Candidate)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.WorkerID) == "" {
		return nil, errors.New("forge: worker ID is required")
	}
	switch opts.State {
	case StateRunning:
		if c.State == StateRunning && c.Worker.ID == opts.WorkerID && c.Worker.PID == opts.PID && c.Worker.ProcessStart == opts.ProcessStart {
			return m, nil
		}
		if c.State != StateCreated {
			return nil, fmt.Errorf("forge: candidate %s cannot transition from %s to running", c.ID, c.State)
		}
		token, err := ProcessStartToken(opts.PID)
		if err != nil || token != opts.ProcessStart {
			return nil, errors.New("forge: worker liveness token is not currently provable")
		}
		c.Worker = Worker{ID: opts.WorkerID, PID: opts.PID, ProcessStart: opts.ProcessStart, StartedAt: now()}
		c.State = StateRunning
	case StateFinished, StateFailed:
		if (c.State == opts.State || c.State == StateExtracted || c.State == StateMerged || c.State == StateCleaned) &&
			c.Worker.ID == opts.WorkerID {
			return m, nil
		}
		if c.State != StateRunning || c.Worker.ID != opts.WorkerID {
			return nil, fmt.Errorf("forge: candidate %s terminal transition does not match its live owner", c.ID)
		}
		c.State = opts.State
		c.Worker.FinishedAt = now()
		c.Worker.Result = string(opts.State)
	default:
		return nil, fmt.Errorf("forge: unsupported candidate state %q", opts.State)
	}
	c.LastTransition = now()
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func RecordWinner(root, runID string, winner CandidateID, recorder string) (*Manifest, error) {
	m, path, err := Load(root, runID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(recorder) == "" || !validCandidateID(winner) {
		return nil, errors.New("forge: valid winner and recorder identity are required")
	}
	if _, err := m.Candidate(winner); err != nil {
		return nil, err
	}
	if m.Winner.Candidate != "" {
		if m.Winner.Candidate == winner && m.Winner.RecordedBy == recorder {
			return m, nil
		}
		return nil, errors.New("forge: winner is already recorded and immutable")
	}
	m.Winner = Winner{Candidate: winner, RecordedBy: recorder, RecordedAt: now()}
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func RecordVerification(root, runID string, passed bool, recorder string) (*Manifest, error) {
	m, path, err := Load(root, runID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(recorder) == "" || m.Merge.State != "landed" {
		return nil, errors.New("forge: landed merge and verifier identity are required")
	}
	state := "failed"
	if passed {
		state = "verified"
	}
	if m.Verification.State != "" {
		if m.Verification.State == state && m.Verification.RecordedBy == recorder {
			return m, nil
		}
		return nil, errors.New("forge: verification result is already recorded and immutable")
	}
	m.Verification = VerificationState{State: state, RecordedBy: recorder, RecordedAt: now()}
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func Extract(root, runID string, candidateID CandidateID) (*Manifest, error) {
	m, path, err := Load(root, runID)
	if err != nil {
		return nil, err
	}
	c, err := m.Candidate(candidateID)
	if err != nil {
		return nil, err
	}
	if c.State == StateExtracted || c.State == StateMerged || c.State == StateCleaned {
		return verifyExtracted(m, c, path)
	}
	if c.State != StateFinished {
		return nil, fmt.Errorf("forge: candidate %s must be finished before extraction", c.ID)
	}
	if live, reason := workerLiveness(c.Worker); live != livenessDead {
		return nil, fmt.Errorf("forge: candidate %s preserved: %s", c.ID, reason)
	}
	if err := validateCandidateRepo(m, c); err != nil {
		return nil, err
	}
	head, err := gitText(c.Worktree, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, err
	}
	if _, err := gitBytes(c.Worktree, nil, "merge-base", "--is-ancestor", c.InitialBase, head); err != nil {
		return nil, fmt.Errorf("forge: candidate %s no longer descends from its initial base", c.ID)
	}
	if ignored, err := gitBytes(c.Worktree, nil, "ls-files", "--others", "--ignored", "--exclude-standard", "-z"); err != nil {
		return nil, err
	} else if len(ignored) != 0 {
		return preserve(path, m, c, "ignored content is not representable in the candidate tree")
	}
	if _, err := gitBytes(c.Worktree, nil, "add", "-A", "--", "."); err != nil {
		return nil, err
	}
	changed, err := gitBytes(c.Worktree, nil, "diff", "--cached", "--quiet", "--exit-code")
	if err != nil {
		_ = changed
		if _, commitErr := gitBytes(c.Worktree, nil,
			"-c", "user.name=DevRites Forge",
			"-c", "user.email=forge@devrites.local",
			"commit", "--no-gpg-sign", "--no-verify",
			"-m", "devrites forge snapshot "+m.RunID+" "+string(c.ID)); commitErr != nil {
			return nil, commitErr
		}
	}
	c.Commit, err = gitText(c.Worktree, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return nil, err
	}
	c.Tree, err = gitText(c.Worktree, "rev-parse", "--verify", "HEAD^{tree}")
	if err != nil {
		return nil, err
	}
	if err := rejectPrivateOrGitlinkChanges(c); err != nil {
		return preserve(path, m, c, err.Error())
	}
	delta, err := gitBytes(c.Worktree, nil, "diff", "--binary", "--full-index", "--no-renames", c.InitialBase, c.Commit, "--")
	if err != nil {
		return nil, err
	}
	c.DeltaSHA256 = sha256Hex(delta)
	if dirty, err := dirtyStatus(c.Worktree, true, nil); err != nil {
		return nil, err
	} else if dirty {
		return preserve(path, m, c, "candidate remains dirty after snapshot")
	}
	c.State = StateExtracted
	c.Preservation = ""
	c.LastTransition = now()
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func verifyExtracted(m *Manifest, c *Candidate, path string) (*Manifest, error) {
	if c.Commit == "" || c.Tree == "" || c.DeltaSHA256 == "" {
		return nil, fmt.Errorf("forge: candidate %s has incomplete extraction state", c.ID)
	}
	if c.State == StateCleaned {
		return m, nil
	}
	if err := validateCandidateRepo(m, c); err != nil {
		return nil, err
	}
	head, err := gitText(c.Worktree, "rev-parse", "HEAD^{commit}")
	if err != nil || head != c.Commit {
		return nil, fmt.Errorf("forge: candidate %s extracted commit drifted", c.ID)
	}
	delta, err := gitBytes(c.Worktree, nil, "diff", "--binary", "--full-index", "--no-renames", c.InitialBase, c.Commit, "--")
	if err != nil || sha256Hex(delta) != c.DeltaSHA256 {
		return nil, fmt.Errorf("forge: candidate %s extracted delta drifted", c.ID)
	}
	return m, nil
}

func Merge(root, runID string, winnerID CandidateID) (*Manifest, error) {
	m, path, err := Load(root, runID)
	if err != nil {
		return nil, err
	}
	if m.Winner.Candidate == "" || m.Winner.Candidate != winnerID {
		return nil, errors.New("forge: merge winner does not match the recorded judge winner")
	}
	winner, err := m.Candidate(winnerID)
	if err != nil {
		return nil, err
	}
	for i := range m.Candidates {
		c := &m.Candidates[i]
		if c.State != StateExtracted && c.State != StateMerged && c.State != StateCleaned {
			return nil, fmt.Errorf("forge: candidate %s has not been extracted", c.ID)
		}
	}
	id, err := inspectRepo(m.PrimaryRoot)
	if err != nil {
		return nil, err
	}
	if id.commonDir != m.GitCommonDir || id.branch != m.Primary.Branch {
		return nil, errors.New("forge: primary repository identity drifted")
	}
	manifestFiles, err := validatedManifestPaths(m.PrimaryRoot)
	if err != nil {
		return nil, err
	}
	if dirty, err := dirtyStatus(m.PrimaryRoot, false, manifestFiles); err != nil {
		return nil, err
	} else if dirty {
		return nil, errors.New("forge: primary index or worktree is dirty")
	}
	if id.head == winner.Commit && id.tree == winner.Tree {
		m.Merge = MergeState{State: "landed", Commit: winner.Commit, Tree: winner.Tree, RecordedAt: now()}
		winner.State = StateMerged
		winner.LastTransition = now()
		if err := save(path, m); err != nil {
			return nil, err
		}
		return m, nil
	}
	if id.head != m.Primary.BaseCommit || id.tree != m.Primary.BaselineTree {
		return nil, errors.New("forge: primary baseline changed before merge")
	}
	for i := range m.Candidates {
		c := &m.Candidates[i]
		if _, err := verifyExtracted(m, c, path); err != nil {
			return nil, err
		}
		if live, reason := workerLiveness(c.Worker); live != livenessDead {
			return nil, fmt.Errorf("forge: candidate %s preserved: %s", c.ID, reason)
		}
	}
	branchHead, err := gitText(m.PrimaryRoot, "rev-parse", "--verify", "refs/heads/"+winner.Branch+"^{commit}")
	if err != nil || branchHead != winner.Commit {
		return nil, errors.New("forge: winner branch no longer pins the recorded commit")
	}
	delta, err := gitBytes(m.PrimaryRoot, nil, "diff", "--binary", "--full-index", "--no-renames", winner.InitialBase, winner.Commit, "--")
	if err != nil || sha256Hex(delta) != winner.DeltaSHA256 {
		return nil, errors.New("forge: winner delta hash mismatch")
	}
	if _, err := gitBytes(m.PrimaryRoot, delta, "apply", "--check", "--index", "--binary", "--whitespace=nowarn", "-"); err != nil {
		return nil, fmt.Errorf("forge: merge conflict preflight failed; primary was not changed: %w", err)
	}
	m.Merge = MergeState{State: "preflight", Commit: winner.Commit, Tree: winner.Tree, RecordedAt: now()}
	if err := save(path, m); err != nil {
		return nil, err
	}
	if _, err := gitBytes(m.PrimaryRoot, nil, "merge", "--ff-only", "--no-edit", winner.Commit); err != nil {
		return nil, fmt.Errorf("forge: fast-forward merge failed; inspect primary before retry: %w", err)
	}
	id, err = inspectRepo(m.PrimaryRoot)
	if err != nil || id.head != winner.Commit || id.tree != winner.Tree {
		return nil, errors.New("forge: merge result did not match recorded winner")
	}
	if dirty, err := dirtyStatus(m.PrimaryRoot, false, manifestFiles); err != nil || dirty {
		return nil, errors.New("forge: primary is not clean after merge")
	}
	m.Merge = MergeState{State: "landed", Commit: winner.Commit, Tree: winner.Tree, RecordedAt: now()}
	winner.State = StateMerged
	winner.LastTransition = now()
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func Cleanup(root, runID string) (*Manifest, error) {
	m, path, err := Load(root, runID)
	if err != nil {
		return nil, err
	}
	if m.Merge.State != "landed" || m.Verification.State != "verified" {
		return nil, errors.New("forge: cleanup requires a landed winner and recorded successful verification")
	}
	if _, err := validatePrimary(m, true); err != nil {
		return nil, err
	}
	preserved := map[string]string{}
	for i := range m.Candidates {
		c := &m.Candidates[i]
		removed, reason := removeEligible(m, c)
		if reason != "" {
			preserved[string(c.ID)] = reason
			c.Preservation = reason
			continue
		}
		if removed {
			c.State = StateCleaned
			c.Preservation = ""
			c.LastTransition = now()
			if err := save(path, m); err != nil {
				return nil, err
			}
		}
	}
	state := "complete"
	if len(preserved) != 0 {
		state = "partial"
	}
	m.Cleanup = CleanupState{State: state, Preserved: preserved, RecordedAt: now()}
	if err := save(path, m); err != nil {
		return nil, err
	}
	return m, nil
}

func Reap(root, slug string) ([]ReapResult, error) {
	root, err := physicalExisting(root)
	if err != nil {
		return nil, err
	}
	if slug != "" && !safeSlug.MatchString(slug) {
		return nil, fmt.Errorf("forge: invalid feature slug %q", slug)
	}
	paths, err := manifestPaths(root, slug)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, path := range paths {
		counts[filepath.Base(filepath.Dir(path))]++
	}
	loaded := make(map[string]*Manifest, len(paths))
	for _, path := range paths {
		m, loadErr := loadAt(root, path)
		if loadErr == nil {
			loaded[path] = m
		}
	}
	results := make([]ReapResult, 0, len(paths))
	for _, path := range paths {
		m, ok := loaded[path]
		if !ok {
			_, loadErr := loadAt(root, path)
			results = append(results, ReapResult{Error: loadErr.Error()})
			continue
		}
		if counts[m.RunID] != 1 {
			results = append(results, ReapResult{RunID: m.RunID, Error: "duplicate manifest; preserved"})
			continue
		}
		result := ReapResult{RunID: m.RunID, Preserved: map[string]string{}}
		changed := false
		for i := range m.Candidates {
			c := &m.Candidates[i]
			removed, reason := removeEligible(m, c)
			if reason != "" {
				result.Preserved[string(c.ID)] = reason
				if c.Preservation != reason {
					c.Preservation = reason
					c.LastTransition = now()
					changed = true
				}
				continue
			}
			if removed {
				result.Removed = append(result.Removed, c.ID)
				c.State = StateCleaned
				c.Preservation = ""
				c.LastTransition = now()
				changed = true
			}
		}
		if len(result.Preserved) == 0 {
			result.Preserved = nil
		}
		if changed {
			if err := save(path, m); err != nil {
				result.Error = err.Error()
			}
		}
		results = append(results, result)
	}
	return results, nil
}

func removeEligible(m *Manifest, c *Candidate) (bool, string) {
	if c.State == StateCleaned {
		return false, ""
	}
	if c.State != StateExtracted && c.State != StateMerged {
		return false, "candidate is not terminal and extracted"
	}
	if live, reason := workerLiveness(c.Worker); live != livenessDead {
		return false, reason
	}
	if c.Commit == "" || c.Tree == "" {
		return false, "candidate content is not recorded as a reachable commit"
	}
	head, err := gitText(m.PrimaryRoot, "rev-parse", "--verify", "refs/heads/"+c.Branch+"^{commit}")
	if err != nil || head != c.Commit {
		return false, "candidate branch no longer pins the recorded commit"
	}
	if _, err := gitBytes(m.PrimaryRoot, nil, "cat-file", "-e", c.Commit+"^{commit}"); err != nil {
		return false, "candidate commit is not reachable"
	}
	if _, err := os.Lstat(c.Worktree); errors.Is(err, os.ErrNotExist) {
		registered, listErr := registeredWorktrees(m.PrimaryRoot)
		if listErr != nil {
			return false, "cannot verify missing worktree registration"
		}
		if _, exists := registered[c.Worktree]; exists {
			return false, "worktree path is missing but remains registered"
		}
		return true, ""
	} else if err != nil {
		return false, "cannot inspect candidate worktree"
	}
	if err := validateCandidateRepo(m, c); err != nil {
		return false, err.Error()
	}
	if dirty, err := dirtyStatus(c.Worktree, true, nil); err != nil {
		return false, err.Error()
	} else if dirty {
		return false, "candidate contains tracked, untracked, or ignored dirt"
	}
	if _, err := gitBytes(m.PrimaryRoot, nil, "worktree", "remove", c.Worktree); err != nil {
		return false, err.Error()
	}
	if _, err := os.Lstat(c.Worktree); !errors.Is(err, os.ErrNotExist) {
		return false, "candidate worktree still exists after removal"
	}
	return true, ""
}

func preserve(path string, m *Manifest, c *Candidate, reason string) (*Manifest, error) {
	c.Preservation = reason
	c.LastTransition = now()
	if err := save(path, m); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("forge: candidate %s preserved: %s", c.ID, reason)
}

func rejectPrivateOrGitlinkChanges(c *Candidate) error {
	names, err := gitBytes(c.Worktree, nil, "diff", "--name-only", "-z", c.InitialBase, c.Commit, "--")
	if err != nil {
		return err
	}
	for _, name := range bytes.Split(names, []byte{0}) {
		clean := filepath.ToSlash(string(name))
		if clean == ".devrites" || strings.HasPrefix(clean, ".devrites/") {
			return errors.New("candidate changed root-owned .devrites content")
		}
	}
	tree, err := gitBytes(c.Worktree, nil, "ls-tree", "-r", "-z", c.Commit)
	if err != nil {
		return err
	}
	for _, entry := range bytes.Split(tree, []byte{0}) {
		if bytes.HasPrefix(entry, []byte("160000 ")) {
			return errors.New("candidate tree contains a gitlink")
		}
	}
	return nil
}

func validateStagingParent(staging string) error {
	parent := filepath.Dir(staging)
	if info, err := os.Lstat(parent); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("forge: staging parent is a symlink: %s", parent)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("forge: inspect staging parent: %w", err)
	}
	return nil
}

func validateManifestParentBeforeCreate(root, path string) error {
	current := root
	rel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return errors.New("forge: manifest path escapes primary root")
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("forge: manifest parent is a symlink: %s", current)
		}
	}
	return nil
}

func degraded(reason string) PlanResult {
	return PlanResult{Status: "degraded", Mode: "serial", Reason: strings.TrimPrefix(reason, "forge: ")}
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func parsePID(value string) (int, error) {
	pid, err := strconv.Atoi(value)
	if err != nil || pid <= 0 || strconv.Itoa(pid) != value {
		return 0, errors.New("forge: PID must be a canonical positive integer")
	}
	return pid, nil
}
