# CI/CD & automation

Read this only when creating or changing a build/deploy pipeline. [`hooks.md`](hooks.md) owns local checks, [`development-workflow.md`](development-workflow.md) owns trunk health, and [`deprecation.md`](deprecation.md) owns migrations.

## Pipeline gates

- Put each check at the earliest affordable stage: editor → pre-commit → pre-push → CI.
- Keep batches small so failures are attributable and releases remain reversible.
- Run lint → type-check → unit → build → integration → audit → E2E as applicable. A red gate stops the line; fix the defect rather than disabling the rule, weakening the test, or skipping the gate. CI is the source of truth for green.

## Failure loop and ownership

Read the specific failure, fix its root cause, verify locally, then push again. Do not blind-rerun a flaky pipeline. Use `devrites-debug-recovery` when a test or build failure needs reproduction.

A designated Build Cop (a human/team role, not a pack agent) owns restoring a broken trunk by fixing or reverting, whichever is faster. Restoring trunk outranks feature work.

## Deploy versus release

Keep incomplete or risky behavior disabled behind a flag so deploy and release remain separate and rollback does not require a redeploy. Every flag has an owner and a removal trigger; remove it through the [`deprecation.md`](deprecation.md) expand/contract path.

## Deployment order and configuration

- Derive an explicit order for schema, application, worker, contract, config, and flag
  changes. Prove the system remains safe at every intermediate old/new combination;
  "deploy together" is not an atomicity guarantee across units.
- Validate required configuration at startup or the earliest safe boundary. Record names,
  owners, environments, safe defaults, and rollback values without recording secrets.
  A staging value copied by memory is not production evidence; a configuration mismatch
  blocks exposure until the exact target value/owner is verified.
- Documentation drift in commands, configuration, migration order, or rollback steps is a
  delivery defect: update the canonical doc and execute the documented path before exposure.
- Migration and destructive steps apply [`data-integrity.md`](data-integrity.md); service,
  queue, webhook, and cache changes apply
  [`integration-reliability.md`](integration-reliability.md). Their recovery/observability
  gates precede exposure.
- Feature flags separate exposure only when both states are tested, the off path preserves
  current behavior, and disabling the flag stops the risky effect. A flag cannot undo an
  already destructive schema or data change.

## Secrets

Commit `.env.example` without values; never commit real `.env` files. Inject CI secrets from the platform store and scope them to the job. Build runners do not receive production credentials.

## Slow pipelines

When wall time exceeds roughly ten minutes, measure before and after. Improve in order: cache dependencies, parallelize independent jobs, path-filter, shard the slow suite, re-tier genuinely slow tests, then consider larger runners.

## Scope

Change only the pipeline surface in scope; record wider CI redesign as follow-up work.
