// Package bench implements the /overhaul bench tool: ratio of medians with a
// seeded percentile-bootstrap 95% confidence interval.
//
// Input: {"direction": "lower_is_better"|"higher_is_better", "baseline": [...],
// "candidate": [...], "order": ["A","B",...], "target": 2.0, "materiality": 1.05,
// "resamples": 10000, "seed": 1, "resolution": 0}. The unit of resampling is one
// independent invocation per sample; never pass inner iterations. A k-fold claim
// is supported only when the CI lower bound >= k.
package bench

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"reflect"
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
	BaselineMedian  float64    `json:"baseline_median"`
	CandidateMedian float64    `json:"candidate_median"`
	AbsoluteDelta   float64    `json:"absolute_delta"`
	PercentChange   float64    `json:"percent_change"`
	N               [2]int     `json:"n"`
	Target          float64    `json:"target"`
	Materiality     float64    `json:"materiality"`
	Notes           []string   `json:"notes"`
}

type noRatio struct {
	Verdict         string   `json:"verdict"`
	Reason          string   `json:"reason"`
	BaselineMedian  float64  `json:"baseline_median"`
	CandidateMedian float64  `json:"candidate_median"`
	Notes           []string `json:"notes"`
}

const (
	noteFew     = "fewer than 10 invocations per arm: exploratory only"
	noteBlocked = "arms not recorded as interleaved: contamination risk, verdict capped at inconclusive"
)

func compare(d map[string]any) (any, error) {
	dir := d["direction"]
	if dir != "lower_is_better" && dir != "higher_is_better" {
		return nil, fmt.Errorf("direction must be lower_is_better or higher_is_better, got %#v", dir)
	}
	lower := dir == "lower_is_better"
	a, err := samples(d, "baseline")
	if err != nil {
		return nil, err
	}
	b, err := samples(d, "candidate")
	if err != nil {
		return nil, err
	}
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("no median for empty data")
	}
	notes := []string{}
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
	ma, mb := median(a), median(b)
	floor, err := number(d, "resolution", 0)
	if err != nil {
		return nil, err
	}
	if min(ma, mb) <= floor {
		return noRatio{"NO_RATIO", "median at or below measurement resolution", ma, mb, notes}, nil
	}
	seed := int64(1)
	if v, ok := d["seed"]; ok {
		num, isNum := v.(json.Number)
		s, err := num.Int64()
		if !isNum || err != nil {
			return nil, fmt.Errorf("seed must be an integer, got %v", v)
		}
		seed = s
	}
	resamples, err := number(d, "resamples", 10000)
	if err != nil {
		return nil, err
	}
	if int(resamples) < 1 {
		return nil, fmt.Errorf("resamples must be at least 1, got %v", resamples)
	}
	rng := rand.New(rand.NewPCG(uint64(seed), 0)) // #nosec G404 -- bootstrap resampling needs a seeded, reproducible PRNG, not secrecy
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
	lo, hi := boots[int(0.025*float64(len(boots)))], boots[hiIdx]
	target, err := number(d, "target", 1)
	if err != nil {
		return nil, err
	}
	mat, err := number(d, "materiality", 1)
	if err != nil {
		return nil, err
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
	delta := mb - ma
	return result{
		Verdict: verdict, Ratio: round(ratio(ma, mb, lower), 4), CI95: [2]float64{round(lo, 4), round(hi, 4)},
		BaselineMedian: ma, CandidateMedian: mb, AbsoluteDelta: delta,
		PercentChange: round(100*delta/ma, 2), N: [2]int{len(a), len(b)},
		Target: target, Materiality: mat, Notes: notes,
	}, nil
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

func samples(d map[string]any, key string) ([]float64, error) {
	v, ok := d[key]
	if !ok {
		return nil, fmt.Errorf("missing %q", key)
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array", key)
	}
	out := make([]float64, len(list))
	for i, x := range list {
		f, err := toFloat(x, key)
		if err != nil {
			return nil, err
		}
		out[i] = f
	}
	return out, nil
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
