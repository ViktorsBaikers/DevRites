package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/orient"
	"github.com/devrites/devrites/internal/safepath"
)

const (
	agentDispatchStateVersion = "devrites-agent-dispatch/v1"
	agentDispatchStateDir     = "devrites-agent-dispatch-v1"
	agentDispatchMetadataRole = "devrites-skill-contract-error"
	agentDispatchSkillGuard   = "devrites-skill-dispatch-guard"
	maxCodexRolloutLine       = 8 << 20
)

var (
	devritesAgentPathRe = regexp.MustCompile(`\.codex/agents/(devrites-[a-z0-9-]+)\.toml`)
	devritesAgentNameRe = regexp.MustCompile(`^devrites-[a-z0-9-]+$`)
	skillInvocationRe   = regexp.MustCompile(`(?i)(?:^|[^a-z0-9_./-])(?:\$|/)([a-z][a-z0-9-]*)\b`)
	reconcileCloseRe    = regexp.MustCompile(`(?m)(?:^|[;&|]\s*)(?:rtk\s+)?(?:[A-Za-z0-9_./-]+/)?devrites-engine\s+reconcile\s+close\b`)
	dispatchWaiveRe     = regexp.MustCompile(`^\s*(?:rtk\s+)?(?:[A-Za-z0-9_./-]+/)?devrites-engine\s+dispatch-waive\s+([a-z-]+)\s*$`)
)

type agentDispatchHookInput struct {
	HookEventName        string          `json:"hook_event_name"`
	SessionID            string          `json:"session_id"`
	TurnID               string          `json:"turn_id"`
	ToolName             string          `json:"tool_name"`
	ToolUseID            string          `json:"tool_use_id"`
	AgentType            string          `json:"agent_type"`
	AgentID              string          `json:"agent_id"`
	Prompt               string          `json:"prompt"`
	StopHookActive       bool            `json:"stop_hook_active"`
	LastAssistantMessage string          `json:"last_assistant_message"`
	ToolResponse         string          `json:"-"`
	ToolInputRaw         json.RawMessage `json:"-"`
	ToolInput            struct {
		Command           string   `json:"command"`
		Cmd               string   `json:"cmd"`
		AgentType         string   `json:"agent_type"`
		ForkTurns         string   `json:"fork_turns"`
		Message           string   `json:"message"`
		TaskName          string   `json:"task_name"`
		ReceiverThreadIDs []string `json:"receiver_thread_ids"`
		IDs               []string `json:"ids"`
		AgentIDs          []string `json:"agent_ids"`
		Target            string   `json:"target"`
		TimeoutMS         int      `json:"timeout_ms"`
	} `json:"tool_input"`
}

type agentDispatchEvent struct {
	Version      string `json:"version"`
	Event        string `json:"event"`
	TurnID       string `json:"turn_id"`
	ToolUseID    string `json:"tool_use_id,omitempty"`
	AgentID      string `json:"agent_id,omitempty"`
	AgentType    string `json:"agent_type,omitempty"`
	Role         string `json:"role,omitempty"`
	WindowID     string `json:"window_id,omitempty"`
	ResultSHA256 string `json:"result_sha256,omitempty"`
	Reason       string `json:"reason,omitempty"`
	AtUnixNano   int64  `json:"at_unix_nano"`
}

type agentDispatchAttempt struct {
	Role         string
	AgentType    string
	ToolUseID    string
	AgentID      string
	WindowID     string
	Started      bool
	Stopped      bool
	Waited       bool
	ResultSHA256 string
}

type codexRolloutRecord struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Payload   struct {
		Type            string `json:"type"`
		SessionID       string `json:"session_id"`
		ID              string `json:"id"`
		ParentThreadID  string `json:"parent_thread_id"`
		CWD             string `json:"cwd"`
		AgentPath       string `json:"agent_path"`
		AgentRole       string `json:"agent_role"`
		Name            string `json:"name"`
		Arguments       string `json:"arguments"`
		Author          string `json:"author"`
		Recipient       string `json:"recipient"`
		Role            string `json:"role"`
		LastAgentResult string `json:"last_agent_message"`
		Content         []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Metadata struct {
			TurnID string `json:"turn_id"`
		} `json:"internal_chat_message_metadata_passthrough"`
	} `json:"payload"`
}

type codexV2Spawn struct {
	Role       string
	AgentPath  string
	SpawnLine  int
	ResultLine int
	SpawnedAt  time.Time
	Result     string
}

func parseAgentDispatchHookInput(r io.Reader) (agentDispatchHookInput, error) {
	var in agentDispatchHookInput
	data, err := io.ReadAll(r)
	if err != nil {
		return in, err
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return in, err
	}
	var raw struct {
		ToolInput    json.RawMessage `json:"tool_input"`
		ToolResponse json.RawMessage `json:"tool_response"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return in, err
	}
	in.ToolInputRaw = raw.ToolInput
	if len(raw.ToolResponse) > 0 {
		if raw.ToolResponse[0] == '"' {
			_ = json.Unmarshal(raw.ToolResponse, &in.ToolResponse)
		} else {
			in.ToolResponse = string(raw.ToolResponse)
		}
	}
	in.HookEventName = strings.TrimSpace(in.HookEventName)
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.TurnID = strings.TrimSpace(in.TurnID)
	in.ToolName = strings.TrimSpace(in.ToolName)
	in.ToolUseID = strings.TrimSpace(in.ToolUseID)
	in.AgentType = strings.TrimSpace(in.AgentType)
	in.AgentID = strings.TrimSpace(in.AgentID)
	in.ToolInput.AgentType = strings.TrimSpace(in.ToolInput.AgentType)
	in.ToolInput.ForkTurns = strings.TrimSpace(in.ToolInput.ForkTurns)
	in.ToolInput.TaskName = strings.TrimSpace(in.ToolInput.TaskName)
	return in, nil
}

func agentDispatchRoot() (string, bool) {
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return "", false
	}
	return root, true
}

func agentDispatchLogPath(root, sessionID string) (string, error) {
	if strings.TrimSpace(sessionID) == "" {
		return "", fmt.Errorf("missing session_id")
	}
	rootSum := sha256.Sum256([]byte(filepath.Clean(root)))
	sessionSum := sha256.Sum256([]byte(sessionID))
	dir := filepath.Join(os.TempDir(), agentDispatchStateDir, hex.EncodeToString(rootSum[:]))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, hex.EncodeToString(sessionSum[:])+".jsonl"), nil
}

func appendAgentDispatchEvent(root, sessionID string, event agentDispatchEvent) error {
	path, err := agentDispatchLogPath(root, sessionID)
	if err != nil {
		return err
	}
	event.Version = agentDispatchStateVersion
	event.AtUnixNano = time.Now().UnixNano()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func readAgentDispatchEvents(root, sessionID string) ([]agentDispatchEvent, error) {
	path, err := agentDispatchLogPath(root, sessionID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var events []agentDispatchEvent
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var event agentDispatchEvent
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("invalid agent dispatch state: %w", err)
		}
		if event.Version != agentDispatchStateVersion {
			return nil, fmt.Errorf("unsupported agent dispatch state %q", event.Version)
		}
		events = append(events, event)
	}
	return events, nil
}

func explicitRolesInText(text string) []string {
	matches := devritesAgentPathRe.FindAllStringSubmatch(text, -1)
	seen := map[string]struct{}{}
	var roles []string
	for _, match := range matches {
		role := match[1]
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func requiredAgentRolesFromSkill(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("skill contract is not a regular file")
	}
	if info.Size() > 256<<10 {
		return nil, fmt.Errorf("skill contract exceeds 256 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("skill contract is missing frontmatter")
	}
	value := ""
	found := false
	closed := false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		if !strings.HasPrefix(line, "required-agent-roles:") {
			continue
		}
		if found {
			return nil, fmt.Errorf("required-agent-roles is duplicated")
		}
		found = true
		value = strings.TrimSpace(strings.TrimPrefix(line, "required-agent-roles:"))
	}
	if !closed {
		return nil, fmt.Errorf("skill contract has unterminated frontmatter")
	}
	if !found {
		return nil, fmt.Errorf("skill contract is missing required-agent-roles")
	}
	value = strings.Trim(value, `"'`)
	if value == "none" {
		return nil, nil
	}
	if value == "" {
		return nil, fmt.Errorf("required-agent-roles is empty")
	}
	seen := map[string]struct{}{}
	var roles []string
	for _, item := range strings.Split(value, ",") {
		role := strings.TrimSpace(item)
		if !devritesAgentNameRe.MatchString(role) {
			return nil, fmt.Errorf("invalid required agent role %q", role)
		}
		if _, ok := seen[role]; ok {
			return nil, fmt.Errorf("duplicate required agent role %q", role)
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles, nil
}

func requiredRolesInPrompt(root, text string) ([]string, bool, error) {
	seen := map[string]struct{}{}
	skillInvoked := false
	skillsDir := filepath.Join(filepath.Dir(root), ".agents", "skills")
	for _, match := range skillInvocationRe.FindAllStringSubmatch(text, -1) {
		skill := strings.ToLower(match[1])
		path := filepath.Join(skillsDir, skill, "SKILL.md")
		if !safepath.WithinResolved(path, skillsDir) {
			return nil, false, fmt.Errorf("unsafe skill path for %s", skill)
		}
		roles, err := requiredAgentRolesFromSkill(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, false, fmt.Errorf("%s: %w", skill, err)
		}
		skillInvoked = true
		for _, role := range roles {
			seen[role] = struct{}{}
		}
	}
	roles := make([]string, 0, len(seen))
	for role := range seen {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles, skillInvoked, nil
}

func expectedAgentType(role string) string {
	if role == "devrites-slice-wright" {
		return "worker"
	}
	return "explorer"
}

func agentRoleDeveloperInstructions(root, role string) (string, error) {
	agentsDir := filepath.Join(filepath.Dir(root), ".codex", "agents")
	path := filepath.Join(agentsDir, role+".toml")
	if !safepath.WithinResolved(path, agentsDir) {
		return "", fmt.Errorf("unsafe role path")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("role contract is not a regular file")
	}
	if info.Size() > 256<<10 {
		return "", fmt.Errorf("role contract exceeds 256 KiB")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	wantName := `name = "` + role + `"`
	nameMatches := false
	start := -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case wantName:
			nameMatches = true
		case "developer_instructions = '''":
			start = i + 1
		}
		if start >= 0 {
			break
		}
	}
	if !nameMatches || start < 0 {
		return "", fmt.Errorf("role contract name or developer_instructions is invalid")
	}
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "'''" {
			continue
		}
		instructions := strings.TrimSpace(strings.Join(lines[start:i], "\n"))
		if instructions == "" {
			return "", fmt.Errorf("developer_instructions is empty")
		}
		return instructions, nil
	}
	return "", fmt.Errorf("developer_instructions is unterminated")
}

func currentReconcileWindowID() string {
	_, _, dir, ok := resolveWorkspace()
	if !ok {
		return ""
	}
	// The wright-start canonical fingerprint is captured by the spawn hook
	// before the child starts. It narrows the delta but does not change source
	// authorization.
	names := []string{".reconcile-base", ".reconcile-allowlist", ".reconcile-devrites"}
	h := sha256.New()
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return ""
		}
		_, _ = io.WriteString(h, name)
		_, _ = h.Write([]byte{0})
		_, _ = h.Write(data)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func dispatchTurnState(events []agentDispatchEvent, turnID string) (map[string]struct{}, []*agentDispatchAttempt) {
	armed := map[string]struct{}{}
	attemptByTool := map[string]*agentDispatchAttempt{}
	attemptByAgent := map[string]*agentDispatchAttempt{}
	var attempts []*agentDispatchAttempt
	for _, event := range events {
		if event.TurnID != turnID {
			continue
		}
		switch event.Event {
		case "armed":
			armed[event.Role] = struct{}{}
		case "waived":
			delete(armed, event.Role)
		case "pending":
			armed[event.Role] = struct{}{}
			attempt := &agentDispatchAttempt{
				Role:      event.Role,
				AgentType: event.AgentType,
				ToolUseID: event.ToolUseID,
				WindowID:  event.WindowID,
			}
			attemptByTool[event.ToolUseID] = attempt
			attempts = append(attempts, attempt)
		case "started":
			attempt := attemptByTool[event.ToolUseID]
			if attempt == nil {
				attempt = &agentDispatchAttempt{
					Role:      event.Role,
					AgentType: event.AgentType,
					ToolUseID: event.ToolUseID,
					WindowID:  event.WindowID,
				}
				attempts = append(attempts, attempt)
			}
			attempt.Started = true
			attempt.AgentID = event.AgentID
			attemptByAgent[event.AgentID] = attempt
		case "waited":
			if attempt := attemptByAgent[event.AgentID]; attempt != nil {
				attempt.Waited = true
			}
		case "stopped":
			if attempt := attemptByAgent[event.AgentID]; attempt != nil {
				attempt.Stopped = true
				attempt.ResultSHA256 = event.ResultSHA256
			}
		case "verified":
			attempts = append(attempts, &agentDispatchAttempt{
				Role:         event.Role,
				AgentType:    event.AgentType,
				AgentID:      event.AgentID,
				WindowID:     event.WindowID,
				Started:      true,
				Stopped:      true,
				Waited:       true,
				ResultSHA256: event.ResultSHA256,
			})
		}
	}
	return armed, attempts
}

func durableCodexV2DispatchAttempts(
	root, sessionID, turnID string,
	armed map[string]struct{},
) ([]*agentDispatchAttempt, error) {
	parentPath, sessionsDir, err := findCodexParentRollout(sessionID)
	if err != nil || parentPath == "" {
		return nil, err
	}
	projectRoot := filepath.Dir(root)
	spawns, waitLines, err := readCodexV2ParentRollout(parentPath, projectRoot, sessionID, turnID, armed)
	if err != nil {
		return nil, err
	}
	var attempts []*agentDispatchAttempt
	for _, spawn := range spawns {
		role := spawn.Role
		var childPath string
		var childMeta codexRolloutRecord
		if strings.TrimSpace(spawn.Result) != "" {
			childPath, childMeta, err = findCodexV2ChildRollout(
				sessionsDir, parentPath, projectRoot, sessionID, spawn.AgentPath,
			)
			if err != nil {
				return nil, err
			}
			if childPath != "" {
				role = childMeta.Payload.AgentRole
			}
		}
		if !devritesAgentNameRe.MatchString(role) || role == agentDispatchSkillGuard {
			if strings.TrimSpace(spawn.Result) != "" {
				return nil, fmt.Errorf("codex V2 completed a default or non-DevRites child during a DevRites skill turn; this turn is tainted and cannot be repaired by another dispatch. Do not use the child result or advance workflow state. End this turn, then retry the skill in a fresh turn; %s", conditionalDispatchInstruction())
			}
			continue
		}
		armed[role] = struct{}{}
		attempt := &agentDispatchAttempt{
			Role:      role,
			AgentType: role,
			Started:   true,
		}
		attempts = append(attempts, attempt)
		if strings.TrimSpace(spawn.Result) == "" || !lineBetween(waitLines, spawn.SpawnLine, spawn.ResultLine) {
			continue
		}
		if childPath == "" || childMeta.Payload.AgentRole != role {
			continue
		}
		instructions, err := agentRoleDeveloperInstructions(root, role)
		if err != nil {
			return nil, fmt.Errorf("named role %s is not loadable: %w", role, err)
		}
		childResult, err := codexChildResult(childPath, instructions)
		if err != nil {
			return nil, err
		}
		if childMeta.Payload.ID == "" || strings.TrimSpace(childResult) == "" {
			continue
		}
		attempt.AgentID = childMeta.Payload.ID
		attempt.Stopped = true
		attempt.Waited = true
		if role == "devrites-slice-wright" {
			attempt.WindowID = currentReconcileWindowID()
			if attempt.WindowID == "" || !reconcileWindowPredates(spawn.SpawnedAt) {
				continue
			}
		}
		sum := sha256.Sum256([]byte(childResult))
		attempt.ResultSHA256 = hex.EncodeToString(sum[:])
	}
	return attempts, nil
}

func findCodexParentRollout(sessionID string) (string, string, error) {
	codeHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codeHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", nil
		}
		codeHome = filepath.Join(home, ".codex")
	}
	sessionsDir := filepath.Join(codeHome, "sessions")
	wantSuffix := "-" + sessionID + ".jsonl"
	var parentPath string
	err := filepath.WalkDir(sessionsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), wantSuffix) {
			return nil
		}
		parentPath = path
		return fs.SkipAll
	})
	if os.IsNotExist(err) {
		return "", sessionsDir, nil
	}
	return parentPath, sessionsDir, err
}

func readCodexV2ParentRollout(
	path, projectRoot, sessionID, turnID string,
	armed map[string]struct{},
) (map[string]*codexV2Spawn, []int, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	spawns := map[string]*codexV2Spawn{}
	var waitLines []int
	metaSeen := false
	_, guarded := armed[agentDispatchSkillGuard]
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), maxCodexRolloutLine)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"session_meta"`)) &&
			!bytes.Contains(line, []byte(`"spawn_agent"`)) &&
			!bytes.Contains(line, []byte(`"wait_agent"`)) &&
			!bytes.Contains(line, []byte(`"agent_message"`)) {
			continue
		}
		var record codexRolloutRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, nil, fmt.Errorf("invalid Codex parent rollout: %w", err)
		}
		if record.Type == "session_meta" {
			if record.Payload.ID != sessionID || !sameResolvedPath(record.Payload.CWD, projectRoot) {
				return nil, nil, fmt.Errorf("codex parent rollout does not match the current session root")
			}
			metaSeen = true
			continue
		}
		if record.Payload.Metadata.TurnID != turnID {
			continue
		}
		switch {
		case record.Payload.Type == "function_call" && record.Payload.Name == "spawn_agent":
			var args struct {
				AgentType string `json:"agent_type"`
				TaskName  string `json:"task_name"`
				ForkTurns string `json:"fork_turns"`
			}
			if json.Unmarshal([]byte(record.Payload.Arguments), &args) != nil {
				continue
			}
			_, required := armed[args.AgentType]
			if (!required && !guarded) || args.TaskName == "" ||
				strings.Contains(args.TaskName, "/") || args.ForkTurns != "none" {
				continue
			}
			spawnedAt, err := time.Parse(time.RFC3339Nano, record.Timestamp)
			if err != nil {
				return nil, nil, fmt.Errorf("codex spawn timestamp is invalid: %w", err)
			}
			spawns["/root/"+args.TaskName] = &codexV2Spawn{
				Role:      args.AgentType,
				AgentPath: "/root/" + args.TaskName,
				SpawnLine: lineNumber,
				SpawnedAt: spawnedAt,
			}
		case record.Payload.Type == "function_call" && record.Payload.Name == "wait_agent":
			waitLines = append(waitLines, lineNumber)
		case record.Payload.Type == "agent_message" &&
			record.Payload.Recipient == "/root":
			spawn := spawns[record.Payload.Author]
			if spawn == nil {
				continue
			}
			spawn.Result = rolloutInputText(record.Payload.Content)
			spawn.ResultLine = lineNumber
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("read Codex parent rollout: %w", err)
	}
	if !metaSeen {
		return nil, nil, fmt.Errorf("codex parent rollout is missing session metadata")
	}
	return spawns, waitLines, nil
}

func findCodexV2ChildRollout(
	sessionsDir, parentPath, projectRoot, sessionID, agentPath string,
) (string, codexRolloutRecord, error) {
	var childPath string
	var childMeta codexRolloutRecord
	err := filepath.WalkDir(sessionsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() || path == parentPath || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		record, ok, err := firstCodexRolloutRecord(path)
		if err != nil || !ok {
			return err
		}
		if record.Type != "session_meta" ||
			record.Payload.ParentThreadID != sessionID ||
			record.Payload.AgentPath != agentPath ||
			!sameResolvedPath(record.Payload.CWD, projectRoot) {
			return nil
		}
		if childPath != "" {
			return fmt.Errorf("multiple Codex child rollouts match %s", agentPath)
		}
		childPath = path
		childMeta = record
		return nil
	})
	return childPath, childMeta, err
}

func firstCodexRolloutRecord(path string) (codexRolloutRecord, bool, error) {
	var record codexRolloutRecord
	file, err := os.Open(path)
	if err != nil {
		return record, false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), maxCodexRolloutLine)
	if !scanner.Scan() {
		return record, false, scanner.Err()
	}
	if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
		return record, false, nil
	}
	return record, true, nil
}

func codexChildResult(path, instructions string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), maxCodexRolloutLine)
	instructionsLoaded := false
	result := ""
	for scanner.Scan() {
		line := scanner.Bytes()
		if !bytes.Contains(line, []byte(`"task_complete"`)) &&
			!bytes.Contains(line, []byte(`"developer"`)) {
			continue
		}
		var record codexRolloutRecord
		if json.Unmarshal(line, &record) != nil {
			continue
		}
		if record.Payload.Type == "message" && record.Payload.Role == "developer" &&
			strings.Contains(rolloutInputText(record.Payload.Content), instructions) {
			instructionsLoaded = true
		}
		if record.Payload.Type == "task_complete" &&
			strings.TrimSpace(record.Payload.LastAgentResult) != "" {
			result = record.Payload.LastAgentResult
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if !instructionsLoaded {
		return "", nil
	}
	return result, nil
}

func rolloutInputText(content []struct {
	Type string `json:"type"`
	Text string `json:"text"`
}) string {
	var text []string
	for _, item := range content {
		if item.Type == "input_text" && strings.TrimSpace(item.Text) != "" {
			text = append(text, item.Text)
		}
	}
	return strings.Join(text, "\n")
}

func lineBetween(lines []int, after, before int) bool {
	for _, line := range lines {
		if line > after && line < before {
			return true
		}
	}
	return false
}

func sameResolvedPath(left, right string) bool {
	resolve := func(path string) string {
		absolute, err := filepath.Abs(path)
		if err == nil {
			path = absolute
		}
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		return filepath.Clean(path)
	}
	return resolve(left) == resolve(right)
}

func reconcileWindowPredates(at time.Time) bool {
	_, _, dir, ok := resolveWorkspace()
	if !ok {
		return false
	}
	for _, name := range []string{".reconcile-base", ".reconcile-allowlist", ".reconcile-devrites"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.ModTime().After(at) {
			return false
		}
	}
	return true
}

func dispatchAttemptComplete(attempt *agentDispatchAttempt) bool {
	return attempt.Started && attempt.Stopped && attempt.Waited && attempt.ResultSHA256 != ""
}

func roleSatisfied(role, windowID string, attempts []*agentDispatchAttempt) bool {
	for _, attempt := range attempts {
		if attempt.Role != role || !dispatchAttemptComplete(attempt) {
			continue
		}
		if windowID == "" || attempt.WindowID == windowID {
			return true
		}
	}
	return false
}

func dispatchInstruction(role string) string {
	return fmt.Sprintf(
		"On MultiAgent V2 call spawn_agent with agent_type=%s, a unique task_name, and fork_turns=\"none\" so Codex loads .codex/agents/%s.toml natively. GPT-5.6 V2 may omit agent_type from the visible tool schema even though the runtime accepts it; send agent_type anyway rather than using a default child. On V1 use agent_type=%s with fork_turns=\"none\" and name that role TOML in the message. Wait for the returned child and use its non-empty result. Do not call wait before spawn_agent and do not synthesize the agent result.",
		role, role, expectedAgentType(role),
	)
}

func conditionalDispatchInstruction() string {
	return "CONDITIONAL DISPATCH RULE — If this skill reaches a child-agent step, call spawn_agent with the exact named agent_type=devrites-<role> specified by the skill, a unique task_name, and fork_turns=\"none\". GPT-5.6 V2 may omit agent_type from the visible tool schema; send agent_type anyway and never use a default child. Wait for the returned child and use its non-empty result."
}

func incompleteDispatchReason(armed map[string]struct{}, attempts []*agentDispatchAttempt) string {
	roles := make([]string, 0, len(armed))
	for role := range armed {
		if role == agentDispatchSkillGuard {
			continue
		}
		if !roleSatisfied(role, "", attempts) {
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		return ""
	}
	sort.Strings(roles)
	role := roles[0]
	if role == agentDispatchMetadataRole {
		return "DevRites could not determine this skill's required agents because its installed required-agent-roles contract is invalid. Reinstall or repair the DevRites skill pack before continuing."
	}
	return fmt.Sprintf("DevRites dispatch for %s is not complete. %s", role, dispatchInstruction(role))
}

func preToolDeny(h harness.Harness, reason string, stdout, stderr io.Writer) int {
	out, err := h.PreToolDeny(reason)
	if err != nil {
		debugf(stderr, "agent-dispatch: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

func preToolRewrite(h harness.Harness, toolInput json.RawMessage, message string, stdout, stderr io.Writer) int {
	var updated map[string]any
	if err := json.Unmarshal(toolInput, &updated); err != nil {
		return preToolDeny(h, "DevRites could not bind developer_instructions to the spawn input.", stdout, stderr)
	}
	updated["message"] = message
	out, err := json.Marshal(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":      "PreToolUse",
			"permissionDecision": "allow",
			"updatedInput":       updated,
		},
	})
	if err != nil {
		return preToolDeny(h, "DevRites could not encode the bound spawn input.", stdout, stderr)
	}
	fmt.Fprintln(stdout, string(out))
	return exitOK
}

func stopDispatchBlock(h harness.Harness, reason string, stdout, stderr io.Writer) int {
	out, err := h.StopBlock(reason)
	if err != nil {
		debugf(stderr, "agent-dispatch: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

func stopDispatchFailure(reason string, stdout, stderr io.Writer) int {
	out, err := json.Marshal(map[string]any{
		"continue":      false,
		"stopReason":    reason,
		"systemMessage": reason,
	})
	if err != nil {
		debugf(stderr, "agent-dispatch: %v", err)
		return exitOK
	}
	fmt.Fprintln(stdout, string(out))
	return exitOK
}

func refreshPendingWrightBoundary(
	root string,
	in agentDispatchHookInput,
	armed map[string]struct{},
	attempts []*agentDispatchAttempt,
) error {
	if in.AgentID != "" || in.AgentType != "" || currentReconcileWindowID() == "" {
		return nil
	}
	for _, attempt := range attempts {
		if attempt.Role == "devrites-slice-wright" {
			return nil
		}
	}
	boundaryArmed := make(map[string]struct{}, len(armed)+1)
	for role := range armed {
		boundaryArmed[role] = struct{}{}
	}
	boundaryArmed["devrites-slice-wright"] = struct{}{}
	durable, err := durableCodexV2DispatchAttempts(root, in.SessionID, in.TurnID, boundaryArmed)
	if err != nil {
		return err
	}
	for _, attempt := range durable {
		if attempt.Role == "devrites-slice-wright" {
			return nil
		}
	}
	workspaceRoot, slug, _, ok := resolveWorkspace()
	if !ok || !sameResolvedPath(workspaceRoot, root) {
		return fmt.Errorf("active reconcile window is unavailable")
	}
	return lib.CaptureReconcileWrightBoundary(root, slug)
}

func hookAgentDispatch(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	if h != harness.Codex {
		_, _ = io.Copy(io.Discard, stdin)
		return exitOK
	}
	in, err := parseAgentDispatchHookInput(stdin)
	if err != nil {
		return exitOK
	}
	root, ok := agentDispatchRoot()
	if !ok || in.SessionID == "" || in.TurnID == "" {
		return exitOK
	}

	switch in.HookEventName {
	case "UserPromptSubmit":
		roles, skillInvoked, rolesErr := requiredRolesInPrompt(root, in.Prompt)
		if rolesErr != nil {
			if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
				Event:  "armed",
				TurnID: in.TurnID,
				Role:   agentDispatchMetadataRole,
			}); err != nil {
				return stopDispatchBlock(h, "DevRites could not record the invalid agent-role contract: "+err.Error(), stdout, stderr)
			}
			return stopDispatchBlock(h, "DevRites could not read the required agent roles: "+rolesErr.Error(), stdout, stderr)
		}
		if skillInvoked {
			if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
				Event:  "armed",
				TurnID: in.TurnID,
				Role:   agentDispatchSkillGuard,
			}); err != nil {
				return stopDispatchBlock(h, "DevRites could not arm the skill dispatch guard: "+err.Error(), stdout, stderr)
			}
		}
		for _, role := range roles {
			if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
				Event:  "armed",
				TurnID: in.TurnID,
				Role:   role,
			}); err != nil {
				return stopDispatchBlock(h, "DevRites could not arm the required agent dispatch: "+err.Error(), stdout, stderr)
			}
		}
		events, stateErr := readAgentDispatchEvents(root, in.SessionID)
		if stateErr == nil {
			armed, attempts := dispatchTurnState(events, in.TurnID)
			if err := refreshPendingWrightBoundary(root, in, armed, attempts); err != nil {
				debugf(stderr, "agent-dispatch pending wright boundary: %v", err)
			}
		}
		if len(roles) > 0 {
			instructions := make([]string, 0, len(roles))
			for _, role := range roles {
				instructions = append(instructions, dispatchInstruction(role))
			}
			fmt.Fprintf(
				stdout,
				"MANDATORY DISPATCH THIS TURN — At the skill's dispatch step, execute every required child before finishing or claiming completion; a success phrase is not evidence.\n%s",
				strings.Join(instructions, "\n"),
			)
		}
		if skillInvoked {
			if len(roles) > 0 {
				fmt.Fprintln(stdout)
			}
			fmt.Fprint(stdout, conditionalDispatchInstruction())
		}
		return exitOK

	case "PreToolUse":
		return hookAgentDispatchPreTool(h, root, in, stdout, stderr)

	case "PostToolUse":
		events, err := readAgentDispatchEvents(root, in.SessionID)
		if err != nil {
			debugf(stderr, "agent-dispatch post-tool state: %v", err)
			return exitOK
		}
		armed, attempts := dispatchTurnState(events, in.TurnID)
		if reason, ok := dispatchWaiverReason(agentDispatchCommand(in)); ok &&
			strings.Contains(in.ToolResponse, "dispatch-waive: accepted "+reason) {
			if err := recordDispatchWaiver(root, in, reason, armed, attempts); err != nil {
				debugf(stderr, "agent-dispatch post-tool waiver: %v", err)
			}
		}
		if isWaitTool(in.ToolName) {
			if waitToolSucceeded(in.ToolResponse) {
				if err := recordCompletedWaits(root, in, attempts); err != nil {
					debugf(stderr, "agent-dispatch post-tool wait: %v", err)
				}
			} else if err := recordCancelledWaits(root, in, attempts); err != nil {
				debugf(stderr, "agent-dispatch post-tool wait cancellation: %v", err)
			}
		}
		if err := refreshPendingWrightBoundary(root, in, armed, attempts); err != nil {
			debugf(stderr, "agent-dispatch post-tool wright boundary: %v", err)
		}
		return exitOK

	case "SubagentStop":
		if in.AgentID == "" {
			return exitOK
		}
		attempt, found, lookupErr := boundDispatchAttempt(root, in.SessionID, in.AgentID)
		if lookupErr != nil {
			return stopDispatchBlock(h, "DevRites could not identify the stopped subagent: "+lookupErr.Error(), stdout, stderr)
		}
		if !found {
			return exitOK
		}
		resultSum := ""
		if strings.TrimSpace(in.LastAssistantMessage) != "" {
			sum := sha256.Sum256([]byte(in.LastAssistantMessage))
			resultSum = hex.EncodeToString(sum[:])
		}
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:        "stopped",
			TurnID:       attempt.TurnID,
			AgentID:      in.AgentID,
			AgentType:    attempt.AgentType,
			ResultSHA256: resultSum,
		}); err != nil {
			return stopDispatchBlock(h, "DevRites could not record the subagent result: "+err.Error(), stdout, stderr)
		}
		if active, activeErr := activeWaitIntent(root, in.SessionID, attempt.TurnID, in.AgentID, time.Now()); activeErr != nil {
			return stopDispatchBlock(h, "DevRites could not verify the awaited subagent result: "+activeErr.Error(), stdout, stderr)
		} else if active {
			if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
				Event:   "waited",
				TurnID:  attempt.TurnID,
				AgentID: in.AgentID,
			}); err != nil {
				return stopDispatchBlock(h, "DevRites could not record the awaited subagent result: "+err.Error(), stdout, stderr)
			}
		}
		return exitOK

	case "Stop":
		events, err := readAgentDispatchEvents(root, in.SessionID)
		if err != nil {
			reason := "DevRites could not verify the required agent dispatch: " + err.Error()
			if in.StopHookActive {
				return stopDispatchFailure(reason, stdout, stderr)
			}
			return stopDispatchBlock(h, reason, stdout, stderr)
		}
		armed, attempts := dispatchTurnState(events, in.TurnID)
		_, guarded := armed[agentDispatchSkillGuard]
		if reason := incompleteDispatchReason(armed, attempts); reason != "" || guarded && len(attempts) == 0 {
			durable, durableErr := durableCodexV2DispatchAttempts(
				root, in.SessionID, in.TurnID, armed,
			)
			if durableErr != nil {
				reason := "DevRites could not verify the Codex V2 agent dispatch: " + durableErr.Error()
				if in.StopHookActive {
					return stopDispatchFailure(reason, stdout, stderr)
				}
				return stopDispatchBlock(h, reason, stdout, stderr)
			}
			attempts = append(attempts, durable...)
		}
		if reason := incompleteDispatchReason(armed, attempts); reason != "" {
			if in.StopHookActive {
				return stopDispatchFailure(reason, stdout, stderr)
			}
			return stopDispatchBlock(h, reason, stdout, stderr)
		}
		return exitOK
	}
	return exitOK
}

func hookAgentDispatchPreTool(h harness.Harness, root string, in agentDispatchHookInput, stdout, stderr io.Writer) int {
	events, err := readAgentDispatchEvents(root, in.SessionID)
	if err != nil {
		return preToolDeny(h, "DevRites could not verify agent dispatch state: "+err.Error(), stdout, stderr)
	}
	armed, attempts := dispatchTurnState(events, in.TurnID)

	command := agentDispatchCommand(in)
	if isShellTool(in.ToolName) && strings.Contains(command, "dispatch-waive") {
		reason, ok := dispatchWaiverReason(command)
		if !ok || !lib.ValidDispatchWaiver(reason) {
			return preToolDeny(h, "DevRites dispatch waiver must be one exact standalone engine command with an approved reason.", stdout, stderr)
		}
		if len(attempts) != 0 {
			return preToolDeny(h, "DevRites dispatch waiver is only valid before any specialist spawn attempt.", stdout, stderr)
		}
		required := 0
		for role := range armed {
			if strings.HasPrefix(role, "devrites-") && role != agentDispatchSkillGuard && role != agentDispatchMetadataRole {
				required++
			}
		}
		if required == 0 {
			return preToolDeny(h, "DevRites dispatch waiver requires an armed mandatory role in this turn.", stdout, stderr)
		}
		return exitOK
	}

	if isAgentDispatchTool(in.ToolName) {
		agentType := in.ToolInput.AgentType
		if agentType == "" {
			roles := explicitRolesInText(in.ToolInput.Message + "\n" + in.ToolInput.TaskName)
			if len(roles) == 1 {
				return preToolDeny(h, "DevRites conditional dispatch requires the exact named specialist. "+dispatchInstruction(roles[0]), stdout, stderr)
			}
			if reason := incompleteDispatchReason(armed, attempts); reason != "" {
				return preToolDeny(h, reason, stdout, stderr)
			}
			if _, guarded := armed[agentDispatchSkillGuard]; guarded {
				return preToolDeny(h, conditionalDispatchInstruction(), stdout, stderr)
			}
			return exitOK
		}
		role := ""
		if strings.HasPrefix(agentType, "devrites-") {
			role = agentType
		} else {
			roles := explicitRolesInText(in.ToolInput.Message)
			if len(roles) == 1 {
				role = roles[0]
			}
		}
		if role == "" {
			if agentType == "explorer" || agentType == "worker" {
				if reason := incompleteDispatchReason(armed, attempts); reason != "" {
					return preToolDeny(h, reason, stdout, stderr)
				}
				if _, guarded := armed[agentDispatchSkillGuard]; !guarded {
					return exitOK // Unrelated generic subagent in a DevRites repository.
				}
				return preToolDeny(h, "DevRites generic dispatch must name exactly one .codex/agents/devrites-<role>.toml contract.", stdout, stderr)
			}
			if reason := incompleteDispatchReason(armed, attempts); reason != "" {
				return preToolDeny(h, reason, stdout, stderr)
			}
			if _, guarded := armed[agentDispatchSkillGuard]; guarded {
				return preToolDeny(h, conditionalDispatchInstruction(), stdout, stderr)
			}
			return exitOK
		}
		instructions, contractErr := agentRoleDeveloperInstructions(root, role)
		if contractErr != nil {
			return preToolDeny(h, "DevRites role contract .codex/agents/"+role+".toml cannot be loaded: "+contractErr.Error()+".", stdout, stderr)
		}
		if in.ToolInput.ForkTurns != "none" {
			return preToolDeny(h, "DevRites agents require fork_turns=\"none\" so the child starts in the role contract rather than inheriting the root type.", stdout, stderr)
		}
		if agentType == "worker" && role != "devrites-slice-wright" {
			return preToolDeny(h, "Generic worker is reserved for devrites-slice-wright; use generic explorer for read-only DevRites roles.", stdout, stderr)
		}
		if agentType == "explorer" && role == "devrites-slice-wright" {
			return preToolDeny(h, "devrites-slice-wright requires the generic worker identity and exact wright allowlist.", stdout, stderr)
		}
		if agentType != "" && agentType != role && agentType != expectedAgentType(role) {
			return preToolDeny(h, fmt.Sprintf("DevRites role %s requires agent_type=%s or its exposed named role. %s", role, expectedAgentType(role), dispatchInstruction(role)), stdout, stderr)
		}
		if in.ToolUseID == "" {
			return preToolDeny(h, "DevRites cannot bind a spawn without tool_use_id.", stdout, stderr)
		}
		windowID := currentReconcileWindowID()
		if role == "devrites-slice-wright" && windowID == "" {
			return preToolDeny(h, "devrites-slice-wright requires an active reconcile snapshot before spawn_agent.", stdout, stderr)
		}
		if role == "devrites-slice-wright" {
			workspaceRoot, slug, _, ok := resolveWorkspace()
			if !ok || !sameResolvedPath(workspaceRoot, root) {
				return preToolDeny(h, "DevRites could not resolve the active wright reconcile window before spawn_agent.", stdout, stderr)
			}
			if err := lib.CaptureReconcileWrightBoundary(root, slug); err != nil {
				return preToolDeny(h, "DevRites could not capture the wright canonical boundary before spawn_agent: "+err.Error(), stdout, stderr)
			}
		}
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:     "pending",
			TurnID:    in.TurnID,
			ToolUseID: in.ToolUseID,
			AgentType: agentType,
			Role:      role,
			WindowID:  windowID,
		}); err != nil {
			return preToolDeny(h, "DevRites could not record the spawn attempt: "+err.Error(), stdout, stderr)
		}
		if agentType != role {
			message := fmt.Sprintf(
				"DevRites bound developer_instructions for `%s`:\n\n%s\n\n## Unchanged parent packet\n%s",
				role, instructions, in.ToolInput.Message,
			)
			return preToolRewrite(h, in.ToolInputRaw, message, stdout, stderr)
		}
		return exitOK
	}

	switch toolBaseName(in.ToolName) {
	case "wait", "wait_agent":
		if len(armed) == 0 {
			return exitOK
		}
		var started []*agentDispatchAttempt
		for _, attempt := range attempts {
			if attempt.Started {
				started = append(started, attempt)
			}
		}
		if len(started) == 0 {
			reason := incompleteDispatchReason(armed, attempts)
			if reason == "" {
				reason = conditionalDispatchInstruction()
			}
			return preToolDeny(h, reason, stdout, stderr)
		}
		targets := waitTargets(in)
		if len(targets) > 0 {
			known := map[string]*agentDispatchAttempt{}
			for _, attempt := range started {
				known[attempt.AgentID] = attempt
			}
			for _, agentID := range targets {
				if known[agentID] == nil {
					return preToolDeny(h, "DevRites wait target is not a confirmed child from this turn.", stdout, stderr)
				}
			}
		}
		if err := recordWaitIntents(root, in, started); err != nil {
			return preToolDeny(h, "DevRites could not record the wait intent: "+err.Error(), stdout, stderr)
		}
		return exitOK
	}

	if isShellTool(in.ToolName) && reconcileCloseRe.MatchString(command) {
		windowID := currentReconcileWindowID()
		if windowID == "" {
			return exitOK
		}
		if !roleSatisfied("devrites-slice-wright", windowID, attempts) {
			durable, durableErr := durableCodexV2DispatchAttempts(
				root, in.SessionID, in.TurnID, armed,
			)
			if durableErr != nil {
				return preToolDeny(h, "DevRites could not verify the Codex V2 wright result: "+durableErr.Error(), stdout, stderr)
			}
			attempts = append(attempts, durable...)
			if !roleSatisfied("devrites-slice-wright", windowID, attempts) {
				return preToolDeny(h, "DevRites reconcile close requires a confirmed, awaited devrites-slice-wright result bound to the active reconcile snapshot.", stdout, stderr)
			}
			for _, attempt := range durable {
				if attempt.Role != "devrites-slice-wright" || attempt.WindowID != windowID ||
					!dispatchAttemptComplete(attempt) {
					continue
				}
				if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
					Event:        "verified",
					TurnID:       in.TurnID,
					AgentID:      attempt.AgentID,
					AgentType:    attempt.AgentType,
					Role:         attempt.Role,
					WindowID:     attempt.WindowID,
					ResultSHA256: attempt.ResultSHA256,
				}); err != nil {
					return preToolDeny(h, "DevRites could not persist the verified Codex V2 wright result: "+err.Error(), stdout, stderr)
				}
			}
		}
	}
	return exitOK
}

func isWaitTool(toolName string) bool {
	switch toolBaseName(toolName) {
	case "wait", "wait_agent":
		return true
	default:
		return false
	}
}

func agentDispatchCommand(in agentDispatchHookInput) string {
	if in.ToolInput.Command != "" {
		return in.ToolInput.Command
	}
	return in.ToolInput.Cmd
}

func dispatchWaiverReason(command string) (string, bool) {
	match := dispatchWaiveRe.FindStringSubmatch(command)
	if len(match) != 2 {
		return "", false
	}
	return match[1], true
}

func recordDispatchWaiver(
	root string,
	in agentDispatchHookInput,
	reason string,
	armed map[string]struct{},
	attempts []*agentDispatchAttempt,
) error {
	if len(attempts) != 0 || !lib.ValidDispatchWaiver(reason) {
		return fmt.Errorf("waiver is not valid after a spawn attempt")
	}
	recorded := 0
	for role := range armed {
		if !strings.HasPrefix(role, "devrites-") || role == agentDispatchSkillGuard || role == agentDispatchMetadataRole {
			continue
		}
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:  "waived",
			TurnID: in.TurnID,
			Role:   role,
			Reason: reason,
		}); err != nil {
			return err
		}
		recorded++
	}
	if recorded == 0 {
		return fmt.Errorf("no armed mandatory role to waive")
	}
	return nil
}

func waitTargets(in agentDispatchHookInput) []string {
	if len(in.ToolInput.ReceiverThreadIDs) > 0 {
		return in.ToolInput.ReceiverThreadIDs
	}
	if len(in.ToolInput.IDs) > 0 {
		return in.ToolInput.IDs
	}
	return in.ToolInput.AgentIDs
}

func waitToolSucceeded(response string) bool {
	response = strings.ToLower(strings.TrimSpace(response))
	if response == "" {
		return false
	}
	for _, failure := range []string{
		`"timed_out":true`,
		`"timed_out": true`,
		"timed out",
		"timeout summary",
		"no activity",
	} {
		if strings.Contains(response, failure) {
			return false
		}
	}
	return true
}

func waitTargetIDs(in agentDispatchHookInput, attempts []*agentDispatchAttempt) []string {
	targets := waitTargets(in)
	if len(targets) > 0 {
		return targets
	}
	targets = make([]string, 0, len(attempts))
	for _, attempt := range attempts {
		if attempt.Started && attempt.AgentID != "" {
			targets = append(targets, attempt.AgentID)
		}
	}
	return targets
}

func agentWaitTimeout(in agentDispatchHookInput) time.Duration {
	const defaultTimeout = 30 * time.Second
	const maxTimeout = time.Hour
	if in.ToolInput.TimeoutMS <= 0 {
		return defaultTimeout
	}
	timeout := time.Duration(in.ToolInput.TimeoutMS) * time.Millisecond
	if timeout > maxTimeout {
		return maxTimeout
	}
	return timeout
}

func recordWaitIntents(root string, in agentDispatchHookInput, attempts []*agentDispatchAttempt) error {
	timeoutMS := strconv.FormatInt(agentWaitTimeout(in).Milliseconds(), 10)
	for _, agentID := range waitTargetIDs(in, attempts) {
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:   "waiting",
			TurnID:  in.TurnID,
			AgentID: agentID,
			Reason:  timeoutMS,
		}); err != nil {
			return err
		}
	}
	return nil
}

func recordCancelledWaits(root string, in agentDispatchHookInput, attempts []*agentDispatchAttempt) error {
	for _, agentID := range waitTargetIDs(in, attempts) {
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:   "wait-cancelled",
			TurnID:  in.TurnID,
			AgentID: agentID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func activeWaitIntent(root, sessionID, turnID, agentID string, now time.Time) (bool, error) {
	events, err := readAgentDispatchEvents(root, sessionID)
	if err != nil {
		return false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.TurnID != turnID || event.AgentID != agentID {
			continue
		}
		switch event.Event {
		case "waited", "wait-cancelled":
			return false, nil
		case "waiting":
			timeoutMS, err := strconv.ParseInt(event.Reason, 10, 64)
			if err != nil || timeoutMS <= 0 {
				return false, nil
			}
			deadline := time.Unix(0, event.AtUnixNano).Add(time.Duration(timeoutMS) * time.Millisecond)
			return !now.After(deadline), nil
		}
	}
	return false, nil
}

func recordCompletedWaits(root string, in agentDispatchHookInput, attempts []*agentDispatchAttempt) error {
	targetSet := map[string]struct{}{}
	for _, agentID := range waitTargets(in) {
		targetSet[agentID] = struct{}{}
	}
	for _, attempt := range attempts {
		if !attempt.Started || attempt.Waited {
			continue
		}
		if len(targetSet) > 0 {
			if _, ok := targetSet[attempt.AgentID]; !ok {
				continue
			}
		}
		if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
			Event:   "waited",
			TurnID:  in.TurnID,
			AgentID: attempt.AgentID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func bindAgentDispatchStart(root string, in agentDispatchHookInput) (string, bool, error) {
	events, err := readAgentDispatchEvents(root, in.SessionID)
	if err != nil {
		return "", false, err
	}
	started := map[string]struct{}{}
	for _, event := range events {
		if event.Event == "started" {
			started[event.ToolUseID] = struct{}{}
		}
	}
	var candidates []agentDispatchEvent
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Event != "pending" || event.AgentType != in.AgentType {
			continue
		}
		if _, ok := started[event.ToolUseID]; ok {
			continue
		}
		if strings.HasPrefix(in.AgentType, "devrites-") && event.Role != in.AgentType {
			continue
		}
		candidates = append(candidates, event)
	}
	if len(candidates) == 0 {
		return "", false, nil
	}
	if len(candidates) != 1 {
		return "", false, fmt.Errorf("ambiguous subagent start: %d pending %s dispatches lack a parent tool-use binding", len(candidates), in.AgentType)
	}
	event := candidates[0]
	if err := appendAgentDispatchEvent(root, in.SessionID, agentDispatchEvent{
		Event:     "started",
		TurnID:    event.TurnID,
		ToolUseID: event.ToolUseID,
		AgentID:   in.AgentID,
		AgentType: in.AgentType,
		Role:      event.Role,
		WindowID:  event.WindowID,
	}); err != nil {
		return "", false, err
	}
	return event.Role, true, nil
}

type boundAgentDispatchAttempt struct {
	TurnID    string
	AgentType string
	Role      string
}

func boundDispatchAttempt(root, sessionID, agentID string) (boundAgentDispatchAttempt, bool, error) {
	events, err := readAgentDispatchEvents(root, sessionID)
	if err != nil {
		return boundAgentDispatchAttempt{}, false, err
	}
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event.Event == "started" && event.AgentID == agentID {
			return boundAgentDispatchAttempt{
				TurnID:    event.TurnID,
				AgentType: event.AgentType,
				Role:      event.Role,
			}, true, nil
		}
	}
	return boundAgentDispatchAttempt{}, false, nil
}

type devritesAgentGuardBinding struct {
	Kind devritesAgentKind
	Role string
}

func devritesAgentBindingForGuard(h harness.Harness, in harness.GuardInput) devritesAgentGuardBinding {
	kind := devritesAgent(in.AgentType)
	binding := devritesAgentGuardBinding{Kind: kind, Role: declaredDevritesRole(in.AgentType)}
	generic := in.AgentType == "" || in.AgentType == "explorer" || in.AgentType == "worker"
	if h != harness.Codex || !generic ||
		os.Getenv("DEVRITES_CODEX_GENERIC_AGENT_COMPAT") != "1" ||
		in.SessionID == "" || in.AgentID == "" {
		return binding
	}
	root, ok := agentDispatchRoot()
	if !ok {
		return devritesAgentGuardBinding{Kind: devritesAgentInvalid}
	}
	attempt, found, err := boundDispatchAttempt(root, in.SessionID, in.AgentID)
	if err != nil {
		return devritesAgentGuardBinding{Kind: devritesAgentInvalid}
	}
	if !found {
		return binding
	}
	if attempt.Role == "devrites-slice-wright" {
		return devritesAgentGuardBinding{Kind: devritesAgentGenericWright, Role: attempt.Role}
	}
	if strings.HasPrefix(attempt.Role, "devrites-") {
		return devritesAgentGuardBinding{Kind: devritesAgentReadonly, Role: attempt.Role}
	}
	return devritesAgentGuardBinding{Kind: devritesAgentInvalid}
}
