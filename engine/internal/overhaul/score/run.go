// Package score implements the /overhaul score tool (calculation
// overhaul-score/1): a deterministic readiness index over a frozen control
// catalog, its results and its gate records. All arithmetic is exact
// (math/big.Rat) and thresholds compare at full precision. Exit 2 means
// invalid input: nothing may be treated as scored.
package score

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"os"
	"slices"
	"sort"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const calc = "overhaul-score/1"

const usage = `usage: overhaul score --rubric <r.json> --results <s.json> --gates <g.json> [--out <scorecard.json>]
       overhaul score compare --rubric <r.json> --baseline <a.json> --candidate <b.json>`

var (
	statuses     = set("PASS", "FAIL", "UNKNOWN", "BLOCKED", "NOT_RUN", "DEFERRED", "ACCEPTED_RISK")
	gateStatuses = set("PASS", "FAIL", "UNKNOWN", "NOT_APPLICABLE", "WAIVED_OPERATIONAL")
	auditGates   = []string{"G-MANDATORY", "G-NO-CRITICAL-HIGH", "G-NO-OPEN-SERIOUS-LEAD", "G-COVERAGE",
		"G-CATALOG-ADEQUACY", "G-LANE-RECEIPTS", "G-CONCURRENCY", "G-FRESHNESS", "G-REPORT-CONSISTENCY"}
	repairGates = []string{"G-APPROVAL", "G-TASKS-COMPLETE", "G-ORACLES", "G-NO-REGRESSION", "G-PERF-TARGETS",
		"G-PRESERVATION", "G-FINGERPRINTS", "G-OWNERSHIP", "G-FINAL-CHALLENGE"}
	mayBeNA = set("G-PERF-TARGETS", "G-LANE-RECEIPTS", "G-CONCURRENCY")
	// domainOrder is the fixed domain set in its documented order, which also
	// orders the per-domain threshold rows.
	domainOrder   = []string{"correctness", "security", "reliability", "performance", "tests", "architecture", "ux", "operations"}
	defaultPolicy = [][2]string{{"global_min", "9.7"}, {"lane_min", "9.7"}, {"domain_min", "9.0"}, {"lane_domain_min", "9.0"}}
	transitions   = map[[2]string]string{
		{"FAIL", "PASS"}:    "fixed-or-newly-proven",
		{"UNKNOWN", "PASS"}: "evidence-only",
		{"PASS", "FAIL"}:    "regression-or-new-failure",
		{"UNKNOWN", "FAIL"}: "newly-demonstrated-failure",
		{"PASS", "UNKNOWN"}: "evidence-invalidated",
	}
)

func set(xs ...string) map[string]bool {
	m := map[string]bool{}
	for _, x := range xs {
		m[x] = true
	}
	return m
}

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	out, err := run(args)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	fmt.Fprint(stdout, out)
	return 0
}

func run(args []string) (string, error) {
	if len(args) > 0 && args[0] == "compare" {
		o, err := flags(args[1:], []string{"--rubric", "--baseline", "--candidate"}, nil)
		if err != nil {
			return "", err
		}
		docs, _, err := load(o["--rubric"], o["--baseline"], o["--candidate"])
		if err != nil {
			return "", err
		}
		res, err := compare(docs[0], docs[1], docs[2])
		if err != nil {
			return "", err
		}
		b, err := ovio.MarshalIndent(res)
		return string(b), err
	}
	o, err := flags(args, []string{"--rubric", "--results", "--gates"}, []string{"--out"})
	if err != nil {
		return "", err
	}
	docs, digs, err := load(o["--rubric"], o["--results"], o["--gates"])
	if err != nil {
		return "", err
	}
	if d := docs[1]["rubric_digest"]; d != nil && d != any(digs[0]) {
		return "", fmt.Errorf("results were produced for a different rubric digest")
	}
	out, err := score(docs[0], docs[1], docs[2])
	if err != nil {
		return "", err
	}
	out["inputs"] = map[string]any{"rubric_sha256": digs[0], "results_sha256": digs[1], "gates_sha256": digs[2]}
	b, err := ovio.MarshalIndent(out)
	if err != nil {
		return "", err
	}
	path, ok := o["--out"]
	if !ok {
		return string(b), nil
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s -> %s\n", out["readiness_verdict"], path), nil
}

// flags parses "--name value" pairs; every required flag must be present and
// no other flag is accepted.
func flags(args, required, optional []string) (map[string]string, error) {
	if len(args)%2 != 0 {
		return nil, fmt.Errorf("every flag needs a value\n%s", usage)
	}
	o := map[string]string{}
	for i := 0; i < len(args); i += 2 {
		if !slices.Contains(required, args[i]) && !slices.Contains(optional, args[i]) {
			return nil, fmt.Errorf("unknown argument %q\n%s", args[i], usage)
		}
		o[args[i]] = args[i+1]
	}
	for _, f := range required {
		if _, ok := o[f]; !ok {
			return nil, fmt.Errorf("missing %s\n%s", f, usage)
		}
	}
	return o, nil
}

func load(paths ...string) ([]map[string]any, []string, error) {
	var docs []map[string]any
	var digs []string
	for _, p := range paths {
		m, err := ovio.LoadObject(p)
		if err != nil {
			return nil, nil, err
		}
		d, err := ovio.SHA256File(p)
		if err != nil {
			return nil, nil, err
		}
		docs, digs = append(docs, m), append(digs, d)
	}
	return docs, digs, nil
}

// rat parses a JSON number (or numeric string) exactly.
func rat(v any) (*big.Rat, bool) {
	s, ok := v.(string)
	if n, isNum := v.(json.Number); isNum {
		s, ok = n.String(), true
	}
	if !ok {
		return nil, false
	}
	return new(big.Rat).SetString(s)
}

func num(v any, what string) (*big.Rat, error) {
	r, ok := rat(v)
	if !ok {
		return nil, fmt.Errorf("%s is not a finite number: %v", what, v)
	}
	if r.Sign() <= 0 {
		return nil, fmt.Errorf("%s must be > 0: %v", what, v)
	}
	return r, nil
}

func needReason(v any, what string) error {
	m := ovio.Obj(v)
	if m == nil || !ovio.Truthy(m["reason"]) || !ovio.Truthy(m["evidence"]) {
		return fmt.Errorf("%s needs reason and evidence", what)
	}
	return nil
}

type control struct {
	id, domain      string
	lanes           []string
	weight          *big.Rat
	na, hard, indep bool
}

type rubricSpec struct {
	policy   map[string]*big.Rat
	dw       map[string]*big.Rat
	excluded map[string]bool
	lanes    []string
	laneExcl map[[2]string]bool
	controls []control
}

func checkRubric(r map[string]any) (*rubricSpec, error) {
	if r["schema"] != "overhaul.rubric/1" {
		return nil, fmt.Errorf("rubric schema must be overhaul.rubric/1")
	}
	errPolicy := fmt.Errorf("policy may only raise the default thresholds (global_min 9.7, lane_min 9.7, domain_min 9.0, lane_domain_min 9.0) and has no other keys")
	pol := ovio.Obj(r["policy"])
	if pol == nil && ovio.Truthy(r["policy"]) {
		return nil, errPolicy
	}
	sp := &rubricSpec{policy: map[string]*big.Rat{}, dw: map[string]*big.Rat{}, excluded: map[string]bool{},
		laneExcl: map[[2]string]bool{}}
	for _, kd := range defaultPolicy {
		def, _ := new(big.Rat).SetString(kd[1])
		val := def
		if v, ok := pol[kd[0]]; ok {
			if val, ok = rat(v); !ok {
				return nil, fmt.Errorf("policy %s is not a finite number: %v", kd[0], v)
			}
		}
		if val.Cmp(def) < 0 {
			return nil, errPolicy
		}
		sp.policy[kd[0]] = val
	}
	for k := range pol {
		if sp.policy[k] == nil {
			return nil, errPolicy
		}
	}
	doms := ovio.Obj(r["domains"])
	for _, d := range slices.Sorted(maps.Keys(doms)) {
		w, err := num(doms[d], "domain weight "+d)
		if err != nil {
			return nil, err
		}
		sp.dw[d] = w
	}
	for _, e := range ovio.List(r["domain_exclusions"]) {
		d, ok := ovio.Obj(e)["domain"].(string)
		if err := needReason(e, "domain exclusion "+d); err != nil {
			return nil, err
		}
		if !ok {
			return nil, fmt.Errorf("domain exclusion needs a domain")
		}
		sp.excluded[d] = true
	}
	known := set(domainOrder...)
	for d := range sp.dw {
		if !known[d] {
			return nil, fmt.Errorf("every domain in %v needs a weight or a recorded exclusion; no others", domainOrder)
		}
	}
	for _, d := range domainOrder {
		if sp.dw[d] == nil && !sp.excluded[d] {
			return nil, fmt.Errorf("every domain in %v needs a weight or a recorded exclusion; no others", domainOrder)
		}
	}
	laneSet := map[string]bool{}
	for _, l := range ovio.List(r["lanes"]) {
		s, ok := l.(string)
		if !ok || laneSet[s] {
			return nil, fmt.Errorf("duplicate or non-string lane name: %v", l)
		}
		laneSet[s] = true
		sp.lanes = append(sp.lanes, s)
	}
	for _, e := range ovio.List(r["lane_domain_exclusions"]) {
		l, okL := ovio.Obj(e)["lane"].(string)
		d, okD := ovio.Obj(e)["domain"].(string)
		if err := needReason(e, fmt.Sprintf("lane-domain exclusion %s/%s", l, d)); err != nil {
			return nil, err
		}
		if !okL || !okD {
			return nil, fmt.Errorf("lane-domain exclusion needs a lane and a domain")
		}
		sp.laneExcl[[2]string{l, d}] = true
	}
	seen := map[string]bool{}
	for _, cv := range ovio.List(r["controls"]) {
		c := ovio.Obj(cv)
		id, _ := c["id"].(string)
		if id == "" || seen[id] {
			return nil, fmt.Errorf("missing or duplicate control id: %q", id)
		}
		seen[id] = true
		dom, _ := c["domain"].(string)
		if sp.dw[dom] == nil {
			return nil, fmt.Errorf("%s: unknown domain %v", id, c["domain"])
		}
		var lanes []string
		var bad []any
		for _, l := range ovio.List(c["lanes"]) {
			if s, ok := l.(string); ok && laneSet[s] {
				lanes = append(lanes, s)
			} else {
				bad = append(bad, l)
			}
		}
		if len(bad) > 0 {
			return nil, fmt.Errorf("%s: unknown lanes %v", id, bad)
		}
		na := c["na"]
		if na != nil {
			if err := needReason(na, id+" N/A"); err != nil {
				return nil, err
			}
			if ovio.Obj(na)["approved"] != true {
				return nil, fmt.Errorf("%s: N/A needs an approved rationale", id)
			}
		}
		w, err := num(c["weight"], id+" weight")
		if err != nil {
			return nil, err
		}
		sp.controls = append(sp.controls, control{id: id, domain: dom, lanes: lanes, weight: w, na: na != nil,
			hard: ovio.Truthy(c["hard_gate"]), indep: ovio.Truthy(c["independent_verification"])})
	}
	return sp, nil
}

// credit returns the control's effective status. Binary credit: only PASS
// earns; PASS without evidence or required verification counts as UNKNOWN.
func credit(c control, res map[string]any, notes *[]any) (string, error) {
	st := "UNKNOWN"
	if v, ok := res["status"]; ok {
		s, isStr := v.(string)
		if !isStr || !statuses[s] {
			return "", fmt.Errorf("%s: unknown status %v", c.id, v)
		}
		st = s
	}
	if st == "PASS" && !ovio.Truthy(res["evidence"]) {
		*notes = append(*notes, map[string]any{"control": c.id, "from": "PASS", "to": "UNKNOWN", "reason": "no evidence"})
		st = "UNKNOWN"
	}
	if st == "PASS" && c.indep && !ovio.Truthy(res["verified_by"]) {
		*notes = append(*notes, map[string]any{"control": c.id, "from": "PASS", "to": "UNKNOWN", "reason": "no independent verification"})
		st = "UNKNOWN"
	}
	return st, nil
}

// agg is 10 * SUM(w*p) / SUM(w) over the kept controls; nil on a zero denominator.
func agg(live []control, pass map[string]bool, keep func(control) bool) *big.Rat {
	den, sum := new(big.Rat), new(big.Rat)
	for _, c := range live {
		if keep(c) {
			den.Add(den, c.weight)
			if pass[c.id] {
				sum.Add(sum, c.weight)
			}
		}
	}
	if den.Sign() == 0 {
		return nil
	}
	sum.Mul(sum, big.NewRat(10, 1))
	return sum.Quo(sum, den)
}

// weighted is SUM(W_d * v_d) / SUM(W_d); nil when ds is empty or any v_d is nil.
func weighted(ds []string, w, v map[string]*big.Rat) (q, den *big.Rat) {
	den, sum := new(big.Rat), new(big.Rat)
	ok := len(ds) > 0
	for _, d := range ds {
		den.Add(den, w[d])
		if v[d] == nil {
			ok = false
			continue
		}
		sum.Add(sum, new(big.Rat).Mul(w[d], v[d]))
	}
	if !ok {
		return nil, den
	}
	return sum.Quo(sum, den), den
}

func show(x *big.Rat) any {
	if x == nil {
		return nil
	}
	return map[string]any{"exact": x.Num().String() + "/" + x.Denom().String(), "decimal": x.FloatString(12)}
}

func score(rubric, results, gates map[string]any) (map[string]any, error) {
	sp, err := checkRubric(rubric)
	if err != nil {
		return nil, err
	}
	if results["schema"] != "overhaul.results/1" {
		return nil, fmt.Errorf("results schema must be overhaul.results/1")
	}
	rres := ovio.Obj(results["results"])
	if rres == nil && ovio.Truthy(results["results"]) {
		return nil, fmt.Errorf("results.results must be an object")
	}
	ids := map[string]bool{}
	for _, c := range sp.controls {
		ids[c.id] = true
	}
	var unknown []string
	for k := range rres {
		if !ids[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("results for controls not in rubric: %v", unknown)
	}

	notes := []any{}
	status := map[string]string{}
	pass := map[string]bool{}
	var live []control
	passes := 0
	for _, c := range sp.controls {
		if c.na {
			continue
		}
		rv, present := rres[c.id]
		if present && ovio.Obj(rv) == nil {
			return nil, fmt.Errorf("%s: result must be an object", c.id)
		}
		st, err := credit(c, ovio.Obj(rv), &notes)
		if err != nil {
			return nil, err
		}
		status[c.id], pass[c.id] = st, st == "PASS"
		if st == "PASS" {
			passes++
		}
		live = append(live, c)
	}

	var domains []string
	for _, d := range domainOrder {
		if sp.dw[d] != nil && !sp.excluded[d] {
			domains = append(domains, d)
		}
	}
	dom, domShow := map[string]*big.Rat{}, map[string]any{}
	for _, d := range domains {
		dom[d] = agg(live, pass, func(c control) bool { return c.domain == d })
		domShow[d] = show(dom[d])
	}
	q, _ := weighted(domains, sp.dw, dom)

	type check struct {
		scope    string
		v, least *big.Rat
	}
	checks := []check{{"global", q, sp.policy["global_min"]}}
	for _, d := range domains {
		checks = append(checks, check{"domain " + d, dom[d], sp.policy["domain_min"]})
	}
	laneOut := map[string]any{}
	appearances := 0
	for _, c := range live {
		appearances += len(c.lanes)
	}
	for _, l := range sp.lanes {
		var lds []string
		for _, d := range domains {
			if !sp.laneExcl[[2]string{l, d}] {
				lds = append(lds, d)
			}
		}
		ld, ldShow := map[string]*big.Rat{}, map[string]any{}
		for _, d := range lds {
			ld[d] = agg(live, pass, func(c control) bool { return c.domain == d && slices.Contains(c.lanes, l) })
			ldShow[d] = show(ld[d])
		}
		ql, den := weighted(lds, sp.dw, ld)
		laneOut[l] = map[string]any{"Q": show(ql), "domains": ldShow, "weight_denominator": den.RatString()}
		checks = append(checks, check{"lane " + l, ql, sp.policy["lane_min"]})
		for _, d := range lds {
			checks = append(checks, check{"lane " + l + " / " + d, ld[d], sp.policy["lane_domain_min"]})
		}
	}

	rows := []any{}
	assessable, allPass := true, true
	for _, ck := range checks {
		res := "PASS"
		switch {
		case ck.v == nil:
			res, assessable = "NOT_ASSESSABLE", false
		case ck.v.Cmp(ck.least) < 0:
			res, allPass = "FAIL", false
		}
		rows = append(rows, map[string]any{"scope": ck.scope, "value": show(ck.v), "min": ck.least.RatString(), "result": res})
	}
	mandatory := []string{}
	for _, c := range live {
		if c.hard && !pass[c.id] {
			mandatory = append(mandatory, c.id)
		}
	}

	mode := gates["mode"]
	if mode != "audit" && mode != "remediation" {
		return nil, fmt.Errorf("gates.mode must be audit or remediation")
	}
	gateOut := map[string]string{"G-THRESHOLDS": "FAIL", "G-MANDATORY": "PASS"}
	if !assessable {
		gateOut["G-THRESHOLDS"] = "UNKNOWN"
	} else if allPass {
		gateOut["G-THRESHOLDS"] = "PASS"
	}
	if len(mandatory) > 0 {
		gateOut["G-MANDATORY"] = "FAIL"
	}
	names := auditGates
	if mode == "remediation" {
		names = append(slices.Clone(auditGates), repairGates...)
	}
	given := ovio.Obj(gates["gates"])
	for _, g := range names {
		if gateOut[g] != "" {
			continue
		}
		e := ovio.Obj(given[g])
		if e == nil && ovio.Truthy(given[g]) {
			return nil, fmt.Errorf("%s: gate record must be an object", g)
		}
		st := "UNKNOWN"
		if v, ok := e["status"]; ok {
			s, isStr := v.(string)
			if !isStr || !gateStatuses[s] {
				return nil, fmt.Errorf("%s: unknown gate status %v", g, v)
			}
			st = s
		}
		if st == "PASS" && !ovio.Truthy(e["evidence"]) {
			st = "UNKNOWN"
			notes = append(notes, map[string]any{"gate": g, "from": "PASS", "to": "UNKNOWN", "reason": "no evidence"})
		}
		if st == "NOT_APPLICABLE" && (!mayBeNA[g] || !ovio.Truthy(e["reason"])) {
			return nil, fmt.Errorf("%s cannot be NOT_APPLICABLE without an allowed reason", g)
		}
		if st == "WAIVED_OPERATIONAL" && (g != "G-CONCURRENCY" || !ovio.Truthy(e["approval"])) {
			return nil, fmt.Errorf("only G-CONCURRENCY may be WAIVED_OPERATIONAL, with an approval reference")
		}
		gateOut[g] = st
	}
	open := []string{}
	for _, g := range slices.Sorted(maps.Keys(gateOut)) {
		if s := gateOut[g]; s != "PASS" && s != "NOT_APPLICABLE" {
			open = append(open, g)
		}
	}
	verdict := "NOT_READY"
	switch {
	case !assessable:
		verdict = "NOT_ASSESSABLE"
	case len(open) == 0:
		verdict = "MEETS_TARGETS"
	case len(open) == 1 && open[0] == "G-CONCURRENCY" && gateOut["G-CONCURRENCY"] == "WAIVED_OPERATIONAL":
		verdict = "MEETS_TARGETS_WITH_APPROVED_DEGRADATION"
	}
	excluded := slices.AppendSeq([]string{}, maps.Keys(sp.excluded))
	sort.Strings(excluded)

	return map[string]any{
		"schema": "overhaul.scorecard/1", "calculation": calc, "subject": results["subject"],
		"fingerprint": results["fingerprint"], "rubric_rev": rubric["rev"],
		"global": map[string]any{"Q": show(q), "domains": domShow, "excluded_domains": excluded},
		"lanes":  laneOut, "thresholds": rows, "gates": gateOut,
		"mandatory_failures": mandatory, "downgrades": notes, "status_by_control": status,
		"counts": map[string]any{"controls": len(sp.controls), "not_applicable": len(sp.controls) - len(live),
			"pass": passes, "lane_appearances": appearances},
		"readiness_verdict": verdict, "open_gates": open,
	}, nil
}

// compare scores baseline and candidate on one rubric and labels each
// per-control status transition.
func compare(rubric, a, b map[string]any) (map[string]any, error) {
	sa, err := score(rubric, a, map[string]any{"mode": "audit"})
	if err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}
	sb, err := score(rubric, b, map[string]any{"mode": "audit"})
	if err != nil {
		return nil, fmt.Errorf("candidate: %w", err)
	}
	before, after := sa["status_by_control"].(map[string]string), sb["status_by_control"].(map[string]string)
	rows := []any{}
	for _, id := range slices.Sorted(maps.Keys(before)) {
		if before[id] != after[id] {
			kind := transitions[[2]string{before[id], after[id]}]
			if kind == "" {
				kind = "status-change"
			}
			rows = append(rows, map[string]any{"control": id, "from": before[id], "to": after[id], "kind": kind})
		}
	}
	return map[string]any{"baseline_Q": sa["global"].(map[string]any)["Q"],
		"candidate_Q": sb["global"].(map[string]any)["Q"], "transitions": rows}, nil
}
