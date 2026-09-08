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

SDK element-access functions, mutable access,
artifact/cache transport, stable CLI and installed SDK admission remain separate.
No stable grammar, SDK source/version, runtime ABI or cache format is promoted.
