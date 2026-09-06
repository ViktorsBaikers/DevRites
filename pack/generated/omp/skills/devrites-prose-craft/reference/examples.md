# Before / after — DevRites artifacts & replies

Real shapes DevRites emits. Each pair keeps the technical content and strips the voice. Note
how the calibration preserves precise lists and identifiers while cutting the prose slop.

## 1. `spec.md` overview

**Before (slop):**
> In today's fast-paced data landscape, this feature delivers a robust, scalable, and seamless
> CSV export capability. It's not just about exporting data — it's about empowering users to
> unlock the full potential of their information. This is a pivotal step forward.

**After:**
> Export the report table to CSV. Users currently copy rows by hand; this adds a one-click
> download. Scope: the reports table only, server-side streaming so memory stays flat on large
> exports.

Cut the jargon, the negative-parallelism hook, and the importance inflation. Kept what the
feature is, who it's for, and the scope boundary.

## 2. Acceptance criteria (technical register — precision stays)

**Keep this as-is — do NOT "humanize" it:**
> - AC-001: A `GET /reports/:id/export?format=csv` returns `200` with `Content-Type: text/csv`.
> - AC-002: Rows stream; peak memory stays under 50 MB for a 1M-row export.
> - AC-003: An unknown `format` returns `400` with `{ "error": "unsupported format" }`.

Three items, exact identifiers, exact status codes. This is the technical register working
correctly. The skill leaves it alone.

## 3. `decisions.md` entry

**Before (slop):**
> We made the strategic decision to leverage a streaming approach. This robust solution
> seamlessly handles large datasets and represents a best-in-class architecture that will
> empower the system to scale effortlessly.

**After:**
> Stream rows to the response instead of buffering the full file. Buffering OOM-ed on a
> 1M-row export in testing (see `evidence.md`). Trade-off: streaming rules out a total-row
> count in the header. Revisit if the UI needs an upfront progress bar.

Named the decision, the reason (with evidence), and the trade-off. That's what an ADR is for.

## 4. `review.md` / `seal.md` finding

**Before (slop):**
> It's worth noting that the error handling here could potentially be more robust. This is a
> critical area that fundamentally matters for production readiness.

**After:**
> Critical — `export.ts:42` catches every error and returns `500 "something went wrong"`. A
> bad `format` param should be a `400` with the specific message (AC-003). Catch the parse error
> narrowly; let the rest propagate.

Severity label, `file:line`, the specific problem, the fix. No hedging, no "fundamentally".

## 5. Chat reply

**Before (slop):**
> Great question! I've gone ahead and implemented the export feature. It's not just functional
> — it's production-ready. Let me know if you need anything else!

**After:**
> Built slice 3 (CSV streaming). Tests pass (`npm test -- export` → 14 passing), peak memory
> 38 MB on the 1M-row fixture. Next: slice 4 (pagination). Want me to continue?

Dropped the pleasantries and the "not just X" inflation. Led with what's done and the evidence.

## 6. Commit body

**Before (slop):**
> This commit introduces a comprehensive and robust solution that seamlessly enables users to
> leverage powerful new export capabilities.

**After:**
> Buffering the whole CSV OOM-ed on >100k rows. Stream rows to the response so memory stays
> flat. Rules out an upfront total-count header; tracked in #123.

Followed `git-workflow.md`: the *why*, wrapped, in plain language.

## 7. Fidelity when evidence is incomplete

**Before (slop, but informative):**
> It is important to note that the importer retries twice after a `429`. The available evidence
> does not clearly indicate whether a third attempt is dropped or queued. In either case, the
> alert text is `Import delayed`.

**After:**
> The importer retries twice after a `429`. We do not have a production trace showing whether a
> third attempt is dropped or queued. The alert text remains `Import delayed`.

Cut the opener and vague phrasing. Kept the retry count, status code, evidence gap, two possible
outcomes, and exact alert text. A smoother sentence that picked one outcome would be fabrication.

## 8. `review.md` finding (narrated candor)

**Before (slop):**
> I want to be upfront: two caveats I would rather flag than let you discover later. First,
> Windows is untested. Second, the numbers in the commit message don't reproduce on this
> hardware. Let me think step by step about what that means for seal.

**After:**
> Two caveats: Windows is untested, and the numbers in the commit message don't reproduce on
> this hardware. Seal stays NO-GO until those are measured or scoped out.

Cut the candor frame and the reasoning-chain opener. Kept both caveats and the gate implication.

## 9. Chat reply (prompt restatement)

**Before (slop):**
> That's a great question. You're asking about whether we should stream the export. Thanks for
> the rollback plan you already wrote — that legwork is what made this possible. Breaking this
> down: streaming avoids the OOM.

**After:**
> Stream the export. Buffering OOM-ed on the 1M-row fixture (see `evidence.md`). Your rollback
> plan still applies.

Dropped the prompt echo, the recap-flattery, and "Breaking this down". Led with the decision
and the evidence.

## The pattern across all nine

1. Lead with the concrete thing (what it is, what's done, what's wrong).
2. Replace every "robust/seamless/leverage" flourish with the specific behavior + its proof.
3. Drop negative-parallelism hooks ("not just X, it's Y"), importance labels ("pivotal"),
   candor frames ("I want to be upfront"), and reasoning-chain scaffolding ("let me think
   step by step").
4. Keep the technical register intact: exact identifiers, status codes, numbered criteria,
   real enumerations. Precision is not slop.
