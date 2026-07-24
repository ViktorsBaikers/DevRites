package toolpolicy

import (
	"strings"
	"testing"
)

func TestClassifyGitCommandDestructiveForms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		reason  ReasonID
	}{
		{"reset hard", "git reset --hard HEAD", ReasonDestructiveReset},
		{"reset default", "git reset HEAD", ReasonDestructiveReset},
		{"reset soft rewrites ref", "git reset --soft HEAD^", ReasonDestructiveHistory},
		{"forced clean", "git clean -fdx", ReasonDestructiveClean},
		{"forced checkout", "git checkout --force main", ReasonDestructiveCheckout},
		{"path checkout", "git checkout HEAD -- file.txt", ReasonDestructiveCheckout},
		{"patch checkout", "git checkout -p file.txt", ReasonDestructiveCheckout},
		{"restore worktree", "git restore --worktree -- file.txt", ReasonDestructiveRestore},
		{"forced switch", "git switch --discard-changes main", ReasonDestructiveSwitch},
		{"forced create switch", "git switch -C main HEAD^", ReasonDestructiveSwitch},
		{"forced tracked remove", "git rm -f file.txt", ReasonDestructiveRemove},
		{"forced index remove", "git update-index --force-remove file.txt", ReasonDestructiveRemove},
		{"branch delete", "git branch -D topic", ReasonDestructiveBranch},
		{"forced branch move", "git branch -f topic HEAD^", ReasonDestructiveBranch},
		{"tag delete", "git tag --delete v1", ReasonDestructiveTag},
		{"update ref delete", "git update-ref -d refs/heads/topic", ReasonDestructiveUpdateRef},
		{"stash drop", "git stash drop stash@{0}", ReasonDestructiveStash},
		{"stash clear", "git stash clear", ReasonDestructiveStash},
		{"reflog expiry", "git reflog expire --expire=now --all", ReasonDestructiveReflog},
		{"prune now gc", "git gc --prune=now", ReasonDestructivePrune},
		{"direct prune", "git prune", ReasonDestructivePrune},
		{"forced worktree remove", "git worktree remove --force ../old", ReasonDestructiveWorktree},
		{"history rebase", "git rebase main", ReasonDestructiveHistory},
		{"history amend", "git commit --amend --no-edit", ReasonDestructiveHistory},
		{"history filter", "git filter-branch -- --all", ReasonDestructiveHistory},
		{"force push", "git push --force-with-lease origin main", ReasonDestructivePushForce},
		{"clustered force push", "git push -fu origin main", ReasonDestructivePushForce},
		{"plus refspec push", "git push origin +main:main", ReasonDestructivePushForce},
		{"delete push flag", "git push --delete origin topic", ReasonDestructivePushDelete},
		{"delete push refspec", "git push origin :topic", ReasonDestructivePushDelete},
		{"pruning fetch", "git fetch --prune origin", ReasonDestructiveRefDeletion},
		{"remote prune", "git remote prune origin", ReasonDestructiveRefDeletion},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(tt.command)
			if got.Verdict != VerdictDestructive || got.ReasonID != tt.reason {
				t.Fatalf("ClassifyGitCommand(%q) = (%q, %q), want (%q, %q): %#v",
					tt.command, got.Verdict, got.ReasonID, VerdictDestructive, tt.reason, got)
			}
			if got.Digest == "" {
				t.Fatalf("destructive result lacks normalized digest: %#v", got)
			}
		})
	}
}

func TestClassifyGitCommandSafeControls(t *testing.T) {
	t.Parallel()
	commands := []string{
		"",
		"git status",
		"git diff -- reset",
		"git show HEAD",
		"git switch main",
		"git add file.txt",
		"git commit -m message",
		"git push origin main",
		"git clean -ndx",
		"git clean -fn",
		"git clean -- -f",
		"git branch --list",
		"git tag --list",
		"git stash list",
		"git reflog show",
		"git gc",
		"git prune --dry-run",
		"git rm --dry-run file.txt",
		"git rm file.txt",
		"git push --dry-run --force origin main",
		"git reflog expire --dry-run --expire=now --all",
		"git worktree remove --dry-run ../old",
		"git worktree remove ../old",
		"git fetch --dry-run --prune origin",
		"git remote prune --dry-run origin",
		"git replace",
		"git reset --help",
		"git push -- --force",
		`grep 'git reset --hard' notes.txt`,
		`printf '%s\n' 'git clean -fdx'`,
		`sh -c "printf '%s\n' 'git reset --hard'"`,
		`eval "printf '%s\n' 'git clean -fdx'"`,
		`echo reset clean git`,
		`/tmp/git-helper reset --hard`,
		`printf git reset --hard`,
		`git status "$path"`,
		`command -v git`,
		`git --version`,
	}
	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(command)
			if got.Verdict != VerdictSafe {
				t.Fatalf("ClassifyGitCommand(%q) = %#v, want safe", command, got)
			}
		})
	}
}

func TestClassifyGitCommandGlobalOptionsAndAliases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		verdict Verdict
		reason  ReasonID
	}{
		{"dash C", "git -C repo reset --hard", VerdictDestructive, ReasonDestructiveReset},
		{"attached dash C", "git -Crepo clean -fd", VerdictDestructive, ReasonDestructiveClean},
		{"work trees", "git --git-dir=.git --work-tree=. clean -fd", VerdictDestructive, ReasonDestructiveClean},
		{"namespace value", "git --namespace=ns --no-pager reset --hard", VerdictDestructive, ReasonDestructiveReset},
		{"repeated config", "git -c user.name=Dev -c core.abbrev=9 status", VerdictSafe, ""},
		{"destructive alias", "git -c alias.nuke='reset --hard' nuke", VerdictDestructive, ReasonDestructiveReset},
		{"benign alias", "git -c alias.st=status st", VerdictSafe, ""},
		{"alias keeps args", "git -c alias.wipe=clean wipe -fdx", VerdictDestructive, ReasonDestructiveClean},
		{"payload alias with external-shaped name", "git -c alias.gui='reset --hard' gui", VerdictDestructive, ReasonDestructiveReset},
		{"nested alias", "git -c alias.a=b -c alias.b='reset --hard' a", VerdictDestructive, ReasonDestructiveReset},
		{"alias cycle", "git -c alias.a=b -c alias.b=a a", VerdictAmbiguous, ReasonAmbiguousAlias},
		{"alias cannot shadow builtin", "git -c alias.reset=status reset --hard", VerdictDestructive, ReasonDestructiveReset},
		{"shell alias", "git -c alias.nuke='!git reset --hard' nuke", VerdictAmbiguous, ReasonAmbiguousAlias},
		{"unknown ambient alias", "git nuke", VerdictAmbiguous, ReasonAmbiguousAlias},
		{"malformed global value", "git -C", VerdictAmbiguous, ReasonAmbiguousGlobalOption},
		{"unsupported global", "git --mystery reset --hard", VerdictAmbiguous, ReasonAmbiguousGlobalOption},
		{"stdin ref updates", "git update-ref --stdin", VerdictAmbiguous, ReasonAmbiguousStdin},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(tt.command)
			if got.Verdict != tt.verdict || got.ReasonID != tt.reason {
				t.Fatalf("ClassifyGitCommand(%q) = (%q, %q), want (%q, %q): %#v",
					tt.command, got.Verdict, got.ReasonID, tt.verdict, tt.reason, got)
			}
		})
	}
}

func TestClassifyGitCommandCriticBypasses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		verdict Verdict
		reason  ReasonID
	}{
		{
			"semantic clean config",
			"git -c clean.requireForce=false clean -d",
			VerdictAmbiguous,
			ReasonAmbiguousGlobalOption,
		},
		{
			"checkout paths without separator",
			"git checkout HEAD .",
			VerdictDestructive,
			ReasonDestructiveCheckout,
		},
		{
			"nohup wrapper",
			"nohup git clean -fd </dev/null",
			VerdictDestructive,
			ReasonDestructiveClean,
		},
		{
			"exec wrapper",
			"exec git reset --hard HEAD",
			VerdictDestructive,
			ReasonDestructiveReset,
		},
		{
			"versioned interpreter",
			`python3.12 -c "import os; os.system('git reset --hard HEAD')"`,
			VerdictAmbiguous,
			ReasonAmbiguousDynamic,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(tt.command)
			if got.Verdict != tt.verdict || got.ReasonID != tt.reason {
				t.Fatalf("ClassifyGitCommand(%q) = (%q, %q), want (%q, %q): %#v",
					tt.command, got.Verdict, got.ReasonID, tt.verdict, tt.reason, got)
			}
		})
	}
}

func TestClassifyGitCommandCriticBypassSafeNeighbors(t *testing.T) {
	t.Parallel()
	commands := []string{
		"git -c color.ui=false clean -nd",
		"git -c color.ui=false checkout -b topic HEAD",
		"git -c color.ui=false reset --hard HEAD",
		"git -c clean.requireForce=false clean -nd",
		"git -c clean.requireForce=false clean --help",
		"git -c user.name=Dev status",
		"git checkout -b topic HEAD",
		"git checkout --detach HEAD",
		"nohup git status",
		"nohup -- git status",
		"nohup --help",
		"exec git status",
		`python3.12 -c "print(1)"`,
		"python3.12-config --help",
	}
	for _, command := range commands {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(command)
			if command == "git -c color.ui=false reset --hard HEAD" {
				if got.Verdict != VerdictDestructive || got.ReasonID != ReasonDestructiveReset {
					t.Fatalf("ClassifyGitCommand(%q) = %#v, want destructive reset", command, got)
				}
				return
			}
			if got.Verdict != VerdictSafe {
				t.Fatalf("ClassifyGitCommand(%q) = %#v, want safe", command, got)
			}
		})
	}
}

func TestClassifyGitCommandListsWrappersAndQuoting(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		count   int
	}{
		{"semicolon", "echo ok; git reset --hard; echo done", 1},
		{"newline", "echo ok\ngit clean -fd\ntrue", 1},
		{"and", "true && git restore -- file", 1},
		{"or", "false || git branch -D topic", 1},
		{"pipeline", "printf x | git reset --hard | cat", 1},
		{"two operations", "git reset --hard && git clean -fdx", 2},
		{"quoted executable", `"/usr/bin/git" "reset" "--hard"`, 1},
		{"transparent wrappers", "env X=1 command rtk git reset --hard", 1},
		{"rtk raw proxy", "rtk proxy git reset --hard", 1},
		{"timeout nice", "timeout --signal TERM 5 nice -n 2 git clean -fd", 1},
		{"escaped executable", `g\it re\set --ha\rd`, 1},
		{"quoted operator path", `git restore -- 'dir/a;b && c|d'`, 1},
		{"redirection prefix", `>/tmp/a git reset --hard`, 1},
		{"redirection suffix", `git clean -fd 2>/tmp/error`, 1},
		{"comment", "echo ok # git reset --hard\ntrue", 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(tt.command)
			if tt.count == 0 {
				if got.Verdict != VerdictSafe {
					t.Fatalf("ClassifyGitCommand(%q) = %#v, want safe", tt.command, got)
				}
				return
			}
			if got.Verdict != VerdictDestructive {
				t.Fatalf("ClassifyGitCommand(%q) = %#v, want destructive", tt.command, got)
			}
			destructive := 0
			for _, finding := range got.Findings {
				if finding.Verdict == VerdictDestructive {
					destructive++
				}
			}
			if destructive != tt.count {
				t.Fatalf("ClassifyGitCommand(%q) has %d destructive findings, want %d: %#v",
					tt.command, destructive, tt.count, got)
			}
		})
	}
}

func TestClassifyGitCommandAmbiguousForms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		reason  ReasonID
	}{
		{"unmatched quote", `git reset --hard "HEAD`, ReasonAmbiguousSyntax},
		{"command substitution", `git reset --hard "$(git rev-parse HEAD)"`, ReasonAmbiguousDynamic},
		{"backticks", "git clean -fd `printf path`", ReasonAmbiguousDynamic},
		{"dynamic executable", `"$cmd" reset --hard`, ReasonAmbiguousDynamic},
		{"dynamic git verb", `git "$verb" --hard`, ReasonAmbiguousDynamic},
		{"dynamic push refspec", `git push "$refspec"`, ReasonAmbiguousDynamic},
		{"dynamic wrapper value", `env TARGET="$target" git reset --hard`, ReasonAmbiguousDynamic},
		{"dynamic redirect", `git clean -fd >"$log"`, ReasonAmbiguousDynamic},
		{"checkout operand can be branch or path", `git checkout file.txt`, ReasonAmbiguousSyntax},
		{"eval", `eval 'git reset --hard'`, ReasonAmbiguousDynamic},
		{"shell interpreter", `sh -c 'git clean -fdx'`, ReasonAmbiguousDynamic},
		{"node interpreter", `node -e "require('child_process').execSync('git reset --hard')"`, ReasonAmbiguousDynamic},
		{"env split", `env -S 'git reset --hard'`, ReasonAmbiguousDynamic},
		{"heredoc", "sh <<'EOF'\ngit reset --hard\nEOF", ReasonAmbiguousDynamic},
		{"compound", `(git reset --hard)`, ReasonAmbiguousSyntax},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ClassifyGitCommand(tt.command)
			if got.Verdict != VerdictAmbiguous || got.ReasonID != tt.reason {
				t.Fatalf("ClassifyGitCommand(%q) = (%q, %q), want (%q, %q): %#v",
					tt.command, got.Verdict, got.ReasonID, VerdictAmbiguous, tt.reason, got)
			}
			if got.Remediation != directCommandRemediation {
				t.Fatalf("ambiguous remediation = %q, want %q", got.Remediation, directCommandRemediation)
			}
		})
	}
}

func TestClassifyGitCommandAmbiguityOutranksDestructiveFinding(t *testing.T) {
	t.Parallel()
	got := ClassifyGitCommand(`git reset --hard; eval 'git clean -fd'`)
	if got.Verdict != VerdictAmbiguous || got.ReasonID != ReasonAmbiguousDynamic {
		t.Fatalf("result = %#v, want dynamic ambiguity as primary", got)
	}
	if len(got.Findings) != 2 {
		t.Fatalf("findings = %#v, want both direct and ambiguous operations", got.Findings)
	}
}

func TestClassifyGitCommandDigestIsCanonicalAndExact(t *testing.T) {
	t.Parallel()
	equivalent := []string{
		`git reset --hard HEAD`,
		`"git" 'reset' --hard HEAD`,
		`g\it reset --hard HEAD;`,
	}
	want := ClassifyGitCommand(equivalent[0])
	for _, command := range equivalent[1:] {
		got := ClassifyGitCommand(command)
		if got.Digest != want.Digest {
			t.Fatalf("equivalent command %q normalized differently:\n%#v\n%#v", command, want, got)
		}
	}

	for _, command := range []string{
		`git reset --hard HEAD^`,
		`echo ok; git reset --hard HEAD`,
		`timeout 5 git reset --hard HEAD`,
		`git reset --hard HEAD &`,
	} {
		got := ClassifyGitCommand(command)
		if got.Digest == want.Digest {
			t.Fatalf("different exact operation %q reused digest %q", command, got.Digest)
		}
	}

	literal := ClassifyGitCommand(`git reset --hard '$ref'`)
	dynamic := ClassifyGitCommand(`git reset --hard "$ref"`)
	if literal.Digest == dynamic.Digest {
		t.Fatalf("literal and dynamic tokens reused digest %q", literal.Digest)
	}
}

func TestClassifyGitCommandBound(t *testing.T) {
	t.Parallel()
	got := ClassifyGitCommand(strings.Repeat("x", MaxCommandBytes+1))
	if got.Verdict != VerdictAmbiguous || got.ReasonID != ReasonInputTooLarge {
		t.Fatalf("oversized result = %#v", got)
	}
	if got.Digest != "" {
		t.Fatalf("oversized input must not receive an authorizable digest: %#v", got)
	}
}

func FuzzClassifyGitCommand(f *testing.F) {
	for _, seed := range []string{
		"",
		"git status",
		"git reset --hard HEAD",
		"git -C repo clean -fdx",
		"git -c alias.nuke='reset --hard' nuke",
		"env X=1 command rtk git push --force origin main",
		"grep 'git reset --hard' notes",
		"git reset --hard \"unterminated",
		"sh -c 'git clean -fd'",
		"\x00\xffgit reset --hard",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, command string) {
		got := ClassifyGitCommand(command)
		switch got.Verdict {
		case VerdictSafe, VerdictDestructive, VerdictAmbiguous:
		default:
			t.Fatalf("invalid verdict %q for %q", got.Verdict, command)
		}
		if got.Verdict != VerdictSafe && got.ReasonID == "" {
			t.Fatalf("non-safe result lacks reason: %#v", got)
		}
		if got.Digest != "" {
			prefix := NormalizationVersion + ":sha256:"
			if len(got.Digest) != len(prefix)+64 || !strings.HasPrefix(got.Digest, prefix) {
				t.Fatalf("invalid digest contract: %#v", got)
			}
		}
	})
}
