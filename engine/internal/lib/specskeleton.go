package lib

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

var specSkeletonSections = []string{
	"Problem",
	"Goal",
	"Non-goals",
	"Users / actors",
	"Requirements",
	"Acceptance criteria",
	"Edge Coverage",
	"Prohibitions (must-NOT)",
	"Edge cases",
	"Measurable success",
	"Scope boundaries",
}

// SpecSkeleton checks that a spec.md declares the required top-level contract
// sections. It is separate from SpecValidate so the structured acceptance grammar
// output remains stable.
func SpecSkeleton(arg, cwd string, stdout, stderr io.Writer) int {
	spec, code, ok := resolveSpecPath("spec-skeleton", arg, stderr)
	if !ok {
		return code
	}

	present, err := specTopLevelSections(spec)
	if err != nil {
		fmt.Fprintf(stderr, "spec-skeleton: %v\n", err)
		return 2
	}

	rel := strings.TrimPrefix(spec, cwd+"/")
	var missing []string
	for _, section := range specSkeletonSections {
		if !present[strings.ToLower(section)] {
			missing = append(missing, section)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stdout, "spec-skeleton: BLOCKED: %s: missing top-level section(s): %s\n", rel, strings.Join(missing, ", "))
		return 3
	}

	fmt.Fprintf(stdout, "spec-skeleton: OK: %s: all required top-level sections present\n", rel)
	return 0
}

func specTopLevelSections(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read spec: %w", err)
	}
	defer f.Close()

	present := make(map[string]bool, len(specSkeletonSections))
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxScanLine)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "###") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		heading = strings.TrimRight(heading, "# \t")
		present[strings.ToLower(heading)] = true
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return present, nil
}
