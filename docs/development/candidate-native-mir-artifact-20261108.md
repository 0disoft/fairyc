# Source-bound native MIR artifacts, 2026-11-08

Authority: ADR 0551. This is the private `mir_model` source/execution pipeline,
separate from the `module_unit_*` cleanup/body IR and existing project envelopes.
It does not grant the latter families closure admission or a project cache hit.

## Exact format

`FYNMIR00` has no closure feature or closure table. `FYNMIR01` requires an
actual closure feature, including a signature-only consumer with zero literal
targets. A scalar-only program in 01, closure records in 00, unknown versions,
trailing bytes and noncanonical flags/u32 words fail. Existing project/unit
artifact markers, decoders and no-closure bytes are unchanged.

All integer fields use eight big-endian bytes. Booleans are exactly 0/1;
u32 fields require zero upper words. After the marker come module index,
length-prefixed owned source and literal payload. Count-prefixed tables follow
in this order (field count per row): shapes (5), fields (6), ABI facts (6), SDK
scopes (3), drops (4), targets (14), signatures (1), functions (11), locals (7),
blocks (20), operations (17), edges (1), then closure targets (2) only in 01.
These are the corresponding authored model fields in declaration order.

Closure targets record a symbolic target ID and capture count; target rows
retain source coordinates and signatures. Construction/call rows and ordered
capture edges retain operands. There is no native invoke pointer in this data.
The decoder checks count, fixed-width remaining bytes and arena caps before
allocating each table, then enters the explicit family assembler and the full
MIR verifier. Owned source is bounded at 1 MiB, payload at 688128 bytes and the
complete artifact at 8 MiB. The shared encoding/decoding ledger is 16 MiB.

## Current-source authority

`native_mir_artifact_current::restore(raw,current,module_index,work)` requires
caller-owned original source. Detached decoding establishes structure and flow
only. The source-bound owner checks source/module identity, discards that
detached owner, and re-enters scanner, executable parser/private preparation,
name resolution, type checking, HIR and MIR lowering. It canonically encodes the
newly checked model and compares every stored byte before returning that new
model. It never executes the untrusted decoded body as current-source authority.

This deliberately rechecks the source; it is not a warm-cache acceleration
claim. Frontend phases keep their own bounded work/arena/depth policies. The
outer ledger charges input/arena admission fees, not unobservable CPU steps.

## Validation

Windows acceptance is owned by `TestCandidateNativeMIRArtifact`. Both low and
optimized Fairy producers start from original source, check canonical snapshots
and work limits, and execute seven source-binding groups. A fresh process reads
each serialized artifact through both detached and current-source owners and
must emit the same C. That C runs with both consumer profiles, including checked
trap and ordering observations. Helper swaps, capture rewrites, coherent checked
body rewrites and literal payload changes must fail source binding even when
detached MIR remains valid. Unsafe/effectful targets and bad signatures must
fail earlier.

The coordinated command passed on Windows: package **69.159s**, acceptance
**69.03s**:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateNativeMIRArtifact$' -count=1 -v -timeout=240s
```

Eight original-source cases, two producer profiles and two emitted-C profiles
give **32 program executions**. There are **16 fresh detached restores** and
**16 fresh source-bound restores**, each pair requiring byte-identical emitted
C. Both producers execute seven source-binding groups, including ten forged
record cases, and reject fourteen additional wire/MIR mutations each (**28**).
Canonical artifact bytes match between the producer profiles. Exact and
one-over shared-work limits, truncations and noncanonical versions also pass.

Initial combined fixture attempts failed honestly: one-sided helper retargeting
left an orphan metadata row, and a body rewrite failed to update its checked
terminator. The fixtures now swap both constructors and coherently rewrite
guard/operation/terminator, so detached acceptance is established before the
source-bound refusal is asserted. No production check was weakened.

After integration of the independently verified generic source-policy fix and
safe-function dependency traversal, the coordinator reran this acceptance and
their focused groups in one sequential command. It passed: package **153.822s**,
native artifact acceptance **78.74s**, source-policy native **35.61s**, source
dependency native **27.64s**. The local full output is retained at
`.cache/closure-artifact-source-integration-20261108.log`.

Linux,
source-bound project/unit execution, persistent project-cache restoration,
whole-compiler self-build, distribution and hosted CI remain separate work.
No normative source-of-truth document, public SDK ABI, database, dependency,
runner setting or remote state is changed by this slice.
