# ADR 0505: Explicit source state read borrows, views and copies

- Status: Accepted for the experimental state graph source entry point
- Extends: ADRs 0381, 0503 and 0504

Admit immutable untyped source locals initialized with
`storage::borrow("cell")` or `storage::borrow("map", literal_key)`.
The existing five source codecs and exact canonical key rules apply. These
locals are opaque owner-bound handles, never ordinary scalar values. Same-block
immutable aliases retain the handle and frozen presence result.

`storage::view(borrow, decimal_u32_offset, positive_decimal_u32_length)` creates
a bounded nested handle; `storage::copy(borrow)` explicitly creates an ordinary
copied value retaining its source codec. A borrowed handle must be copied before
it can supply a write value or dynamic map key. Copy results may have an exact ordinary
type annotation; borrow/view locals may not. There is no implicit dereference,
conversion, mutable view, raw pointer, helper-owned borrow or borrow argument.

`storage::found(borrow)` exposes the original read's frozen presence bit. The
bit is an ordinary bool; it does not keep a handle alive or assert current
storage presence after later mutation. Missing borrow copy/view fails at
execution, with no publication. A safe conditional pattern first reads a value
and tests its presence, then creates/copies the borrow inside the selected arm.

## Block and mutation lifetimes

Borrow handles are never graph edge arguments or target parameters. A source
branch ends their lifetime even if an outer lexical name is still visible.
The adapter carries ordinary copied values and frozen presence bits to
successors, while dropping all handle authority. A later view/copy/alias of
the ended handle refuses. Creating a fresh borrow inside an arm is permitted.

All represented operations re-enter existing state-MIR verification. Exact-key
replacement or deletion invalidates prior handles, even for equal bytes.
Different literal map keys preserve them; a dynamic mutation of that map while
a borrow is live remains conservatively closed. A writing helper invalidates
prior handles under ADR 0504, while read-only helpers and values copied before
a writing call retain their independent lifetimes. Nested views preserve origin.

View bounds reject negative, computed, zero-length and wider-than-u32 source
operands. Runtime range checks retain subtraction-based overflow safety and
the current graph/transport limits. A copied partial fixed-width value is still
that codec; a later typed write/key must be canonical and may fail. There is no
padding, truncation or automatic reinterpretation to bytes. Dynamic bytes/text
schema source admission remains a separate task, although those codecs are
already available to independently authored HIR/native graphs.

## Windows evidence

Source tests cover same-block aliases/presence, copied values crossing joins,
borrow creation inside presence-selected arms, handle escape refusals, exact
and ambiguous map mutation, helper read/write boundaries, type/operand refusal,
u32 maximum/overflow and exact/outside view ranges. They assert atomic refusal
without MIR/state publication or runtime failure without persistent writes.

The authored `examples/contract_state_graph/borrows/transaction.fy` performs
nested views around a read-only helper, copies before a writing helper, then
uses the original copied 42 after the source is replaced with 99 and after a
branch. Both branches and commit/reject/trap pass real Windows low/optimized
C execution, store close/reopen and independent Go comparisons: 24 attempts.
Ten more native attempts cover exact literal map borrows with unrelated-key
mutation for every source codec. Eight actual local-store failures cover absent
borrow, one-past-end and UINT32_MAX view bounds, and noncanonical fixed-width
copy/write. Every failing attempt has one snapshot, zero CAS and unchanged
durable bytes. Both profiles also pass the developer command's init/execute/
snapshot path for the helper and borrow sources, including exact costs 12/16.

The source AST remains detached, original ordinary/single-cell admissions stay
closed and native execution uses actual generated C. The call-only projection
is not Complete/source-cache authority. Linux, dynamic bytes/text source,
Complete/artifact/cache/ordinary-CLI and final acceptance gates remain queued.
No stable language revision/CLI/SDK ABI, DB/provider, dependency, default limit,
runtime ABI, repository-policy or runner changes occur here.
