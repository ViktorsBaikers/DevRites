package lib

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func auditRenderRecheckPacket(t *testing.T, root, workspace, skills, wave, inventory string) string {
	t.Helper()
	contextIntegrityRun(t, root, []string{"feature", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--skills-root", skills, "--recheck", inventory}, 0)
	if wave != "r1" {
		sums := readTestFile(t, filepath.Join(workspace, "packets", "r1.sha256"))
		writeFile(t, filepath.Join(workspace, "packets", wave+".sha256"), strings.ReplaceAll(sums, "packets/r1.diff", "packets/"+wave+".diff"))
		writeFile(t, filepath.Join(workspace, "packets", wave+".diff"), readTestFile(t, filepath.Join(workspace, "packets", "r1.diff")))
	}
	if code, out := runPacketT(t, root, "--wave", wave, "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck"); code != 0 {
		t.Fatalf("render %s: code=%d %s", wave, code, out)
	}
	return filepath.Join(workspace, "packets", wave+".code-reviewer.md")
}

// Resolve the rendered Read first entry, not the reusable context cache name.
func auditPacketContextPath(t *testing.T, root, packet string) string {
	t.Helper()
	_, reads, ok := strings.Cut(readTestFile(t, packet), "\nRead first:\n")
	if !ok {
		t.Fatal("packet has no read-set")
	}
	entry, _, _ := strings.Cut(reads, "\n")
	if !strings.HasPrefix(entry, "- ") {
		t.Fatalf("packet has no first read-set entry: %q", entry)
	}
	path := filepath.FromSlash(strings.TrimPrefix(entry, "- "))
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(root), path)
	}
	return path
}

func auditOpenRecheckPacket(root, wave, packet string) (int, string) {
	var stdout, stderr bytes.Buffer
	code := RunDispatch(root, []string{"feature", "open", "--phase", "build", "--wave", wave, "--role", "code-reviewer", "--slice", "SLICE-001", "--packet", packet}, &stdout, &stderr)
	return code, stdout.String() + stderr.String()
}

func TestAuditReacquiredPacketKeepsOriginalRecheckReadSet(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	skills := newSkillsRoot(t)
	const canaryA = "inventory-alpha: reject missing tenant predicate"
	const canaryB = "inventory-bravo: reject duplicate refund receipt"
	writeFile(t, filepath.Join(workspace, "inventory-a.md"), canaryA+"\n")
	packetA := auditRenderRecheckPacket(t, root, workspace, skills, "r1", "inventory-a.md")
	original := readTestFile(t, auditPacketContextPath(t, root, packetA))
	if !strings.Contains(original, canaryA) || strings.Contains(original, canaryB) {
		t.Fatal("first generated read-set does not isolate inventory A")
	}

	writeFile(t, filepath.Join(workspace, "inventory-b.md"), canaryB+"\n")
	packetB := auditRenderRecheckPacket(t, root, workspace, skills, "r2", "inventory-b.md")
	current := readTestFile(t, auditPacketContextPath(t, root, packetB))
	if !strings.Contains(current, canaryB) || strings.Contains(current, canaryA) {
		t.Fatal("second generated read-set does not isolate inventory B")
	}
	if code, out := auditOpenRecheckPacket(root, "r2", packetB); code != 0 {
		t.Fatalf("current packet must remain dispatchable: code=%d %s", code, out)
	}

	// Reacquire A from disk after B has been rendered; no cached bundle bytes help its consumer.
	if err := lintPacketDiffs(root, "feature", packetA, false); err != nil {
		t.Fatalf("A's frozen diff changed during context regeneration: %v", err)
	}
	code, out := auditOpenRecheckPacket(root, "r1", packetA)
	if code == 3 {
		return // A stale binding may be refused instead of retaining an immutable snapshot.
	}
	if code != 0 {
		t.Fatalf("delayed dispatch: code=%d %s", code, out)
	}
	if got := readTestFile(t, auditPacketContextPath(t, root, packetA)); got != original {
		t.Fatalf("delayed packet admitted with replaced read-set: original A=%t, now A=%t B=%t", strings.Contains(original, canaryA), strings.Contains(got, canaryA), strings.Contains(got, canaryB))
	}
}

func TestAuditDispatchRejectsTamperedPacketContext(t *testing.T) {
	for _, damage := range []string{"none", "body", "body-and-sidecar"} {
		t.Run(damage, func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			skills := newSkillsRoot(t)
			writeFile(t, filepath.Join(workspace, "inventory.md"), "inventory-alpha: reject missing tenant predicate\n")
			packet := auditRenderRecheckPacket(t, root, workspace, skills, "r1", "inventory.md")
			if damage != "none" {
				path := auditPacketContextPath(t, root, packet)
				body := []byte(readTestFile(t, path))
				body[0] ^= 1
				if err := os.WriteFile(path, body, 0o600); err != nil {
					t.Fatal(err)
				}
				if damage == "body-and-sidecar" {
					if err := os.WriteFile(path+".sha256", []byte(fmt.Sprintf("%x\n", sha256.Sum256(body))), 0o600); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, _, err := frozenCandidate(workspace, "r1"); err != nil {
				t.Fatalf("context mutation changed frozen candidate: %v", err)
			}
			want := 3
			if damage == "none" {
				want = 0
			}
			if code, out := auditOpenRecheckPacket(root, "r1", packet); code != want {
				t.Fatalf("dispatch %s context: code=%d want=%d %s", damage, code, want, out)
			}
			if damage != "none" {
				if _, err := os.Stat(filepath.Join(workspace, "dispatch.json")); !os.IsNotExist(err) {
					t.Fatalf("tampered context refusal published dispatch state: %v", err)
				}
			}
		})
	}
}

func TestAuditPacketRefusesChangedContextUnderSameWave(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	skills := newSkillsRoot(t)
	writeFile(t, filepath.Join(workspace, "inventory.md"), "inventory-alpha: preserve original read-set\n")
	packet := auditRenderRecheckPacket(t, root, workspace, skills, "r1", "inventory.md")
	prior := readTestFile(t, packet)
	snapshot := auditPacketContextPath(t, root, packet)
	original := readTestFile(t, snapshot)
	if info, err := os.Lstat(snapshot); err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Fatalf("retained context must be private and regular: %v %v", info, err)
	}
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck"); code != 0 {
		t.Fatalf("identical context re-render: code=%d %s", code, out)
	}
	writeFile(t, filepath.Join(workspace, "inventory.md"), "inventory-bravo: different read-set\n")
	contextIntegrityRun(t, root, []string{"feature", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--skills-root", skills, "--recheck", "inventory.md"}, 0)
	if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck"); code != 3 {
		t.Fatalf("changed same-wave context: code=%d %s", code, out)
	}
	if got := readTestFile(t, packet); got != prior {
		t.Fatal("refused re-render overwrote historical packet")
	}
	if got := readTestFile(t, snapshot); got != original {
		t.Fatal("refused re-render overwrote historical read-set")
	}
	if code, out := auditOpenRecheckPacket(root, "r1", packet); code != 0 {
		t.Fatalf("original read-set no longer dispatchable: code=%d %s", code, out)
	}
}

func TestAuditGeneratedContextBindingAcrossPhasesAndLanes(t *testing.T) {
	for _, phase := range []string{"build", "vet", "review", "seal"} {
		for _, override := range []string{"", "absolute", "relative", "external-absolute", "external-relative"} {
			t.Run(phase+"/"+override, func(t *testing.T) {
				root, workspace, _ := packetTestWorkspace(t)
				if strings.HasPrefix(override, "external-") {
					authorizedRoot := filepath.Join(t.TempDir(), "authorized-workspace")
					if err := os.Rename(root, authorizedRoot); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(authorizedRoot, root); err != nil {
						t.Skipf("symlink unavailable: %v", err)
					}
					workspace = filepath.Join(authorizedRoot, "work", "feature")
				}
				if strings.HasSuffix(override, "absolute") {
					t.Setenv("DEVRITES_WORKSPACE", workspace)
				} else if strings.HasSuffix(override, "relative") {
					t.Setenv("DEVRITES_WORKSPACE", ".devrites/work/feature")
				}
				skills := newSkillsRoot(t)
				writeFile(t, filepath.Join(skills, "rite-"+phase, "SKILL.md"), `<!-- loads: {"always":["devrites-lib/reference/standards/core.md"],"workspace":["spec.md"]} -->`+"\n")
				const role = "devrites-code-reviewer.backend"
				const canary = "tenant read-set: preserve exact source bytes"
				writeFile(t, filepath.Join(workspace, "spec.md"), "# Spec\n"+canary+"\n")
				contextArgs := []string{"feature", "--phase", phase, "--role", role, "--skills-root", skills}
				packetArgs := []string{"--wave", "r1", "--phase", phase, "--role", role}
				dispatchArgs := []string{"feature", "open", "--phase", phase, "--wave", "r1", "--role", role}
				source := phase + "-" + role
				if phase == "build" {
					contextArgs = append(contextArgs, "--slice", "SLICE-001")
					packetArgs = append(packetArgs, "--slice", "SLICE-001")
					dispatchArgs = append(dispatchArgs, "--slice", "SLICE-001")
					source += ".SLICE-001"
				}
				contextIntegrityRun(t, root, contextArgs, 0)
				cache := filepath.Join(workspace, "ctx", source+".bundle.md")
				expected := readTestFile(t, cache)
				if !strings.Contains(expected, canary) {
					t.Fatal("source context lost fixture read-set")
				}
				if code, out := runPacketT(t, root, packetArgs...); code != 0 {
					t.Fatalf("render: code=%d %s", code, out)
				}
				packet := filepath.Join(workspace, "packets", "r1."+role+".md")
				bound := auditPacketContextPath(t, root, packet)
				if got := readTestFile(t, bound); got != expected {
					t.Fatal("packet did not retain exact selected context bytes")
				}
				ref, err := filepath.Rel(projectDir(root), bound)
				if err != nil {
					t.Fatal(err)
				}
				// Relative, absolute and workspace-local names bind the same retained leaf.
				for _, alias := range []string{filepath.ToSlash(ref), filepath.ToSlash(bound), "packets/" + filepath.Base(bound)} {
					raw := []byte(fmt.Sprintf("Read first:\n- %s\ncontext %x packets/%s\n", alias, sha256.Sum256([]byte(expected)), filepath.Base(bound)))
					if err := lintPacketContexts(root, "feature", raw); err != nil {
						t.Fatalf("selected workspace context alias %q: %v", alias, err)
					}
				}
				// A regenerated cache and its sidecar must not redirect the packet.
				replacement := []byte("unrelated later cache\n")
				writeFile(t, cache, string(replacement))
				writeFile(t, cache+".sha256", fmt.Sprintf("%x\n", sha256.Sum256(replacement)))
				var stdout, stderr bytes.Buffer
				if code := RunDispatch(root, append(dispatchArgs, "--packet", packet), &stdout, &stderr); code != 0 {
					t.Fatalf("cache mutation redirected admission: code=%d %s%s", code, &stdout, &stderr)
				}
				if got := readTestFile(t, auditPacketContextPath(t, root, packet)); got != expected {
					t.Fatal("dispatch admitted different read-set bytes")
				}
			})
		}
	}
}

func TestAuditManualBundleReferencesNeedPacketChecksum(t *testing.T) {
	for _, scope := range []string{"feature", "project", "absolute", "absolute-spaces"} {
		for _, damage := range []string{"unbound", "none", "body-and-sidecar", "conflicting", "malformed"} {
			t.Run(scope+"/"+damage, func(t *testing.T) {
				root, workspace, _ := packetTestWorkspace(t)
				path := filepath.Join(workspace, "ctx", "manual.bundle.md")
				if scope == "absolute" || scope == "absolute-spaces" {
					name := "manual.bundle.md"
					if scope == "absolute-spaces" {
						name = "manual read-set.bundle.md"
					}
					path = filepath.Join(t.TempDir(), name)
				}
				const body = "original manual context bytes\n"
				writeFile(t, path, body)
				ref := "ctx/manual.bundle.md"
				if scope == "project" {
					ref = ".devrites/work/feature/" + ref
				} else if scope == "absolute" || scope == "absolute-spaces" {
					ref = filepath.ToSlash(path)
				}
				text := "Read first:\n- " + ref + "\n"
				if damage != "unbound" {
					digest := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
					if damage == "malformed" {
						digest = "not-a-digest"
					}
					text += "context " + digest + " " + ref + "\n"
				}
				if damage == "conflicting" {
					text += "context " + strings.Repeat("0", 64) + " " + ref + "\n"
				}
				packet := writePacket(t, text)
				if damage == "body-and-sidecar" {
					replacement := "tampered manual context bytes\n"
					writeFile(t, path, replacement)
					writeFile(t, path+".sha256", fmt.Sprintf("%x\n", sha256.Sum256([]byte(replacement))))
				}
				var stdout, stderr bytes.Buffer
				code := RunDispatch(root, []string{"feature", "open", "--phase", "build", "--wave", "manual", "--role", "code-reviewer", "--slice", "SLICE-001", "--packet", packet}, &stdout, &stderr)
				want := 3
				if damage == "none" {
					want = 0
				}
				if code != want {
					t.Fatalf("manual context admission: code=%d want=%d %s%s", code, want, &stdout, &stderr)
				}
				if damage != "none" {
					if _, err := os.Stat(filepath.Join(workspace, "dispatch.json")); !os.IsNotExist(err) {
						t.Fatalf("refused manual context published dispatch state: %v", err)
					}
				}
			})
		}
	}
}

func TestAuditPacketWithoutBundleRetainsManualGrammar(t *testing.T) {
	root, _, _ := packetTestWorkspace(t)
	packet := writePacket(t, "Read first:\n- spec.md\n- tasks.md\nOpen findings:\n- F-1 src/a.go:12 tenant predicate missing\nReturn: Outcome: <findings|no-findings|gap>\n")
	var stdout, stderr bytes.Buffer
	if code := RunDispatch(root, []string{"feature", "open", "--phase", "build", "--wave", "manual", "--role", "code-reviewer", "--slice", "SLICE-001", "--packet", packet}, &stdout, &stderr); code != 0 {
		t.Fatalf("bundle-free manual packet refused: code=%d %s%s", code, &stdout, &stderr)
	}
}

func TestAuditWrightDispatchChecksRetainedContextBytes(t *testing.T) {
	for _, tamper := range []bool{false, true} {
		t.Run(fmt.Sprintf("tamper=%t", tamper), func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			if code, out := runPacketT(t, root, "--wave", "w1", "--phase", "build", "--role", "slice-wright", "--slice", "SLICE-001", "--checks", "G-UNIT"); code != 0 {
				t.Fatalf("render wright: code=%d %s", code, out)
			}
			packet := filepath.Join(workspace, "packets", "w1.contract.md")
			if tamper {
				path := auditPacketContextPath(t, root, packet)
				body := []byte(readTestFile(t, path))
				body[0] ^= 1
				writeFile(t, path, string(body))
				writeFile(t, path+".sha256", fmt.Sprintf("%x\n", sha256.Sum256(body)))
			}
			var stdout, stderr bytes.Buffer
			code := RunDispatch(root, []string{"feature", "open", "--phase", "build", "--wave", "w1", "--role", "slice-wright", "--slice", "SLICE-001", "--contract", packet}, &stdout, &stderr)
			want := 0
			if tamper {
				want = 3
			}
			if code != want {
				t.Fatalf("wright context admission: code=%d want=%d %s%s", code, want, &stdout, &stderr)
			}
			if tamper {
				if _, err := os.Stat(filepath.Join(workspace, "dispatch.json")); !os.IsNotExist(err) {
					t.Fatalf("refused writer context published dispatch state: %v", err)
				}
			}
		})
	}
}

func TestAuditPacketRejectsUnsafeContextSource(t *testing.T) {
	for _, damage := range []string{"empty", "oversized", "directory", "fifo", "symlink", "parent-symlink"} {
		t.Run(damage, func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			path := filepath.Join(workspace, "ctx", "review-code-reviewer.bundle.md")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch damage {
			case "empty":
				writeFile(t, path, "")
			case "oversized":
				writeFile(t, path, "bundle\n")
				if err := os.Truncate(path, workspaceArtifactLimit+1); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				contextFIFO(t, path)
			case "symlink":
				target := filepath.Join(t.TempDir(), "outside.bundle.md")
				writeFile(t, target, "outside bytes\n")
				if err := os.Symlink(target, path); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			case "parent-symlink":
				external := t.TempDir()
				writeFile(t, filepath.Join(external, filepath.Base(path)), "outside bytes\n")
				if err := os.Rename(filepath.Dir(path), filepath.Dir(path)+"-original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(external, filepath.Dir(path)); err != nil {
					t.Skipf("symlink unavailable: %v", err)
				}
			}
			if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "review", "--role", "code-reviewer"); code != 3 {
				t.Fatalf("unsafe context source: code=%d %s", code, out)
			}
			if _, err := os.Stat(filepath.Join(workspace, "packets", "r1.code-reviewer.md")); !os.IsNotExist(err) {
				t.Fatalf("unsafe source published packet: %v", err)
			}
		})
	}
}

func TestAuditPacketNeverRebindsDamagedHistoricalSnapshot(t *testing.T) {
	for _, damage := range []string{"missing", "replaced"} {
		t.Run(damage, func(t *testing.T) {
			root, workspace, _ := packetTestWorkspace(t)
			skills := newSkillsRoot(t)
			writeFile(t, filepath.Join(workspace, "inventory.md"), "original inventory read-set\n")
			packet := auditRenderRecheckPacket(t, root, workspace, skills, "r1", "inventory.md")
			prior := readTestFile(t, packet)
			snapshot := auditPacketContextPath(t, root, packet)
			cache := filepath.Join(workspace, "ctx", "build-code-reviewer.SLICE-001.recheck.bundle.md")
			if damage == "missing" {
				if err := os.Remove(snapshot); err != nil {
					t.Fatal(err)
				}
			} else {
				const changed = "different read-set with rewritten digest\n"
				writeFile(t, snapshot, changed)
				writeFile(t, snapshot+".sha256", fmt.Sprintf("%x\n", sha256.Sum256([]byte(changed))))
				writeFile(t, cache, changed)
				writeFile(t, cache+".sha256", fmt.Sprintf("%x\n", sha256.Sum256([]byte(changed))))
			}
			if code, out := runPacketT(t, root, "--wave", "r1", "--phase", "build", "--role", "code-reviewer", "--slice", "SLICE-001", "--recheck"); code != 3 {
				t.Fatalf("historical snapshot %s was rebound: code=%d %s", damage, code, out)
			}
			if got := readTestFile(t, packet); got != prior {
				t.Fatal("refused historical rebind replaced packet authority")
			}
			if code, out := auditOpenRecheckPacket(root, "r1", packet); code != 3 {
				t.Fatalf("damaged historical read-set admitted: code=%d %s", code, out)
			}
		})
	}
}

func TestAuditPacketContextRejectsForeignAndEscapingNamespaces(t *testing.T) {
	root, workspace, _ := packetTestWorkspace(t)
	const body = "bound context bytes\n"
	const leaf = "read-set.bundle.md"
	selected := filepath.Join(workspace, "ctx", leaf)
	foreign := filepath.Join(filepath.Dir(workspace), "other", "ctx", leaf)
	escaped := filepath.Join(filepath.Dir(workspace), "outside.bundle.md")
	for _, path := range []string{selected, foreign, escaped} {
		writeFile(t, path, body)
	}
	symlink := filepath.Join(workspace, "ctx", "link.bundle.md")
	if err := os.Symlink(selected, symlink); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	parent := filepath.Join(workspace, "linked")
	if err := os.Symlink(filepath.Join(workspace, "ctx"), parent); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	refs := []string{
		"../outside.bundle.md",
		workspaceRel(root, workspace) + "/../outside.bundle.md",
		workspaceRel(root, workspace) + "/ctx/../ctx/" + leaf,
		workspaceRel(root, filepath.Dir(foreign)) + "/" + leaf,
		filepath.ToSlash(foreign),
		"ctx/link.bundle.md",
		"linked/" + leaf,
	}
	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			raw := []byte(fmt.Sprintf("Read first:\n- %s\ncontext %x %s\n", ref, sha256.Sum256([]byte(body)), ref))
			if err := lintPacketContexts(root, "feature", raw); err == nil {
				t.Fatalf("unsafe or foreign context %q accepted", ref)
			}
		})
	}
}

func TestAuditExternalOverrideRejectsForeignContextAuthority(t *testing.T) {
	root, _, _ := packetTestWorkspace(t)
	authorizedRoot := filepath.Join(t.TempDir(), "authorized-workspace")
	if err := os.Rename(root, authorizedRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(authorizedRoot, root); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	workspace := filepath.Join(authorizedRoot, "work", "feature")
	t.Setenv("DEVRITES_WORKSPACE", workspace)
	const body = "foreign context bytes\n"
	selected := filepath.Join(workspace, "ctx", "selected.bundle.md")
	writeFile(t, selected, body)
	alias := filepath.ToSlash(filepath.Join(root, "work", "feature", "ctx", filepath.Base(selected)))
	selectedPacket := []byte(fmt.Sprintf("Read first:\n- %s\ncontext %x ctx/%s\n", alias, sha256.Sum256([]byte(body)), filepath.Base(selected)))
	if err := lintPacketContexts(root, "feature", selectedPacket); err != nil {
		t.Fatalf("selected workspace absolute alias refused: %v", err)
	}
	if err := os.Symlink(filepath.Join(workspace, "ctx"), filepath.Join(workspace, "linked")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	foreign := filepath.Join(authorizedRoot, "work", "other", "ctx", "foreign.bundle.md")
	writeFile(t, foreign, body)
	refs := []string{
		filepath.ToSlash(foreign),
		workspaceRel(root, filepath.Dir(foreign)) + "/" + filepath.Base(foreign),
		filepath.ToSlash(filepath.Join(root, "work", "other", "ctx", filepath.Base(foreign))),
		workspaceRel(root, workspace) + "/../other/ctx/" + filepath.Base(foreign),
		filepath.ToSlash(filepath.Join(root, "work", "feature", "linked", filepath.Base(selected))),
	}
	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			raw := []byte(fmt.Sprintf("Read first:\n- %s\ncontext %x %s\n", ref, sha256.Sum256([]byte(body)), ref))
			if err := lintPacketContexts(root, "feature", raw); err == nil {
				t.Fatalf("foreign override context %q accepted", ref)
			}
		})
	}
}
