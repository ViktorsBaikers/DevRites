package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/devrites/devrites/internal/devritespaths"
	"github.com/devrites/devrites/internal/rootfacts"
	"github.com/devrites/devrites/internal/state"
)

type rootMode uint8

const (
	rootUnused rootMode = iota
	rootLenient
	rootStrict
)

// rootModeFor is the single policy boundary between diagnostic/read-only
// commands, which may degrade outside a workspace, and commands that can write
// workspace or Git state, which must never fall back after an unsafe root
// selection.
func rootModeFor(command string, args []string) rootMode {
	subcommand := firstRootOperand(args)
	switch command {
	case "secret-scan":
		return rootLenient
	case "state":
		switch subcommand {
		case "resolve", "close":
			return rootStrict
		}
		return rootUnused
	default:
		return rootUnused
	}
}

func firstRootOperand(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(args[0]))
}

// resolveRootFor resolves exactly once per command invocation. Strict commands
// retain root-safety refusals; lenient readers preserve their historical
// diagnostic/no-workspace behavior.
func resolveRootFor(command string, args []string) (string, int, error) {
	switch rootModeFor(command, args) {
	case rootUnused:
		return "", exitOK, nil
	case rootLenient:
		return resolveRootLenient(), exitOK, nil
	case rootStrict:
		root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
		if err == nil {
			return root, exitOK, nil
		}
		if errors.Is(err, rootfacts.ErrUnsafeRoot) {
			return "", exitBlocked, err
		}
		if errors.Is(err, rootfacts.ErrNoRoot) {
			return fallbackRoot(), exitOK, nil
		}
		return "", exitUsage, err
	default:
		panic("unknown root resolution mode")
	}
}

// resolveRootLenient resolves the .devrites root for read-only and diagnostic
// commands that must degrade cleanly outside a workspace. When no root is found,
// it returns a nonexistent <cwd>/.devrites so those commands see an empty
// workspace and keep their established no-workspace behavior.
func resolveRootLenient() string {
	if root, err := state.ResolveRoot(os.Getenv("DEVRITES_ROOT")); err == nil {
		return root
	}
	return fallbackRoot()
}

func fallbackRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ".devrites"
	}
	return filepath.Join(cwd, devritespaths.DevritesRootName)
}
