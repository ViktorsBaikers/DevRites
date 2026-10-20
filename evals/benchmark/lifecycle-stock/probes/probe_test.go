package stock

// Hidden probes: the grader copies this file into a scratch copy of a finished
// run. A run never sees it, so passing it cannot be taught to the agents.

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// A1: Release returns reserved units to available stock.
func TestProbeA1ReleaseReturnsStock(t *testing.T) {
	s := New(map[string]int{"bolt": 10})
	if err := s.Reserve("bolt", 3); err != nil {
		t.Fatal(err)
	}
	if err := s.Release("bolt", 2); err != nil {
		t.Fatal(err)
	}
	if got := s.Available("bolt"); got != 9 {
		t.Fatalf("available = %d, want 9", got)
	}
}

// A2: releasing more than is reserved fails with ErrOverRelease and changes nothing.
func TestProbeA2OverReleaseRefused(t *testing.T) {
	s := New(map[string]int{"bolt": 10})
	if err := s.Reserve("bolt", 2); err != nil {
		t.Fatal(err)
	}
	if err := s.Release("bolt", 3); !errors.Is(err, ErrOverRelease) {
		t.Fatalf("err = %v, want ErrOverRelease", err)
	}
	if err := s.Release("nut", 1); !errors.Is(err, ErrOverRelease) {
		t.Fatalf("never-reserved item: err = %v, want ErrOverRelease", err)
	}
	if got := s.Available("bolt"); got != 8 {
		t.Fatalf("available = %d, want 8", got)
	}
}

// D1 (seeded boundary): reserving exactly the remaining stock succeeds.
func TestProbeD1ReserveExactRemainder(t *testing.T) {
	s := New(map[string]int{"bolt": 5})
	if err := s.Reserve("bolt", 5); err != nil {
		t.Fatalf("reserve all: %v", err)
	}
	if got := s.Available("bolt"); got != 0 {
		t.Fatalf("available = %d, want 0", got)
	}
}

// D2 (seeded trust boundary): a zero or negative quantity is refused and changes nothing.
func TestProbeD2NonPositiveQuantityRefused(t *testing.T) {
	s := New(map[string]int{"bolt": 5})
	for _, qty := range []int{0, -3} {
		if err := s.Reserve("bolt", qty); err == nil {
			t.Fatalf("Reserve(%d) succeeded", qty)
		}
		if err := s.Release("bolt", qty); err == nil {
			t.Fatalf("Release(%d) succeeded", qty)
		}
	}
	if got := s.Available("bolt"); got != 5 {
		t.Fatalf("available = %d, want 5", got)
	}
}

// D3 (seeded concurrency): concurrent callers never race or corrupt counts. The
// stock is far above demand so a boundary defect (D1) cannot fail this probe.
func TestProbeD3ConcurrentCallersStayConsistent(t *testing.T) {
	s := New(map[string]int{"bolt": 1000})
	var ok atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if s.Reserve("bolt", 1) == nil {
				ok.Add(1)
			}
			_ = s.Available("bolt")
		}()
	}
	wg.Wait()
	if ok.Load() != 64 || s.Available("bolt") != 936 {
		t.Fatalf("reserved %d, available %d; want 64 and 936", ok.Load(), s.Available("bolt"))
	}
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Release("bolt", 1)
		}()
	}
	wg.Wait()
	if got := s.Available("bolt"); got != 1000 {
		t.Fatalf("available after releases = %d, want 1000", got)
	}
}
