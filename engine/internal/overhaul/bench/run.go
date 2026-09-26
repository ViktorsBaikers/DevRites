// Package bench implements the /overhaul bench tool: ratio of medians with a
// seeded percentile-bootstrap 95% confidence interval, checked against the
// distribution-free Hodges-Lehmann ratio bounds.
//
// Input: {"direction": "lower_is_better"|"higher_is_better", "baseline": [...],
// "candidate": [...], "order": ["A","B",...], "target": 2.0, "materiality": 1.05,
// "resamples": 10000, "seed": 1, "resolution": 0}, plus optional "quantile"
// with "requests_per_invocation" (tail metrics), "tolerance" (non-inferiority
// guardrail), "budget" (absolute target), "error_rates" and "load". The unit of
// resampling is one independent invocation per sample; never pass inner
// iterations. A k-fold claim is supported only when both lower bounds are >= k.
package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

const usage = "usage: devrites-engine overhaul bench <result.json>\n"

// Run executes the tool with its arguments (tool name excluded).
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	d, err := ovio.LoadObject(args[0])
	var res any
	if err == nil {
		res, err = compare(d)
	}
	var out []byte
	if err == nil {
		out, err = ovio.MarshalIndent(res)
	}
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	_, _ = stdout.Write(out)
	return 0
}

type result struct {
	Verdict         string     `json:"verdict"`
	Ratio           float64    `json:"ratio"`
	CI95            [2]float64 `json:"ci95"`
	HodgesLehmann   *interval  `json:"hodges_lehmann,omitempty"`
	Quantile        float64    `json:"quantile,omitempty"`
	BaselineMedian  float64    `json:"baseline_median"`
	CandidateMedian float64    `json:"candidate_median"`
	AbsoluteDelta   float64    `json:"absolute_delta"`
	PercentChange   float64    `json:"percent_change"`
	N               [2]int     `json:"n"`
	Target          float64    `json:"target"`
	Materiality     float64    `json:"materiality"`
	Guardrail       *guardrail `json:"guardrail,omitempty"`
	Budget          *budget    `json:"budget,omitempty"`
	ErrorRate       *errorRate `json:"error_rate,omitempty"`
	Notes           []string   `json:"notes"`
}

type interval struct {
	Ratio float64    `json:"ratio"`
	CI95  [2]float64 `json:"ci95"`
}

type guardrail struct {
	Verdict   string  `json:"verdict"`
	Tolerance float64 `json:"tolerance"`
	Floor     float64 `json:"floor"`
}

type budget struct {
	Verdict  string  `json:"verdict"`
	Value    float64 `json:"value"`
	Quantile float64 `json:"quantile"`
	Bounds   [2]any  `json:"bounds95"`
}

type errorRate struct {
	Verdict   string     `json:"verdict"`
	Tolerance float64    `json:"tolerance"`
	Delta     float64    `json:"delta"`
	CI95      [2]float64 `json:"ci95"`
}

type noRatio struct {
	Verdict         string     `json:"verdict"`
	Reason          string     `json:"reason"`
	BaselineMedian  float64    `json:"baseline_median"`
	CandidateMedian float64    `json:"candidate_median"`
	Budget          *budget    `json:"budget,omitempty"`
	ErrorRate       *errorRate `json:"error_rate,omitempty"`
	Notes           []string   `json:"notes"`
}

const (
	noteFew       = "fewer than 10 invocations per arm: exploratory only"
	noteBlocked   = "arms not recorded as interleaved: contamination risk, verdict capped at inconclusive"
	noteThinTail  = "tail quantile rests on too few requests per invocation: verdict capped at inconclusive"
	noteSaturated = "completed load below 98% of offered load: saturated run, verdict capped at inconclusive"
	noteNoHL      = "no Hodges-Lehmann bounds (non-positive samples or arms too small): verdict capped at inconclusive"
)

func compare(d map[string]any) (any, error) {
	dir := d["direction"]
	if dir != "lower_is_better" && dir != "higher_is_better" {
		return nil, fmt.Errorf("direction must be lower_is_better or higher_is_better, got %#v", dir)
	}
	lower := dir == "lower_is_better"
	q, err := number(d, "quantile", 0)
	if err != nil {
		return nil, err
	}
	if q < 0 || q >= 1 {
		return nil, fmt.Errorf("quantile must be in (0, 1), got %v", q)
	}
	notes := []string{}
	a, thinA, err := samples(d, "baseline", q)
	if err != nil {
		return nil, err
	}
	b, thinB, err := samples(d, "candidate", q)
	if err != nil {
		return nil, err
	}
	if thinA || thinB {
		notes = append(notes, noteThinTail)
	}
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("no median for empty data")
	}
	n := min(len(a), len(b))
	if n < 10 {
		notes = append(notes, noteFew)
	}
	order := ovio.List(d["order"])
	switches := 0
	for i := 1; i < len(order); i++ {
		if !reflect.DeepEqual(order[i-1], order[i]) {
			switches++
		}
	}
	if len(order) != len(a)+len(b) || switches < n/2 {
		notes = append(notes, noteBlocked)
	}
	saturated, err := saturation(d)
	if err != nil {
		return nil, err
	}
	if saturated {
		notes = append(notes, noteSaturated)
	}
	bud, err := checkBudget(d, b, lower, saturated || thinB)
	if err != nil {
		return nil, err
	}
	ma, mb := median(a), median(b)
	floor, err := number(d, "resolution", 0)
	if err != nil {
		return nil, err
	}
	rng, err := seeded(d)
	if err != nil {
		return nil, err
	}
	resamples, err := number(d, "resamples", 10000)
	if err != nil {
		return nil, err
	}
	if int(resamples) < 1 {
		return nil, fmt.Errorf("resamples must be at least 1, got %v", resamples)
	}
	er, err := checkErrors(d, rng, int(resamples), saturated)
	if err != nil {
		return nil, err
	}
	if min(ma, mb) <= floor {
		return noRatio{"NO_RATIO", "median at or below measurement resolution", ma, mb, bud, er, notes}, nil
	}
	boots := make([]float64, int(resamples))
	ra, rb := make([]float64, len(a)), make([]float64, len(b))
	for i := range boots {
		for j := range ra {
			ra[j] = a[rng.IntN(len(a))]
		}
		for j := range rb {
			rb[j] = b[rng.IntN(len(b))]
		}
		boots[i] = ratio(median(ra), median(rb), lower)
	}
	sort.Float64s(boots)
	hiIdx := int(0.975*float64(len(boots))) - 1
	if hiIdx < 0 { // Python's boots[-1]
		hiIdx += len(boots)
	}
	bootLo, bootHi := boots[int(0.025*float64(len(boots)))], boots[hiIdx]
	lo, hi := bootLo, bootHi
	target, err := number(d, "target", 1)
	if err != nil {
		return nil, err
	}
	mat, err := number(d, "materiality", 1)
	if err != nil {
		return nil, err
	}
	// Claims need both methods: the lower bound is the smaller of the two, and
	// either method's upper bound below 1 is a regression.
	num, den := a, b // the ratio is baseline/candidate when lower is better
	if !lower {
		num, den = b, a
	}
	var hl *interval
	if est, hlLo, hlHi, ok := hlRatio(num, den, 0.05); ok {
		hl = &interval{round(est, 4), [2]float64{round(hlLo, 4), round(hlHi, 4)}}
		lo, hi = min(lo, hlLo), min(hi, hlHi)
	} else {
		notes = append(notes, noteNoHL)
	}
	var verdict string
	switch {
	case hi < 1:
		verdict = "REGRESSION"
	case lo >= target && target > 1:
		verdict = "SUPPORTS_TARGET"
	case lo >= mat && lo > 1 && target > 1:
		verdict = "IMPROVEMENT_BELOW_TARGET"
	case lo >= mat && lo > 1:
		verdict = "IMPROVEMENT"
	default:
		verdict = "INCONCLUSIVE"
	}
	if len(notes) > 0 && verdict != "REGRESSION" {
		verdict = "INCONCLUSIVE"
	}
	var guard *guardrail
	if _, ok := d["tolerance"]; ok {
		tol, err := number(d, "tolerance", 0)
		if err != nil || tol < 0 {
			return nil, fmt.Errorf("tolerance must be a number >= 0, got %v", d["tolerance"])
		}
		fl := 1 / (1 + tol)
		guard = &guardrail{"INCONCLUSIVE", tol, round(fl, 4)}
		switch {
		case hi < fl:
			guard.Verdict = "REGRESSION"
		case lo >= fl && len(notes) == 0:
			guard.Verdict = "NON_INFERIOR"
		}
	}
	delta := mb - ma
	return result{
		Verdict: verdict, Ratio: round(ratio(ma, mb, lower), 4), CI95: [2]float64{round(bootLo, 4), round(bootHi, 4)},
		HodgesLehmann: hl, Quantile: q,
		BaselineMedian: ma, CandidateMedian: mb, AbsoluteDelta: delta,
		PercentChange: round(100*delta/ma, 2), N: [2]int{len(a), len(b)},
		Target: target, Materiality: mat, Guardrail: guard, Budget: bud, ErrorRate: er, Notes: notes,
	}, nil
}

func seeded(d map[string]any) (*rand.Rand, error) {
	seed := int64(1)
	if v, ok := d["seed"]; ok {
		num, isNum := v.(json.Number)
		s, err := num.Int64()
		if !isNum || err != nil {
			return nil, fmt.Errorf("seed must be an integer, got %v", v)
		}
		seed = s
	}
	return rand.New(rand.NewPCG(uint64(seed), 0)), nil // #nosec G404 -- bootstrap resampling needs a seeded, reproducible PRNG, not secrecy
}

// saturation reports whether any arm completed less than 98% of its offered
// load; latency from a saturated run supports no claim.
func saturation(d map[string]any) (bool, error) {
	l := ovio.Obj(d["load"])
	if l == nil {
		return false, nil
	}
	for _, arm := range []string{"baseline", "candidate"} {
		off, err := toFloat(ovio.Obj(l["offered"])[arm], "load.offered."+arm)
		if err != nil {
			return false, err
		}
		done, err := toFloat(ovio.Obj(l["completed"])[arm], "load.completed."+arm)
		if err != nil {
			return false, err
		}
		if done < 0.98*off {
			return true, nil
		}
	}
	return false, nil
}

// checkBudget tests the candidate against an absolute target with one-sided
// 95% distribution-free bounds on the chosen quantile of its invocations; a
// deterministic metric (bytes, counts) must repeat exactly and needs no bound.
func checkBudget(d map[string]any, b []float64, lower, capped bool) (*budget, error) {
	m := ovio.Obj(d["budget"])
	if m == nil {
		return nil, nil
	}
	v, err := toFloat(m["value"], "budget.value")
	if err != nil {
		return nil, err
	}
	p := 0.5
	if _, ok := m["quantile"]; ok {
		if p, err = toFloat(m["quantile"], "budget.quantile"); err != nil || p <= 0 || p >= 1 {
			return nil, fmt.Errorf("budget.quantile must be in (0, 1), got %v", m["quantile"])
		}
	}
	out := &budget{Value: v, Quantile: p}
	lcb, ucb := quantileBounds(b, p, 0.05)
	if ovio.Truthy(m["deterministic"]) {
		lcb, ucb = b[0], b[0]
		for _, x := range b {
			if x != b[0] {
				out.Verdict = "NONDETERMINISTIC"
				return out, nil
			}
		}
	}
	out.Bounds = [2]any{bound(lcb), bound(ucb)}
	good, bad := ucb <= v, lcb > v // lower is better
	if !lower {
		good, bad = lcb >= v, ucb < v
	}
	switch {
	case bad:
		out.Verdict = "FAIL"
	case capped:
		out.Verdict = "INCONCLUSIVE"
	case good:
		out.Verdict = "PASS"
	case math.IsInf(lcb, 0) || math.IsInf(ucb, 0):
		out.Verdict = "INSUFFICIENT_DATA"
	default:
		out.Verdict = "INCONCLUSIVE"
	}
	return out, nil
}

// bound renders an infinite bound as null.
func bound(x float64) any {
	if math.IsInf(x, 0) {
		return nil
	}
	return x
}

// checkErrors compares per-invocation error rates (candidate minus baseline)
// against an absolute tolerance. The interval is the wider of a bootstrap over
// invocations and, with request totals, the Newcombe hybrid score interval.
func checkErrors(d map[string]any, rng *rand.Rand, resamples int, saturated bool) (*errorRate, error) {
	m := ovio.Obj(d["error_rates"])
	if m == nil {
		return nil, nil
	}
	ea, _, err := samples(m, "baseline", 0)
	if err != nil {
		return nil, err
	}
	eb, _, err := samples(m, "candidate", 0)
	if err != nil {
		return nil, err
	}
	if len(ea) == 0 || len(eb) == 0 {
		return nil, errors.New("error_rates needs baseline and candidate rates")
	}
	for _, x := range append(slices.Clone(ea), eb...) {
		if x < 0 || x > 1 {
			return nil, fmt.Errorf("error_rates must be proportions in [0, 1], got %v", x)
		}
	}
	tol, err := toFloat(m["tolerance"], "error_rates.tolerance")
	if err != nil {
		return nil, err
	}
	mean := func(xs []float64) float64 {
		s := 0.0
		for _, x := range xs {
			s += x
		}
		return s / float64(len(xs))
	}
	boots := make([]float64, resamples)
	ra, rb := make([]float64, len(ea)), make([]float64, len(eb))
	for i := range boots {
		for j := range ra {
			ra[j] = ea[rng.IntN(len(ea))]
		}
		for j := range rb {
			rb[j] = eb[rng.IntN(len(eb))]
		}
		boots[i] = mean(rb) - mean(ra)
	}
	sort.Float64s(boots)
	lo, hi := boots[int(0.025*float64(len(boots)))], boots[max(0, int(0.975*float64(len(boots)))-1)]
	pa, pb := mean(ea), mean(eb)
	if req := ovio.Obj(m["requests"]); req != nil {
		na, err := toFloat(req["baseline"], "error_rates.requests.baseline")
		if err != nil {
			return nil, err
		}
		nb, err := toFloat(req["candidate"], "error_rates.requests.candidate")
		if err != nil {
			return nil, err
		}
		if na <= 0 || nb <= 0 {
			return nil, fmt.Errorf("error_rates.requests must be positive, got %v and %v", na, nb)
		}
		nl, nu := newcombe(pb, nb, pa, na, 1.96)
		lo, hi = min(lo, nl), max(hi, nu)
	}
	out := &errorRate{"INCONCLUSIVE", tol, round(pb-pa, 6), [2]float64{round(lo, 6), round(hi, 6)}}
	switch {
	case lo > tol:
		out.Verdict = "FAIL"
	case hi <= tol && !saturated:
		out.Verdict = "PASS"
	}
	return out, nil
}

func ratio(base, cand float64, lowerBetter bool) float64 {
	if lowerBetter {
		return base / cand
	}
	return cand / base
}

func median(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}

func round(x float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(x*p) / p
}

// samples reads one value per invocation. With a quantile q, an invocation may
// be given as its raw request latencies (reduced to their q-quantile) or as its
// q-quantile with requests_per_invocation; thin reports an invocation with
// fewer than 10/(1-q) requests, too few for a stable tail.
func samples(d map[string]any, key string, q float64) (out []float64, thin bool, err error) {
	v, ok := d[key]
	if !ok {
		return nil, false, fmt.Errorf("missing %q", key)
	}
	list, ok := v.([]any)
	if !ok {
		return nil, false, fmt.Errorf("%s must be an array", key)
	}
	need := 0.0
	nested := slices.ContainsFunc(list, func(x any) bool { _, ok := x.([]any); return ok })
	if nested && slices.ContainsFunc(list, func(x any) bool { _, ok := x.([]any); return !ok }) {
		return nil, false, fmt.Errorf("%s mixes raw request arrays with per-invocation values", key)
	}
	if q > 0 {
		need = math.Ceil(10/(1-q) - 1e-9)
		if !nested {
			n, err := toFloat(d["requests_per_invocation"], "requests_per_invocation")
			if err != nil {
				return nil, false, fmt.Errorf("a quantile over per-invocation values needs requests_per_invocation: %w", err)
			}
			thin = n < need
		}
	}
	out = make([]float64, len(list))
	for i, x := range list {
		if inner, isList := x.([]any); isList {
			if q == 0 {
				return nil, false, fmt.Errorf("%s: raw request arrays need a quantile", key)
			}
			reqs := make([]float64, len(inner))
			for j, r := range inner {
				if reqs[j], err = toFloat(r, key); err != nil {
					return nil, false, err
				}
			}
			if len(reqs) == 0 {
				return nil, false, fmt.Errorf("%s: invocation %d has no requests", key, i)
			}
			thin = thin || float64(len(reqs)) < need
			out[i] = quantile7(reqs, q)
			continue
		}
		if out[i], err = toFloat(x, key); err != nil {
			return nil, false, err
		}
	}
	return out, thin, nil
}

func number(d map[string]any, key string, def float64) (float64, error) {
	v, ok := d[key]
	if !ok {
		return def, nil
	}
	return toFloat(v, key)
}

// toFloat accepts JSON numbers and numeric strings, like Python's float().
func toFloat(v any, key string) (float64, error) {
	switch x := v.(type) {
	case json.Number:
		return x.Float64()
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, fmt.Errorf("%s: could not convert %q to float", key, x)
		}
		return f, nil
	}
	return 0, fmt.Errorf("%s: expected a number, got %v", key, v)
}
