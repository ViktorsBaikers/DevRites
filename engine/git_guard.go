package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/devrites/devrites/internal/harness"
	"github.com/devrites/devrites/internal/lib"
	"github.com/devrites/devrites/internal/orient"
	drvreason "github.com/devrites/devrites/internal/reason"
	"github.com/devrites/devrites/internal/toolpolicy"
)

const gitGuardSuffix = " (devrites-git-guard)"

// hookGitGuard denies ambiguous high-impact Git and requires an exact, fresh,
// consumed-on-attempt authority record for an unambiguous destructive command.
func hookGitGuard(h harness.Harness, stdin io.Reader, stdout, stderr io.Writer) int {
	in := h.ParsePreToolInput(stdin)
	if !isShellTool(in.ToolName) || strings.TrimSpace(in.Command) == "" {
		return exitOK
	}

	classified := toolpolicy.ClassifyGitCommand(in.Command)
	if classified.Verdict == toolpolicy.VerdictSafe {
		return exitOK
	}
	classifierReasons := gitClassifierReasons(classified)
	if classified.Verdict == toolpolicy.VerdictAmbiguous {
		primary := gitReasonID(classified.ReasonID, drvreason.GitAuthorityUnavailable)
		root, slug := gitGuardWorkspace()
		recordGitPolicy(root, slug, h, primary, classifierReasons, lib.OutcomeDenied)
		return emitGitDeny(h, stdout, stderr,
			fmt.Sprintf("DevRites denied an ambiguous high-impact Git request (%s). %s%s",
				primary, classified.Remediation, gitGuardSuffix))
	}

	root, slug := gitGuardWorkspace()
	if root == "" {
		return emitGitDeny(h, stdout, stderr,
			"DevRites cannot authorize destructive Git without an active DevRites workspace ("+
				string(drvreason.GitWorkspaceUnavailable)+")."+gitGuardSuffix)
	}
	if slug == "" {
		recordGitPolicy(root, "", h, drvreason.GitWorkspaceUnavailable, classifierReasons, lib.OutcomeDenied)
		return emitGitDeny(h, stdout, stderr,
			"DevRites cannot authorize destructive Git without an active DevRites workspace ("+
				string(drvreason.GitWorkspaceUnavailable)+")."+gitGuardSuffix)
	}

	decision := lib.AuthorizeGitOperation(root, slug, classified.Digest, destructiveClassifierReasons(classified))
	if decision.Allowed {
		out, err := h.PreToolAllow("DevRites consumed one fresh exact-operation Git authorization (" +
			string(decision.ReasonID) + ")." + gitGuardSuffix)
		if err != nil {
			recordGitPolicy(root, slug, h, drvreason.GitAuthorityUnavailable, classifierReasons, lib.OutcomeDenied)
			return exitOK
		}
		fmt.Fprintln(stdout, out)
		recordGitPolicy(root, slug, h, decision.ReasonID, classifierReasons, lib.OutcomeAllowed)
		return exitOK
	}

	recordGitPolicy(root, slug, h, decision.ReasonID, classifierReasons, lib.OutcomeDenied)
	return emitGitDeny(h, stdout, stderr, gitAuthorityDenyMessage(decision))
}

func gitGuardWorkspace() (string, string) {
	root, err := orient.ResolveRoot(os.Getenv("DEVRITES_ROOT"))
	if err != nil {
		return "", ""
	}
	slug, err := orient.ActiveSlug(root)
	if err != nil {
		return root, ""
	}
	return root, slug
}

func emitGitDeny(h harness.Harness, stdout, stderr io.Writer, message string) int {
	out, err := h.PreToolDeny(message)
	if err != nil {
		debugf(stderr, "git-guard decision unavailable")
		return exitOK
	}
	fmt.Fprintln(stdout, out)
	return exitOK
}

func gitAuthorityDenyMessage(decision lib.GitAuthorityDecision) string {
	reasonID := string(decision.ReasonID)
	switch decision.ReasonID {
	case drvreason.GitAuthorityPending, drvreason.GitAuthorityExpired, drvreason.GitAuthorityReplayed:
		if decision.QuestionID != "" {
			return fmt.Sprintf(
				`DevRites requires one fresh exact-operation Git authorization (%s). Resolve %s with /rite-resolve %s "%s", then retry.%s`,
				reasonID, decision.QuestionID, decision.QuestionID, lib.GitAuthorityAnswer, gitGuardSuffix)
		}
	case drvreason.GitAuthorityRefused:
		return fmt.Sprintf("DevRites denied this destructive Git operation because its exact authorization was refused (%s).%s",
			reasonID, gitGuardSuffix)
	case drvreason.GitAuthorityCorrupt:
		return fmt.Sprintf("DevRites denied destructive Git because its authority record or consumption ledger "+
			"is invalid (%s). Repair the workspace authority state before retrying.%s",
			reasonID, gitGuardSuffix)
	case drvreason.GitWorkspaceUnavailable:
		return fmt.Sprintf("DevRites cannot authorize destructive Git without an active DevRites workspace (%s).%s",
			reasonID, gitGuardSuffix)
	}
	return fmt.Sprintf("DevRites could not persist or consume destructive Git authority (%s). Repair the workspace or hook, then retry.%s",
		reasonID, gitGuardSuffix)
}

func destructiveClassifierReasons(classified toolpolicy.Result) []toolpolicy.ReasonID {
	seen := map[toolpolicy.ReasonID]bool{}
	var out []toolpolicy.ReasonID
	for _, finding := range classified.Findings {
		if finding.Verdict != toolpolicy.VerdictDestructive || seen[finding.ReasonID] {
			continue
		}
		seen[finding.ReasonID] = true
		out = append(out, finding.ReasonID)
	}
	return out
}

func gitClassifierReasons(classified toolpolicy.Result) []drvreason.ID {
	seen := map[drvreason.ID]bool{}
	var out []drvreason.ID
	for _, finding := range classified.Findings {
		id := gitReasonID(finding.ReasonID, "")
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if len(out) == 0 {
		if id := gitReasonID(classified.ReasonID, ""); id != "" {
			out = append(out, id)
		}
	}
	return out
}

func gitReasonID(value toolpolicy.ReasonID, fallback drvreason.ID) drvreason.ID {
	id, err := drvreason.Parse(string(value))
	if err != nil {
		return fallback
	}
	return id
}

func recordGitPolicy(root, slug string, h harness.Harness, primary drvreason.ID, classifier []drvreason.ID, outcome lib.EventOutcome) {
	if root == "" || !drvreason.Known(primary) {
		return
	}
	ev := lib.NewEventV1(lib.BoundaryGitPolicy, "git-guard", primary)
	ev.GuardStrength = lib.GuardEnforced
	ev.Outcome = outcome
	ev.Host = lib.EventHost(h)
	if ev.Host != lib.HostClaude && ev.Host != lib.HostCodex {
		ev.Host = lib.HostEngine
	}
	ev.RuleIDs = []drvreason.ID{primary}
	for _, id := range classifier {
		if len(ev.RuleIDs) == 16 {
			break
		}
		duplicate := false
		for _, existing := range ev.RuleIDs {
			if existing == id {
				duplicate = true
				break
			}
		}
		if !duplicate {
			ev.RuleIDs = append(ev.RuleIDs, id)
		}
	}
	if err := lib.BindEventWorkspace(root, slug, &ev); err != nil {
		return
	}
	_ = lib.AppendEventV1(root, ev)
}
