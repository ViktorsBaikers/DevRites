# Executable workflow artifacts

Executable files used only to plan, isolate, or prove a DevRites workflow—such as
a controller, harness, proof bundle, fixture generator, or bounded materializer
under the active `.devrites/work/<slug>/`—are **workflow artifacts**, not product
source/tests. They remain excluded from the product candidate and never authorize
the real action they prepare.

## Ownership

After an exact design and path set pass Vet, the **controlling root is the sole materializer**
of these artifacts under its existing `.devrites/**` authority. For these paths,
never dispatch `devrites-slice-wright`,
never widen the wright contract, and never ask a
read-only planner/reviewer for implementation bodies. Product source/tests remain
wright-only.

Missing implementation bytes are an agent-owned materialization task, not
technical-recovery exhaustion, while the vetted behavior and exact paths are
complete. The root may not invent unresolved protocol choices; those return to
Plan/Vet.

## Admission

Before writing:

1. Require the active slug, current readiness binding, resolved human gates, and
   an exact file list in `plan.md` / `test-plan.md`. Reject directories, globs,
   traversal, duplicates, symlinks, paths outside the active feature workspace,
   and any product source/test or dependency path.
2. Bind the complete behavior, interfaces, failure relations, proof commands,
   expected signals, and rollback for the atomic artifact set. A prose placeholder
   or a request that a drafter supply code is a Plan gap, not implementation input.
3. Record preimages or absence for every target plus protected product manifests
   and the current candidate digest. Resolve every parent no-follow before mutation.

## Materialize

The root authors the smallest complete bytes for all admitted targets. Write each
through a same-parent private temporary file, set the planned mode, flush it, and
atomically replace the target; settle the whole set or roll back partial output.
Do not install dependencies, touch Git, use the network, mutate product paths, or
execute the consumptive action.

## Verify and return

Read back the exact path set, modes, and SHA-256 identities. Run only the vetted
compile/static/fixture/mutant commands in their isolated non-consumptive modes;
require every discriminating signal. Recheck protected preimages and require the
candidate digest remains identical. Record the workflow-artifact hashes and proof
in canonical evidence, but never add these paths to the product candidate manifest
or built-slice count. Run the affected narrow Vet, restore the controlling return
cursor, and stop for fresh authorization before any real consumptive action.

An actual host permission failure, unresolved design choice, out-of-scope path, or
failed atomic/proof check blocks truthfully. The absence of a writable specialist
does not: root ownership is the supported path.
