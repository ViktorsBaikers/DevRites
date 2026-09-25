package records

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

func set(vals ...string) map[string]bool {
	m := map[string]bool{}
	for _, v := range vals {
		m[v] = true
	}
	return m
}

var enums = []struct {
	field   string
	allowed map[string]bool
}{
	{"mode", set("full", "pr", "branch")},
	{"phase", set("PREFLIGHT", "AUDITING", "RECONCILING", "AWAITING_APPROVAL", "REMEDIATING", "VERIFYING",
		"SCORING", "FINAL_CHALLENGE", "COMPLETE", "PAUSED", "STOPPED", "BLOCKED")},
	{"readiness_verdict", set("NOT_ASSESSED", "NOT_ASSESSABLE", "NOT_READY", "MEETS_TARGETS",
		"MEETS_TARGETS_WITH_APPROVED_DEGRADATION")},
	{"execution_outcome", set("RUNNING", "AUDIT_COMPLETE", "AUDIT_INCOMPLETE", "AWAITING_APPROVAL",
		"READY_FOR_USER_REVIEW", "COMPLETED_WITH_APPROVED_DEGRADATION", "STOPPED_BY_USER",
		"BLOCKED_NEEDS_USER", "BLOCKED_ENVIRONMENT", "BLOCKED_PARALLEL_CAPABILITY",
		"BUDGET_EXHAUSTED", "INCONCLUSIVE")},
}

var (
	kinds      = set("defect", "hardening", "maintainability", "opportunity")
	origins    = set("introduced", "exposed-existing", "pre-existing")
	severities = set("critical", "high", "medium", "low", "informational")
	moves      = map[string]map[string]bool{ // allowed finding status transitions
		"candidate":        set("needs-validation", "confirmed", "rejected"),
		"needs-validation": set("confirmed", "rejected"),
		"confirmed":        set("approved-for-fix", "deferred", "accepted-risk", "rejected"),
		"approved-for-fix": set("in-progress", "deferred"),
		"in-progress":      set("fixed-unverified", "approved-for-fix"),
		"fixed-unverified": set("verified", "in-progress"),
		"verified":         set("confirmed"),
		"deferred":         set("approved-for-fix", "confirmed"),
		"accepted-risk":    set("approved-for-fix", "confirmed"),
		"rejected":         set("candidate"),
	}
	severityOK   = set("confirmed", "approved-for-fix", "in-progress", "fixed-unverified", "verified", "deferred", "accepted-risk")
	attemptState = set("dispatched", "running", "returned", "admitted", "rejected", "cancelled", "timed-out", "unknown")
	live         = set("dispatched", "running", "unknown", "timed-out") // writer not proven stopped
	writers      = set("implementer")
	taskOutcomes = set("completed", "blocked", "deferred", "failed", "not-started", "superseded")
	required     = []string{"run.json", "coverage.json", "findings.json", "evidence.json", "dispatch.json",
		"approval.json", "cycles.json"}
	channels = set("conversation", "host-structured-action")
)

// Repr renders a decoded JSON value the way Python's repr() does, so rule
// messages read the same as the original recipe's.
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case string:
		return quote(x)
	case json.Number:
		return x.String()
	case []string:
		parts := make([]string, len(x))
		for i, s := range x {
			parts[i] = quote(s)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []any:
		parts := make([]string, len(x))
		for i, e := range x {
			parts[i] = Repr(e)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, len(keys))
		for i, k := range keys {
			parts[i] = quote(k) + ": " + Repr(x[k])
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(v)
}

func quote(s string) string {
	q := '\''
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		q = '"'
	}
	var b strings.Builder
	b.WriteRune(q)
	for _, r := range s {
		switch {
		case r == '\\' || r == q:
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\x%02x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteRune(q)
	return b.String()
}

// PyStr is Python's str(): strings as-is, anything else as Repr.
func PyStr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return Repr(v)
}

// join mirrors os.path.join: an absolute part replaces everything before it.
func join(base string, parts ...string) string {
	for _, p := range parts {
		switch {
		case strings.HasPrefix(p, "/"):
			base = p
		case base == "" || strings.HasSuffix(base, "/"):
			base += p
		default:
			base += "/" + p
		}
	}
	return base
}

func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	return ovio.Resolve(abs)
}

func inside(base, rel string) bool {
	root := realpath(base)
	return strings.HasPrefix(realpath(join(root, rel)), root+string(os.PathSeparator))
}

func isFile(p string) bool {
	fi, err := os.Stat(p) // #nosec G703 -- stat only; path is inside the operator-selected run area
	return err == nil && fi.Mode().IsRegular()
}

func isDir(p string) bool {
	fi, err := os.Stat(p) // #nosec G703 -- stat only; path is inside the operator-selected run area
	return err == nil && fi.IsDir()
}

func sha(p string) (string, error) { return ovio.SHA256File(p) }

// keyError mimics Python's KeyError text for a missing required field.
func keyError(k string) error { return fmt.Errorf("'%s'", k) }

// BadPath returns why p is not an exact repository-relative file path, or ""
// when it is one.
func BadPath(p any, repo string) string {
	s, ok := p.(string)
	if !ok || s == "" || strings.HasSuffix(s, "/") || path.IsAbs(s) || filepath.IsAbs(s) || strings.ContainsAny(s, "*?") {
		return "not an exact relative file path"
	}
	if path.Clean(s) != s || strings.Split(s, "/")[0] == ".." {
		return "not normalized or escapes the repository"
	}
	for _, part := range strings.Split(s, "/") {
		if part == ".git" {
			return "names a directory or git metadata"
		}
	}
	if isDir(join(repo, s)) {
		return "names a directory or git metadata"
	}
	if !inside(repo, s) {
		return "resolves outside the repository (symlink or traversal)"
	}
	return ""
}

// vset is a set of JSON values keyed by their Repr, keeping the value.
type vset map[string]any

func (s vset) add(v any) { s[Repr(v)] = v }

func (s vset) has(v any) bool {
	_, ok := s[Repr(v)]
	return ok
}

func setOf(vals []any) vset {
	s := vset{}
	for _, v := range vals {
		s.add(v)
	}
	return s
}

// sorted mirrors Python's sorted(set) for the string members the records use.
func (s vset) sorted() []any {
	out := make([]any, 0, len(s))
	for _, v := range s {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return PyStr(out[i]) < PyStr(out[j]) })
	return out
}

func uniq(items []any, key func(map[string]any) string, label string, add func(string, ...any)) map[string]bool {
	seen := map[string]bool{}
	for _, it := range items {
		k := key(ovio.Obj(it))
		if seen[k] {
			add("%s: duplicate id %s", label, k)
		}
		seen[k] = true
	}
	return seen
}

func idKey(m map[string]any) string { return Repr(m["id"]) }

func str(m map[string]any, k string) string { return ovio.Str(m[k]) }

func eq(a, b any) bool { return reflect.DeepEqual(a, b) }

// prefixEq reports whether list starts with prefix (Python list[:len(p)] == p).
func prefixEq(list, prefix []any) bool {
	if len(list) < len(prefix) {
		return false
	}
	for i := range prefix {
		if !eq(list[i], prefix[i]) {
			return false
		}
	}
	return true
}

func decode(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

func closure(tasks map[string]map[string]any, picked []any) vset {
	need, stack := vset{}, append([]any(nil), picked...)
	for len(stack) > 0 {
		t := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, d := range ovio.List(tasks[Repr(t)]["deps"]) {
			if !need.has(d) {
				need.add(d)
				stack = append(stack, d)
			}
		}
	}
	for _, p := range picked {
		delete(need, Repr(p))
	}
	return need
}

func cyclic(tasks map[string]map[string]any) bool {
	state := map[string]int{}
	var visit func(k string) bool
	visit = func(k string) bool {
		switch state[k] {
		case 1:
			return true
		case 2:
			return false
		}
		state[k] = 1
		hit := false
		for _, d := range ovio.List(tasks[k]["deps"]) {
			if _, ok := tasks[Repr(d)]; ok && visit(Repr(d)) {
				hit = true
				break
			}
		}
		state[k] = 2
		return hit
	}
	for k := range tasks {
		if visit(k) {
			return true
		}
	}
	return false
}

// Check validates one generation directory (optionally against the previous
// one) and returns rule violations. An error means an unreadable input.
func Check(run, genDir, prevDir string) ([]string, error) {
	var errs []string
	add := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	R := map[string]map[string]any{}
	for _, name := range required {
		p := join(genDir, name)
		if !isFile(p) {
			add("missing %s", name)
			continue
		}
		b, err := os.ReadFile(p) // #nosec G304 G703 -- fixed record name inside the generation directory
		if err != nil {
			return nil, err
		}
		v, err := decode(b)
		if err == nil {
			if m, ok := v.(map[string]any); ok {
				R[name] = m
				continue
			}
			err = fmt.Errorf("expected a JSON object")
		}
		add("%s: invalid JSON (%v)", name, err)
	}
	if len(errs) > 0 {
		return errs, nil
	}
	for _, name := range required {
		want := "overhaul." + strings.TrimSuffix(name, ".json") + "/1"
		if !eq(R[name]["schema"], want) {
			add("%s: schema must be %s", name, want)
		}
	}
	rj := R["run.json"]
	for _, e := range enums {
		if !e.allowed[str(rj, e.field)] {
			keys := make([]string, 0, len(e.allowed))
			for k := range e.allowed {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			add("run.json: %s=%s not in %s", e.field, Repr(rj[e.field]), Repr(keys))
		}
	}
	repo := str(rj, "repo_root")
	if !isDir(repo) {
		add("run.json: repo_root is not a directory")
	}
	ev := ovio.List(R["evidence.json"]["items"])
	evIDs := uniq(ev, idKey, "evidence", add)
	for _, it := range ev {
		e := ovio.Obj(it)
		p := str(e, "path")
		if p == "" || !inside(run, p) || !isFile(join(run, p)) {
			add("evidence %s: path missing or outside run dir", PyStr(e["id"]))
			continue
		}
		d, err := sha(join(run, p))
		if err != nil {
			return nil, err
		}
		if !eq(d, e["sha256"]) {
			add("evidence %s: sha256 missing or mismatched", PyStr(e["id"]))
		}
	}
	att := ovio.List(R["dispatch.json"]["attempts"])
	vr := map[string]bool{}
	for _, it := range att {
		x := ovio.Obj(it)
		if str(x, "role") == "verifier" && str(x, "status") == "admitted" {
			if r, ok := x["receipt"].(string); ok {
				vr[r] = true
			}
			vr[PyStr(x["task_id"])+"#"+PyStr(x["attempt_id"])] = true
		}
	}
	fs := ovio.List(R["findings.json"]["findings"])
	fIDs := uniq(fs, idKey, "finding", add)
	var fps []any
	for _, f := range fs {
		if ovio.Truthy(ovio.Obj(f)["fingerprint"]) {
			fps = append(fps, f)
		}
	}
	uniq(fps, func(m map[string]any) string { return Repr(m["fingerprint"]) }, "finding fingerprint", add)
	for _, it := range fs {
		f := ovio.Obj(it)
		fid, hist, status := PyStr(f["id"]), ovio.List(f["history"]), str(f, "status")
		if !kinds[str(f, "kind")] || !origins[str(f, "scope_origin")] {
			add("finding %s: bad kind or scope_origin", fid)
		}
		if len(hist) == 0 || str(ovio.Obj(hist[0]), "status") != "candidate" || !eq(ovio.Obj(hist[len(hist)-1])["status"], f["status"]) {
			add("finding %s: history must start at candidate and end at the status", fid)
		}
		for i := 1; i < len(hist); i++ {
			a, b := ovio.Obj(hist[i-1]), ovio.Obj(hist[i])
			if eq(b["status"], a["status"]) && ovio.Truthy(b["reason"]) {
				continue // re-assessment entry (severity or kind change)
			}
			if !moves[str(a, "status")][str(b, "status")] {
				add("finding %s: illegal transition %s -> %s", fid, PyStr(a["status"]), PyStr(b["status"]))
			}
		}
		if f["severity"] != nil && (!severities[str(f, "severity")] || !severityOK[status]) {
			add("finding %s: severity only for established impact", fid)
		}
		if set("confirmed", "verified", "rejected")[status] && !ovio.Truthy(f["evidence"]) {
			add("finding %s: %s needs evidence", fid, status)
		}
		if vb, ok := f["verified_by"].(string); status == "verified" && !(ok && vr[vb]) {
			add("finding %s: verified_by must name an admitted verifier attempt", fid)
		}
		if status == "rejected" && (len(hist) == 0 || !ovio.Truthy(ovio.Obj(hist[len(hist)-1])["reason"])) {
			add("finding %s: rejection needs a reason", fid)
		}
		if status == "needs-validation" && !ovio.Truthy(f["blockers"]) {
			add("finding %s: needs-validation must name the missing fact (blockers)", fid)
		}
		for _, l := range ovio.List(f["locations"]) {
			loc := ovio.Obj(l)
			lp := PyStr(loc["path"])
			var bad bool
			if ovio.Truthy(loc["external"]) {
				bad = path.IsAbs(lp) || filepath.IsAbs(lp) || strings.HasPrefix(lp, "~")
			} else {
				bad = BadPath(lp, repo) != ""
			}
			if bad {
				add("finding %s: location %s is not a repository-relative path or label", fid, Repr(lp))
			}
		}
		for _, ref := range ovio.List(f["evidence"]) {
			if !evIDs[Repr(ref)] {
				add("finding %s: unknown evidence %s", fid, PyStr(ref))
			}
		}
		for _, ref := range ovio.List(f["related"]) {
			if !fIDs[Repr(ref)] {
				add("finding %s: unknown related finding %s", fid, PyStr(ref))
			}
		}
	}
	var plan map[string]any
	tasks := map[string]map[string]any{}
	revs := ovio.Obj(rj["revisions"])
	pr := ovio.Obj(revs["plan"])
	hasPR := len(pr) > 0
	if hasPR && !eq(pr["file"], "revisions/plan-r"+PyStr(pr["rev"])+".json") {
		add("run.json: revisions.plan.file must be revisions/plan-r<rev>.json")
	} else if hasPR {
		var err error
		if plan, err = ovio.LoadObject(join(run, str(pr, "file"))); err != nil {
			return nil, err
		}
		if !eq(plan["rev"], pr["rev"]) {
			add("plan revision number mismatch")
		}
		var controls vset
		rub := ovio.Obj(plan["rubric"])
		if rf := str(rub, "file"); rf == "" {
			add("plan: rubric digest mismatch")
		} else if d, err := sha(join(run, rf)); err != nil {
			return nil, err
		} else if !eq(d, rub["sha256"]) {
			add("plan: rubric digest mismatch")
		} else {
			doc, err := ovio.LoadObject(join(run, rf))
			if err != nil {
				return nil, err
			}
			controls = vset{}
			for _, c := range ovio.List(doc["controls"]) {
				controls.add(ovio.Obj(c)["id"])
			}
		}
		for _, r := range ovio.List(plan["pinned"]) {
			ref := ovio.Obj(r)
			if _, ok := ref["file"]; !ok {
				return nil, keyError("file")
			}
			d, err := sha(join(run, str(ref, "file")))
			if err != nil {
				return nil, err
			}
			if !eq(d, ref["sha256"]) {
				add("plan: pinned %s digest mismatch", PyStr(ref["file"]))
			}
		}
		tl := ovio.List(plan["tasks"])
		uniq(tl, idKey, "task", add)
		var order []any // first-insertion order, like a Python dict
		for _, it := range tl {
			t := ovio.Obj(it)
			if _, ok := tasks[Repr(t["id"])]; !ok {
				order = append(order, t["id"])
			}
			tasks[Repr(t["id"])] = t
		}
		for _, tid := range order {
			t, ts := tasks[Repr(tid)], PyStr(tid)
			for _, d := range ovio.List(t["deps"]) {
				if _, ok := tasks[Repr(d)]; !ok {
					add("task %s: unknown dependency %s", ts, PyStr(d))
				}
			}
			for _, ref := range ovio.List(t["findings"]) {
				if !fIDs[Repr(ref)] {
					add("task %s: unknown finding %s", ts, PyStr(ref))
				}
			}
			if controls != nil {
				for _, c := range ovio.List(t["controls"]) {
					if !controls.has(c) {
						add("task %s: unknown control ids", ts)
						break
					}
				}
			}
			for _, p := range ovio.List(t["allowed_paths"]) {
				if why := BadPath(p, repo); why != "" {
					add("task %s: allowed path %s %s", ts, Repr(p), why)
				}
			}
			if !ovio.Truthy(t["oracle"]) || !ovio.Truthy(t["verifier_role"]) || eq(t["verifier_role"], t["writer_role"]) {
				add("task %s: needs an oracle and a separate verifier", ts)
			}
		}
		if cyclic(tasks) {
			add("plan: task dependencies contain a cycle")
		}
	}
	rr, scp := ovio.Obj(revs["rubric"]), join(genDir, "scorecard.json")
	if isFile(scp) {
		if rf := str(rr, "file"); !(rf != "" && inside(run, rf) && isFile(join(run, rf))) {
			add("run.json: a scorecard needs revisions.rubric naming its rubric file")
		} else if ovio.Truthy(rr["sha256"]) {
			d, err := sha(join(run, rf))
			if err != nil {
				return nil, err
			}
			if !eq(d, rr["sha256"]) {
				add("run.json: revisions.rubric digest mismatch")
			}
		}
		sc, err := ovio.LoadObject(scp)
		if err != nil {
			return nil, err
		}
		if v := sc["readiness_verdict"]; v != nil && !eq(v, rj["readiness_verdict"]) {
			add("run.json: readiness_verdict differs from scorecard.json")
		}
	}
	ap := ovio.List(R["approval.json"]["events"])
	uniq(ap, idKey, "approval", add)
	stopAt := -1
	for i, it := range ap {
		if k := str(ovio.Obj(it), "kind"); k == "stop" || k == "revoke" {
			stopAt = i
		}
	}
	var active, approvals []map[string]any
	granted := map[string]vset{}
	for i, it := range ap {
		a := ovio.Obj(it)
		if str(a, "kind") != "approve" {
			continue
		}
		approvals = append(approvals, a)
		granted[Repr(a["id"])] = setOf(ovio.List(a["tasks"]))
		if str(a, "status") == "active" {
			if i < stopAt {
				add("approval %s: still active after a later stop/revoke", PyStr(a["id"]))
			} else if i > stopAt {
				active = append(active, a)
			}
		}
	}
	anyWriter := false
	for _, it := range att {
		anyWriter = anyWriter || writers[str(ovio.Obj(it), "role")]
	}
	if len(active) > 1 {
		add("approval: more than one active approval")
	}
	if !hasPR && (len(granted) > 0 || anyWriter) {
		add("run.json: approvals and writers need revisions.plan")
	}
	for _, it := range ap {
		a := ovio.Obj(it)
		kind, aid := str(a, "kind"), PyStr(a["id"])
		if (kind == "approve" || kind == "accept-degradation") && (!ovio.Truthy(a["quote"]) || !channels[str(a, "channel")]) {
			add("approval %s: needs the user's verbatim decision and a trusted channel", aid)
		}
		if kind != "approve" {
			continue
		}
		pf := join(run, "revisions", "plan-r"+PyStr(a["plan_rev"])+".json")
		ok := isFile(pf)
		if ok {
			d, err := sha(pf)
			if err != nil {
				return nil, err
			}
			ok = eq(d, a["plan_digest"])
		}
		if !ok {
			add("approval %s: plan digest does not match plan revision %s", aid, PyStr(a["plan_rev"]))
		}
		if str(a, "status") == "active" && plan != nil {
			if !eq(a["plan_rev"], plan["rev"]) {
				add("approval %s: approves a stale plan revision", aid)
			}
			picked := ovio.List(a["tasks"])
			missing, unknown := closure(tasks, picked), vset{}
			for _, t := range picked {
				if _, ok := tasks[Repr(t)]; !ok {
					unknown.add(t)
				}
			}
			if len(unknown) > 0 {
				add("approval %s: unknown tasks %s", aid, Repr(unknown.sorted()))
			}
			if len(missing) > 0 {
				add("approval %s: selected tasks miss prerequisites %s", aid, Repr(missing.sorted()))
			}
		}
	}
	assessmentOnly := ovio.Truthy(rj["assessment_only"])
	if assessmentOnly && len(active) > 0 {
		add("assessment-only run carries an active approval")
	}
	uniq(att, func(x map[string]any) string {
		return "(" + Repr(x["task_id"]) + ", " + Repr(x["attempt_id"]) + ")"
	}, "attempt", add)
	approved := vset{}
	if len(active) > 0 {
		approved = setOf(ovio.List(active[0]["tasks"]))
	}
	liveWrites := map[string]string{}
	var unsettled []any
	for _, it := range att {
		x := ovio.Obj(it)
		xid, status := PyStr(x["task_id"])+"#"+PyStr(x["attempt_id"]), str(x, "status")
		if !attemptState[status] {
			add("attempt %s: bad status", xid)
		}
		if set("returned", "admitted", "rejected")[status] && !(str(x, "receipt") != "" && isFile(join(run, str(x, "receipt")))) {
			add("attempt %s: receipt file missing", xid)
		}
		if set("dispatched", "running", "returned")[status] {
			unsettled = append(unsettled, xid)
		}
		if !writers[str(x, "role")] {
			continue
		}
		if assessmentOnly {
			add("attempt %s: writer dispatched in an assessment-only run", xid)
		}
		if (status == "dispatched" || status == "running") && !approved.has(x["task_id"]) {
			add("attempt %s: writer task is not in the active approval", xid)
		}
		if !granted[Repr(x["authorized_by"])].has(x["task_id"]) {
			add("attempt %s: authorized_by does not name an approval of this task", xid)
		}
		allowed := setOf(ovio.List(tasks[Repr(x["task_id"])]["allowed_paths"]))
		extra := vset{}
		for _, p := range ovio.List(x["changed_paths"]) {
			if !allowed.has(p) {
				extra.add(p)
			}
		}
		if status == "admitted" && len(extra) > 0 {
			add("attempt %s: admitted changes outside allowed paths %s", xid, Repr(extra.sorted()))
		}
		if live[status] {
			for _, p := range ovio.List(x["write_paths"]) {
				k := strings.ToLower(realpath(join(repo, ovio.Str(p))))
				if prev, ok := liveWrites[k]; ok {
					add("single writer: %s and %s both hold %s", prev, xid, PyStr(p))
				}
				liveWrites[k] = xid
			}
		}
	}
	outcome := str(rj, "execution_outcome")
	if outcome != "RUNNING" && len(unsettled) > 0 {
		add("attempts must be admitted, rejected or cancelled before the run leaves RUNNING: %s", Repr(unsettled[:min(5, len(unsettled))]))
	}
	if outcome != "RUNNING" && outcome != "AWAITING_APPROVAL" && len(approvals) > 0 && !assessmentOnly {
		outs := ovio.Obj(rj["task_outcomes"])
		var missing []any
		for _, t := range ovio.List(approvals[len(approvals)-1]["tasks"]) {
			var o any
			if s, ok := t.(string); ok {
				o = outs[s]
			}
			if !taskOutcomes[ovio.Str(o)] {
				missing = append(missing, t)
			}
		}
		if len(missing) > 0 {
			add("approved tasks without exactly one recorded outcome: %s", Repr(missing))
		}
	}
	cov := ovio.List(R["coverage.json"]["files"])
	uniq(cov, func(c map[string]any) string { return Repr(c["path"]) }, "coverage", add)
	for _, it := range cov {
		c := ovio.Obj(it)
		if !ovio.Truthy(c["eligible"]) && !ovio.Truthy(ovio.Get(c, "exclusion.reason")) {
			add("coverage %s: ineligible without an exclusion reason", PyStr(c["path"]))
		}
	}
	if prevDir != "" {
		prev := map[string]map[string]any{}
		for _, n := range []string{"cycles.json", "findings.json", "approval.json", "run.json", "coverage.json"} {
			m, err := ovio.LoadObject(join(prevDir, n))
			if err != nil {
				return nil, err
			}
			prev[n] = m
		}
		for _, k := range []string{"run_id", "repo_root", "mode", "assessment_only"} {
			if !eq(prev["run.json"][k], rj[k]) {
				add("run.json: %s changed; start a new run instead", k)
			}
		}
		if !prefixEq(ovio.List(R["cycles.json"]["cycles"]), ovio.List(prev["cycles.json"]["cycles"])) {
			add("cycles.json is append-only: an earlier cycle changed or disappeared")
		}
		cur := map[string]map[string]any{}
		for _, it := range fs {
			f := ovio.Obj(it)
			cur[Repr(f["id"])] = f
		}
		for _, it := range ovio.List(prev["findings.json"]["findings"]) {
			f := ovio.Obj(it)
			g, ok := cur[Repr(f["id"])]
			fh, isList := f["history"].([]any)
			gh := ovio.List(g["history"])
			if !ok || !isList || !prefixEq(gh, fh) {
				add("finding %s: removed or history rewritten", PyStr(f["id"]))
			} else if (!eq(g["severity"], f["severity"]) || !eq(g["kind"], f["kind"])) && len(gh) == len(fh) {
				add("finding %s: severity or kind changed without a history entry", PyStr(f["id"]))
			}
		}
		kept := vset{}
		for _, c := range cov {
			kept.add(ovio.Obj(c)["path"])
		}
		var gone []any
		for _, c := range ovio.List(prev["coverage.json"]["files"]) {
			if p := ovio.Obj(c)["path"]; !kept.has(p) {
				gone = append(gone, p)
			}
		}
		if len(gone) > 0 {
			add("coverage denominator shrank: %s (exclude with a reason instead)", Repr(gone[:min(5, len(gone))]))
		}
		oldap := ovio.List(prev["approval.json"]["events"])
		same := len(ap) >= len(oldap)
		for i := 0; same && i < len(oldap); i++ {
			same = eq(noStatus(ap[i]), noStatus(oldap[i]))
		}
		if !same {
			add("approval.json events are append-only")
		}
	}
	gen := strings.ReplaceAll(filepath.Base(strings.TrimRight(genDir, "/")), ".tmp", "")
	planStamp := "none"
	if hasPR {
		if _, ok := pr["file"]; !ok {
			return nil, keyError("file")
		}
		d, err := sha(join(run, str(pr, "file")))
		if err != nil {
			return nil, err
		}
		planStamp = "r" + PyStr(pr["rev"]) + ":" + d
	}
	stamp := fmt.Sprintf("overhaul-stamp run=%s generation=%s plan=%s", PyStr(rj["run_id"]), gen, planStamp)
	vdir := join(genDir, "views")
	if isDir(vdir) {
		entries, err := os.ReadDir(vdir)
		if err != nil {
			return nil, err
		}
		for _, v := range entries {
			b, err := os.ReadFile(join(vdir, v.Name())) // #nosec G703 -- view file listed from the generation's own views directory
			if err != nil {
				return nil, err
			}
			if !bytes.Contains(b, []byte(stamp)) {
				add("view %s: stamp does not match this generation (%s)", v.Name(), stamp)
			}
		}
	}
	return errs, nil
}

// noStatus is dict(a, status=None): the event with its mutable status blanked.
func noStatus(v any) map[string]any {
	m := map[string]any{}
	for k, x := range ovio.Obj(v) {
		m[k] = x
	}
	m["status"] = nil
	return m
}
