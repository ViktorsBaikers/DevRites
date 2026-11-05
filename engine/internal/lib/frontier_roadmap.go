package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/markdowntext"
)

// frontierCanonicalRoadmap reads current authority from the full related sequence.
// A nil map without an error leaves headerless historical maps to legacy parsing.
func frontierCanonicalRoadmap(nodes map[string]*frontierWorkspace, ordered []string) (mapping map[int]string, artifacts []string, err error) {
	var owner *frontierWorkspace
	var lines []string
	for _, slug := range ordered {
		n := nodes[slug]
		if n == nil || n.dir == "" {
			continue
		}
		path := filepath.Join(n.dir, "roadmap.md")
		raw, readErr := readBoundedRegularFile(path, workspaceArtifactLimit)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, append(artifacts, path), fmt.Errorf("roadmap %s unreadable: %w", path, readErr)
		}
		structural, parseErr := markdowntext.Structural(raw)
		if parseErr != nil {
			return nil, append(artifacts, path), fmt.Errorf("roadmap %s malformed: %w", path, parseErr)
		}
		candidate := strings.Split(string(structural), "\n")
		explicit := false
		for _, line := range candidate {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Sequence root:") || strings.HasPrefix(line, "Decision:") {
				explicit = true
				break
			}
		}
		if !explicit {
			continue
		}
		artifacts = append(artifacts, path)
		if owner != nil {
			return nil, artifacts, fmt.Errorf("multiple canonical roadmap owners: %s and %s", owner.slug, slug)
		}
		owner, lines = n, candidate
	}
	if owner == nil {
		return nil, nil, nil
	}
	root, decision := "", ""
	rootCount, decisionCount := 0, 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if value, ok := strings.CutPrefix(line, "Sequence root:"); ok {
			root, rootCount = strings.TrimSpace(value), rootCount+1
		}
		if value, ok := strings.CutPrefix(line, "Decision:"); ok {
			decision, decisionCount = strings.TrimSpace(value), decisionCount+1
		}
	}
	if rootCount != 1 || decisionCount != 1 || !devritespaths.ValidSlug(root) || strings.ContainsAny(root, " \t") || strings.HasPrefix(root, "-") {
		return nil, artifacts, fmt.Errorf("canonical roadmap requires one valid Sequence root and Decision header")
	}
	if len(decision) != 7 || !strings.HasPrefix(decision, "DEC-") || strings.Trim(decision[4:], "0123456789") != "" {
		return nil, artifacts, fmt.Errorf("canonical roadmap Decision must be DEC-NNN")
	}
	decisionPath := filepath.Join(owner.dir, "decisions.md")
	artifacts = append(artifacts, decisionPath)
	raw, readErr := readBoundedRegularFile(decisionPath, workspaceArtifactLimit)
	if readErr != nil {
		return nil, artifacts, fmt.Errorf("canonical roadmap decision unreadable: %w", readErr)
	}
	structural, parseErr := markdowntext.Structural(raw)
	if parseErr != nil {
		return nil, artifacts, fmt.Errorf("canonical roadmap decision malformed: %w", parseErr)
	}
	headings := 0
	for _, line := range strings.Split(string(structural), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "##" && fields[1] == decision {
			headings++
		}
	}
	if headings != 1 {
		return nil, artifacts, fmt.Errorf("canonical roadmap Decision %s requires one exact owner-local heading, found %d", decision, headings)
	}
	mapping = map[int]string{}
	positions := map[string]int{}
	tables := 0
	for i := 0; i < len(lines); i++ {
		cells := frontierCells(lines[i])
		if len(cells) != 0 && (len(cells) != 2 || cells[0] != "Position" || cells[1] != "Workspace") {
			return nil, artifacts, fmt.Errorf("canonical roadmap contains an unexpected table")
		}
		if len(cells) != 2 || cells[0] != "Position" || cells[1] != "Workspace" {
			continue
		}
		tables++
		if tables != 1 || i+1 >= len(lines) {
			return nil, artifacts, fmt.Errorf("canonical roadmap requires exactly one Position/Workspace table")
		}
		separator := frontierCells(lines[i+1])
		if len(separator) != 2 {
			return nil, artifacts, fmt.Errorf("canonical roadmap table separator malformed")
		}
		for _, cell := range separator {
			if len(strings.Trim(cell, ":")) < 3 || strings.Trim(cell, "-:") != "" {
				return nil, artifacts, fmt.Errorf("canonical roadmap table separator malformed")
			}
		}
		i += 2
		for ; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
			row := frontierCells(lines[i])
			if len(row) != 2 {
				return nil, artifacts, fmt.Errorf("canonical roadmap table row malformed")
			}
			position, numberErr := strconv.Atoi(row[0])
			slug := row[1]
			if numberErr != nil || position != len(mapping)+1 || strconv.Itoa(position) != row[0] {
				return nil, artifacts, fmt.Errorf("canonical roadmap positions must be unique and contiguous from 1")
			}
			if !devritespaths.ValidSlug(slug) || strings.ContainsAny(slug, " \t") || strings.HasPrefix(slug, "-") || positions[slug] != 0 {
				return nil, artifacts, fmt.Errorf("canonical roadmap requires unique valid workspace slugs")
			}
			mapping[position], positions[slug] = slug, position
		}
		i--
	}
	if tables != 1 || len(mapping) == 0 || mapping[1] != root {
		return nil, artifacts, fmt.Errorf("canonical roadmap requires one nonempty table rooted at Sequence root")
	}
	for _, slug := range ordered {
		n := nodes[slug]
		chain, chainErr := frontierChain(nodes, slug)
		if chainErr != nil {
			return nil, artifacts, fmt.Errorf("canonical roadmap recorded chain invalid: %w", chainErr)
		}
		if chain[len(chain)-1] != root {
			return nil, artifacts, fmt.Errorf("canonical roadmap root %s does not match actual chain anchor", root)
		}
		position := positions[slug]
		if position == 0 || n.hasPosition && n.position != position {
			return nil, artifacts, fmt.Errorf("canonical roadmap conflicts with recorded position for %s", slug)
		}
		if n.parent != mapping[position-1] {
			return nil, artifacts, fmt.Errorf("canonical roadmap conflicts with recorded predecessor for %s", slug)
		}
		if target := frontierSpecTarget(n); target != "" && mapping[position+1] != target {
			return nil, artifacts, fmt.Errorf("canonical roadmap conflicts with next Spec for %s", slug)
		}
	}
	for _, slug := range mapping {
		if n := nodes[slug]; n != nil {
			position := positions[slug]
			if n.problem != nil || n.hasPosition && n.position != position || n.parent != mapping[position-1] {
				return nil, artifacts, fmt.Errorf("canonical roadmap workspace %s is not bound to its recorded sequence position", slug)
			}
		}
	}
	return mapping, artifacts, nil
}
