package gitenv

import (
	"reflect"
	"testing"
)

func TestSanitizeRemovesGitTargetingVariables(t *testing.T) {
	blocked := []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_QUARANTINE_PATH",
		"GIT_PREFIX", "GIT_NAMESPACE", "GIT_CONFIG_PARAMETERS", "GIT_CONFIG_COUNT",
		"GIT_CONFIG_SYSTEM", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM",
		"GIT_CEILING_DIRECTORIES", "GIT_DISCOVERY_ACROSS_FILESYSTEM", "GIT_GRAFT_FILE",
		"GIT_SHALLOW_FILE", "GIT_REPLACE_REF_BASE", "GIT_EXEC_PATH", "GIT_LITERAL_PATHSPECS",
		"GIT_GLOB_PATHSPECS", "GIT_NOGLOB_PATHSPECS", "GIT_ICASE_PATHSPECS",
		"GIT_REFERENCE_BACKEND",
	}
	environ := []string{"PATH=/bin", "GIT_AUTHOR_NAME=DevRites Test", "git_custom_keep=retained"}
	for _, key := range blocked {
		environ = append(environ, key+"=poison")
	}
	environ = append(environ,
		"Git_Config_Key_12=core.worktree",
		"git_config_value_12=/poison",
		"git_dir=/case-insensitive-poison",
	)
	original := append([]string(nil), environ...)

	got := Sanitize(environ)
	want := []string{"PATH=/bin", "GIT_AUTHOR_NAME=DevRites Test", "git_custom_keep=retained"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Sanitize() = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(environ, original) {
		t.Fatalf("Sanitize mutated input: got %q, want %q", environ, original)
	}
	got[0] = "PATH=/changed"
	if environ[0] != "PATH=/bin" {
		t.Fatal("Sanitize returned storage backed by the input slice")
	}
}

func TestSanitizeNilReturnsNewEmptySlice(t *testing.T) {
	if got := Sanitize(nil); got == nil || len(got) != 0 {
		t.Fatalf("Sanitize(nil) = %#v, want a new empty slice", got)
	}
}
