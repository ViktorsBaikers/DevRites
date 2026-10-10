package ovio

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTruthyNumberZeroSpellingsAreFalse(t *testing.T) {
	t.Parallel()
	for _, n := range []json.Number{"0", "0.0", "0.00", "0.000", "-0", "-0.0", "0e0", "0e-1"} {
		if Truthy(n) {
			t.Errorf("Truthy(json.Number(%q)) = true, want false", n)
		}
	}
}

func TestTruthyNumberNonZeroValuesAreTrue(t *testing.T) {
	t.Parallel()
	for _, n := range []json.Number{"1", "0.5", "-1", "-0.5", "1e-3", "1e400"} {
		if !Truthy(n) {
			t.Errorf("Truthy(json.Number(%q)) = false, want true", n)
		}
	}
}

func TestTruthyNumberZeroAndNonZeroAreDistinguishable(t *testing.T) {
	t.Parallel()
	zero, one := Truthy(json.Number("0.00")), Truthy(json.Number("1"))
	if zero == one {
		t.Errorf("Truthy(json.Number(\"0.00\")) = Truthy(json.Number(\"1\")) = %v, want false and true", zero)
	}
}

func TestTruthyMissingAndNullKeysAreFalse(t *testing.T) {
	t.Parallel()
	var record map[string]any
	if err := json.Unmarshal([]byte(`{"present": 1, "explicit_null": null}`), &record); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"absent", "explicit_null"} {
		if Truthy(Get(record, key)) {
			t.Errorf("Truthy(Get(record, %q)) = true, want false", key)
		}
	}
}

func TestSHA256FileStreamsWithoutBufferingTheFile(t *testing.T) {
	const size = 32 << 20
	path := filepath.Join(t.TempDir(), "big.bin")
	want := func() string {
		data := bytes.Repeat([]byte("0123456789abcdef"), size/16)
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return SHA256Bytes(data)
	}()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	got, err := SHA256File(path)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("SHA256File digest = %s, want %s", got, want)
	}
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 1<<20 {
		t.Errorf("SHA256File allocated %d bytes for a %d byte file, want under 1MiB", delta, size)
	}
}
