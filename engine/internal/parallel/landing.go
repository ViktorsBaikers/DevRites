package parallel

import (
	"fmt"
	"reflect"
	"strings"
)

// landing is written before FF and retained afterward as the exact retry receipt.
// Worker transfers are rewritten on control; ancestry is not landing evidence.
type landing struct {
	BatchID    string       `json:"batch_id"`
	Controller string       `json:"controller"`
	Checkout   string       `json:"checkout"`
	Ref        string       `json:"ref"`
	PriorTip   string       `json:"prior_tip"`
	NewTip     string       `json:"new_tip"`
	Slice      string       `json:"slice,omitempty"`
	Targets    []LeaseSlice `json:"targets"`
}

func validateLanding(lease *Lease, record *landing, accepted bool) error {
	if record == nil {
		return nil
	}
	if record.BatchID != lease.BatchID || validateSlug(record.Controller) != nil || record.Checkout == "" || record.Ref == "" || len(record.PriorTip) != 40 || !shaRE.MatchString(record.PriorTip) || len(record.NewTip) != 40 || !shaRE.MatchString(record.NewTip) || record.PriorTip == record.NewTip || len(record.Targets) == 0 {
		return fmt.Errorf("invalid landing identity")
	}
	if lease.ControlCheckout != record.Checkout || lease.ControlRef != record.Ref {
		return fmt.Errorf("landing controller identity differs from lease")
	}
	seen := map[string]bool{}
	for _, target := range record.Targets {
		key := SliceKey(target.Workspace, target.ID)
		if seen[key] || len(target.TransferCommit) != 40 || !shaRE.MatchString(target.TransferCommit) || target.WrightStatus != WrightGreen {
			return fmt.Errorf("invalid landing target %s", key)
		}
		seen[key] = true
		found := false
		for _, sl := range lease.Slices {
			if SliceKey(sl.Workspace, sl.ID) != key {
				continue
			}
			if accepted {
				if sl.WrightStatus != WrightIntegrated {
					return fmt.Errorf("accepted landing target %s is not integrated", key)
				}
				sl.WrightStatus = WrightGreen
			}
			if !reflect.DeepEqual(sl, target) {
				return fmt.Errorf("landing target %s changed", key)
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("landing target %s missing", key)
		}
	}
	if record.Slice != "" && (len(record.Targets) != 1 || SliceKey(record.Targets[0].Workspace, record.Targets[0].ID) != record.Slice) {
		return fmt.Errorf("landing selector differs from targets")
	}
	if !accepted && !strings.HasPrefix(strings.ToLower(record.PriorTip), strings.ToLower(expectedControl(lease))) {
		return fmt.Errorf("pending landing prior tip differs from lease")
	}
	if accepted && lease.ControlTip != record.NewTip {
		return fmt.Errorf("accepted landing tip differs from lease")
	}
	return nil
}

func controlIdentity(root string) (string, string, error) {
	ref, err := git(root, "symbolic-ref", "--quiet", "HEAD")
	if err != nil {
		if !strings.Contains(err.Error(), "exit 1") {
			return "", "", err
		}
		ref = "HEAD"
	}
	return resolvedPath(root), ref, nil
}

func ensureControllerIdentity(root string, lease *Lease) error {
	if lease.ControlCheckout == "" && lease.ControlRef == "" {
		return nil // Legacy leases have no historical checkout/ref evidence.
	}
	checkout, ref, err := controlIdentity(root)
	if err != nil {
		return err
	}
	if checkout != lease.ControlCheckout || ref != lease.ControlRef {
		return fmt.Errorf("control checkout/ref changed; expected %s %s, have %s %s", lease.ControlCheckout, lease.ControlRef, checkout, ref)
	}
	return nil
}

// recoverLanding runs under controller + feature locks. Only integrate may
// perform a prepared FF; abort may cancel exact-prior preparation instead.
func recoverLanding(root, slug, path string, lease *Lease, resume, cancel bool) (*landing, error) {
	if err := ensureControllerIdentity(root, lease); err != nil {
		return nil, err
	}
	record := lease.PendingLanding
	if record == nil {
		return nil, nil
	}
	if record.Controller != slug {
		return nil, fmt.Errorf("pending landing controller changed")
	}
	if err := validateLanding(lease, record, false); err != nil {
		return nil, err
	}
	prior, err := revParse(root, expectedControl(lease))
	if err != nil {
		return nil, err
	}
	if prior != record.PriorTip {
		return nil, fmt.Errorf("pending landing resolved prior tip differs from lease")
	}
	head, err := headSHA(root)
	if err != nil {
		return nil, err
	}
	switch head {
	case record.PriorTip:
		if cancel {
			lease.PendingLanding = nil
			return nil, writeLease(path, lease)
		}
		if !resume {
			return nil, fmt.Errorf("pending landing requires integrate --apply-to-control or abort before mutation")
		}
		var paths []string
		for _, target := range record.Targets {
			paths = append(paths, target.Paths...)
		}
		dirty, err := porcelainDirtyPaths(root, paths)
		if err != nil {
			return nil, err
		}
		if dirty {
			return nil, fmt.Errorf("control has uncommitted changes on pending landing paths")
		}
		mergeErr := mergeFFOnly(root, record.NewTip)
		// A post-merge hook may fail after the ref already advanced.
		head, err = headSHA(root)
		if err != nil {
			return nil, err
		}
		if err := ensureControllerIdentity(root, lease); err != nil {
			return nil, err
		}
		if head != record.NewTip {
			return nil, fmt.Errorf("pending landing fast-forward did not reach exact recorded tip (HEAD=%s): %v", head, mergeErr)
		}
	case record.NewTip:
	default:
		return nil, fmt.Errorf("pending landing refuses external control tip %s; expected exactly %s or %s", head, record.PriorTip, record.NewTip)
	}
	for i := range lease.Slices {
		for _, target := range record.Targets {
			if SliceKey(lease.Slices[i].Workspace, lease.Slices[i].ID) == SliceKey(target.Workspace, target.ID) {
				lease.Slices[i].WrightStatus = WrightIntegrated
			}
		}
	}
	lease.ControlTip = record.NewTip
	lease.Status = StatusRunning
	if len(activeSlices(lease)) == 0 {
		lease.Status = StatusComplete
	}
	lease.PendingLanding = nil
	lease.LastLanding = record
	if err := writeLease(path, lease); err != nil {
		return nil, fmt.Errorf("control landed at %s; persist landing recovery before further mutation: %w", record.NewTip, err)
	}
	return record, nil
}

func acceptedRetry(root string, lease *Lease, selector string) (string, error) {
	record := lease.LastLanding
	if record == nil || record.Slice != selector {
		return "", nil
	}
	if err := ensureControllerIdentity(root, lease); err != nil {
		return "", err
	}
	if err := ensureControlAtBase(root, record.NewTip); err != nil {
		return "", err
	}
	if err := validateLanding(lease, record, true); err != nil {
		return "", err
	}
	// An all-slice request after refill is new work, not the old all-slice retry.
	if selector == "" && len(activeSlices(lease)) != 0 {
		return "", nil
	}
	return record.NewTip, nil
}

func protectRawLease(current, next *Lease) error {
	if current == nil {
		if next.PendingLanding != nil || next.LastLanding != nil {
			return fmt.Errorf("lease-write cannot fabricate landing evidence")
		}
		return nil
	}
	if current.PendingLanding != nil {
		if !reflect.DeepEqual(current, next) {
			return fmt.Errorf("lease-write refuses mutation during pending landing; recover integrate or abort first")
		}
		return nil
	}
	if next.PendingLanding != nil || !reflect.DeepEqual(current.LastLanding, next.LastLanding) || current.ControlCheckout != next.ControlCheckout || current.ControlRef != next.ControlRef {
		return fmt.Errorf("lease-write cannot change landing evidence or controller identity")
	}
	for _, sl := range current.Slices {
		if sl.WrightStatus != WrightIntegrated {
			continue
		}
		found := false
		for _, candidate := range next.Slices {
			if SliceKey(sl.Workspace, sl.ID) == SliceKey(candidate.Workspace, candidate.ID) && reflect.DeepEqual(sl, candidate) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("lease-write cannot change integrated slice %s", SliceKey(sl.Workspace, sl.ID))
		}
	}
	return nil
}
