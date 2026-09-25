package score

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

var doms = []string{"correctness", "security", "reliability", "performance", "tests", "architecture", "ux", "operations"}

func ctl(i int, d string, w any, lanes ...string) map[string]any {
	if len(lanes) == 0 {
		lanes = []string{"backend"}
	}
	return map[string]any{"id": fmt.Sprintf("C-%03d", i), "domain": d, "lanes": lanes, "weight": w, "hard_gate": false}
}

func rubricOf(controls []map[string]any) map[string]any {
	return map[string]any{"schema": "overhaul.rubric/1", "rev": 1, "lanes": []string{"frontend", "backend"},
		"controls": controls, "domains": map[string]any{"correctness": 20, "security": 20, "reliability": 15,
			"performance": 15, "tests": 10, "architecture": 10, "ux": 5, "operations": 5}}
}

func resOf(status map[string]string) map[string]any {
	r := map[string]any{}
	for k, v := range status {
		r[k] = map[string]any{"status": v, "evidence": []string{"EV-1"}}
	}
	return map[string]any{"schema": "overhaul.results/1", "subject": "candidate", "results": r}
}

func passing() map[string]any {
	g := map[string]any{}
	for _, id := range auditGates {
		g[id] = map[string]any{"status": "PASS", "evidence": []string{"EV-g"}}
	}
	return map[string]any{"mode": "audit", "gates": g}
}

func allPass(cs []map[string]any) map[string]string {
	st := map[string]string{}
	for _, c := range cs {
		st[c["id"].(string)] = "PASS"
	}
	return st
}

// norm round-trips v through JSON with UseNumber, exactly as file input arrives.
func norm(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func tryScore(t *testing.T, r, s, g any) (map[string]any, error) {
	t.Helper()
	out, err := score(norm(t, r), norm(t, s), norm(t, g))
	if err != nil {
		return nil, err
	}
	return norm(t, out), nil
}

func mustScore(t *testing.T, r, s, g any) map[string]any {
	t.Helper()
	out, err := tryScore(t, r, s, g)
	if err != nil {
		t.Fatalf("score: %v", err)
	}
	return out
}

func get(v any, path string) string { return fmt.Sprint(ovio.Get(v, path)) }

func exact(t *testing.T, v any, path string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(get(v, path))
	if !ok {
		t.Fatalf("%s is not exact: %v", path, ovio.Get(v, path))
	}
	return r
}

func base() []map[string]any {
	var b []map[string]any
	for i, d := range doms {
		b = append(b, ctl(i, d, 5, "frontend", "backend"))
	}
	return b
}

func workedExample() ([]map[string]any, map[string]string) {
	ex := append(base(), ctl(10, "correctness", 5), ctl(11, "correctness", 5), ctl(12, "correctness", 3), ctl(13, "correctness", 1))
	st := allPass(ex)
	st["C-013"] = "FAIL"
	return ex, st
}

func laneMasking() []map[string]any {
	l := base()
	for k := range 10 {
		l = append(l, ctl(30+k, "correctness", 5, "backend"))
	}
	return append(l, ctl(40, "correctness", 5, "frontend"))
}

// 1. Worked example: correctness 5,5,5,3 PASS + 1 FAIL -> D=180/19, Q=18800/1900.
func TestWorkedExample(t *testing.T) {
	ex, st := workedExample()
	out := mustScore(t, rubricOf(ex), resOf(st), passing())
	if got := get(out, "global.domains.correctness.exact"); got != "180/19" {
		t.Fatalf("correctness = %s", got)
	}
	if got := get(out, "global.Q.exact"); got != "188/19" {
		t.Fatalf("Q = %s", got)
	}
	if got := get(out, "global.Q.decimal"); got != "9.894736842105" {
		t.Fatalf("decimal = %s", got)
	}
	if v := get(out, "readiness_verdict"); v != "MEETS_TARGETS" {
		t.Fatalf("verdict = %s", v)
	}
}

// 2. Same numbers, failing control is a hard gate -> NOT_READY despite Q 9.89.
func TestHardGateBlocks(t *testing.T) {
	ex, st := workedExample()
	ex[len(ex)-1]["hard_gate"] = true
	out := mustScore(t, rubricOf(ex), resOf(st), passing())
	if get(out, "readiness_verdict") != "NOT_READY" || get(out, "gates.G-MANDATORY") != "FAIL" {
		t.Fatalf("verdict %s, G-MANDATORY %s", get(out, "readiness_verdict"), get(out, "gates.G-MANDATORY"))
	}
	if got := get(out, "global.Q.exact"); got != "188/19" {
		t.Fatalf("Q = %s", got)
	}
}

// 3. Q exactly 9.696 fails 9.7 at full precision.
func TestFullPrecisionThreshold(t *testing.T) {
	var cs []map[string]any
	st := map[string]string{}
	i := 100
	for _, d := range doms {
		n := 1
		if d == "correctness" || d == "security" {
			n = 250
		}
		for k := range n {
			cs = append(cs, ctl(i, d, 1, "frontend", "backend"))
			st[fmt.Sprintf("C-%03d", i)] = "PASS"
			if n == 250 && k < 19 {
				st[fmt.Sprintf("C-%03d", i)] = "FAIL"
			}
			i++
		}
	}
	out := mustScore(t, rubricOf(cs), resOf(st), passing())
	if got := get(out, "global.Q.exact"); got != "1212/125" {
		t.Fatalf("Q = %s", got)
	}
	row := ovio.List(out["thresholds"])[0]
	if get(row, "scope") != "global" || get(row, "result") != "FAIL" || get(out, "readiness_verdict") != "NOT_READY" {
		t.Fatalf("row %v verdict %s", row, get(out, "readiness_verdict"))
	}
}

// 4. Zero denominator: an applicable domain whose only control is N/A -> NOT_ASSESSABLE, never 10.
func TestZeroDenominatorNotAssessable(t *testing.T) {
	b := base()
	z := make([]map[string]any, len(b))
	for i, c := range b {
		z[i] = maps.Clone(c)
	}
	z[6]["na"] = map[string]any{"reason": "no UI", "evidence": []string{"EV-2"}, "approved": true}
	out := mustScore(t, rubricOf(z), resOf(allPass(b)), passing())
	if ovio.Get(out, "global.Q") != nil || get(out, "readiness_verdict") != "NOT_ASSESSABLE" {
		t.Fatalf("Q %v verdict %s", ovio.Get(out, "global.Q"), get(out, "readiness_verdict"))
	}
}

// 5. Global 59/6 and backend 10 cannot mask frontend 9.0; shared controls count once globally.
func TestLaneMasking(t *testing.T) {
	l := laneMasking()
	st := allPass(l)
	st["C-040"] = "FAIL"
	out := mustScore(t, rubricOf(l), resOf(st), passing())
	if row := ovio.List(out["thresholds"])[0]; get(row, "result") != "PASS" || get(out, "global.Q.exact") != "59/6" {
		t.Fatalf("global row %v Q %s", row, get(out, "global.Q.exact"))
	}
	if get(out, "lanes.frontend.Q.exact") != "9/1" || get(out, "lanes.backend.Q.exact") != "10/1" {
		t.Fatalf("lanes %v", out["lanes"])
	}
	laneRow := "missing"
	for _, row := range ovio.List(out["thresholds"]) {
		if get(row, "scope") == "lane frontend" {
			laneRow = get(row, "result")
		}
	}
	if laneRow != "FAIL" {
		t.Fatalf("lane frontend threshold row = %s", laneRow)
	}
	if get(out, "readiness_verdict") != "NOT_READY" {
		t.Fatalf("verdict %s", get(out, "readiness_verdict"))
	}
	if get(out, "counts.controls") != "19" || get(out, "counts.lane_appearances") != "27" {
		t.Fatalf("counts %v", out["counts"])
	}
}

// 6. PASS without evidence counts as UNKNOWN; UNKNOWN earns zero, not N/A.
func TestPassWithoutEvidenceIsUnknown(t *testing.T) {
	b := base()
	r := resOf(allPass(b))
	r["results"].(map[string]any)["C-000"].(map[string]any)["evidence"] = []string{}
	out := mustScore(t, rubricOf(b), r, passing())
	if get(out, "status_by_control.C-000") != "UNKNOWN" || get(out, "global.domains.correctness.exact") != "0/1" {
		t.Fatalf("status %s correctness %s", get(out, "status_by_control.C-000"), get(out, "global.domains.correctness.exact"))
	}
	if len(ovio.List(out["downgrades"])) != 1 {
		t.Fatalf("downgrades %v", out["downgrades"])
	}
}

// 7. Fixed-catalog monotonicity: turning any PASS into FAIL never raises any score.
func TestMonotonicity(t *testing.T) {
	l := laneMasking()
	full := allPass(l)
	top := mustScore(t, rubricOf(l), resOf(full), passing())
	for id := range full {
		worse := maps.Clone(full)
		worse[id] = "FAIL"
		low := mustScore(t, rubricOf(l), resOf(worse), passing())
		for _, p := range []string{"global.Q.exact", "lanes.frontend.Q.exact", "lanes.backend.Q.exact"} {
			if exact(t, low, p).Cmp(exact(t, top, p)) > 0 {
				t.Fatalf("failing %s raised %s", id, p)
			}
		}
		if exact(t, low, "global.Q.exact").Cmp(exact(t, top, "global.Q.exact")) >= 0 {
			t.Fatalf("failing %s did not lower global Q", id)
		}
	}
}

// 8. Duplicate control ids and zero weights are invalid input.
func TestInvalidCatalog(t *testing.T) {
	zero := base()
	zero[0] = maps.Clone(zero[0])
	zero[0]["weight"] = 0
	for _, tc := range []struct {
		cs   []map[string]any
		want string
	}{{append(base(), base()[0]), "duplicate control id"}, {zero, "must be > 0"}} {
		if _, err := tryScore(t, rubricOf(tc.cs), resOf(nil), passing()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("want %q, got %v", tc.want, err)
		}
	}
}

// 9. Approved sequential degradation is the only open gate -> qualified outcome.
func TestApprovedDegradation(t *testing.T) {
	b := base()
	g := passing()
	g["gates"].(map[string]any)["G-CONCURRENCY"] = map[string]any{"status": "WAIVED_OPERATIONAL", "approval": "AP-2"}
	out := mustScore(t, rubricOf(b), resOf(allPass(b)), g)
	if v := get(out, "readiness_verdict"); v != "MEETS_TARGETS_WITH_APPROVED_DEGRADATION" {
		t.Fatalf("verdict %s", v)
	}
}

// 10. Missing gate evidence -> UNKNOWN -> NOT_READY.
func TestGatePassWithoutEvidence(t *testing.T) {
	b := base()
	g := passing()
	g["gates"].(map[string]any)["G-COVERAGE"] = map[string]any{"status": "PASS"}
	out := mustScore(t, rubricOf(b), resOf(allPass(b)), g)
	if get(out, "gates.G-COVERAGE") != "UNKNOWN" || get(out, "readiness_verdict") != "NOT_READY" {
		t.Fatalf("gate %s verdict %s", get(out, "gates.G-COVERAGE"), get(out, "readiness_verdict"))
	}
}

// 11. A rubric cannot lower the policy thresholds or drop a domain without a recorded exclusion.
func TestWeakenedRubric(t *testing.T) {
	b := base()
	low := rubricOf(b)
	low["policy"] = map[string]any{"global_min": "0"}
	var four []map[string]any
	for i, d := range []string{"tests", "architecture", "ux", "operations"} {
		four = append(four, ctl(i+1, d, 5))
	}
	partial := rubricOf(four)
	partial["domains"] = map[string]any{"tests": 10, "architecture": 10, "ux": 5, "operations": 5}
	for _, tc := range []struct {
		r    map[string]any
		st   map[string]string
		want string
	}{{low, allPass(b), "policy may only raise"}, {partial, allPass(four), "needs a weight or a recorded exclusion"}} {
		if _, err := tryScore(t, tc.r, resOf(tc.st), passing()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("want %q, got %v", tc.want, err)
		}
	}
}

func writeJSON(t *testing.T, dir, name string, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func runCLI(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestCompare(t *testing.T) {
	var excl []any
	for _, d := range doms[2:] {
		excl = append(excl, map[string]any{"domain": d, "reason": "out of scope", "evidence": []string{"EV-x"}})
	}
	r := map[string]any{"schema": "overhaul.rubric/1", "rev": 1, "lanes": []string{"backend"},
		"domains": map[string]any{"correctness": 50, "security": 50}, "domain_exclusions": excl,
		"controls": []map[string]any{ctl(1, "correctness", 5), ctl(2, "correctness", 5), ctl(3, "security", 5)}}
	dir := t.TempDir()
	code, stdout, stderr := runCLI("compare", "--rubric", writeJSON(t, dir, "r.json", r),
		"--baseline", writeJSON(t, dir, "a.json", resOf(map[string]string{"C-001": "PASS", "C-002": "UNKNOWN", "C-003": "PASS"})),
		"--candidate", writeJSON(t, dir, "b.json", resOf(map[string]string{"C-001": "FAIL", "C-002": "PASS", "C-003": "PASS"})))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	out := norm(t, json.RawMessage(stdout))
	if get(out, "baseline_Q.exact") != "15/2" || get(out, "candidate_Q.exact") != "15/2" {
		t.Fatalf("Q %v / %v", out["baseline_Q"], out["candidate_Q"])
	}
	rows := ovio.List(out["transitions"])
	if len(rows) != 2 ||
		get(rows[0], "control") != "C-001" || get(rows[0], "kind") != "regression-or-new-failure" ||
		get(rows[1], "control") != "C-002" || get(rows[1], "kind") != "evidence-only" {
		t.Fatalf("transitions %v", rows)
	}
}

func TestCLIOutWritesScorecard(t *testing.T) {
	dir := t.TempDir()
	b := base()
	outPath := filepath.Join(dir, "scorecard.json")
	code, stdout, stderr := runCLI("--rubric", writeJSON(t, dir, "r.json", rubricOf(b)),
		"--results", writeJSON(t, dir, "s.json", resOf(allPass(b))),
		"--gates", writeJSON(t, dir, "g.json", passing()), "--out", outPath)
	if code != 0 || stdout != "MEETS_TARGETS -> "+outPath+"\n" {
		t.Fatalf("exit %d stdout %q stderr %q", code, stdout, stderr)
	}
	card, err := ovio.LoadObject(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if get(card, "schema") != "overhaul.scorecard/1" || get(card, "readiness_verdict") != "MEETS_TARGETS" ||
		len(get(card, "inputs.rubric_sha256")) != 64 {
		t.Fatalf("scorecard %v", card)
	}
}

func TestCLIInvalidInputExits2(t *testing.T) {
	dir := t.TempDir()
	b := base()
	bad := rubricOf(b)
	bad["schema"] = "overhaul.rubric/0"
	rubric := writeJSON(t, dir, "r.json", rubricOf(b))
	stale := resOf(allPass(b))
	stale["rubric_digest"] = strings.Repeat("0", 64)
	for name, args := range map[string][]string{
		"bad rubric": {"--rubric", writeJSON(t, dir, "bad.json", bad), "--results", writeJSON(t, dir, "s.json", resOf(allPass(b))),
			"--gates", writeJSON(t, dir, "g.json", passing())},
		"stale digest": {"--rubric", rubric, "--results", writeJSON(t, dir, "stale.json", stale),
			"--gates", filepath.Join(dir, "g.json")},
	} {
		code, stdout, stderr := runCLI(args...)
		if code != 2 || stdout != "" || !strings.HasPrefix(stderr, "ERROR: ") {
			t.Fatalf("%s: exit %d stdout %q stderr %q", name, code, stdout, stderr)
		}
	}
}
