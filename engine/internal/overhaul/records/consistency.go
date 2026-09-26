package records

import (
	"encoding/json"
	"slices"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

var (
	jsonZero      = any(json.Number("0"))
	domains       = set(ovio.Domains...)
	cellStates    = set("applicable", "not-applicable", "blocked")
	rangeStates   = set("inventoried", "tool-scanned", "semantically-reviewed", "cross-boundary-reviewed", "verified", "excluded", "blocked")
	reviewedRange = set("semantically-reviewed", "cross-boundary-reviewed", "verified", "excluded")
	openFinding   = set("confirmed", "approved-for-fix", "in-progress", "fixed-unverified", "deferred", "accepted-risk")
	serious       = set("critical", "high")
)

// consistency checks that findings, the coverage ledger, the applicability
// matrix, the control results and the recorded gates agree with each other, so
// a gate or control cannot claim PASS against the run's own records.
func consistency(run, genDir string, R map[string]map[string]any, add func(string, ...any)) error {
	controls, err := rubricControls(run, R["run.json"])
	if err != nil {
		return err
	}
	fs := ovio.List(R["findings.json"]["findings"])
	for _, it := range fs {
		f := ovio.Obj(it)
		fid := PyStr(f["id"])
		if !domains[str(f, "domain")] {
			add("finding %s: domain must be one of %s", fid, Repr(ovio.Domains))
		}
		if controls == nil {
			continue
		}
		cs := ovio.List(f["controls"])
		if len(cs) == 0 && severityOK[str(f, "status")] {
			add("finding %s: a confirmed finding names the rubric controls it fails", fid)
		}
		for _, c := range cs {
			if !controls.has(c) {
				add("finding %s: unknown control %s", fid, PyStr(c))
			}
		}
	}

	cov := R["coverage.json"]
	cells := ovio.List(cov["applicability"])
	seen := vset{}
	for _, it := range cells {
		c := ovio.Obj(it)
		key := Repr([]any{c["domain"], c["component"], c["lane"]})
		if seen.has(key) {
			add("applicability %s: duplicate cell", key)
		}
		seen.add(key)
		if !domains[str(c, "domain")] || str(c, "component") == "" || str(c, "lane") == "" {
			add("applicability %s: needs a known domain, a component and a lane", key)
		}
		st := str(c, "state")
		if !cellStates[st] {
			add("applicability %s: state must be applicable, not-applicable or blocked", key)
		}
		if st != "applicable" && !ovio.Truthy(c["reason"]) {
			add("applicability %s: %s needs a reason", key, st)
		}
		for _, r := range ovio.List(c["receipts"]) {
			if p := ovio.Str(r); p == "" || !inside(run, p) || !isFile(join(run, p)) {
				add("applicability %s: receipt %s missing or outside the run dir", key, PyStr(r))
			}
		}
	}
	for _, it := range ovio.List(cov["files"]) {
		for _, r := range ovio.List(ovio.Obj(it)["ranges"]) {
			if st := str(ovio.Obj(r), "state"); !rangeStates[st] {
				add("coverage %s: range state %s is not a coverage state", PyStr(ovio.Obj(it)["path"]), Repr(st))
			}
		}
	}

	results := ""
	for _, name := range []string{"results-candidate.json", "results-baseline.json"} {
		if isFile(join(genDir, name)) {
			results = name
			break
		}
	}
	if results != "" {
		doc, err := ovio.LoadObject(join(genDir, results))
		if err != nil {
			return err
		}
		res := ovio.Obj(doc["results"])
		for _, it := range fs {
			f := ovio.Obj(it)
			if !openFinding[str(f, "status")] {
				continue
			}
			for _, c := range ovio.List(f["controls"]) {
				if str(ovio.Obj(res[ovio.Str(c)]), "status") == "PASS" {
					add("finding %s: control %s is PASS in %s while the finding is %s",
						PyStr(f["id"]), PyStr(c), results, str(f, "status"))
				}
			}
		}
	}

	scp := join(genDir, "scorecard.json")
	if !isFile(scp) {
		return nil
	}
	sc, err := ovio.LoadObject(scp)
	if err != nil {
		return err
	}
	gates := ovio.Obj(sc["gates"])
	claims := func(g string) bool { return str(gates, g) == "PASS" }
	for _, it := range fs {
		f := ovio.Obj(it)
		fid, status := PyStr(f["id"]), str(f, "status")
		if claims("G-NO-CRITICAL-HIGH") && openFinding[status] && serious[str(f, "severity")] {
			add("scorecard: G-NO-CRITICAL-HIGH is PASS but finding %s is %s %s", fid, status, str(f, "severity"))
		}
		if claims("G-NO-OPEN-SERIOUS-LEAD") && (status == "candidate" || status == "needs-validation") &&
			serious[str(f, "potential_impact")] {
			add("scorecard: G-NO-OPEN-SERIOUS-LEAD is PASS but lead %s is open with potential impact %s", fid, str(f, "potential_impact"))
		}
	}
	if !claims("G-COVERAGE") {
		return nil
	}
	admitted := map[string]bool{}
	for _, it := range ovio.List(R["dispatch.json"]["attempts"]) {
		if x := ovio.Obj(it); str(x, "status") == "admitted" {
			admitted[str(x, "receipt")] = true
		}
	}
	if len(cells) == 0 {
		add("scorecard: G-COVERAGE is PASS without an applicability matrix in coverage.json")
	}
	for _, it := range cells {
		c := ovio.Obj(it)
		key := Repr([]any{c["domain"], c["component"], c["lane"]})
		switch str(c, "state") {
		case "blocked":
			add("scorecard: G-COVERAGE is PASS but applicability %s is blocked", key)
		case "applicable":
			if !checkedBy(run, str(c, "domain"), ovio.List(c["receipts"]), admitted) {
				add("scorecard: G-COVERAGE is PASS but no admitted, non-gap receipt of applicability %s lists %s in domains_checked", key, str(c, "domain"))
			}
		}
	}
	for _, it := range ovio.List(cov["files"]) {
		f := ovio.Obj(it)
		if !ovio.Truthy(f["eligible"]) {
			continue
		}
		rs := ovio.List(f["ranges"])
		done := len(rs) > 0 || eq(f["lines"], jsonZero)
		for _, r := range rs {
			done = done && reviewedRange[str(ovio.Obj(r), "state")]
		}
		if !done {
			add("scorecard: G-COVERAGE is PASS but eligible file %s has unreviewed ranges", PyStr(f["path"]))
		}
	}
	return nil
}

// checkedBy reports whether any admitted, non-gap receipt records domain in
// domains_checked.
func checkedBy(run, domain string, receipts []any, admitted map[string]bool) bool {
	for _, r := range receipts {
		p := ovio.Str(r)
		if !admitted[p] || !inside(run, p) || !isFile(join(run, p)) {
			continue
		}
		rc, err := ovio.LoadObject(join(run, p))
		if err != nil || str(rc, "outcome") == "gap" {
			continue
		}
		if slices.ContainsFunc(ovio.List(rc["domains_checked"]), func(d any) bool { return d == any(domain) }) {
			return true
		}
	}
	return false
}

// rubricControls returns the control IDs of the run's rubric (revisions.rubric,
// else the plan's pinned rubric), or nil when no rubric is frozen yet.
func rubricControls(run string, rj map[string]any) (vset, error) {
	revs := ovio.Obj(rj["revisions"])
	rf := str(ovio.Obj(revs["rubric"]), "file")
	if rf == "" {
		if pf := str(ovio.Obj(revs["plan"]), "file"); pf != "" && inside(run, pf) && isFile(join(run, pf)) {
			plan, err := ovio.LoadObject(join(run, pf))
			if err != nil {
				return nil, err
			}
			rf = str(ovio.Obj(plan["rubric"]), "file")
		}
	}
	if rf == "" || !inside(run, rf) || !isFile(join(run, rf)) {
		return nil, nil
	}
	doc, err := ovio.LoadObject(join(run, rf))
	if err != nil {
		return nil, err
	}
	ids := vset{}
	for _, c := range ovio.List(doc["controls"]) {
		ids.add(ovio.Obj(c)["id"])
	}
	return ids, nil
}
