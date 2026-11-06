# Candidate state graph borrow verification, 2026-11-07

This slice implements detached root borrow/view/copy verification in
`examples/lexer_candidate/src/module_state_graph_borrows.fy`. The policy owners
are `internal/contractstatemir/borrow_verify.go`, `graph_verify.go`, `verify.go`,
and `helpers.go`. No language, product ABI, artifact format, target maturity,
dependency, runner, or public execution bound changes are introduced.

## API and ownership

The module exports `Ledger`, `empty`, `apply`, `invalidate`, and `invalidate_all`
using the integration signatures agreed with the graph verifier owner. `apply`
accepts the operation, immutable graph/layouts, ordinary global/live value tables,
global borrow identity table, block-local ledger, helper flag, and work ledger.
The core creates a fresh ledger for every block; the global borrow identity table
persists across root blocks. Helpers cannot own any borrow operation.

`Ledger` is nonowning plain metadata backed by fixed scalar arrays with capacity
1024, matching the existing graph operation bound. This deliberately replaces the
initial resource/Vec design: mutable resource-field projections remain closed by
the language contract. This representation keeps that language boundary closed. No
unsafe access or exported mutable ledger getter is used. Count checks precede
array indexing, and a full ledger cannot accept another handle. The ledger itself
is not authority: the sealed verifier constructs and uses its own ledger and
does not accept a caller-supplied one.

Borrow IDs must be nonzero globally unique u32 identities with known codecs.
They use a separate namespace from ordinary ValueIDs: a borrow and its found or
copied value may have the same numeric identity. Found results are bool; copied
results preserve the source borrow codec. Both ordinary result families are
defined consistently in the global and block-live value tables.

## Borrow policy

- Cell and map borrows require the root capability, cost 1, read effect, correct
  layout field kind, one borrow result, and one found result. Unrelated storage,
  call, write, and view fields are rejected.
- Map borrows accept only a canonical literal exact key. Present empty keys are
  allowed for codecs that admit them. Dynamic borrow keys are rejected.
- Views require an available valid input borrow, a fresh borrow identity, a
  nonzero length, and no storage/result/found fields. Offset and length must fit
  u32 because the private model stores them in usize while Go uses uint32.
- A view retains its root field and exact key. The verifier does not compare
  offsets/lengths with runtime encoded widths, add them, or apply static codec
  width restrictions. Go admits those shapes; runtime range/encoding traps are
  a separate gate. Fixtures include all seven value codecs and maximum u32
  offset/length.
- Copy requires an available valid borrow and defines an ordinary value. A copy
  survives subsequent storage mutation and can cross a block edge by the usual
  typed value arguments. Bare borrows cannot do so: core edges resolve ordinary
  live values only. No separate escape API was necessary.
- Cell write/delete invalidates every live borrow/view of the same field. Map
  literal write/delete invalidates only the same field and byte-equal key. Keys
  are immutable graph-data spans; equality does not depend on packet offsets.
- A dynamic map mutation rejects any same-field live borrow, even when no later
  copy uses it. Different fields and already-invalidated borrows do not alias.
- A call with verified transitive write effects invalidates every earlier
  borrow/view, including unrelated fields. Read-only helper calls preserve them.

All ledger scans, result definitions, and byte comparisons charge the existing
16MiB work ledger before the corresponding work. Errors retain the core private
mapping: 1001 malformed, 1002 work/bound, 1003 allocation. Ordinary mutation
operands are validated by the core before `invalidate` runs.

## Independent fixtures and executed checks

`internal/cli/candidate_state_graph_borrow_ir_test.go` builds its own matrix from
`stateGraphIRBase(t)` and checks expected acceptance against
`contractstatemir.Verify` before any native build. It reuses `stateGraphIRPacket`
with its 19-word operations and derives the probe from `stateGraphIRProbe`
without editing the shared base test.

The test-only packet wrapper preserves unknown nonempty callee names using a
nonzero out-of-range sentinel; the shared encoder otherwise represents unknown
names as zero. This prevents malformed call metadata from disappearing before
the candidate verifier sees it. Direct private wire mutations also check that
offset/length values greater than u32 are rejected.

The native probe uses `candidateAtlasProbeSize`, rooted at
`module_state_graph_verify`, with fixture modules `module_state_declarations`
and `candidate_artifact_wire`. It builds only their authored import closure.
Both Windows MSVC low and optimized profiles use the existing 16MiB fixture
stack reserve. The matrix includes a chain of 1024 handles at the unchanged
operation limit, global identity collisions, detached copies, invalidation,
dynamic aliasing, helper effects, malformed metadata, and block lifetime/escape.

Accepted standalone roots exercise the sealed graph factory and its immutable
layout owner. Every accepted context exercises the exact work boundary, boundary
plus one, and an already-exhausted work ledger from the original probe. Helper
fixtures supply the Go-verified transitive effect table to
`verify::validate_function`; this validates borrow behavior in context and does
not manufacture a sealed complete-program proof. The program owner's fixtures
own that separate proof boundary.

Executed successfully on Windows:

```text
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphBorrowIROracle$' -count=1
ok github.com/0disoft/fairyc/internal/cli 0.120s

go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphBorrowIR(Oracle)?$' -count=1 -timeout 5m
ok github.com/0disoft/fairyc/internal/cli 18.247s
```

The final combined run includes the 1024-handle fixture and both native profiles.
The initial resource-field design failed semantic admission before native work;
the fixed-array replacement passed. The first borrow matrix run exposed the
callee transport loss described above; the wrapper correction passed the matrix.
These earlier failures are not counted as passing checks.

## Remaining gates

This is detached graph verification, not current-source graph production or
Complete authority. Candidate graph artifact encoding/decoding, canonical body
hashes, fresh-owner/current-source rebind, and cache admission are separate
unfinished integration gates. Runtime storage access, actual borrow/view/copy
execution, runtime range traps, snapshot/overlay behavior, and native end-to-end
transactions were not implemented or claimed by these verifier fixtures.

Linux native execution is deferred. No heavy full-compiler build or Atlas suite
was run. The parent's existing root regression and the other worker's complete
helper-program matrix are independently owned checks, not results claimed here.
The worker authored only its three assigned new files. Parent integration also
updates the common graph operation/context verifier, test packet and owning ADR.

## Integrated regression

The final combined graph/borrow/helper command passes in **50.456s** on Windows.
The borrow matrix checks 87 independently expected Go cases and executes 89
cases per native profile (178 processes), including two private u32 wire guards.
There are no selected borrow skips. Root/helper interaction is separately replayed
by the whole-program helper tests: read-only preserves a borrow, a write call
invalidates it, and an ordinary pre-call copy survives. Scoped vet, formatting,
UTF-8/LF/final-newline and diff checks pass. Source lock/whole-example results
are recorded with the following helper integration commit.
