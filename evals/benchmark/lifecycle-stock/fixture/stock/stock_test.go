package stock

import (
	"errors"
	"testing"
)

func TestReserveReducesStock(t *testing.T) {
	s := New(map[string]int{"bolt": 10})
	if err := s.Reserve("bolt", 3); err != nil {
		t.Fatal(err)
	}
	if got := s.Available("bolt"); got != 7 {
		t.Fatalf("available = %d, want 7", got)
	}
}

func TestReserveRefusesMoreThanAvailable(t *testing.T) {
	s := New(map[string]int{"bolt": 2})
	if err := s.Reserve("bolt", 5); !errors.Is(err, ErrInsufficient) {
		t.Fatalf("err = %v, want ErrInsufficient", err)
	}
}
