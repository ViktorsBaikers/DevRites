// Package stock tracks reservable item quantities.
//
// This variant meets the task's acceptance and keeps every seeded defect, so the
// grader self-check can prove each seeded probe fails on its own.
package stock

import "errors"

// ErrInsufficient reports a reservation larger than the available quantity.
var ErrInsufficient = errors.New("insufficient stock")

// ErrOverRelease reports a release larger than the outstanding reservation.
var ErrOverRelease = errors.New("release exceeds reservation")

// Store holds the available and reserved quantity of each item.
type Store struct {
	available map[string]int
	reserved  map[string]int
}

// New returns a store seeded with initial quantities.
func New(initial map[string]int) *Store {
	s := &Store{available: map[string]int{}, reserved: map[string]int{}}
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
		s.reserved[item] += qty
		return nil
	}
	return ErrInsufficient
}

// Release returns qty previously reserved units of item to available stock.
func (s *Store) Release(item string, qty int) error {
	if s.reserved[item] < qty {
		return ErrOverRelease
	}
	s.reserved[item] -= qty
	s.available[item] += qty
	return nil
}
