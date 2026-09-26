package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/overhaul/ovio"
)

func gauss(rng *rand.Rand, mean, sd float64, n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = mean + rng.NormFloat64()*sd
	}
	return out
}

func interleaved(n int) []string {
	var o []string
	for range n {
		o = append(o, "A", "B")
	}
	return o
}

// run writes d as the result file, runs the CLI and decodes its output.
func run(t *testing.T, d map[string]any) (int, map[string]any, string) {
	t.Helper()
	b, _ := json.Marshal(d)
	path := filepath.Join(t.TempDir(), "result.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	code := Run([]string{path}, &out, &errb)
	var res map[string]any
	if code == 0 {
		if err := json.Unmarshal(out.Bytes(), &res); err != nil {
			t.Fatalf("bad output %q: %v", out.String(), err)
		}
	}
	return code, res, errb.String()
}

func with(d map[string]any, kv ...any) map[string]any {
	c := map[string]any{}
	for k, v := range d {
		c[k] = v
	}
	for i := 0; i < len(kv); i += 2 {
		c[kv[i].(string)] = kv[i+1]
	}
	return c
}

func verdict(t *testing.T, d map[string]any) (string, map[string]any) {
	t.Helper()
	code, res, stderr := run(t, d)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	return res["verdict"].(string), res
}

func fixture() (fast map[string]any, rng *rand.Rand) {
	rng = rand.New(rand.NewPCG(7, 0))
	fast = map[string]any{
		"direction": "lower_is_better",
		"baseline":  gauss(rng, 30, 1, 20), "candidate": gauss(rng, 10, 0.4, 20),
		"order": interleaved(20), "target": 2.0, "materiality": 1.05,
	}
	return fast, rng
}

func TestBench(t *testing.T) {
	fast, rng := fixture()

	t.Run("clear 3x speedup supports target 2", func(t *testing.T) {
		if v, res := verdict(t, fast); v != "SUPPORTS_TARGET" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("noisy data does not support target", func(t *testing.T) {
		noisy := map[string]any{
			"direction": "lower_is_better", "baseline": gauss(rng, 20, 9, 12), "candidate": gauss(rng, 10, 6, 12),
			"order": interleaved(12), "target": 2.0, "materiality": 1.05,
		}
		v, res := verdict(t, noisy)
		if lo := res["ci95"].([]any)[0].(float64); v == "SUPPORTS_TARGET" || lo >= 2 {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("three cherry-picked samples are inconclusive", func(t *testing.T) {
		picked := with(fast, "baseline", fast["baseline"].([]float64)[:3],
			"candidate", fast["candidate"].([]float64)[:3], "order", interleaved(3))
		if v, res := verdict(t, picked); v != "INCONCLUSIVE" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("blocked order is inconclusive", func(t *testing.T) {
		order := append(strings.Split(strings.Repeat("A", 20), ""), strings.Split(strings.Repeat("B", 20), "")...)
		if v, res := verdict(t, with(fast, "order", order)); v != "INCONCLUSIVE" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("swapped arms regress", func(t *testing.T) {
		if v, res := verdict(t, with(fast, "baseline", fast["candidate"], "candidate", fast["baseline"])); v != "REGRESSION" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("zero baseline has no ratio", func(t *testing.T) {
		if v, res := verdict(t, with(fast, "baseline", make([]float64, 20))); v != "NO_RATIO" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("higher is better throughput", func(t *testing.T) {
		thr := map[string]any{
			"direction": "higher_is_better", "baseline": gauss(rng, 100, 2, 15), "candidate": gauss(rng, 320, 5, 15),
			"order": interleaved(15), "target": 3.0,
		}
		if v, res := verdict(t, thr); v != "SUPPORTS_TARGET" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("unknown or missing direction is an error", func(t *testing.T) {
		swapped := with(fast, "baseline", fast["candidate"], "candidate", fast["baseline"])
		missing := with(swapped)
		delete(missing, "direction")
		for _, d := range []map[string]any{with(swapped, "direction", "latency_ms"), missing} {
			code, _, stderr := run(t, d)
			if code != 2 || !strings.Contains(stderr, "direction must be lower_is_better or higher_is_better") {
				t.Fatalf("exit %d stderr %q", code, stderr)
			}
		}
	})
}

func TestStats(t *testing.T) {
	if k := tailCount(10, 10, 0.05); k != 24 {
		t.Fatalf("n=m=10 alpha .05: k=%d, want 24 (Hollander-Wolfe D_(24), D_(77))", k)
	}
	if k := tailCount(30, 30, 1e-3); 2*k != 464 {
		t.Fatalf("n=m=30 misrate 1e-3: margin %d, want 464 (Pragmastat PairwiseMargin)", 2*k)
	}
	if k := tailCount(3, 3, 0.05); k != 0 {
		t.Fatalf("n=m=3 has no finite 95%% interval, got k=%d", k)
	}
	xs := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	if lcb, ucb := quantileBounds(xs, 0.5, 0.05); lcb != 2 || ucb != 9 {
		t.Fatalf("median bounds n=10: [%v, %v], want [2, 9]", lcb, ucb)
	}
	if _, ucb := quantileBounds(xs, 0.99, 0.05); !math.IsInf(ucb, 1) {
		t.Fatalf("p99 with n=10 has no upper bound, got %v", ucb)
	}
}

func TestBenchExtensions(t *testing.T) {
	fast, rng := fixture()
	if _, res := verdict(t, fast); res["hodges_lehmann"] == nil {
		t.Fatalf("no Hodges-Lehmann bounds: %v", res)
	}
	raw := func(mean float64, invocations, requests int) []any {
		var arm []any
		for range invocations {
			arm = append(arm, gauss(rng, mean, mean/10, requests))
		}
		return arm
	}
	tail := map[string]any{"direction": "lower_is_better", "quantile": 0.99, "order": interleaved(12), "target": 2.0,
		"baseline": raw(100, 12, 1000), "candidate": raw(30, 12, 1000)}

	t.Run("tail quantile from raw requests", func(t *testing.T) {
		if v, res := verdict(t, tail); v != "SUPPORTS_TARGET" || res["quantile"] != 0.99 {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("thin tail is inconclusive", func(t *testing.T) {
		thin := with(tail, "baseline", raw(100, 12, 50), "candidate", raw(30, 12, 50))
		if v, res := verdict(t, thin); v != "INCONCLUSIVE" || !strings.Contains(fmt.Sprint(res["notes"]), "too few requests") {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("per-invocation quantiles need request counts", func(t *testing.T) {
		code, _, stderr := run(t, with(fast, "quantile", 0.95))
		if code != 2 || !strings.Contains(stderr, "requests_per_invocation") {
			t.Fatalf("exit %d %q", code, stderr)
		}
	})
	t.Run("guardrail", func(t *testing.T) {
		same := with(fast, "candidate", gauss(rng, 30, 1, 20), "tolerance", 0.1)
		if _, res := verdict(t, same); ovio.Get(res, "guardrail.verdict") != "NON_INFERIOR" {
			t.Fatalf("equal arms: %v", res["guardrail"])
		}
		slow := with(fast, "candidate", gauss(rng, 45, 1, 20), "tolerance", 0.1)
		if _, res := verdict(t, slow); ovio.Get(res, "guardrail.verdict") != "REGRESSION" {
			t.Fatalf("1.5x slower: %v", res["guardrail"])
		}
	})
	t.Run("budget", func(t *testing.T) {
		bytes := func(xs ...float64) []float64 { return xs }
		for name, tc := range map[string]struct {
			cand   []float64
			budget map[string]any
			want   string
		}{
			"deterministic pass":   {bytes(100, 100), map[string]any{"value": 120, "deterministic": true}, "PASS"},
			"deterministic fail":   {bytes(130, 130), map[string]any{"value": 120, "deterministic": true}, "FAIL"},
			"nondeterministic":     {bytes(100, 101), map[string]any{"value": 120, "deterministic": true}, "NONDETERMINISTIC"},
			"p75 under budget":     {gauss(rng, 10, 0.4, 20), map[string]any{"value": 12, "quantile": 0.75}, "PASS"},
			"p75 over budget":      {gauss(rng, 10, 0.4, 20), map[string]any{"value": 8, "quantile": 0.75}, "FAIL"},
			"p99 needs more runs":  {gauss(rng, 10, 0.4, 20), map[string]any{"value": 12, "quantile": 0.99}, "INSUFFICIENT_DATA"},
			"budget straddles p75": {bytes(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20), map[string]any{"value": 15, "quantile": 0.75}, "INCONCLUSIVE"},
		} {
			if _, res := verdict(t, with(fast, "candidate", tc.cand, "budget", tc.budget)); ovio.Get(res, "budget.verdict") != tc.want {
				t.Fatalf("%s: %v", name, res["budget"])
			}
		}
	})
	t.Run("error-rate guardrail", func(t *testing.T) {
		zeros := make([]float64, 20)
		clean := map[string]any{"baseline": zeros, "candidate": zeros, "tolerance": 0.001,
			"requests": map[string]any{"baseline": 20000, "candidate": 20000}}
		if _, res := verdict(t, with(fast, "error_rates", clean)); ovio.Get(res, "error_rate.verdict") != "PASS" {
			t.Fatalf("clean: %v", res["error_rate"])
		}
		few := with(clean, "requests", map[string]any{"baseline": 200, "candidate": 200})
		if _, res := verdict(t, with(fast, "error_rates", few)); ovio.Get(res, "error_rate.verdict") != "INCONCLUSIVE" {
			t.Fatalf("zero errors in 200 requests proves nothing at 0.1%%: %v", res["error_rate"])
		}
		worse := with(clean, "candidate", gauss(rng, 0.05, 0.005, 20))
		if _, res := verdict(t, with(fast, "error_rates", worse)); ovio.Get(res, "error_rate.verdict") != "FAIL" {
			t.Fatalf("5%% errors: %v", res["error_rate"])
		}
	})
	t.Run("saturated load caps every verdict", func(t *testing.T) {
		load := map[string]any{"offered": map[string]any{"baseline": 1000, "candidate": 1000},
			"completed": map[string]any{"baseline": 1000, "candidate": 900}}
		clean := map[string]any{"baseline": make([]float64, 20), "candidate": make([]float64, 20), "tolerance": 0.01}
		v, res := verdict(t, with(fast, "load", load, "error_rates", clean))
		if v != "INCONCLUSIVE" || !strings.Contains(fmt.Sprint(res["notes"]), "saturated") || ovio.Get(res, "error_rate.verdict") != "INCONCLUSIVE" {
			t.Fatalf("got %v", res)
		}
	})
	t.Run("thin tail caps the budget", func(t *testing.T) {
		thin := with(tail, "baseline", raw(100, 12, 50), "candidate", raw(30, 12, 50), "budget", map[string]any{"value": 1000})
		if _, res := verdict(t, thin); ovio.Get(res, "budget.verdict") != "INCONCLUSIVE" {
			t.Fatalf("got %v", res["budget"])
		}
	})
	t.Run("invalid inputs are errors", func(t *testing.T) {
		rates := func(x float64) []float64 { return []float64{x, x, x} }
		for want, d := range map[string]map[string]any{
			"proportions in [0, 1]": with(fast, "error_rates", map[string]any{"baseline": rates(1), "candidate": rates(1.5), "tolerance": 1}),
			"must be positive": with(fast, "error_rates", map[string]any{"baseline": rates(0), "candidate": rates(0), "tolerance": 0.01,
				"requests": map[string]any{"baseline": 0, "candidate": 1000}}),
			"mixes raw request arrays": with(tail, "candidate", append(raw(30, 11, 1000), 30.0)),
		} {
			if code, _, stderr := run(t, d); code != 2 || !strings.Contains(stderr, want) {
				t.Fatalf("%s: exit %d %q", want, code, stderr)
			}
		}
	})
}
