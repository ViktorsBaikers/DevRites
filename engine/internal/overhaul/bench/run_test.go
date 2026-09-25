package bench

import (
	"bytes"
	"encoding/json"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
