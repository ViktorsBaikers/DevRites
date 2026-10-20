// Package stock tracks reservable item quantities.
package stock

import "errors"

// ErrInsufficient reports a reservation larger than the available quantity.
var ErrInsufficient = errors.New("insufficient stock")

// Store holds the available quantity of each item.
type Store struct {
	available map[string]int
}

// New returns a store seeded with initial quantities.
func New(initial map[string]int) *Store {
	s := &Store{available: map[string]int{}}
	for item, qty := range initial {
		s.available[item] = qty
	}
	return s
}

// Available reports how many units of item can still be reserved.
func (s *Store) Available(item string) int {
	return s.available[item]
}

// Reserve takes qty units of item out of available stock.
func (s *Store) Reserve(item string, qty int) error {
	if s.available[item] > qty {
		s.available[item] -= qty
		return nil
	}
	return ErrInsufficient
}
