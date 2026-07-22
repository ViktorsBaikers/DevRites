# CI/CD & automation

CI is the enforcement mechanism for every other rule: the place a discipline that's optional on
one engineer's machine becomes mandatory for the trunk. This file is the pipeline-and-release
layer; it doesn't restate what its neighbours own: local hook staging by cost is
[`hooks.md`](hooks.md), trunk-always-green + definition-of-done is
[`development-workflow.md`](development-workflow.md), and the safe-migration ladder is
[`deprecation.md`](deprecation.md). Read this when setting up or changing a build/deploy pipeline.

## Two forces that set the shape

- **Shift left.** A defect gets cheaper the earlier it's caught: a lint error costs seconds, the
  same bug in production costs an incident. Push each check to the earliest stage that can catch it
  (editor → pre-commit → pre-push → CI), so CI confirms rather than discovers.
- **Faster is safer.** Small, frequent deploys are *lower* risk, not higher: a release with three
  changes is debuggable; one with thirty is an investigation. Batch size is a safety lever, so
  optimize for a pipeline fast enough that shipping often is painless.

## The gate pipeline: no gate is skippable
CI runs an ordered set of gates: lint → type-check → unit → build → integration → audit → e2e:
and a red gate stops the line. The discipline is the same as a red test ([`testing.md`](testing.md)):
**if lint fails, fix the code (don't disable the rule; if a test fails, fix the code) don't skip
the test.** A gate silenced to get green is a defect shipped with a clean check on top of it. CI,
not a local run, is the source of truth for "green".

## The CI-failure loop
A red pipeline is a signal to act on, not noise to retry. Read the *specific* failure, fix at the
root, verify locally, push again. Map the failure to its fix: lint → the reported rule (not a blanket
`--fix` that masks intent); type error → the flagged location; test failure → `devrites-debug-recovery`
(reproduce before fixing); build → config/dependency drift. A blind re-run of a flaky pipeline is the
CI version of papering over a failing test.

## Build Cop: one owner keeps the trunk green
When the pipeline breaks, the job of getting it green belongs to a designated **Build Cop**, not
diffused across everyone until no one acts, and the fix is **fix or revert**, whichever is faster,
by whoever holds the role, not necessarily the author of the breaking change. A broken trunk blocks
everyone, so restoring it outranks feature work until it's green.

## Feature flags decouple deploy from release
Deploying code and releasing behaviour are separate events. Ship incomplete or risky work **disabled**
behind a flag so the trunk keeps moving; turn it on to release, off to roll back: no redeploy on the
critical path. Every flag is born with an **owner and a removal trigger**; a flag that outlives its
rollout becomes dead branches and combinatorial test cost (staged-rollout mechanics:
[`git-workflow.md`](git-workflow.md), [`deprecation.md`](deprecation.md) expand→contract).

## Secrets never live where they don't have to
Tier secret exposure by environment: `.env.example` committed (no values); real `.env` never
committed; CI secrets injected from the platform's secret store, scoped to the job. **CI must not
hold production secrets**: a build runner is a broad attack surface, and least privilege
([`security.md`](security.md)) says a compromised pipeline shouldn't reach prod credentials.

## When the pipeline is too slow (>~10 min), climb the ladder
Slowness makes engineers skip or batch, which defeats "faster is safer". Fix it in impact order,
cheapest first: cache dependencies → parallelize independent jobs → path-filter to only what changed
→ shard/matrix the slow suite → prune or re-tier genuinely slow tests → larger runners last. Measure
before and after ([`performance.md`](performance.md) measure-first): a "faster" pipeline that didn't
move the wall-clock number is just more config.

## Scope
Most projects own their own CI; DevRites installs *into* them. Treat this as guidance for the
pipeline the change touches, not licence to re-architect a project's whole CI in an unrelated
change: the same feature-scope discipline as the rest of the rules.
