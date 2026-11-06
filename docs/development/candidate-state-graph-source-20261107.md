# Candidate authored state graph source: first bounded producer

Status: implemented and tested on Windows amd64, low and optimized. This is a
private straight-line source subset, not full Go graph-source parity, a Complete,
an executable state transaction, or a cache hit.

## Scope and API

Owner: `examples/lexer_candidate/src/module_state_graph_source.fy`.

```fairy
pub fn Result<Produced,u32> produce(ref u8[] input,ref mut usize work)
pub fn ref u8[] source(ref Produced p)
pub fn Owner owner(ref Produced p)
pub fn ref u8[] body(ref Produced p)
pub fn Span operation_span(ref Produced p,usize index)
pub fn Span exit_span(ref Produced p)
pub fn ref m::Graph graph(ref Produced p)
pub fn ref state::Plan layouts(ref Produced p)
pub fn bool matches(ref Produced p,ref u8[] current)
pub fn usize source_limit()
```

`Produced` has no public constructor or mutable getter. It owns the parser's
original source stream and the independent graph verifier's sealed graph and
copied layout owner. Input bytes are copied before scanning, so later caller
buffer mutation cannot alter source, spans, graph, or layouts.

`Owner` records `module_index`, `package`, one-based syntax `declaration`, catalog
`identity`, `declaration_span`, `body_span`, and `effects_span`. All spans are
half-open offsets into the exact retained source, including CRLF and comments.
The declaration span includes its attribute prefix; the body span includes both
braces; effects span is the complete effects attribute, or `[0,0)` when absent.
Operation spans cover call expressions without their semicolon; exit span covers
the complete terminal statement. Returned owner/span values are copies.

The initial source set is one module with exactly one private `@contract_uid`
state declaration and one private safe nongeneric parameterless `void
transaction()` function. State and root can appear in either source order.
`module_state_declarations::build_graph_metered`, the metered variant of
`build_graph`, derives canonical UID, field order, field prefixes, codecs and
layout digest from this exact source and its freshly built catalog. No caller
layout, catalog, Parsed arena or graph is accepted.

Admitted operations are literal `storage::read`, `storage::write`, and
`storage::delete`, for cells and maps. Discarded read results still receive
distinct monotonically derived value/found IDs. Every operation and exit has
cost 1. Capability is the freshly derived root catalog identity, shared by the
graph and its operations. Effects are read from the root's source attribute and
independently checked against its operations by `module_state_graph_verify`.
Missing required effects, duplicate effects, foreign effects and unknown root
attributes fail. As in the Go root oracle, unused declared storage effects are
allowed; helper exact-effect rules are not widened.

Literal support includes bool, u32, u64, bytes20, bytes32, bytes and UTF-8 text,
including empty dynamic values, embedded binary bytes and escaped field names.
Integer operands must be unsigned decimal source tokens, with checked maxima
`4294967295` and `18446744073709551615`. Decimal multiplication is performed in
4/8 canonical big-endian bytes, with a carry check; it does not narrow a u64
through signed integers or host usize. Hex, binary, separators, signs, floats and
wrong literal families fail closed in this private source profile.

Exactly one explicit final `storage::commit()`, `storage::reject()` or literal
`storage::trap("category")` is required. The independent verifier enforces the
existing lowercase/digit/hyphen trap category policy. Statements after an exit
are rejected rather than silently ignored.

## Bounds and exclusions

- Source limit is privately 65,536 bytes inclusive; 65,537 is refused. This does
  not claim the complete Go source-size family or general workspace admission.
- The existing independent scanner limits and parser limits remain in place.
  The producer charges source admission, literal/field/path work, reversed block
  link traversal, declaration derivation, graph verification and layout copying
  against the supplied 16 MiB phase ledger. Scanner/parser retain their own
  bounded internal work limits; this is not a wall-clock or allocation quota.
- At most 1,024 storage operations plus one exit. Parsed reverse statement links
  are materialized once in a bounded local array, preserving source order with
  O(N) traversal instead of repeated tail scans. Field lookup remains bounded
  linear lookup; independent graph value verification retains its own bounds.
- Local bindings, copied/dynamic operands, branch/join control, helpers, borrows,
  views, generic calls and ordinary statement families return a refusal. Tests
  include Go-accepted locals, branches, borrows and helper source explicitly
  refused here. This source producer does not yet admit the wider detached IR.
- `matches` compares exact source bytes only. No filename, source-root membership,
  package selection, source-set identity, ordinary MIR owner or Complete is
  inferred from that comparison. Catalog identities are source-local coordinates,
  not Go's opaque transaction token or runtime storage authority.

Errors are `1001` malformed/unsupported source, `1002` work/size bound, and `1003`
allocation failure. No partial Produced escapes any error. Actual allocator
failure injection was not performed.

## Validation

Final command:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphSource(Oracle|SealedOwner)?$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_state_graph_source_test.go
git diff --check -- examples/lexer_candidate/src/module_state_graph_source.fy internal/cli/candidate_state_graph_source_test.go docs/development/candidate-state-graph-source-20261107.md
```

Final source tests passed in 29.699s:

- 83 independent Go source/MIR oracle fixtures passed. Positive native output is
  compared against actual `compiler.CheckExperimentalStateGraph` output: layout
  UID/hash and all field prefixes/codecs, source/root/effects/body spans, operation
  order, selected field indexes, literal bytes, read/found IDs, costs, effects and
  terminal category/span. No caller-supplied IR packet substitutes for production.
- 166 native process checks passed: all 83 fixtures in each Windows profile.
  This includes all six cell/map actions, every codec family, u32/u64 maximum
  minus one / maximum / overflow, 1,024 versus 1,025 operations, 1,024 distinct
  reads, the 65,536/65,537-byte boundary, malformed input, wrong effects, wrong
  owners, wrong operand counts/families and closed wider-source shapes.
- Every successful native fixture also reconstructs at exactly the measured work
  allowance, receives `1002` with one unit less, refuses an already-exhausted
  ledger without modifying it, independently revalidates the graph, and checks
  exact-source drift after mutating the caller buffer. These repetitions are
  internal assertions, not extra fixtures in the 166-process count.
- Five semantic refusal checks passed for a forged private constructor and
  mutable graph, layout, source and body escapes.
- Scoped vet, Go formatting, diff hygiene, and owned-file UTF-8/LF/final-newline
  checks passed.

An earlier 76-fixture native set passed 152 processes in 18.531s. Subsequent tests
expanded the boundary coverage and required the exact `1002` exhaustion error.
Initial probe/import/ref-assignment/cast/resource-join issues were corrected
before these passes. One intervening package compilation was blocked by another
worker's temporarily incomplete `stateGraphBodyTypedContexts` test definition;
that worker's file was not modified here. None of those failed commands is
counted as a validation pass.

Linux native execution is deferred for the batched Linux run. Ordinary
`check/build/run/test`, current Complete integration, artifact reuse, cache
restoration, persistence and actual state execution were not tested or granted
by this producer. Other workers own artifact/source-binding work separately.

No source-of-truth contracts, public CLI/SDK API, DB schema, dependency versions,
runner policy, shared graph APIs, lockfile or queue were changed in this slice.
Only the new producer, its new test file and this receipt belong to this worker.
No staging, commit, push, CI dispatch, release or deployment was performed.
