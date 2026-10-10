#!/usr/bin/env bash
# Contract test: the results/1 field list in scoring.md must state the
# identity rules the score calculator enforces, and the calculator must accept
# a results file built only from what the document says.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
command -v python3 >/dev/null 2>&1 || { echo "  SKIP: python3 not available"; exit 0; }
command -v go >/dev/null 2>&1 || [ -x "${DEVRITES_ENGINE_CLI:-${DEVRITES_ENGINE:-}}" ] || { echo "  SKIP: no engine binary and cannot build engine"; exit 0; }

python3 -I - "$ROOT" <<'PY'
#!/usr/bin/env python3
"""Contract oracle for the results-file identity fields documented in
pack/.claude/skills/overhaul/references/scoring.md.

score() in engine/internal/overhaul/score/run.go guards two fields:
`results.subject` must be a non-empty string, and `results.fingerprint` must
decode to exactly sha256.Size bytes. The reference must say so.

This oracle parses the reference and asserts the contract it *states*, then
proves that contract against the real calculator:

  A. the results-schema field list marks `subject` and `fingerprint` required,
     using the same parenthetical convention the list already uses for
     `rubric_digest`;
  B. the reference states the non-empty-string rule for `subject`;
  C. the reference states the 64-hexadecimal-character rule for `fingerprint`;
  D. a results.json built from nothing but what the reference describes, with
     the values the reference's own rules imply, is ACCEPTED by the calculator
     (exit 0) and the scorecard carries both fields verbatim;
  E. every way of violating B and C is REFUSED by the calculator with exit 2,
     empty stdout, an error naming the offending field, and no scorecard file
     written;
  F. `score compare` refuses an unidentified arm with that arm named in the
     error, so the reference's "both arms" clause is enforced too;
  G. no sentence describes either field as optional.

Failing (A), (B) or (C) is a documentation defect. Failing (D), (E) or (F)
means the reference and the calculator disagree, which is worse.

Usage: <repo-root> [engine-binary]

<engine-binary> is optional. When omitted the oracle honours the repository's
existing test convention - $DEVRITES_ENGINE_CLI, then $DEVRITES_ENGINE, as used by
tests/engine-observation-contract-test.sh - and otherwise builds the engine from
<repo-root>/engine into a temporary directory and removes it afterwards.

Exit 0 = the documented contract is stated and is the enforced contract.
Exit 1 = at least one contract assertion failed (each is printed).
"""

import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile

REFERENCE = "pack/.claude/skills/overhaul/references/scoring.md"

# Every domain key must appear in the rubric's `domains` map; the calculator
# rejects a rubric that omits one and invents none.
DOMAIN_WEIGHTS = {
    "correctness": 20, "security": 20, "reliability": 15, "performance": 15,
    "tests": 10, "architecture": 10, "ux": 5, "operations": 5,
}

# Subject violations. Each must be refused by the calculator.
BAD_SUBJECTS = {
    "subject absent": None,
    "subject null": "__NULL__",
    "subject empty string": "",
    "subject not a string": 12,
}

# Fingerprint violations. Each must be refused by the calculator.
BAD_FINGERPRINTS = {
    "fingerprint absent": None,
    "fingerprint null": "__NULL__",
    "fingerprint empty string": "",
    "fingerprint placeholder": "not-a-sha256",
    "fingerprint one hex char short": "a" * 63,
    "fingerprint one hex char long": "a" * 65,
    "fingerprint 64 non-hex characters": "z" * 64,
}

FAILURES = []


def fail(message):
    FAILURES.append(message)
    print("FAIL: " + message)


def ok(message):
    print("  ok: " + message)


# --------------------------------------------------------------------------
# Document parsing
# --------------------------------------------------------------------------

def field_list(text):
    """Return the results-schema field list, or None when it no longer parses.

    Anchors on the schema tag and terminates on the closing brace group of the
    `results{...}` member, so a rewritten list is still found as long as the
    schema tag and that member survive.
    """
    match = re.search(
        r"`overhaul\.results/1`:(.*?)verified_by\}\}`\)", text, re.DOTALL)
    return None if match is None else match.group(1)


def required_fields(listing):
    """Map backticked field name -> its parenthetical, for the whole list.

    Uses the convention the list already follows for `rubric_digest`
    (`name` (required; ...)), so no field is special-cased.
    """
    out = {}
    for name, note in re.findall(r"`([A-Za-z_]+)`\s*\(([^)]*)\)", listing):
        out[name] = note
    return out


def assert_documented(text, listing):
    """Assertions A, B, C and G - what the reference must say."""
    if listing is None:
        fail("%s no longer lists an overhaul.results/1 field list ending in "
             "results{control_id: {status, evidence[], verified_by}}; the "
             "documented input shape is now unparseable" % REFERENCE)
        return {}

    notes = required_fields(listing)
    print("  parsed results-schema fields: %s" % sorted(notes))
    for name in ("subject", "fingerprint"):
        note = notes.get(name)
        if note is None:
            fail("%s lists `%s` in the results-schema fields with no "
                 "parenthetical, so nothing marks it required; the reader is "
                 "left to assume it is optional" % (REFERENCE, name))
        elif "required" not in note:
            fail("%s marks `%s` as (%s), which does not say required; the "
                 "calculator refuses a results file without it"
                 % (REFERENCE, name, note))
        else:
            ok("`%s` is marked required in the results-schema list (%s)"
               % (name, note))

    # B. subject must be stated as a non-empty string, in one sentence.
    if re.search(r"`subject`[^.\n]*non-empty[^.\n]*string", text):
        ok("reference states the non-empty-string rule for `subject`")
    else:
        fail("%s nowhere states that `subject` must be a non-empty string; the "
             "calculator rejects empty, null and non-string subjects" % REFERENCE)

    # C. fingerprint must be stated as exactly 64 hexadecimal characters.
    if re.search(r"`fingerprint`[^.\n]*64[^.\n]*hexadecimal", text):
        ok("reference states the 64-hexadecimal-character rule for `fingerprint`")
    else:
        fail("%s nowhere states that `fingerprint` must be exactly 64 "
             "hexadecimal characters; the calculator rejects anything that does "
             "not decode to 32 bytes" % REFERENCE)

    # G. neither field may be described as optional anywhere in the reference.
    for name in ("subject", "fingerprint"):
        if re.search(r"`%s`[^.\n]*optional" % name, text) or \
           re.search(r"optional[^.\n]*`%s`" % name, text):
            fail("%s describes `%s` as optional; it is required"
                 % (REFERENCE, name))
    ok("neither field is described as optional")

    return notes


# --------------------------------------------------------------------------
# Calculator harness
# --------------------------------------------------------------------------

def resolve_engine(repo_root, given):
    if given:
        if not os.path.isfile(given) or not os.access(given, os.X_OK):
            print("FAIL: engine binary %s is not an executable file" % given)
            sys.exit(1)
        return given, None
    scratch = tempfile.mkdtemp(prefix="scoring-contract-engine-")
    binary = os.path.join(scratch, "devrites-engine")
    build = subprocess.run(
        ["go", "build", "-trimpath", "-o", binary, "."],
        cwd=os.path.join(repo_root, "engine"),
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    if build.returncode != 0:
        print("FAIL: go build of ./engine failed:\n" + build.stdout)
        shutil.rmtree(scratch, ignore_errors=True)
        sys.exit(1)
    return binary, scratch


def write_json(path, doc):
    with open(path, "w", encoding="utf-8") as handle:
        json.dump(doc, handle, indent=2, sort_keys=True)
        handle.write("\n")
    return path


def minimal_rubric():
    """A rubric the calculator accepts, so the results check is what runs.

    One control per domain, all PASS-able, and the default weight table, which
    is what the reference documents as the default.
    """
    controls = []
    for index, domain in enumerate(sorted(DOMAIN_WEIGHTS), start=1):
        controls.append({
            "id": "C-%03d" % index, "domain": domain, "lanes": ["backend"],
            "weight": 1, "hard_gate": False, "evaluation": "executed",
        })
    return {
        "schema": "overhaul.rubric/1", "rev": 1,
        "lanes": ["backend"],
        "controls": controls,
        "domains": dict(DOMAIN_WEIGHTS),
    }


def results_doc(control_ids, subject, fingerprint):
    """Build results/1 from the documented field list.

    `subject` and `fingerprint` are only included when the caller supplies them,
    so the "absent" cases are genuinely absent rather than null.
    """
    doc = {"schema": "overhaul.results/1",
           "results": {cid: {"status": "PASS", "evidence": ["EV-1"]}
                       for cid in control_ids}}
    if subject is not None:
        doc["subject"] = subject
    if fingerprint is not None:
        doc["fingerprint"] = fingerprint
    return doc


def gates_doc(mode="audit"):
    return {"schema": "overhaul.gates/1", "mode": mode, "gates": {}}


def score(engine, work, name, results, gates=None):
    """Run `overhaul score --out`; return (exit, stdout, stderr, scorecard)."""
    rubric_path = write_json(os.path.join(work, "rubric.json"), minimal_rubric())
    with open(rubric_path, "rb") as handle:
        digest = hashlib.sha256(handle.read()).hexdigest()
    results = dict(results)
    results["rubric_digest"] = digest
    results_path = write_json(os.path.join(work, name + ".json"), results)
    gates_path = write_json(os.path.join(work, "gates.json"), gates or gates_doc())
    card = os.path.join(work, name + ".card.json")
    if os.path.exists(card):
        os.remove(card)
    run = subprocess.run(
        [engine, "overhaul", "score", "--rubric", rubric_path,
         "--results", results_path, "--gates", gates_path, "--out", card],
        stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    return run.returncode, run.stdout, run.stderr, card


def refused(engine, work, name, results, field):
    """Assert the calculator refused this results file, naming `field`."""
    code, stdout, stderr, card = score(engine, work, name, results)
    if code != 2:
        fail("%s: expected exit 2, got %d (stdout %r) - the calculator accepted "
             "a results file the reference says must be refused"
             % (name, code, stdout.strip()))
        return
    if stdout != "":
        fail("%s: exit 2 must write nothing to stdout, got %r" % (name, stdout))
    if not re.search(r"^ERROR: results: [^\n]*\b%s\b" % field, stderr):
        fail("%s: stderr %r does not name the offending field `%s`"
             % (name, stderr.strip(), field))
    if os.path.exists(card):
        fail("%s: a scorecard was written for a refused results file" % name)
    ok("refused with exit 2, no stdout, no scorecard: %s" % stderr.strip())


def assert_enforced(engine, work):
    """Assertions D and E - the documented contract against the calculator."""
    control_ids = [c["id"] for c in minimal_rubric()["controls"]]

    # D. The values the reference's own rules imply. subject is a non-empty
    # string; fingerprint is exactly 64 hexadecimal characters.
    subject = "candidate"
    fingerprint = hashlib.sha256(b"the exact candidate being scored").hexdigest()
    if not re.fullmatch(r"[0-9a-fA-F]{64}", fingerprint):
        fail("internal: the oracle's own fingerprint is not 64 hex characters")
    code, stdout, stderr, card = score(
        engine, work, "documented", results_doc(control_ids, subject, fingerprint))
    if code != 0 or stderr != "":
        fail("a results.json built from only what %s describes was refused: exit "
             "%d, stderr %r - the reference is wrong about what the calculator "
             "accepts" % (REFERENCE, code, stderr.strip()))
        return
    with open(card, encoding="utf-8") as handle:
        card_doc = json.load(handle)
    for field, want in (("subject", subject), ("fingerprint", fingerprint)):
        got = card_doc.get(field)
        if got != want:
            fail("scorecard %s is %r, not the value the results file carried "
                 "(%r); the field is not copied verbatim" % (field, got, want))
        else:
            ok("scorecard carries %s verbatim: %r" % (field, got))
    ok("a results.json built only from the documented fields is accepted "
       "(exit 0)")

    # E. Every documented rule is actually enforced.
    for label, value in sorted(BAD_SUBJECTS.items()):
        doc = results_doc(control_ids, "__NULL__", fingerprint)
        if value == "__NULL__":
            doc["subject"] = None
        elif value is None:
            del doc["subject"]
        else:
            doc["subject"] = value
        refused(engine, work, label.replace(" ", "-"), doc, "subject")

    for label, value in sorted(BAD_FINGERPRINTS.items()):
        doc = results_doc(control_ids, subject, "__NULL__")
        if value == "__NULL__":
            doc["fingerprint"] = None
        elif value is None:
            del doc["fingerprint"]
        else:
            doc["fingerprint"] = value
        refused(engine, work, label.replace(" ", "-"), doc, "fingerprint")


def assert_compare_arms(engine, work):
    """Assertion F - the reference's "both arms of `compare`" clause."""
    control_ids = [c["id"] for c in minimal_rubric()["controls"]]
    rubric_path = write_json(os.path.join(work, "cmp-rubric.json"), minimal_rubric())
    with open(rubric_path, "rb") as handle:
        digest = hashlib.sha256(handle.read()).hexdigest()
    good = hashlib.sha256(b"arm").hexdigest()
    blank = results_doc(control_ids, "candidate", None)
    blank["rubric_digest"] = digest
    ok_path = write_json(os.path.join(work, "cmp-ok.json"),
                         results_doc(control_ids, "candidate", good) |
                         {"rubric_digest": digest})
    bad_path = write_json(os.path.join(work, "cmp-bad.json"), blank)
    for arm, baseline, candidate in (("baseline", bad_path, ok_path),
                                     ("candidate", ok_path, bad_path)):
        run = subprocess.run(
            [engine, "overhaul", "score", "compare", "--rubric", rubric_path,
             "--baseline", baseline, "--candidate", candidate],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        if run.returncode != 2 or run.stdout != "" or \
           not run.stderr.startswith("ERROR: %s: " % arm) or \
           "fingerprint" not in run.stderr:
            fail("compare %s arm without an identified code state: exit %d "
                 "stdout %r stderr %r - expected exit 2 naming the arm and the "
                 "field" % (arm, run.returncode, run.stdout, run.stderr.strip()))
        else:
            ok("compare refuses an unidentified %s arm: %s"
               % (arm, run.stderr.strip()))


def main():
    repo_root = sys.argv[1] if len(sys.argv) > 1 else "."
    given_engine = sys.argv[2] if len(sys.argv) > 2 else (
        os.environ.get("DEVRITES_ENGINE_CLI") or os.environ.get("DEVRITES_ENGINE"))
    reference_path = os.path.join(repo_root, REFERENCE)
    try:
        with open(reference_path, encoding="utf-8") as handle:
            text = handle.read()
    except OSError as exc:
        print("FAIL: cannot read %s: %s" % (reference_path, exc))
        return 1

    print("reference: %s" % REFERENCE)
    print("documented contract:")
    assert_documented(text, field_list(text))

    engine, scratch = resolve_engine(repo_root, given_engine)
    work = tempfile.mkdtemp(prefix="scoring-contract-work-")
    try:
        print("enforced contract:")
        assert_enforced(engine, work)
        print("compare arms:")
        assert_compare_arms(engine, work)
    finally:
        shutil.rmtree(work, ignore_errors=True)
        if scratch:
            shutil.rmtree(scratch, ignore_errors=True)

    if FAILURES:
        print("FAIL: %d results-identity contract violation(s)" % len(FAILURES))
        for message in FAILURES:
            print("  - %s" % message)
        return 1
    print("PASS: scoring.md states the results identity contract the calculator "
          "enforces: `subject` required and non-empty, `fingerprint` required "
          "and exactly 64 hexadecimal characters, refused with exit 2 and no "
          "scorecard otherwise, and both copied verbatim into the scorecard.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
PY
echo "overhaul-scoring-contract: PASS"
