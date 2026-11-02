package install

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrites/devrites/internal/hostpack"
	"github.com/devrites/devrites/internal/testutil"
)

func TestAuditEditedPinnedAliasRetainsOwnershipBaseline(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, rel := range []string{".claude/skills/b/SKILL.md", ".agents/skills/b/SKILL.md"} {
		t.Run(rel, func(t *testing.T) {
			target, payload := t.TempDir(), testPayload(t)
			testutil.WriteFile(t, filepath.Join(payload, "claude/skills/rite-build/SKILL.md"), "build\n")
			runInstall(t, target, payload, func(*Options) {})
			pin := exec.Command("bash", filepath.Join("..", "..", "..", "scripts", "pin.sh"), "--target", target, "add", "b", "rite-build")
			pin.Env = append(os.Environ(), "HOME="+t.TempDir())
			if out, err := pin.CombinedOutput(); err != nil {
				t.Fatalf("pin.sh add: %v\n%s", err, out)
			}
			manifest := filepath.Join(target, ManifestName)
			records, err := readManifest(manifest)
			if err != nil {
				t.Fatal(err)
			}
			original := records[rel].Hash
			if original == "" {
				t.Fatal("pin did not record an ownership hash")
			}
			testutil.AppendFile(t, filepath.Join(target, filepath.FromSlash(rel)), "\nUser customization must survive.\n")
			before := hashTree(t, target)
			for _, dryRun := range []bool{true, false} {
				err := applyInstall(target, payload, false, dryRun, nil)
				if err == nil || !strings.Contains(err.Error(), "customized managed file: "+rel) {
					t.Errorf("dry=%t: refresh error=%v, want customized alias refusal", dryRun, err)
				}
				if hashTree(t, target) != before {
					t.Errorf("dry=%t: refused refresh changed project", dryRun)
				}
			}
			opts := DefaultOptions(ModeUninstall)
			opts.Target, opts.KeepBinary = target, true
			opts.Stdout, opts.Stderr = &bytes.Buffer{}, &bytes.Buffer{}
			if err := Apply(opts); err == nil || !strings.Contains(err.Error(), "customized managed file: "+rel) {
				t.Errorf("uninstall error=%v, want customized alias refusal", err)
			}
			if hashTree(t, target) != before {
				t.Error("refused uninstall changed project")
			}
			records, err = readManifest(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if records[rel].Hash != original {
				t.Errorf("ownership hash=%q, want original %q", records[rel].Hash, original)
			}
			if err := applyInstall(target, payload, true, false, nil); err != nil {
				t.Fatalf("forced refresh: %v", err)
			}
			records, err = readManifest(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if records[rel].Hash != original {
				t.Errorf("forced refresh adopted edited pin hash %q, want %q", records[rel].Hash, original)
			}
			before = hashTree(t, target)
			if err := Apply(opts); err == nil || !strings.Contains(err.Error(), "customized managed file: "+rel) {
				t.Errorf("uninstall after forced refresh error=%v, want customized alias refusal", err)
			}
			if hashTree(t, target) != before {
				t.Error("uninstall after forced refresh changed project")
			}
		})
	}
}

func TestAuditNonClaudeMarkerOwnership(t *testing.T) {
	t.Setenv("DEVRITES_NO_BINARY", "1")
	for _, merge := range []hostpack.MarkerMerge{
		hostpack.CodexAgentsMerge, hostpack.PiAgentsMerge,
		hostpack.DevinAgentsMerge, hostpack.DshAgentsMerge, hostpack.CodexConfigMerge,
	} {
		for _, allHosts := range []bool{false, true} {
			for _, identical := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/all=%t/identical=%t", merge.MarkerRel, allHosts, identical), func(t *testing.T) {
					target, payload := t.TempDir(), testPayload(t)
					marker := "user-owned marker\n"
					if identical {
						marker = merge.MarkerText + "\n"
					}
					testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(merge.MarkerRel)), marker)
					userText := "User project guidance.\n"
					if merge.MarkerRel == hostpack.CodexConfigMerge.MarkerRel {
						userText = "model = \"user-model\"\n"
					}
					testutil.WriteFile(t, filepath.Join(target, filepath.FromSlash(merge.TargetRel)), userText)
					opts := DefaultOptions(ModeInstall)
					opts.Target, opts.PayloadDir = target, payload
					opts.WithCodex = allHosts || merge.MarkerRel == hostpack.CodexAgentsMerge.MarkerRel || merge.MarkerRel == hostpack.CodexConfigMerge.MarkerRel
					opts.WithPi = allHosts || merge.MarkerRel == hostpack.PiAgentsMerge.MarkerRel
					opts.WithDevin = allHosts || merge.MarkerRel == hostpack.DevinAgentsMerge.MarkerRel
					opts.WithDsh = allHosts || merge.MarkerRel == hostpack.DshAgentsMerge.MarkerRel
					opts.Stdout, opts.Stderr = &bytes.Buffer{}, &bytes.Buffer{}
					before := hashTree(t, target)
					for _, dryRun := range []bool{true, false} {
						opts.DryRun = dryRun
						err := Apply(opts)
						if identical {
							if err != nil {
								t.Fatalf("dry=%t: identical marker adoption: %v", dryRun, err)
							}
						} else {
							if err == nil || !strings.Contains(err.Error(), "without owning "+merge.MarkerRel) {
								t.Errorf("dry=%t: error=%v, want marker ownership refusal", dryRun, err)
							}
							if hashTree(t, target) != before {
								t.Errorf("dry=%t: refused marker collision changed project", dryRun)
							}
						}
					}
					if !identical {
						opts.DryRun, opts.Force = false, true
						if err := Apply(opts); err != nil {
							t.Fatalf("forced marker replacement: %v", err)
						}
					}
					records, err := readManifest(filepath.Join(target, ManifestName))
					if err != nil {
						t.Fatal(err)
					}
					if records[merge.MarkerRel].Hash == "" {
						t.Fatal("successful install did not acquire marker ownership")
					}
					runUninstall(t, target)
					got := testutil.ReadFile(t, filepath.Join(target, filepath.FromSlash(merge.TargetRel)))
					if strings.Contains(got, merge.Begin) || strings.TrimSpace(got) != strings.TrimSpace(userText) {
						t.Fatalf("uninstall left managed bytes or lost user content: %q", got)
					}
				})
			}
		}
	}
}
