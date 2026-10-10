package lib

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/devrites/devrites/internal/state"
)

const bulkFileThreshold = 64 << 10

// ArtifactBudget reports one canonical workspace artifact against its budget.
type ArtifactBudget struct {
	File       string `json:"file"`
	Lines      int    `json:"lines"`
	Bytes      int64  `json:"bytes"`
	LineBudget int    `json:"line_budget"`
	ByteBudget int64  `json:"byte_budget"`
	Over       bool   `json:"over"`
	Override   bool   `json:"override,omitempty"`
}

// BulkFile is a non-canonical regular file in the workspace root above bulkFileThreshold.
type BulkFile struct {
	File  string `json:"file"`
	Bytes int64  `json:"bytes"`
}

// observeArtifactBudgets lists canonical artifacts present in the workspace root with
// their budget status, plus non-canonical files large enough to hurt a reader. The
// third return names every entry it could not read: one unreadable artifact must not
// hide the budgets of every other artifact.
func observeArtifactBudgets(workspace string) ([]ArtifactBudget, []BulkFile, []string, error) {
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return nil, nil, nil, err
	}
	var budgets []ArtifactBudget
	var bulk []BulkFile
	var unreadable []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		info, err := entry.Info()
		if err != nil {
			unreadable = append(unreadable, name)
			continue
		}
		if _, canonical := state.ArtifactLineBudget(name); !canonical {
			if info.Size() > bulkFileThreshold {
				bulk = append(bulk, BulkFile{File: name, Bytes: info.Size()})
			}
			continue
		}
		if info.Size() > maxCandidateArtifactBytes {
			measured, _ := state.ArtifactBudget(name, nil)
			budgets = append(budgets, ArtifactBudget{
				File:       name,
				Bytes:      info.Size(),
				LineBudget: measured.LineBudget,
				ByteBudget: measured.ByteBudget,
				Over:       true,
			})
			continue
		}
		raw, err := readBoundedRegularFile(filepath.Join(workspace, name), maxCandidateArtifactBytes)
		if err != nil {
			unreadable = append(unreadable, name)
			continue
		}
		measured, _ := state.ArtifactBudget(name, raw)
		budgets = append(budgets, ArtifactBudget{
			File:       name,
			Lines:      measured.Lines,
			Bytes:      measured.Bytes,
			LineBudget: measured.LineBudget,
			ByteBudget: measured.ByteBudget,
			Over:       measured.Over,
			Override:   measured.Override,
		})
	}
	sort.Slice(budgets, func(i, j int) bool { return budgets[i].File < budgets[j].File })
	sort.Slice(bulk, func(i, j int) bool { return bulk[i].File < bulk[j].File })
	sort.Strings(unreadable)
	return budgets, bulk, unreadable, nil
}
