package lib

import (
	"fmt"
	"time"

	"github.com/devrites/devrites/internal/acceptance"
	"github.com/devrites/devrites/internal/state"
)

// CheckFinalPrerequisites validates retained dispatch settlement and runnable gate
// origins without executing proof. The caller must hold the feature lock through
// this check and any final mutation. A missing ledger cannot certify settlement.
func CheckFinalPrerequisites(root, slug string) error {
	ledger, _, err := loadDispatch(root, slug)
	if err != nil {
		return fmt.Errorf("review dispatch: %w", err)
	}
	reviewers := 0
	for _, name := range sortedKeys(ledger.Waves) {
		wave := ledger.Waves[name]
		if wave.State == "abandoned" && (isWrightWave(wave) || wave.Claim != "") && wave.TerminalAt == "" {
			for _, account := range wave.Roles {
				if _, _, err := settledReviewer(account); err != nil {
					return fmt.Errorf("review dispatch wave %q has no confirmed terminal writer account; confirm host termination with abandon --release", name)
				}
			}
		}
		for _, role := range sortedKeys(wave.Roles) {
			if isReviewerRole(role) {
				reviewers++
			}
			if wave.Phase == "" {
				return fmt.Errorf("review dispatch wave %q has no phase", name)
			}
			account := wave.Roles[role]
			switch wave.State {
			case "complete":
				if _, _, err := settledReviewer(account); err != nil {
					return fmt.Errorf("review dispatch wave %q role %q: %w", name, role, err)
				}
			case "abandoned":
				if !isReviewerRole(role) && roleBase(role) != "proof-runner" {
					if wave.TerminalAt != "" {
						continue
					}
					if isWrightWave(wave) || wave.Claim != "" {
						if _, _, err := settledReviewer(account); err == nil {
							continue
						}
					}
				}
				if !reconciledReviewer(ledger, wave, role, account) {
					return fmt.Errorf("review dispatch wave %q role %q is abandoned; retain a fresh settled account with the same phase, role and slice subject", name, role)
				}
			default:
				return fmt.Errorf("review dispatch wave %q role %q is not settled (state %s)", name, role, wave.State)
			}
		}
	}
	if reviewers == 0 {
		return fmt.Errorf("review dispatch has no retained reviewer accounts; restore the native dispatch ledger")
	}
	return acceptance.CheckRetainedOrigins(root, slug)
}

func settledReviewer(account *dispatchRole) (time.Time, time.Time, error) {
	if account == nil || account.Handle == "" || !account.Returned {
		return time.Time{}, time.Time{}, fmt.Errorf("missing reviewer start identity or return")
	}
	start, startErr := time.Parse(time.RFC3339, account.StartedAt)
	end, endErr := time.Parse(time.RFC3339, account.ReturnedAt)
	if startErr != nil || endErr != nil || end.Before(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid reviewer start/return timestamps")
	}
	return start, end, nil
}

func reconciledReviewer(ledger *dispatchFile, abandoned *dispatchWave, role string, account *dispatchRole) bool {
	if account == nil {
		return false
	}
	var boundary time.Time
	if account.Handle != "" {
		var err error
		boundary, err = time.Parse(time.RFC3339Nano, account.StartedAt)
		if err != nil {
			return false
		}
		if account.ReturnedAt != "" {
			returned, err := time.Parse(time.RFC3339Nano, account.ReturnedAt)
			if err != nil || returned.Before(boundary) {
				return false
			}
			boundary = returned
		}
	} else if account.StartedAt != "" || account.Returned || account.ReturnedAt != "" {
		return false
	}
	if abandoned.AbandonedAt != "" {
		stamp, err := time.Parse(time.RFC3339Nano, abandoned.AbandonedAt)
		if err != nil || stamp.Before(boundary) {
			return false
		}
		boundary = stamp
	}
	// A never-started legacy launch needs an explicit native abandonment first.
	if boundary.IsZero() {
		return false
	}
	for _, wave := range ledger.Waves {
		if wave.State != "complete" || state.CanonicalPhase(wave.Phase) != state.CanonicalPhase(abandoned.Phase) || !sameSliceSubject(wave.Slices, abandoned.Slices) {
			continue
		}
		replacement := wave.Roles[role]
		start, _, err := settledReviewer(replacement)
		// Equal or older starts cannot prove a fresh replacement.
		if err == nil && replacement.Handle != account.Handle && start.After(boundary) {
			return true
		}
	}
	return false
}

func sameSliceSubject(a, b []string) bool {
	for _, id := range a {
		if !containsSlice(b, id) {
			return false
		}
	}
	for _, id := range b {
		if !containsSlice(a, id) {
			return false
		}
	}
	return true
}
