// Package toolpolicy contains pure, host-independent tool-call policy helpers.
package toolpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	// MaxCommandBytes bounds policy work on one structured shell command.
	MaxCommandBytes = 64 << 10
	// NormalizationVersion identifies the exact token grammar hashed by Digest.
	NormalizationVersion = "drv-git-op-v1"
)

// Verdict is the classifier's policy-neutral description of a command.
type Verdict string

const (
	VerdictSafe        Verdict = "safe"
	VerdictDestructive Verdict = "destructive"
	VerdictAmbiguous   Verdict = "ambiguous"
)

// ReasonID is a stable semantic diagnostic identifier.
type ReasonID string

const (
	ReasonInputTooLarge          ReasonID = "DRV-GIT-INPUT-TOO-LARGE"
	ReasonAmbiguousSyntax        ReasonID = "DRV-GIT-AMBIGUOUS-SYNTAX"
	ReasonAmbiguousDynamic       ReasonID = "DRV-GIT-AMBIGUOUS-DYNAMIC"
	ReasonAmbiguousAlias         ReasonID = "DRV-GIT-AMBIGUOUS-ALIAS"
	ReasonAmbiguousGlobalOption  ReasonID = "DRV-GIT-AMBIGUOUS-GLOBAL-OPTION"
	ReasonAmbiguousStdin         ReasonID = "DRV-GIT-AMBIGUOUS-STDIN"
	ReasonDestructiveReset       ReasonID = "DRV-GIT-DESTRUCTIVE-RESET"
	ReasonDestructiveClean       ReasonID = "DRV-GIT-DESTRUCTIVE-CLEAN"
	ReasonDestructiveCheckout    ReasonID = "DRV-GIT-DESTRUCTIVE-CHECKOUT"
	ReasonDestructiveRestore     ReasonID = "DRV-GIT-DESTRUCTIVE-RESTORE"
	ReasonDestructiveSwitch      ReasonID = "DRV-GIT-DESTRUCTIVE-SWITCH"
	ReasonDestructiveRemove      ReasonID = "DRV-GIT-DESTRUCTIVE-RM"
	ReasonDestructiveBranch      ReasonID = "DRV-GIT-DESTRUCTIVE-BRANCH"
	ReasonDestructiveTag         ReasonID = "DRV-GIT-DESTRUCTIVE-TAG"
	ReasonDestructiveUpdateRef   ReasonID = "DRV-GIT-DESTRUCTIVE-UPDATE-REF"
	ReasonDestructiveStash       ReasonID = "DRV-GIT-DESTRUCTIVE-STASH"
	ReasonDestructiveReflog      ReasonID = "DRV-GIT-DESTRUCTIVE-REFLOG"
	ReasonDestructivePrune       ReasonID = "DRV-GIT-DESTRUCTIVE-PRUNE"
	ReasonDestructiveHistory     ReasonID = "DRV-GIT-DESTRUCTIVE-HISTORY"
	ReasonDestructivePushForce   ReasonID = "DRV-GIT-DESTRUCTIVE-PUSH-FORCE"
	ReasonDestructivePushDelete  ReasonID = "DRV-GIT-DESTRUCTIVE-PUSH-DELETE"
	ReasonDestructiveWorktree    ReasonID = "DRV-GIT-DESTRUCTIVE-WORKTREE"
	ReasonDestructiveRefDeletion ReasonID = "DRV-GIT-DESTRUCTIVE-REF-DELETE"
)

const directCommandRemediation = "Use a direct, literal git command without shell expansion, eval, an interpreter payload, or an undeclared alias."

// Finding describes one destructive or ambiguous Git operation.
type Finding struct {
	Verdict     Verdict
	ReasonID    ReasonID
	Segment     int
	Remediation string
}

// Result is deterministic and contains no repository or environment facts.
// Digest is SHA-256 over a versioned canonical token stream; normalized command
// text is deliberately not exported because tool arguments may be sensitive.
// A future authority check may compare it only when Verdict is Destructive;
// Ambiguous commands must be rewritten as direct literal commands instead.
type Result struct {
	Verdict     Verdict
	ReasonID    ReasonID
	Digest      string
	Findings    []Finding
	Remediation string
}

// ClassifyGitCommand scans one structured shell-command string for direct,
// syntactically provable destructive Git operations. It is not a shell
// firewall: unsupported dynamic forms are ambiguous only when they visibly
// may synthesize a high-impact Git command.
func ClassifyGitCommand(command string) Result {
	if len(command) > MaxCommandBytes {
		return primaryResult(nil, Finding{
			Verdict:     VerdictAmbiguous,
			ReasonID:    ReasonInputTooLarge,
			Segment:     -1,
			Remediation: directCommandRemediation,
		})
	}

	shell, err := scanShell(command)
	if err != nil {
		if !potentialHighImpactGit(command) {
			return Result{Verdict: VerdictSafe}
		}
		reason := ReasonAmbiguousSyntax
		if failure, ok := err.(*scanFailure); ok && failure.dynamic {
			reason = ReasonAmbiguousDynamic
		}
		return primaryResult(nil, Finding{
			Verdict:     VerdictAmbiguous,
			ReasonID:    reason,
			Segment:     -1,
			Remediation: directCommandRemediation,
		})
	}

	normalized := normalizeTokens(shell.tokens)
	result := Result{
		Verdict: VerdictSafe,
		Digest:  digestNormalized(normalized),
	}
	for _, segment := range shell.segments {
		result.Findings = append(result.Findings, classifySegment(segment)...)
	}
	return primaryResult(&result)
}

func primaryResult(base *Result, extra ...Finding) Result {
	result := Result{Verdict: VerdictSafe}
	if base != nil {
		result = *base
	}
	result.Findings = append(result.Findings, extra...)

	for _, wanted := range []Verdict{VerdictAmbiguous, VerdictDestructive} {
		for _, finding := range result.Findings {
			if finding.Verdict != wanted {
				continue
			}
			result.Verdict = finding.Verdict
			result.ReasonID = finding.ReasonID
			result.Remediation = finding.Remediation
			return result
		}
	}
	return result
}

func digestNormalized(normalized string) string {
	sum := sha256.Sum256([]byte(normalized))
	return NormalizationVersion + ":sha256:" + hex.EncodeToString(sum[:])
}

func classifySegment(segment shellSegment) []Finding {
	words, err := commandWords(segment.tokens)
	if err != nil || len(words) == 0 {
		return nil
	}
	original := words

	peeled, terminal, peelReason := peelTransparent(words)
	if terminal || len(peeled) == 0 {
		return nil
	}
	if peelReason != "" {
		if wordsPotentialHighImpactGit(original) {
			return []Finding{ambiguousFinding(segment.index, peelReason)}
		}
		return nil
	}

	executable := basename(peeled[0].text)
	if payload, ok := interpreterPayload(executable, peeled); ok {
		nested := ClassifyGitCommand(payload)
		if nested.Verdict != VerdictSafe || opaqueInterpreter(executable) && potentialHighImpactGit(payload) {
			return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousDynamic)}
		}
		return nil
	}

	if peeled[0].dynamic {
		if wordsHaveHighImpactHint(peeled[1:]) {
			return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousDynamic)}
		}
		return nil
	}
	if isCompoundReserved(executable) && wordsPotentialHighImpactGit(peeled) {
		return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousSyntax)}
	}
	if executable != "git" {
		return nil
	}

	invocation, finding := parseGitInvocation(peeled, segment.index)
	if finding != nil {
		return []Finding{*finding}
	}
	if invocation.verb == "update-ref" && hasArg(invocation.args, "--stdin") {
		return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousStdin)}
	}
	reasons := destructiveReasons(invocation)
	if invocation.semanticConfig && dynamicSensitiveGitCommand(invocation.verb) &&
		!gitOperationIsDryRun(invocation) &&
		!hasArg(invocation.args, "--help") && !hasArg(invocation.args, "-h") {
		return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousGlobalOption)}
	}
	if len(reasons) > 0 && segmentHasDynamic(segment.tokens) ||
		invocation.dynamic && dynamicSensitiveGitCommand(invocation.verb) {
		return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousDynamic)}
	}
	if len(reasons) == 0 && invocation.verb == "checkout" &&
		!hasArg(invocation.args, "--help") && !hasArg(invocation.args, "-h") &&
		checkoutOperandIsAmbiguous(invocation.args) {
		return []Finding{ambiguousFinding(segment.index, ReasonAmbiguousSyntax)}
	}

	findings := make([]Finding, 0, len(reasons))
	for _, reason := range reasons {
		findings = append(findings, Finding{
			Verdict:     VerdictDestructive,
			ReasonID:    reason,
			Segment:     segment.index,
			Remediation: "Obtain fresh one-shot authority for this exact normalized operation.",
		})
	}
	return findings
}

func segmentHasDynamic(tokens []shellToken) bool {
	for _, tok := range tokens {
		if tok.dynamic {
			return true
		}
	}
	return false
}

func ambiguousFinding(segment int, reason ReasonID) Finding {
	return Finding{
		Verdict:     VerdictAmbiguous,
		ReasonID:    reason,
		Segment:     segment,
		Remediation: directCommandRemediation,
	}
}

func peelTransparent(words []shellToken) ([]shellToken, bool, ReasonID) {
	for {
		for len(words) > 0 && isAssignment(words[0].text) {
			words = words[1:]
		}
		for len(words) > 0 && words[0].text == "!" {
			words = words[1:]
		}
		if len(words) == 0 {
			return nil, true, ""
		}

		switch basename(words[0].text) {
		case "command":
			var terminal bool
			words, terminal = peelCommand(words[1:])
			if terminal {
				return nil, true, ""
			}
		case "env":
			var reason ReasonID
			words, reason = peelEnv(words[1:])
			if reason != "" {
				return words, false, reason
			}
		case "rtk":
			words = words[1:]
			if len(words) > 0 && words[0].text == "--" {
				words = words[1:]
			}
			if len(words) > 0 && words[0].text == "proxy" {
				words = words[1:]
				if len(words) > 0 && words[0].text == "--" {
					words = words[1:]
				}
			}
		case "timeout":
			var reason ReasonID
			words, reason = peelTimeout(words[1:])
			if reason != "" {
				return words, false, reason
			}
		case "nice":
			var reason ReasonID
			words, reason = peelNice(words[1:])
			if reason != "" {
				return words, false, reason
			}
		case "nohup":
			var terminal bool
			var reason ReasonID
			words, terminal, reason = peelNohup(words[1:])
			if terminal {
				return nil, true, ""
			}
			if reason != "" {
				return words, false, reason
			}
		case "exec":
			var reason ReasonID
			words, reason = peelExec(words[1:])
			if reason != "" {
				return words, false, reason
			}
		default:
			return words, false, ""
		}
	}
}

func peelCommand(words []shellToken) ([]shellToken, bool) {
	for len(words) > 0 {
		switch words[0].text {
		case "--":
			return words[1:], false
		case "-p":
			words = words[1:]
		case "-v", "-V":
			return nil, true
		default:
			return words, false
		}
	}
	return nil, true
}

func peelEnv(words []shellToken) ([]shellToken, ReasonID) {
	for len(words) > 0 {
		word := words[0].text
		switch {
		case word == "--":
			return words[1:], ""
		case isAssignment(word):
			words = words[1:]
		case word == "-i" || word == "--ignore-environment" || word == "-0" || word == "--null":
			words = words[1:]
		case word == "-u" || word == "--unset" || word == "-C" || word == "--chdir":
			if len(words) < 2 {
				return words, ReasonAmbiguousSyntax
			}
			words = words[2:]
		case strings.HasPrefix(word, "--unset=") || strings.HasPrefix(word, "--chdir="):
			words = words[1:]
		case word == "-S" || word == "--split-string" || strings.HasPrefix(word, "--split-string="):
			return words, ReasonAmbiguousDynamic
		case strings.HasPrefix(word, "-"):
			return words, ReasonAmbiguousSyntax
		default:
			return words, ""
		}
	}
	return nil, ""
}

func peelTimeout(words []shellToken) ([]shellToken, ReasonID) {
	for len(words) > 0 {
		word := words[0].text
		switch {
		case word == "--":
			words = words[1:]
			goto duration
		case word == "-s" || word == "--signal" || word == "-k" || word == "--kill-after":
			if len(words) < 2 {
				return words, ReasonAmbiguousSyntax
			}
			words = words[2:]
		case strings.HasPrefix(word, "--signal=") || strings.HasPrefix(word, "--kill-after="):
			words = words[1:]
		case word == "--foreground" || word == "--preserve-status" || word == "-v" || word == "--verbose":
			words = words[1:]
		case strings.HasPrefix(word, "-s") && len(word) > 2:
			words = words[1:]
		case strings.HasPrefix(word, "-k") && len(word) > 2:
			words = words[1:]
		case strings.HasPrefix(word, "-"):
			return words, ReasonAmbiguousSyntax
		default:
			goto duration
		}
	}
	return nil, ""

duration:
	if len(words) == 0 {
		return nil, ReasonAmbiguousSyntax
	}
	return words[1:], ""
}

func peelNice(words []shellToken) ([]shellToken, ReasonID) {
	for len(words) > 0 {
		word := words[0].text
		switch {
		case word == "--":
			return words[1:], ""
		case word == "-n" || word == "--adjustment":
			if len(words) < 2 {
				return words, ReasonAmbiguousSyntax
			}
			words = words[2:]
		case strings.HasPrefix(word, "--adjustment="):
			words = words[1:]
		case isSignedDecimal(word):
			words = words[1:]
		case strings.HasPrefix(word, "-"):
			return words, ReasonAmbiguousSyntax
		default:
			return words, ""
		}
	}
	return nil, ""
}

func peelNohup(words []shellToken) ([]shellToken, bool, ReasonID) {
	if len(words) == 0 {
		return nil, true, ""
	}
	switch words[0].text {
	case "--":
		return words[1:], false, ""
	case "--help", "--version":
		return nil, true, ""
	default:
		if strings.HasPrefix(words[0].text, "-") {
			return words, false, ReasonAmbiguousSyntax
		}
		return words, false, ""
	}
}

func peelExec(words []shellToken) ([]shellToken, ReasonID) {
	for len(words) > 0 {
		word := words[0].text
		switch {
		case word == "--":
			return words[1:], ""
		case word == "-a":
			if len(words) < 2 {
				return words, ReasonAmbiguousSyntax
			}
			words = words[2:]
		case strings.HasPrefix(word, "-a") && len(word) > 2:
			words = words[1:]
		case word == "-c" || word == "-l" || word == "-cl" || word == "-lc":
			words = words[1:]
		case strings.HasPrefix(word, "-"):
			return words, ReasonAmbiguousSyntax
		default:
			return words, ""
		}
	}
	return nil, ""
}

func isSignedDecimal(s string) bool {
	if len(s) < 2 || (s[0] != '-' && s[0] != '+') {
		return false
	}
	return isDigits(s[1:])
}

func interpreterPayload(executable string, words []shellToken) (string, bool) {
	if versionedExecutable(executable, "python") {
		return optionPayload(words, "-c", "")
	}
	if versionedExecutable(executable, "node") {
		return optionPayload(words, "-e", "--eval")
	}
	if versionedExecutable(executable, "ruby") || versionedExecutable(executable, "perl") {
		return optionPayload(words, "-e", "")
	}

	switch executable {
	case "eval":
		if len(words) < 2 {
			return "", false
		}
		parts := make([]string, 0, len(words)-1)
		for _, word := range words[1:] {
			parts = append(parts, word.text)
		}
		return strings.Join(parts, " "), true
	case "sh", "bash", "dash", "zsh", "ksh":
		for i := 1; i < len(words); i++ {
			if words[i].text == "-c" || (strings.HasPrefix(words[i].text, "-") && strings.Contains(words[i].text[1:], "c")) {
				if i+1 < len(words) {
					return words[i+1].text, true
				}
				return "", false
			}
		}
	}
	return "", false
}

func versionedExecutable(executable, base string) bool {
	if executable == base {
		return true
	}
	version := strings.TrimPrefix(executable, base)
	if version == executable || version == "" || version[0] == '.' || version[len(version)-1] == '.' {
		return false
	}
	for i, r := range version {
		switch {
		case r >= '0' && r <= '9':
		case r == '.' && i > 0 && version[i-1] != '.':
		default:
			return false
		}
	}
	return true
}

func opaqueInterpreter(executable string) bool {
	return versionedExecutable(executable, "python") ||
		versionedExecutable(executable, "node") ||
		versionedExecutable(executable, "ruby") ||
		versionedExecutable(executable, "perl")
}

func optionPayload(words []shellToken, short, long string) (string, bool) {
	for i := 1; i < len(words); i++ {
		if words[i].text == short || (long != "" && words[i].text == long) {
			if i+1 < len(words) {
				return words[i+1].text, true
			}
			return "", false
		}
		if long != "" && strings.HasPrefix(words[i].text, long+"=") {
			return strings.TrimPrefix(words[i].text, long+"="), true
		}
	}
	return "", false
}
