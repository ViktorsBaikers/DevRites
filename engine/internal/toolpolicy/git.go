package toolpolicy

import (
	"strings"
	"unicode"
)

type aliasDefinition struct {
	value   string
	dynamic bool
}

type gitInvocation struct {
	verb           string
	args           []string
	dynamic        bool
	semanticConfig bool
}

func parseGitInvocation(words []shellToken, segment int) (gitInvocation, *Finding) {
	aliases := map[string]aliasDefinition{}
	dynamic := false
	semanticConfig := false
	i := 1
	for i < len(words) {
		word := words[i]
		dynamic = dynamic || word.dynamic
		switch {
		case word.text == "--":
			i++
			goto verb
		case word.text == "-C" || word.text == "-c" ||
			word.text == "--git-dir" || word.text == "--work-tree" ||
			word.text == "--namespace" || word.text == "--config-env":
			if i+1 >= len(words) {
				finding := ambiguousFinding(segment, ReasonAmbiguousGlobalOption)
				return gitInvocation{}, &finding
			}
			value := words[i+1]
			dynamic = dynamic || value.dynamic
			if word.text == "-c" {
				recordAlias(aliases, value)
				semanticConfig = semanticConfig || gitConfigAffectsSemantics(value.text)
			} else if word.text == "--config-env" {
				semanticConfig = true
			}
			i += 2
		case strings.HasPrefix(word.text, "-C") && len(word.text) > 2:
			i++
		case strings.HasPrefix(word.text, "-c") && len(word.text) > 2:
			config := shellToken{
				kind:    wordToken,
				text:    strings.TrimPrefix(word.text, "-c"),
				dynamic: word.dynamic,
			}
			recordAlias(aliases, config)
			semanticConfig = semanticConfig || gitConfigAffectsSemantics(config.text)
			i++
		case longOptionValue(word.text, "--git-dir") != "" ||
			longOptionValue(word.text, "--work-tree") != "" ||
			longOptionValue(word.text, "--namespace") != "":
			i++
		case longOptionValue(word.text, "--config-env") != "":
			semanticConfig = true
			i++
		case word.text == "--bare" || word.text == "--no-pager" ||
			word.text == "--paginate" || word.text == "-p" ||
			word.text == "--no-replace-objects" || word.text == "--literal-pathspecs" ||
			word.text == "--glob-pathspecs" || word.text == "--noglob-pathspecs" ||
			word.text == "--icase-pathspecs" || word.text == "--no-optional-locks":
			i++
		case word.text == "--version" || word.text == "-v" ||
			word.text == "--help" || word.text == "-h" ||
			word.text == "--html-path" || word.text == "--man-path" ||
			word.text == "--info-path" || word.text == "--exec-path":
			return gitInvocation{}, nil
		case strings.HasPrefix(word.text, "--exec-path=") ||
			strings.HasPrefix(word.text, "--list-cmds=") ||
			strings.HasPrefix(word.text, "--attr-source="):
			i++
		case strings.HasPrefix(word.text, "-"):
			finding := ambiguousFinding(segment, ReasonAmbiguousGlobalOption)
			return gitInvocation{}, &finding
		default:
			goto verb
		}
	}

verb:
	if i >= len(words) {
		return gitInvocation{}, nil
	}
	verbWord := words[i]
	if verbWord.dynamic {
		finding := ambiguousFinding(segment, ReasonAmbiguousDynamic)
		return gitInvocation{}, &finding
	}
	args := append([]shellToken(nil), words[i+1:]...)

	verbName := strings.ToLower(verbWord.text)
	for depth := 0; depth < 8; depth++ {
		// Git never lets an alias shadow a built-in command.
		if knownGitCommands[verbName] {
			break
		}
		alias, ok := aliases[verbName]
		if !ok {
			break
		}
		expanded, aliasDynamic, ok := expandAlias(alias)
		if !ok || len(expanded) == 0 {
			finding := ambiguousFinding(segment, ReasonAmbiguousAlias)
			return gitInvocation{}, &finding
		}
		verbWord = expanded[0]
		verbName = strings.ToLower(verbWord.text)
		args = append(expanded[1:], args...)
		dynamic = dynamic || alias.dynamic || aliasDynamic || verbWord.dynamic
		if depth == 7 {
			finding := ambiguousFinding(segment, ReasonAmbiguousAlias)
			return gitInvocation{}, &finding
		}
	}

	if !knownGitCommands[verbName] {
		finding := ambiguousFinding(segment, ReasonAmbiguousAlias)
		return gitInvocation{}, &finding
	}
	argStrings := make([]string, 0, len(args))
	for _, arg := range args {
		argStrings = append(argStrings, arg.text)
		dynamic = dynamic || arg.dynamic
	}
	return gitInvocation{
		verb:           verbName,
		args:           argStrings,
		dynamic:        dynamic,
		semanticConfig: semanticConfig,
	}, nil
}

func gitConfigAffectsSemantics(config string) bool {
	key, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(config)), "=")
	if strings.HasPrefix(key, "alias.") {
		return false
	}
	// color.ui only changes presentation and cannot alter the selected operation.
	return key != "color.ui"
}

func recordAlias(aliases map[string]aliasDefinition, config shellToken) {
	key, value, ok := strings.Cut(config.text, "=")
	if !ok {
		return
	}
	key = strings.ToLower(key)
	if !strings.HasPrefix(key, "alias.") || len(key) == len("alias.") {
		return
	}
	aliases[strings.TrimPrefix(key, "alias.")] = aliasDefinition{
		value:   value,
		dynamic: config.dynamic,
	}
}

func expandAlias(alias aliasDefinition) ([]shellToken, bool, bool) {
	value := strings.TrimSpace(alias.value)
	if value == "" || strings.HasPrefix(value, "!") {
		return nil, alias.dynamic, false
	}
	shell, err := scanShell(value)
	if err != nil || len(shell.segments) != 1 {
		return nil, alias.dynamic, false
	}
	for _, tok := range shell.segments[0].tokens {
		if tok.kind != wordToken {
			return nil, alias.dynamic, false
		}
	}
	words, err := commandWords(shell.segments[0].tokens)
	if err != nil {
		return nil, alias.dynamic, false
	}
	dynamic := alias.dynamic
	for _, word := range words {
		dynamic = dynamic || word.dynamic
	}
	return words, dynamic, true
}

func longOptionValue(word, name string) string {
	prefix := name + "="
	if !strings.HasPrefix(word, prefix) {
		return ""
	}
	return strings.TrimPrefix(word, prefix)
}

func destructiveReasons(inv gitInvocation) []ReasonID {
	args := inv.args
	if hasArg(args, "--help") || hasArg(args, "-h") {
		return nil
	}
	switch inv.verb {
	case "reset":
		if !hasArg(args, "--soft") {
			return []ReasonID{ReasonDestructiveReset}
		}
		return []ReasonID{ReasonDestructiveHistory}
	case "clean":
		if isDryRun(args) {
			return nil
		}
		if hasArg(args, "--force") || hasShortFlag(args, 'f') {
			return []ReasonID{ReasonDestructiveClean}
		}
	case "checkout":
		if hasArg(args, "--force") || hasShortFlag(args, 'f') ||
			hasArg(args, "--patch") || hasShortFlag(args, 'p') ||
			hasArg(args, "--ours") || hasArg(args, "--theirs") ||
			hasPathSeparatorWithValue(args) || checkoutHasPathOperands(args) {
			return []ReasonID{ReasonDestructiveCheckout}
		}
	case "checkout-index":
		if hasArg(args, "--force") || hasShortFlag(args, 'f') {
			return []ReasonID{ReasonDestructiveCheckout}
		}
	case "restore":
		return []ReasonID{ReasonDestructiveRestore}
	case "switch":
		if hasArg(args, "--discard-changes") || hasArg(args, "--force") ||
			hasShortFlag(args, 'f') || hasArg(args, "--force-create") ||
			hasShortFlag(args, 'C') {
			return []ReasonID{ReasonDestructiveSwitch}
		}
	case "rm":
		if isDryRun(args) {
			return nil
		}
		if hasArg(args, "--force") || hasShortFlag(args, 'f') {
			return []ReasonID{ReasonDestructiveRemove}
		}
	case "update-index":
		if hasArg(args, "--force-remove") {
			return []ReasonID{ReasonDestructiveRemove}
		}
	case "branch":
		if hasArg(args, "--delete") || hasShortFlag(args, 'd') || hasShortFlag(args, 'D') ||
			hasArg(args, "--move") && hasArg(args, "--force") || hasShortFlag(args, 'M') ||
			hasArg(args, "--force") || hasShortFlag(args, 'f') {
			return []ReasonID{ReasonDestructiveBranch}
		}
	case "tag":
		if hasArg(args, "--delete") || hasShortFlag(args, 'd') || hasArg(args, "--force") || hasShortFlag(args, 'f') {
			return []ReasonID{ReasonDestructiveTag}
		}
	case "update-ref":
		if hasArg(args, "--delete") || hasShortFlag(args, 'd') {
			return []ReasonID{ReasonDestructiveUpdateRef}
		}
	case "symbolic-ref":
		if hasArg(args, "--delete") {
			return []ReasonID{ReasonDestructiveUpdateRef}
		}
	case "stash":
		if firstPositional(args) == "drop" || firstPositional(args) == "clear" {
			return []ReasonID{ReasonDestructiveStash}
		}
	case "reflog":
		if !isDryRun(args) && (firstPositional(args) == "expire" || firstPositional(args) == "delete") {
			return []ReasonID{ReasonDestructiveReflog}
		}
	case "gc":
		if optionEquals(args, "--prune", "now") || optionEquals(args, "--prune", "all") {
			return []ReasonID{ReasonDestructivePrune}
		}
	case "prune":
		if !isDryRun(args) {
			return []ReasonID{ReasonDestructivePrune}
		}
	case "worktree":
		sub := firstPositional(args)
		if sub == "remove" && !isDryRun(args) &&
			(hasArg(args, "--force") || hasShortFlag(args, 'f')) {
			return []ReasonID{ReasonDestructiveWorktree}
		}
		if sub == "prune" && !isDryRun(args) &&
			(optionEquals(args, "--expire", "now") || optionEquals(args, "--expire", "all")) {
			return []ReasonID{ReasonDestructiveWorktree}
		}
	case "rebase", "filter-branch", "fast-import":
		return []ReasonID{ReasonDestructiveHistory}
	case "replace":
		if firstPositional(args) != "" && !hasArg(args, "--list") && !hasShortFlag(args, 'l') {
			return []ReasonID{ReasonDestructiveHistory}
		}
	case "commit":
		if hasArg(args, "--amend") {
			return []ReasonID{ReasonDestructiveHistory}
		}
	case "push":
		if isDryRun(args) {
			return nil
		}
		var reasons []ReasonID
		if pushForces(args) {
			reasons = append(reasons, ReasonDestructivePushForce)
		}
		if pushDeletes(args) {
			reasons = append(reasons, ReasonDestructivePushDelete)
		}
		return reasons
	case "fetch":
		if !hasArg(args, "--dry-run") && (hasArg(args, "--prune") || hasShortFlag(args, 'p')) {
			return []ReasonID{ReasonDestructiveRefDeletion}
		}
	case "remote":
		if !isDryRun(args) && firstPositional(args) == "prune" {
			return []ReasonID{ReasonDestructiveRefDeletion}
		}
	}
	return nil
}

func dynamicSensitiveGitCommand(verb string) bool {
	switch verb {
	case "reset", "clean", "checkout", "checkout-index", "restore", "switch",
		"rm", "update-index", "branch", "tag", "update-ref", "symbolic-ref", "stash", "reflog",
		"gc", "prune", "worktree", "rebase", "filter-branch", "fast-import",
		"replace", "commit", "push", "fetch", "remote":
		return true
	default:
		return false
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == want {
			return true
		}
	}
	return false
}

func isDryRun(args []string) bool {
	return hasArg(args, "--dry-run") || hasShortFlag(args, 'n')
}

func gitOperationIsDryRun(inv gitInvocation) bool {
	switch inv.verb {
	case "clean", "rm", "prune", "push":
		return isDryRun(inv.args)
	case "reflog", "worktree", "fetch", "remote":
		return hasArg(inv.args, "--dry-run")
	default:
		return false
	}
}

func hasShortFlag(args []string, want byte) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if len(arg) < 2 || arg[0] != '-' || strings.HasPrefix(arg, "--") {
			continue
		}
		if strings.ContainsRune(arg[1:], rune(want)) {
			return true
		}
	}
	return false
}

func hasPathSeparatorWithValue(args []string) bool {
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) {
			return true
		}
	}
	return false
}

func checkoutHasPathOperands(args []string) bool {
	paths, _ := checkoutOperandMode(args)
	return paths
}

func checkoutOperandIsAmbiguous(args []string) bool {
	_, ambiguous := checkoutOperandMode(args)
	return ambiguous
}

func checkoutOperandMode(args []string) (paths, ambiguous bool) {
	branchMode := false
	positionals := 0
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			return i+1 < len(args), false
		case arg == "--pathspec-from-file" && i+1 < len(args),
			strings.HasPrefix(arg, "--pathspec-from-file="):
			return true, false
		case arg == "-b" || arg == "-B" || arg == "--orphan":
			branchMode = true
			i++
		case strings.HasPrefix(arg, "-b") && len(arg) > 2,
			strings.HasPrefix(arg, "-B") && len(arg) > 2,
			strings.HasPrefix(arg, "--orphan="):
			branchMode = true
		case arg == "--detach" || arg == "-t" || arg == "--track":
			branchMode = true
		case arg == "--conflict":
			i++
		case strings.HasPrefix(arg, "-"):
		default:
			positionals++
		}
	}
	if branchMode {
		return false, false
	}
	if positionals >= 2 {
		return true, false
	}
	return false, positionals == 1
}

func firstPositional(args []string) string {
	skipValue := false
	for _, arg := range args {
		if skipValue {
			skipValue = false
			continue
		}
		if arg == "--" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "-m", "--message", "--expire", "--prune", "--force-with-lease":
				skipValue = true
			}
			continue
		}
		return strings.ToLower(arg)
	}
	return ""
}

func optionEquals(args []string, option, value string) bool {
	for i, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == option+"="+value {
			return true
		}
		if arg == option && i+1 < len(args) && args[i+1] == value {
			return true
		}
	}
	return false
}

func pushForces(args []string) bool {
	options := true
	for _, arg := range args {
		if arg == "--" {
			options = false
			continue
		}
		if options && (arg == "--force" || hasShortFlag([]string{arg}, 'f') ||
			strings.HasPrefix(arg, "--force-with-lease") ||
			arg == "--force-if-includes" || arg == "--mirror") ||
			strings.HasPrefix(arg, "+") {
			return true
		}
	}
	return false
}

func pushDeletes(args []string) bool {
	options := true
	for _, arg := range args {
		if arg == "--" {
			options = false
			continue
		}
		if options && (arg == "--delete" || hasShortFlag([]string{arg}, 'd') ||
			arg == "--mirror" || arg == "--prune") {
			return true
		}
		if strings.HasPrefix(arg, ":") && len(arg) > 1 {
			return true
		}
	}
	return false
}

func wordsPotentialHighImpactGit(words []shellToken) bool {
	hasGit := false
	joined := make([]string, 0, len(words))
	for _, word := range words {
		joined = append(joined, word.text)
		if basename(strings.ToLower(word.text)) == "git" {
			hasGit = true
		}
	}
	return hasGit && wordsHaveHighImpactHint(words) ||
		potentialHighImpactGit(strings.Join(joined, " "))
}

func wordsHaveHighImpactHint(words []shellToken) bool {
	for _, word := range words {
		for _, part := range relaxedParts(word.text) {
			if highImpactHints[part] {
				return true
			}
		}
	}
	return false
}

func potentialHighImpactGit(command string) bool {
	parts := relaxedParts(command)
	hasGit := false
	hasDynamic := strings.ContainsAny(command, "$`")
	hasImpact := false
	for _, part := range parts {
		hasGit = hasGit || part == "git"
		hasImpact = hasImpact || highImpactHints[part]
	}
	return hasImpact && (hasGit || hasDynamic)
}

func relaxedParts(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
}

func isCompoundReserved(executable string) bool {
	switch executable {
	case "if", "then", "else", "elif", "fi", "for", "while", "until",
		"case", "esac", "do", "done", "function", "{", "}":
		return true
	default:
		return false
	}
}

var highImpactHints = map[string]bool{
	"reset":          true,
	"clean":          true,
	"checkout":       true,
	"checkout-index": true,
	"restore":        true,
	"switch":         true,
	"rm":             true,
	"branch":         true,
	"tag":            true,
	"update-ref":     true,
	"symbolic-ref":   true,
	"stash":          true,
	"reflog":         true,
	"gc":             true,
	"prune":          true,
	"worktree":       true,
	"rebase":         true,
	"filter-branch":  true,
	"fast-import":    true,
	"replace":        true,
	"commit":         true,
	"push":           true,
	"fetch":          true,
	"remote":         true,
}

var knownGitCommands = map[string]bool{
	"add": true, "am": true, "annotate": true, "apply": true, "archive": true,
	"bisect": true, "blame": true, "branch": true, "bugreport": true, "bundle": true,
	"cat-file": true, "check-attr": true, "check-ignore": true, "check-mailmap": true,
	"check-ref-format": true, "checkout": true, "checkout-index": true, "cherry": true,
	"cherry-pick": true, "clean": true, "clone": true, "column": true, "commit": true,
	"commit-graph": true, "commit-tree": true, "config": true, "count-objects": true,
	"credential": true, "credential-cache": true, "credential-store": true,
	"describe": true, "diagnose": true, "diff": true, "diff-files": true, "diff-index": true,
	"diff-tree": true, "difftool": true, "fast-export": true, "fast-import": true,
	"fetch": true, "filter-branch": true, "fmt-merge-msg": true, "for-each-ref": true,
	"format-patch": true, "fsck": true, "fsmonitor--daemon": true, "gc": true,
	"get-tar-commit-id": true, "grep": true, "hash-object": true,
	"help": true, "hook": true, "index-pack": true, "init": true,
	"interpret-trailers": true, "log": true, "ls-files": true, "ls-remote": true,
	"ls-tree": true, "mailinfo": true, "mailsplit": true, "maintenance": true,
	"merge": true, "merge-base": true, "merge-file": true, "merge-index": true,
	"merge-tree": true, "mktag": true,
	"mktree": true, "multi-pack-index": true, "mv": true, "name-rev": true,
	"notes": true, "pack-objects": true, "pack-redundant": true, "pack-refs": true,
	"patch-id": true, "prune": true, "push": true, "range-diff": true,
	"read-tree": true, "rebase": true, "reflog": true, "remote": true,
	"repack": true, "replace": true, "rerere": true,
	"reset": true, "restore": true, "rev-list": true, "rev-parse": true,
	"rm": true, "shortlog": true, "show": true,
	"show-branch": true, "show-index": true, "show-ref": true, "sparse-checkout": true,
	"stage": true, "stash": true, "status": true, "stripspace": true, "switch": true,
	"symbolic-ref": true, "tag": true, "unpack-file": true, "unpack-objects": true,
	"update-index": true, "update-ref": true, "var": true, "verify-commit": true,
	"verify-pack": true, "verify-tag": true, "whatchanged": true, "worktree": true,
	"write-tree": true,
}
