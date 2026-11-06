# ADR 0503: Explicit source admission for state graphs

- Status: Accepted for a compiler-private source entry point
- Extends: ADRs 0378, 0383 and 0502

The subsequent [ADR 0504](0504-source-state-graph-helpers.md) adds bounded
source helpers and graph call admission; the original single-root checkpoint
below retains its scope. [ADR 0505](0505-source-state-read-borrows.md) adds
root read borrows; the full integration queue remains open.

`compiler.CheckExperimentalStateGraph` admits one parsed source file containing
one state declaration and one private, safe, nongeneric, parameterless
`fn void transaction()`. The original `CheckExperimentalState`, ordinary
compiler entry points and stable CLI keep their existing admission. This does
not add an SDK module or promote experimental storage calls to ordinary calls.

Normal semantic analysis owns the source/module/package identity, layout,
signature and declared effects. The explicit adapter checks every executable
statement structurally, then attaches its graph to that same HIR owner. Ordinary
and state MIR are published together only after their existing verification.
The caller's AST and its slices are never modified. No unadmitted body is
silently omitted from execution.

## Operations and values

The existing source schema admits `bool`, `u32`, `u64`, `bytes20` and `bytes32`
cells and maps. Read/write/delete take a literal field name; maps additionally
take one key operand before a write's value. Operands are exact typed prior
read values or literals: booleans, decimal unsigned integers, and exact-length
byte strings for fixed bytes. Scalar overflow and fixed-byte length mismatch
refuse, rather than truncate, pad or infer another codec.

`let name = storage::read(...)` binds the immutable copied value and its
presence result. A missing read has the state-MIR codec's canonical zero value;
`storage::found(name)` tests its separately retained presence. Immutable aliases
retain that presence; an alias of a found result is a plain bool. Optional type
annotations must match the storage codec. Discarded reads still execute and
retain observations. Assignment, mutable locals and local shadowing refuse.

Structured `if` uses a read bool or a presence result. Block arms can contain
further reads, writes, deletes and nested branches. Current outer copied values
and their presence bits pass through explicit typed SSA block parameters on
every branch and join; arm-local names do not escape. The parser's final-if
tail form has the same statement behavior. There is no implicit value tail,
mutable phi, loop, arithmetic, arbitrary call or inferred branch condition.

Every reachable root path must explicitly commit, reject or trap. Statements
after a closed path refuse. Trap categories re-enter the existing state-MIR
category checks. Every actual storage action and control terminator costs one;
aliases/presence projection do not synthesize another read. The existing 64
block and 1,024 operation bounds also apply during source graph construction,
before unbounded work or native source generation. Native slot/transport limits
remain separate unchanged checks.

## Remaining sequence

This is the first source-graph slice, not completion of native state integration.
The checked-in example and Windows tests pass the compiled owner through
`PrepareCompiled`, actual generated graph C, the sealed response admission and
one-attempt local-store CAS. The admitted subset also has explicit native
developer selection under ADR 0500. Source helper/borrow syntax and dynamic byte/text
schema admission, ordinary CLI selection, current Complete/source provenance,
artifact/cache invalidation and full acceptance remain required. The broader
seven-codec/helper/borrow HIR/native runtime remains independently available.

No persistence provider, public ABI, target choice, cache hit authority,
dependency, default timeout, runner or hosted execution changes here. Linux
native verification remains in the combined batch.

Evidence owners: `internal/compiler/state_graph_source*.go`,
`internal/compiler/state_graph_operands.go`,
`internal/contractstatenative/graph_source_windows_test.go`, and
`examples/contract_state_graph/transaction.fy`.
