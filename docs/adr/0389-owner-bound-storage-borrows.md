# ADR 0389: Owner-bound storage borrows

- Status: Accepted private source, MIR and Windows native prerequisite
- Date: 2026-09-09
- Depends on: ADR 0388

## Decision

Prepare a compiler-private shared `storage_borrow<T>(owner, address)` operation
for non-consuming vector element access. Private generic expansion alone may
produce its AST node. It requires exactly one supported scalar or nominal type
argument, one shared safe owner borrow, one opaque raw pointer and an enclosing
unsafe scope. Its HIR result is `ref T`, with the exact element type and source
unsafe scope retained. It neither moves the element nor grants release authority.

The result inherits the first argument's borrow origins and loans, never an
independent lifetime inferred from the raw pointer. Existing function-return
summaries propagate that relationship through direct and relay calls. Source
analysis rejects owner moves before last use, local-origin returns and malformed
or non-unsafe construction. Raw address validity, alignment, initialized element
state and correspondence to the owner remain unsafe-caller obligations, not facts
established by this source operation.

## Temporary borrow compatibility

The source integration exposed a missing check for explicit borrows that exist
only as call arguments. Borrow creation now compares retained loans live after
that expression and loans retained by earlier arguments of the enclosing call.
Two shared borrows remain compatible; a mutable borrow conflicts with a live
shared or mutable loan. Temporary argument borrows are released at call completion
instead of becoming function-long facts. Only borrow-creation expressions retain
the additional backward-liveness snapshots.

This restores the existing E2101 access rule for these paths, including a
retained view passed as an earlier or later argument and two conflicting direct
argument borrows. Separate calls, compatible shared arguments and mutation after
the last view use remain accepted. This is not a claim of general alias analysis
or mutable element-view support.

## Evidence and remaining work

Compiler source tests cover scalar and resource element references, original AST
preservation, exact Param(0) return origins through two functions, malformed
owner/pointer/arity, unsafe omission, local escape, owner move and exclusive access.
Independent ordinary byte-array tests cover temporary argument conflicts and
last-use release. Semantic regression and existing low/optimized MSVC vector and
cache tests pass.

Private lowering now preserves the shared owner and opaque address as separate
operands, freezes the owner before address evaluation, and retains exact result,
owner and address types plus unsafe source identity. The transfer verifier admits
this operation only with its closed shape, live operands and matching unsafe
provenance; ordinary and query-only verification reject it. Source-derived loan
and return-origin checks remain part of compiler publication. This does not add
independent MIR alias analysis or prove raw-address validity from operand types.
Mutation tests cover missing or dead owners, mutable results/owners, invalid
element/address types, result aliasing, unsafe identity and ownership flags.

The private C backend checks the actual owner/address operand types against the
retained MIR types and rejects swapped operands. It requires a supported nonzero
element layout and emits a shared pointer assignment, not a value copy or move.
Generated Fairy programs read scalar and resource elements through a returned
borrow and a relay function under both low and optimized Windows MSVC profiles.
The independent C probe checks a value of 41, one owner cleanup and zero borrowed
element cleanups; the process must exit 42 without output. The source rejection
and MIR mutation tests remain active. This proves the private Windows transport,
not a general guarantee about arbitrary unsafe addresses or another target.

The explicit vector candidate now exposes `at<T>(ref Vec<T>, usize) -> ref T`.
It checks the initialized-prefix bound before calling a closed read-only slot
transport, checks the returned address, and binds the reference to the vector.
The raw helper validates descriptor shape, element size and index without changing
length, capacity or payload. Candidate Buffer tests borrow first/last initialized
elements, preserve ownership and allocation counts, and then pop after last use.
Empty and past-end accesses trap under low/optimized MSVC using runtime-provided
indices. Stable SDK loading remains byte-identical.

The integration exposed a missing direct resource-move loan check. Binding and
assignment RHS moves now use the same E2105 owner-loan rule as consuming calls;
grouped moves are included. Independent resource tests reject live-loan moves and
accept moves after last use. Candidate tests reject clear/reserve/pop with E2101
and direct vector moves with E2105 while an element reference remains live.

Private package artifacts now stabilize and rebind the storage-borrow source
identity alongside its unsafe scope and operand types. A resource-borrow artifact
round trip with different package/function numbering executes under low/optimized
MSVC; ordinary writing and schema-substituted ordinary reading remain rejected.
Manifest-loaded candidate programs containing `at` obtain both package hits from
memory and from a reopened disk-cache session, retain the borrow operation, and
execute under both profiles. Invalid source and active-view mutation are rejected
before cache reads/writes change cache statistics.

Optional or mutable element access, stable CLI and installed SDK admission remain separate.
No stable grammar, SDK source/version, runtime ABI or cache format is promoted.

## SDK-only operation admission prerequisite

The private integration layer now accepts an explicit set of package owners for
storage layout, take, put and borrow operations. It rejects empty, duplicated or
absent owners and enables these intrinsics only in their defining expanders.
Consumer generic calls still instantiate the selected provider's implementation;
module names and import aliases confer no privilege. The prior unrestricted
private test lane remains separate and unchanged.

`AnalyzeOwnedStorageTransferPackages` binds those selections through the current
package-coordinate map. Full SDK Buffer vector construction, reads and cleanup
execute through the restricted path under both MSVC profiles; the same consumer
cannot directly use a storage-take intrinsic. Source tests cover all five private
intrinsic spellings, exact owner selection and detached input preservation.

This selection API is not source authentication. The candidate workspace now
supplies that missing check: `VectorCandidateStorageOwners` requires explicit
candidate loading, exact SDK kind/identity/runtime/digest/module catalog, and the
complete ordered source paths, bytes and parsed ASTs from the embedded bundle.
It rejects test overlays inside the SDK and SDK claims on other package paths.
Only the verified catalog's resolved SDK owners receive storage authority.

`NewSDKStorageSession` applies this check before analysis/cache lookup and
uses the owner-restricted compiler path. Its compiler-build key has a separate
SDK-owned discriminator from the unrestricted private test session. Real memory
and reopened disk hits still execute Buffer reads and cleanup under both MSVC
profiles. Consumer intrinsics, mutated SDK ASTs and invalid current source leave
cache statistics unchanged. Loader mutation tests cover bytes, AST, source paths,
order/count, test flags, digest, labels, runtime and module catalog.

The shared uncached `workspacecompiler.Check` and bounded `SourceSession` now
recognize an explicitly candidate-loaded workspace. They verify its source
authority before selecting owned storage analysis; default-loaded workspaces
retain the previous ordinary/generic admission rules. SourceSession keeps one
active cache family across candidate/ordinary transitions. Tests require memory
entries to be discarded on a family change, exact disk entries to survive a
round trip, and forged SDK input not to alter the active cache.

The Windows workspace build path now carries the verified SDK storage selection
into the storage-aware emitter. Its support composition includes one system heap,
the closed vector runtime and the existing product/run trap transport. Consumer
declarations of private vector runtime symbols are rejected before native build.
Focused CLI tests load the candidate bundle and execute Buffer element access,
console output and out-of-range assertion traps under low/optimized MSVC. Ordinary
support composition is unchanged and unknown private vector symbols are rejected.

Native test dispatch now shares this exact SDK authority check. The selected
test harness uses storage-transfer MIR verification and the same storage-aware
Windows emitter; ordinary native tests retain ordinary verification. Candidate
test overlays execute through discovery, analysis, isolated native processes and
the existing report writer. Both MSVC profiles preserve an assertion-failing
vector test followed by a passing Buffer-borrow test, captured console bytes and
source locations in the JSON report. Existing native runner isolation tests pass.

At this native-dispatch prerequisite stage, default loader/SDK selection remained
unchanged and installed SDK promotion was separate. ADR 0390 now supersedes that
exposure boundary while retaining the source/borrow guarantees established here.

Before default SDK admission, the workspace integration uses a distinct
`AnalyzeSDKStoragePackages` path. Only authenticated SDK owners may declare
generic aggregates or instantiate templates with nominal arguments. Consumer
generic functions retain builtin-scalar constraints, including transitive calls;
ordinary nominal values may still be passed to SDK Vec operations. The broader
owner-restricted private compiler API remains unchanged for experimental evidence.
The SDK-owned cache discriminator advances to v1. Memory/disk admission tests
reject consumer aggregate declarations and nominal function instances before
cache statistics change; scalar consumer functions plus Vec<Buffer> still pass.
