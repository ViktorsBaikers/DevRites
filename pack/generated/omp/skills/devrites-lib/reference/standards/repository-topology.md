# Repository topology

Load this when work spans a monorepo member, nested project, multiple languages,
multiple services, or more than one repository. The purpose is to select the real
owners and roots before planning paths or commands.

## Establish the topology from live evidence

1. Find the repository root and any nested roots. Corroborate manifests, workspace
   declarations, lockfiles, build files, CI commands, and scoped repository guidance.
2. Name each affected deployable, package, service, database, shared library, and
   generated or vendored surface. A directory is not automatically an ownership
   boundary.
3. For every command, record its working directory and the file that establishes
   that root. Do not run a root command in every child or a child command at the root
   by guesswork.
4. For every cross-root change, name one canonical contract owner and read-only
   consumers. Existing schemas, types, fixtures, or interface definitions outrank a
   new coordination document.

Record the dirty working tree baseline before planning paths. Preserve unrelated user
changes and separate existing generated/vendor modifications from the candidate. **Failing
case:** Build attributes pre-existing dirty-tree diffs to the slice and advances
without a recorded baseline. Missing or
contradictory documentation is a gap to resolve against live source/tests/config; missing documentation is not a reason
to invent a root or convention.

## Ownership rules

- **One fact, one writable owner.** A provider/consumer contract is edited at its
  canonical source and consumed from there; do not maintain matching prose or types
  independently in each service.
- **Generated and vendored code are destinations, not design owners.** Change their
  declared source or dependency. If generation cannot run in the authorized scope,
  stop with the exact missing proof instead of hand-editing output.
- **Repository guidance is scoped.** Apply the nearest validated instructions to a
  path; same-level conflicts that affect behavior, safety, or acceptance are an open
  decision, not permission to pick the convenient file.
- **Shared files serialize work.** Parallel slices must not edit the same contract,
  migration chain, lockfile, generated target, shared state, port, or deployment
  resource. File-disjoint work can still conflict through those resources.

## Architecture checks

- Draw repository/service/package edges with their contract and direction. A missing
  edge is not "internal" merely because both sides live in one monorepo.
- Give mutable state one owner. If two services can write the same fact, define the
  authority, conflict rule, and reconciliation path before build.
- **Topology records diverge from defaults, not ecosystem basics.** An entry earns its
  line by stating what this repository does differently from the platform default; a
  restatement of default behavior is noise that hides the entry that matters. **Failing
  case:** “Postgres stores relational data” listed as a topology fact while the actual
  cross-root contract goes unrecorded.
- A dependency cycle is a boundary defect. Break it with an existing lower-level
  contract, dependency inversion, or a deliberately owned integration seam; do not
  hide it behind duplicated types or runtime import tricks.
- For mixed languages or runtimes, prove the contract at the serialized boundary and
  use each member's native checks. One language's typecheck cannot prove another
  member consumes the contract correctly.
- For multiple repositories, keep the shared behavioral contract in its established
  planning/contract owner. Component plans reference it and own their local paths,
  rollout, and proof; references never imply cross-repository write authority.

## Required plan output

When applicable, `plan.md` names:

| Root/deployable | Owner | Contract or state owned | Command cwd | Change/proof |
| --- | --- | --- | --- | --- |
| `<path/service>` | `<module/team>` | `<artifact/fact>` | `<cwd>` | `<slice + evidence>` |

Also record dependency edges, shared mutable resources, deployment order, and the
smallest independently reversible unit. `Topology impact: none — <specific reason>`
is sufficient for a single-root change.

## Evidence and stop conditions

Evidence is the live root/manifest/config plus consumer- and provider-side checks of
the same contract. Stop on competing roots, lockfiles, owners, or writable contract
copies; an unproven root makes downstream path and command claims unreliable.
