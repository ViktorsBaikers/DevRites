package lib

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// claimAnchorAge separates a claiming session's long-lived host processes from
// the per-command shells and launchers that start moments before `claim add`.
const claimAnchorAge = time.Minute

// claimProc names one process by pid and start time, so a reused pid cannot
// impersonate it.
type claimProc struct {
	PID   int    `json:"pid"`
	Start string `json:"start"`
}

type procInfo struct {
	ppid  int
	start string
	comm  string
}

// shells outlive the agent sessions they launch (a terminal pane, a
// multiplexer's login shell), so a shell never stands for the holder.
var shells = map[string]bool{
	"sh": true, "bash": true, "zsh": true, "fish": true, "dash": true, "ksh": true,
	"tcsh": true, "csh": true, "nu": true, "pwsh": true, "powershell": true, "cmd": true,
}

// processTable lists running processes keyed by pid, or nil when the host
// cannot say. Tests replace it.
var processTable = psProcessTable

func psProcessTable() map[int]procInfo {
	cmd := exec.Command("ps", "-A", "-o", "pid=,ppid=,lstart=,comm=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	table := map[int]procInfo{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 8 {
			continue
		}
		pid, perr := strconv.Atoi(f[0])
		ppid, pperr := strconv.Atoi(f[1])
		start, serr := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(f[2:7], " "), time.Local)
		if perr != nil || pperr != nil || serr != nil {
			continue
		}
		table[pid] = procInfo{ppid: ppid, start: start.UTC().Format(time.RFC3339), comm: strings.Join(f[7:], " ")}
	}
	if len(table) == 0 {
		return nil
	}
	return table
}

// lineage walks from pid up to, but not including, pid 1.
func lineage(table map[int]procInfo, pid int) []claimProc {
	var out []claimProc
	seen := map[int]bool{}
	for pid > 1 && !seen[pid] {
		seen[pid] = true
		p, ok := table[pid]
		if !ok {
			break
		}
		out = append(out, claimProc{PID: pid, Start: p.start})
		pid = p.ppid
	}
	return out
}

// holderProbe answers whether a claim's holding session has exited. It only
// ever proves exit on this host; anything it cannot see counts as alive.
type holderProbe struct {
	table map[int]procInfo
	host  string
	self  []claimProc
}

func newHolderProbe() holderProbe {
	table := processTable()
	host, _ := os.Hostname()
	return holderProbe{table: table, host: host, self: lineage(table, os.Getppid())}
}

// anchors returns the caller's ancestors that started more than
// claimAnchorAge before now: the session's host processes.
func (p holderProbe) anchors(now time.Time) []claimProc {
	var out []claimProc
	for _, proc := range p.self {
		start, err := time.Parse(time.RFC3339, proc.Start)
		if err == nil && now.Sub(start) > claimAnchorAge {
			out = append(out, proc)
		}
	}
	return out
}

// holder returns the nearest anchor that is not a shell: the agent session
// process that owns the claim. A shell or terminal multiplexer above it can
// outlive the session, so only the session's own exit proves release.
func (p holderProbe) holder(now time.Time) *claimProc {
	for _, proc := range p.anchors(now) {
		name := strings.TrimPrefix(filepath.Base(p.table[proc.PID].comm), "-")
		if name != "" && !shells[name] {
			return &proc
		}
	}
	return nil
}

// gone reports whether rec's holder is proven exited: the claim was taken on
// this host and its recorded holder session has exited, or, for a claim with
// no recorded holder, every anchor it does not share with the caller has.
func (p holderProbe) gone(rec *claimRecord) bool {
	if p.table == nil || rec.Host == "" || rec.Host != p.host {
		return false
	}
	if h := rec.Holder; h != nil {
		if slices.Contains(p.self, *h) {
			return false
		}
		proc, ok := p.table[h.PID]
		return !ok || proc.start != h.Start
	}
	unique := 0
	for _, a := range rec.Anchors {
		if slices.Contains(p.self, a) {
			continue
		}
		unique++
		if proc, ok := p.table[a.PID]; ok && proc.start == a.Start {
			return false
		}
	}
	return unique > 0
}

// withoutGone drops claims whose holder session has exited.
func (p holderProbe) withoutGone(claims []*claimRecord) []*claimRecord {
	return slices.DeleteFunc(claims, p.gone)
}
