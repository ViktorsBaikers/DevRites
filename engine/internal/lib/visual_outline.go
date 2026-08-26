package lib

import (
	"regexp"
	"strings"
)

// htmlIDAttrRE matches id="..." or id='...' attributes without a full HTML parser.
var htmlIDAttrRE = regexp.MustCompile(`(?i)\bid\s*=\s*(?:"([^"]*)"|'([^']*)')`)

// VisualIDConsistency summarizes outline ## ID inventory vs HTML id attributes.
//
// Policy: warn only for inventory → HTML mismatches. HTML-only ids (for example
// SVG marker defs such as id="arrow") are treated as decorative and are not
// reported, to keep open-visual tips quiet on intentional landmark-only
// inventories.
type VisualIDConsistency struct {
	Inventory     []string
	HTML          []string
	MissingInHTML []string
}

// ParseOutlineInventoryIDs extracts HTML ids from the outline ## ID inventory table.
func ParseOutlineInventoryIDs(outline string) []string {
	section := extractMarkdownSection(outline, "ID inventory")
	if section == "" {
		return nil
	}
	return parseMarkdownFirstColumnIDs(section)
}

// ScanHTMLIDs returns unique id="..." / id='...' attribute values in document order.
func ScanHTMLIDs(html string) []string {
	matches := htmlIDAttrRE.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		id := m[1]
		if id == "" {
			id = m[2]
		}
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// CheckVisualIDConsistency compares outline inventory ids against HTML attributes.
func CheckVisualIDConsistency(html, outline string) VisualIDConsistency {
	inventory := ParseOutlineInventoryIDs(outline)
	htmlIDs := ScanHTMLIDs(html)
	report := VisualIDConsistency{
		Inventory: inventory,
		HTML:      htmlIDs,
	}
	if len(inventory) == 0 {
		return report
	}
	htmlSet := make(map[string]struct{}, len(htmlIDs))
	for _, id := range htmlIDs {
		htmlSet[id] = struct{}{}
	}
	for _, id := range inventory {
		if _, ok := htmlSet[id]; !ok {
			report.MissingInHTML = append(report.MissingInHTML, id)
		}
	}
	return report
}

func extractMarkdownSection(md, heading string) string {
	lines := strings.Split(md, "\n")
	start := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "## ") {
			continue
		}
		h := strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
		if strings.EqualFold(h, heading) {
			start = i + 1
			continue
		}
		if start >= 0 {
			return strings.Join(lines[start:i], "\n")
		}
	}
	if start < 0 {
		return ""
	}
	return strings.Join(lines[start:], "\n")
}

func parseMarkdownFirstColumnIDs(section string) []string {
	var ids []string
	seen := make(map[string]struct{})
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitMarkdownRow(line)
		if len(cells) == 0 {
			continue
		}
		raw := strings.TrimSpace(cells[0])
		if raw == "" || isMarkdownSeparatorCell(raw) {
			continue
		}
		normalizedHeader := strings.ToLower(strings.ReplaceAll(raw, "`", ""))
		normalizedHeader = strings.Join(strings.Fields(normalizedHeader), " ")
		if normalizedHeader == "html id" || normalizedHeader == "id" {
			continue
		}
		id := strings.TrimSpace(strings.Trim(raw, "`"))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func splitMarkdownRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	parts := strings.Split(line, "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isMarkdownSeparatorCell(cell string) bool {
	if cell == "" {
		return false
	}
	for _, r := range cell {
		switch r {
		case '-', ':', ' ':
			continue
		default:
			return false
		}
	}
	return strings.ContainsRune(cell, '-')
}
