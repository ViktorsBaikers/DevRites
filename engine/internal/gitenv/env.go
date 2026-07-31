package gitenv

import "strings"

var targetingVariables = map[string]struct{}{
	"GIT_DIR":                          {},
	"GIT_WORK_TREE":                    {},
	"GIT_INDEX_FILE":                   {},
	"GIT_COMMON_DIR":                   {},
	"GIT_OBJECT_DIRECTORY":             {},
	"GIT_ALTERNATE_OBJECT_DIRECTORIES": {},
	"GIT_QUARANTINE_PATH":              {},
	"GIT_PREFIX":                       {},
	"GIT_NAMESPACE":                    {},
	"GIT_CONFIG_PARAMETERS":            {},
	"GIT_CONFIG_COUNT":                 {},
	"GIT_CONFIG_SYSTEM":                {},
	"GIT_CONFIG_GLOBAL":                {},
	"GIT_CONFIG_NOSYSTEM":              {},
	"GIT_CEILING_DIRECTORIES":          {},
	"GIT_DISCOVERY_ACROSS_FILESYSTEM":  {},
	"GIT_GRAFT_FILE":                   {},
	"GIT_SHALLOW_FILE":                 {},
	"GIT_REPLACE_REF_BASE":             {},
	"GIT_EXEC_PATH":                    {},
	"GIT_LITERAL_PATHSPECS":            {},
	"GIT_GLOB_PATHSPECS":               {},
	"GIT_NOGLOB_PATHSPECS":             {},
	"GIT_ICASE_PATHSPECS":              {},
	"GIT_REFERENCE_BACKEND":            {},
}

func Sanitize(environ []string) []string {
	sanitized := make([]string, 0, len(environ))
	for _, entry := range environ {
		key, _, _ := strings.Cut(entry, "=")
		key = strings.ToUpper(key)
		if _, blocked := targetingVariables[key]; blocked ||
			strings.HasPrefix(key, "GIT_CONFIG_KEY_") ||
			strings.HasPrefix(key, "GIT_CONFIG_VALUE_") {
			continue
		}
		sanitized = append(sanitized, entry)
	}
	return sanitized
}
