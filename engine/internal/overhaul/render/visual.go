package render

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

// The HTML view leads with pictures of the gaps: a score answer, a domain ×
// lane gap map, where the control weight is lost, code hotspots and review
// coverage. Charts are inline SVG with <title> tooltips (no script); every
// value they draw is also printed beside them and kept in the records tables.

var (
	gateText = map[string]string{
		"G-THRESHOLDS":           "Scores are below their minimums",
		"G-MANDATORY":            "Hard-gate controls fail",
		"G-NO-CRITICAL-HIGH":     "Critical or high findings are open",
		"G-NO-OPEN-SERIOUS-LEAD": "Serious leads are not validated yet",
		"G-COVERAGE":             "Review coverage is incomplete",
		"G-CATALOG-ADEQUACY":     "The rubric has not passed its adequacy review",
		"G-LANE-RECEIPTS":        "A frontend or backend review is missing",
		"G-CONCURRENCY":          "Frontend and backend review did not overlap",
		"G-FRESHNESS":            "Some evidence is stale",
		"G-REPORT-CONSISTENCY":   "Records or views are inconsistent",
		"G-APPROVAL":             "No current approval of this plan",
		"G-TASKS-COMPLETE":       "Approved tasks are not complete",
		"G-ORACLES":              "Regression tests are not proven red then green",
		"G-NO-REGRESSION":        "A regression or skipped check remains",
		"G-PERF-TARGETS":         "Performance targets are not demonstrated",
		"G-PRESERVATION":         "Behavior preservation is not proven",
		"G-FINGERPRINTS":         "Source changed during proof",
		"G-OWNERSHIP":            "Changes left their approved paths",
		"G-FINAL-CHALLENGE":      "The final challenge has not passed",
	}
	verdictText = map[string]string{
		"NOT_READY": "Not ready", "MEETS_TARGETS": "Meets targets", "NOT_ASSESSED": "Not assessed",
		"NOT_ASSESSABLE": "Not assessable", "MEETS_TARGETS_WITH_APPROVED_DEGRADATION": "Meets targets (approved degradation)",
	}
	// gapSteps are the gap-map buckets: how far a score sits below its floor.
	gapSteps = []struct {
		cls, label string
		upTo       *big.Rat // largest gap in this bucket; nil = unbounded
	}{
		{"g0", "meets its floor", big.NewRat(0, 1)},
		{"g4", "up to 2 below", big.NewRat(2, 1)},
		{"g3", "2–5 below", big.NewRat(5, 1)},
		{"g2", "5–8 below", big.NewRat(8, 1)},
		{"g1", "more than 8 below", nil},
	}
	navName = map[string]string{"plan": "Repair plan", "findings": "Findings", "opportunities": "Opportunities", "leads": "Unvalidated leads",
		"repairs": "Repairs", "performance": "Performance", "cycles": "Iterations"}
	sevWeight = map[string]int{"critical": 8, "high": 4, "medium": 2, "low": 1}
	ten       = big.NewRat(10, 1)
)

// ratOf reads an exact score: {"exact":"n/d"}, "n/d", or a JSON number.
func ratOf(x any) *big.Rat {
	switch t := x.(type) {
	case map[string]any:
		return ratOf(t["exact"])
	case string:
		if r, ok := new(big.Rat).SetString(t); ok {
			return r
		}
	case json.Number:
		if r, ok := new(big.Rat).SetString(t.String()); ok {
			return r
		}
	}
	return nil
}

// scoreFor returns a scope's score from the scorecard: "global", "lane X" or
// "domain X" (global domain), falling back to the thresholds table.
func scoreFor(sc map[string]any, scope string) *big.Rat {
	var r *big.Rat
	switch kind, name, _ := strings.Cut(scope, " "); kind {
	case "global":
		r = ratOf(ovio.Get(sc, "global.Q"))
	case "lane":
		r = ratOf(ovio.Obj(ovio.Obj(ovio.Obj(sc["lanes"])[name])["Q"]))
	case "domain":
		r = ratOf(ovio.Obj(ovio.Obj(ovio.Get(sc, "global.domains"))[name]))
	}
	if r == nil {
		r, _, _ = threshold(sc, scope)
	}
	return r
}

func laneDomain(sc map[string]any, lane, domain string) *big.Rat {
	return ratOf(ovio.Obj(ovio.Obj(ovio.Obj(ovio.Obj(sc["lanes"])[lane])["domains"])[domain]))
}

// floorFor is the scope's recorded minimum, else the skill's default.
func floorFor(sc map[string]any, scope string) *big.Rat {
	if _, least, _ := threshold(sc, scope); least != nil {
		return least
	}
	if strings.HasPrefix(scope, "domain ") {
		return big.NewRat(9, 1)
	}
	return big.NewRat(97, 10)
}

func gapClass(val, floor *big.Rat) string {
	if val == nil {
		return "gx"
	}
	gap := new(big.Rat).Sub(floor, val)
	for _, s := range gapSteps {
		if s.upTo == nil || gap.Cmp(s.upTo) <= 0 {
			return s.cls
		}
	}
	return "g1"
}

func f2(r *big.Rat) string {
	if r == nil {
		return "–"
	}
	return floor2(r)
}

// frac maps r/of onto 0..100 for SVG geometry.
func frac(r, of *big.Rat) float64 {
	if r == nil || of.Sign() == 0 {
		return 0
	}
	f, _ := new(big.Rat).Quo(r, of).Float64()
	return max(0, min(100, f*100))
}

// bullet draws a 0–10 score bar with the floor as a tick and, when given, the
// baseline as a hollow marker. Geometry is in a stretched 0–100 box.
func bullet(val, floor, base *big.Rat, cls, label string) string {
	var w strings.Builder
	fmt.Fprintf(&w, `<svg class="bar" viewBox="0 0 100 10" preserveAspectRatio="none" role="img" aria-label="%s"><title>%s</title>`, esc(label), esc(label))
	w.WriteString(`<rect class="trk" x="0" y="2" width="100" height="6"/>`)
	if val != nil {
		fmt.Fprintf(&w, `<rect class="%s" x="0" y="2" width="%.2f" height="6"/>`, cls, frac(val, ten))
	}
	if base != nil && (val == nil || base.Cmp(val) != 0) {
		fmt.Fprintf(&w, `<rect class="was" x="%.2f" y="0" width="0.8" height="10"/>`, max(0, frac(base, ten)-0.4))
	}
	fmt.Fprintf(&w, `<rect class="tgt" x="%.2f" y="0" width="0.6" height="10"/></svg>`, max(0, frac(floor, ten)-0.3))
	return w.String()
}

// stack draws one horizontal stacked bar: segments are (class, value, label)
// scaled against total, separated by a small surface gap.
func stack(segs [][3]any, total float64, label string) string {
	var w strings.Builder
	fmt.Fprintf(&w, `<svg class="bar" viewBox="0 0 100 10" preserveAspectRatio="none" role="img" aria-label="%s"><title>%s</title>`, esc(label), esc(label))
	x := 0.0
	for _, s := range segs {
		v := s[1].(float64)
		if v <= 0 || total <= 0 {
			continue
		}
		wd := v / total * 100
		fmt.Fprintf(&w, `<rect class="%s" x="%.2f" y="1" width="%.2f" height="8"><title>%s</title></rect>`,
			s[0].(string), x, max(0, wd-0.4), esc(s[2].(string)))
		x += wd
	}
	return w.String() + "</svg>"
}

func legend(items ...[2]string) string {
	var w strings.Builder
	w.WriteString(`<ul class="legend">`)
	for _, it := range items {
		w.WriteString(`<li><span class="key ` + it[0] + `"></span>` + it[1] + `</li>`)
	}
	return w.String() + "</ul>"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

// current is the scorecard the page scores against: the candidate scorecard
// when one exists, else the baseline. base is set only when both exist.
func (b *built) current() (cur, base map[string]any) {
	if len(b.sc) > 0 && (b.sc["thresholds"] != nil || b.sc["global"] != nil) {
		if len(b.base) > 0 {
			return b.sc, b.base
		}
		return b.sc, nil
	}
	return b.base, nil
}

func (v view) summaryBlock(b *built, summary string, decide bool) string {
	cur, base := b.current()
	verdict := pyStr(b.rj["readiness_verdict"])
	var w strings.Builder
	w.WriteString(`<header id="summary" class="hero">`)
	fmt.Fprintf(&w, `<p role="status"><strong>%s</strong></p>`, esc(summary))

	q, qf := scoreFor(cur, "global"), floorFor(cur, "global")
	text := verdictText[verdict]
	if text == "" {
		text = verdict
	}
	w.WriteString(`<div class="answer"><div class="score">`)
	fmt.Fprintf(&w, `<h1 class="verdict %s">%s</h1>`, tone(verdict), esc(v.redact(text)))
	if q != nil {
		fmt.Fprintf(&w, `<p class="hero-num">%s<span> / 10</span></p>`, f2(q))
		gap := new(big.Rat).Sub(qf, q)
		if gap.Sign() > 0 {
			fmt.Fprintf(&w, `<p class="muted">Target %s · %s to go</p>`, f2(qf), f2(gap))
		} else {
			fmt.Fprintf(&w, `<p class="muted">Target %s · met</p>`, f2(qf))
		}
		w.WriteString(bullet(q, qf, scoreFor(base, "global"), gapClass(q, qf), "Global score "+f2(q)+" of 10, target "+f2(qf)))
	} else {
		w.WriteString(`<p class="hero-num muted">No score yet</p>`)
	}
	lanes := b.scoreLanes(cur)
	if len(lanes) > 0 {
		w.WriteString(`<ul class="rows lanes">`)
		for _, l := range lanes {
			s, fl := scoreFor(cur, "lane "+l), floorFor(cur, "lane "+l)
			fmt.Fprintf(&w, `<li><span class="lbl">%s</span>%s<span class="val">%s</span></li>`, esc(v.redact(l)),
				bullet(s, fl, scoreFor(base, "lane "+l), gapClass(s, fl), l+" lane "+f2(s)+" of 10, target "+f2(fl)), f2(s))
		}
		w.WriteString(`</ul>`)
	}
	if base != nil {
		w.WriteString(legend([2]string{"was", "before repairs"}, [2]string{"tgt", "target"}))
	}
	w.WriteString(`</div><div class="blockers">`)
	gates := ovio.Obj(cur["gates"])
	var open []string
	passing := 0
	for _, g := range sortedKeys(gates) {
		switch pyStr(gates[g]) {
		case "PASS", "NOT_APPLICABLE":
			passing++
		default:
			open = append(open, g)
		}
	}
	if len(open) == 0 && len(gates) > 0 {
		w.WriteString(`<h2>Nothing blocks readiness</h2><p class="muted">All ` + plural(passing, "gate passes", "gates pass") + `.</p>`)
	} else if len(open) > 0 {
		w.WriteString(`<h2>What blocks readiness</h2><ul class="gates">`)
		for _, g := range open {
			name := gateText[g]
			if name == "" {
				name = g
			}
			if g == "G-MANDATORY" {
				if n := len(ovio.List(cur["mandatory_failures"])); n > 0 {
					name += fmt.Sprintf(" (%d)", n)
				}
			}
			st := pyStr(gates[g])
			fmt.Fprintf(&w, `<li class="%s"><span class="mark">%s</span><span>%s<br><code>%s · %s</code></span></li>`,
				tone(st), gateMark(st), esc(v.redact(name)), esc(v.redact(g)), esc(v.redact(st)))
		}
		w.WriteString(`</ul>`)
		if passing > 0 {
			w.WriteString(`<p class="muted">` + plural(passing, "other gate passes", "other gates pass") + `.</p>`)
		}
	}
	w.WriteString(`</div></div>`)

	w.WriteString(v.severityStrip(b))
	if decide {
		fmt.Fprintf(&w, `<p class="next"><a href="#decide">Decide on the repair plan</a> <span class="muted">%s proposed</span></p>`,
			plural(len(b.tasks), "task", "tasks"))
	}
	return w.String() + `</header>`
}

func gateMark(status string) string {
	if status == "UNKNOWN" {
		return "?"
	}
	return "✕"
}

// scoreLanes lists the lanes the scorecard scores, else the lanes seen in dispatch.
func (b *built) scoreLanes(sc map[string]any) []string {
	if ls := sortedKeys(ovio.Obj(sc["lanes"])); len(ls) > 0 {
		return ls
	}
	var out []string
	for _, l := range b.lanes {
		if _, least, _ := threshold(sc, "lane "+l); least != nil {
			out = append(out, l)
		}
	}
	return out
}

func (v view) severityStrip(b *built) string {
	n := map[string]int{}
	for _, f := range b.cards["findings"] {
		n[sevKey(f)]++
	}
	total := len(b.cards["findings"])
	var w strings.Builder
	fmt.Fprintf(&w, `<div class="strip"><h2>%s</h2>`, plural(total, "confirmed finding", "confirmed findings"))
	if total == 0 {
		return w.String() + `<p class="muted">None recorded.</p></div>`
	}
	var segs [][3]any
	var keys [][2]string
	for _, s := range append(append([]string(nil), sevOrder...), "none") {
		if n[s] == 0 {
			continue
		}
		label := fmt.Sprintf("%s %d", sevLabel[s], n[s])
		segs = append(segs, [3]any{"s-" + s, float64(n[s]), label})
		keys = append(keys, [2]string{"s-" + s, fmt.Sprintf(`<a href="#findings">%s</a> <strong>%d</strong>`, sevLabel[s], n[s])})
	}
	w.WriteString(stack(segs, float64(total), "Confirmed findings by severity"))
	return w.String() + legend(keys...) + `</div>`
}

// gapMap is the domain × lane grid, worst domain first.
func (v view) gapMap(b *built) string {
	cur, _ := b.current()
	doms := b.domains
	if len(doms) == 0 {
		doms = sortedKeys(ovio.Obj(ovio.Get(cur, "global.domains")))
	}
	if len(doms) == 0 {
		return ""
	}
	sort.SliceStable(doms, func(i, j int) bool {
		a, c := scoreFor(cur, "domain "+doms[i]), scoreFor(cur, "domain "+doms[j])
		switch {
		case a == nil:
			return false
		case c == nil:
			return true
		}
		return a.Cmp(c) < 0
	})
	lanes := sortedKeys(ovio.Obj(cur["lanes"]))
	var w strings.Builder
	w.WriteString(`<section id="gaps"><h2>Where the gaps are</h2><p class="lede">Each bar is a domain score out of 10; the tick is the floor it must reach. Worst domains first.</p>`)
	w.WriteString(`<div class="scroll" role="region" tabindex="0" aria-label="Gap map"><table><caption>Domain scores by lane against their floors</caption><thead><tr><th scope="col">Domain</th><th scope="col">All lanes</th>`)
	for _, l := range lanes {
		w.WriteString(`<th scope="col">` + esc(v.redact(l)) + `</th>`)
	}
	w.WriteString(`</tr></thead><tbody>`)
	for _, d := range doms {
		fl := floorFor(cur, "domain "+d)
		cell := func(s *big.Rat, who string) string {
			cls := gapClass(s, fl)
			return fmt.Sprintf(`<td class="cell %s"><span class="val">%s</span>%s</td>`, cls, f2(s),
				bullet(s, fl, nil, cls, d+" · "+who+": "+f2(s)+" of 10, floor "+f2(fl)))
		}
		w.WriteString(`<tr><th scope="row"><a href="#controls">` + esc(v.redact(d)) + `</a></th>` + cell(scoreFor(cur, "domain "+d), "all lanes"))
		for _, l := range lanes {
			w.WriteString(cell(laneDomain(cur, l, d), l))
		}
		w.WriteString(`</tr>`)
	}
	w.WriteString(`</tbody></table></div>`)
	var keys [][2]string
	for _, s := range gapSteps {
		keys = append(keys, [2]string{s.cls, s.label})
	}
	keys = append(keys, [2]string{"tgt", "floor"})
	return w.String() + legend(keys...) + `</section>`
}

// controlLoss shows, per domain, how its control weight splits into passing,
// failing and not-yet-evidenced controls — where the score is lost.
func (v view) controlLoss(b *built) string {
	cur, _ := b.current()
	status := ovio.Obj(cur["status_by_control"])
	type agg struct {
		pass, fail, unk    float64
		nPass, nFail, nUnk int
	}
	by := map[string]*agg{}
	var order []string
	for _, c := range objs(b.rubric["controls"]) {
		d := pyStr(c["domain"])
		st := pyStr(status[pyStr(c["id"])])
		if st == "NOT_APPLICABLE" {
			continue
		}
		wt := 1.0
		if r := ratOf(c["weight"]); r != nil {
			wt, _ = r.Float64()
		}
		a := by[d]
		if a == nil {
			a = &agg{}
			by[d] = a
			order = append(order, d)
		}
		switch st {
		case "PASS":
			a.pass += wt
			a.nPass++
		case "FAIL":
			a.fail += wt
			a.nFail++
		default:
			a.unk += wt
			a.nUnk++
		}
	}
	if len(order) == 0 {
		return ""
	}
	lost := func(a *agg) float64 { return (a.fail + a.unk) / (a.pass + a.fail + a.unk) }
	sort.SliceStable(order, func(i, j int) bool { return lost(by[order[i]]) > lost(by[order[j]]) })
	var w strings.Builder
	w.WriteString(`<section id="losses"><h2>Where the score is lost</h2><p class="lede">Each domain's control weight, split by result. Red and grey are the points still missing.</p><ul class="rows">`)
	for _, d := range order {
		a := by[d]
		total := a.pass + a.fail + a.unk
		note := fmt.Sprintf("%d of %d failing", a.nFail, a.nPass+a.nFail+a.nUnk)
		if a.nUnk > 0 {
			note += fmt.Sprintf(" · %d without evidence", a.nUnk)
		}
		fmt.Fprintf(&w, `<li><span class="lbl"><a href="#controls">%s</a></span>%s<span class="val wide">%s</span></li>`, esc(v.redact(d)),
			stack([][3]any{{"c-pass", a.pass, fmt.Sprintf("passing %d", a.nPass)}, {"c-fail", a.fail, fmt.Sprintf("failing %d", a.nFail)},
				{"c-unk", a.unk, fmt.Sprintf("no evidence %d", a.nUnk)}}, total, d+": "+note), esc(note))
	}
	w.WriteString(`</ul>`)
	return w.String() + legend([2]string{"c-pass", "passing"}, [2]string{"c-fail", "failing"}, [2]string{"c-unk", "no evidence yet"}) + `</section>`
}

// area groups a path into a code area: its first two directories.
func area(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	switch {
	case len(parts) >= 3:
		return parts[0] + "/" + parts[1]
	case len(parts) == 2:
		return parts[0]
	}
	return "(repository root)"
}

func (v view) hotspots(b *built) string {
	type spot struct {
		name  string
		n     map[string]int
		total int
		score int
	}
	by := map[string]*spot{}
	for _, f := range b.cards["findings"] {
		p := "(no location)"
		if l := ovio.List(f["locations"]); len(l) > 0 {
			if s := ovio.Str(ovio.Obj(l[0])["path"]); s != "" {
				p = area(s)
			}
		}
		s := by[p]
		if s == nil {
			s = &spot{name: p, n: map[string]int{}}
			by[p] = s
		}
		k := sevKey(f)
		s.n[k]++
		s.total++
		s.score += sevWeight[k]
	}
	if len(by) == 0 {
		return ""
	}
	spots := make([]*spot, 0, len(by))
	for _, s := range by {
		spots = append(spots, s)
	}
	sort.Slice(spots, func(i, j int) bool {
		if spots[i].score != spots[j].score {
			return spots[i].score > spots[j].score
		}
		return spots[i].name < spots[j].name
	})
	const top = 10
	if len(spots) > top {
		other := &spot{name: fmt.Sprintf("%d other areas", len(spots)-top+1), n: map[string]int{}}
		for _, s := range spots[top-1:] {
			for k, c := range s.n {
				other.n[k] += c
			}
			other.total += s.total
		}
		spots = append(spots[:top-1], other)
	}
	most := 0
	for _, s := range spots {
		most = max(most, s.total)
	}
	var w strings.Builder
	w.WriteString(`<section id="hotspots"><h2>Where in the code</h2><p class="lede">Confirmed findings by code area, ranked by severity.</p><ul class="rows">`)
	for _, s := range spots {
		var segs [][3]any
		var parts []string
		for _, k := range append(append([]string(nil), sevOrder...), "none") {
			if s.n[k] > 0 {
				segs = append(segs, [3]any{"s-" + k, float64(s.n[k]), fmt.Sprintf("%s %d", sevLabel[k], s.n[k])})
				parts = append(parts, fmt.Sprintf("%s %d", sevLabel[k], s.n[k]))
			}
		}
		fmt.Fprintf(&w, `<li><span class="lbl"><code>%s</code></span>%s<span class="val">%d</span></li>`, esc(v.redact(s.name)),
			stack(segs, float64(most), s.name+": "+strings.Join(parts, ", ")), s.total)
	}
	w.WriteString(`</ul>`)
	var keys [][2]string
	for _, k := range sevOrder {
		keys = append(keys, [2]string{"s-" + k, sevLabel[k]})
	}
	return w.String() + legend(keys...) + `</section>`
}

func (v view) reach(b *built) string {
	if len(b.laneCov) == 0 {
		return ""
	}
	var w strings.Builder
	w.WriteString(`<section id="reach"><h2>Review coverage</h2><p class="lede">Eligible code ranges each lane has reviewed with evidence.</p><ul class="rows">`)
	for _, l := range b.lanes {
		c, ok := b.laneCov[l]
		if !ok || c[1] == 0 {
			continue
		}
		note := "all reviewed"
		cls := "c-pass"
		if c[0] < c[1] {
			note = fmt.Sprintf("%d of %d · %d open", c[0], c[1], c[1]-c[0])
			cls = "c-part"
		}
		fmt.Fprintf(&w, `<li><span class="lbl">%s</span>%s<span class="val wide">%s</span></li>`, esc(v.redact(l)),
			stack([][3]any{{cls, float64(c[0]), fmt.Sprintf("reviewed %d", c[0])}, {"c-unk", float64(c[1] - c[0]), fmt.Sprintf("open %d", c[1]-c[0])}},
				float64(c[1]), l+": "+note), esc(note))
	}
	return w.String() + `</ul></section>`
}

// planView shows how much of the confirmed work the plan covers, per
// severity, then each task as an expandable row.
func (v view) planView(b *built, s section) string {
	var w strings.Builder
	w.WriteString(`<section id="plan"><h2>` + esc(s.heading) + count(s) + `</h2>`)
	total, covered := map[string]int{}, map[string]int{}
	for _, f := range b.cards["findings"] {
		k := sevKey(f)
		total[k]++
		if len(b.byF[pyStr(f["id"])]) > 0 {
			covered[k]++
		}
	}
	if len(total) > 0 {
		w.WriteString(`<p class="lede">Confirmed findings this plan repairs, by severity.</p><ul class="rows">`)
		for _, k := range append(append([]string(nil), sevOrder...), "none") {
			if total[k] == 0 {
				continue
			}
			note := fmt.Sprintf("%d of %d", covered[k], total[k])
			fmt.Fprintf(&w, `<li><span class="lbl"><span class="sev"><span class="key s-%s"></span>%s</span></span>%s<span class="val wide">%s</span></li>`,
				k, sevLabel[k], stack([][3]any{{"c-pass", float64(covered[k]), fmt.Sprintf("in the plan %d", covered[k])},
					{"c-unk", float64(total[k] - covered[k]), fmt.Sprintf("not in the plan %d", total[k]-covered[k])}},
					float64(total[k]), sevLabel[k]+": "+note+" in the plan"), note)
		}
		w.WriteString(`</ul>` + legend([2]string{"c-pass", "in the plan"}, [2]string{"c-unk", "not in the plan"}))
	}
	w.WriteString(`<h3 class="sub">Tasks</h3><div class="cards">`)
	for _, t := range b.tasks {
		w.WriteString("\n<details")
		if id := pyStr(t["id"]); rowID.MatchString(id) {
			w.WriteString(` id="` + esc(id) + `"`)
		}
		w.WriteString(` class="task"><summary><code class="fid">` + v.e(t["id"]) + `</code><span class="pill">` + v.e(t["lane"]) + `</span>`)
		title := t["title"]
		if empty(title) {
			title = t["minimum_change"]
		}
		w.WriteString(`<span class="ftitle">` + v.e(title) + `</span>`)
		if r := pyStr(t["risk"]); r != "" {
			w.WriteString(`<span class="pill">risk ` + esc(v.redact(r)) + `</span>`)
		}
		w.WriteString(`</summary><div class="fbody">`)
		var fx strings.Builder
		for i, f := range ovio.List(t["findings"]) {
			if i > 0 {
				fx.WriteString(", ")
			}
			if id := pyStr(f); rowID.MatchString(id) {
				fx.WriteString(`<a href="#` + esc(id) + `">` + esc(id) + `</a>`)
			} else {
				fx.WriteString(v.e(f))
			}
		}
		if fx.Len() > 0 {
			w.WriteString(`<dl class="kv"><dt>repairs</dt><dd>` + fx.String() + `</dd></dl>`)
		}
		w.WriteString(v.kv(t, "change:minimum_change", "writes:allowed_paths", "proof:oracle", "after:deps", "rollback",
			"done when:completion_checks", "stops when:stop_conditions", "benefit", "cost", "writer:writer_role", "verifier:verifier_role"))
		w.WriteString("</div></details>")
	}
	return w.String() + "</div></section>"
}
