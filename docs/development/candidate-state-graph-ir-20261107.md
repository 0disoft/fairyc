# Fairy detached state graph IR, 2026-11-07

Scope: [ADR 0511](../adr/0511-candidate-state-graph-ir.md), following the
[seven-codec layout prerequisite](candidate-state-graph-20261107.md).

## Implemented owner

`module_state_graph_model` owns bounded, opaque graph data with typed parameters,
operands, operations and explicit terminators. Its assembly function grants no
proof. `module_state_graph_values` checks canonical codecs, metered definitions,
block-local lookup and trap categories. `module_state_graph_verify` independently
checks the selected layout, all packed spans, transaction identity/effect/cost,
global value uniqueness, local availability, typed edges, reachability and
acyclicity. Its sealed `Verified` resource retains the graph and its own immutable
layout copy; it exports only immutable borrows.

The initial graph family admits six cell/map operations and typed
jump/branch/commit/reject/trap. Root borrowing/view/copy, helper calls, current
source lowering/Complete binding, exact state-body identities, artifact/cache
restoration, native execution and ordinary commands remain separate. Neither
an assembly call, numeric identity nor detached proof grants those authorities.

## Independent Windows execution

The new test constructs original HIR and lowers it through the existing Go
state-MIR owner, retaining its independently created transaction identity. A
test-only packet carries its structural facts to authored Fairy assembly; the
production Fairy verifier supplies every acceptance decision. Independent Go
`contractstatemir.Verify` supplies expected acceptance/refusal for each program.
The packet reader is a fixture, not a production artifact protocol or proof.

There are **56 programs per profile / 112 independent native processes**:

- Typed branch/join with cell reads, found values, literal map keys and dynamic
  map/cell writes; globally unique definitions and explicit block parameters.
- Cross-arm/future/unknown values, duplicate definitions, missing/aliased found
  results, non-bool branches, mismatched/missing/out-of-range edge arguments,
  field/kind/type drift, missing/foreign effects and transaction identities.
- Canonical operation/terminator costs, dense IDs, entry parameters,
  unreachable blocks and a reachable cycle.
- 63/64/65 blocks and 1,023/1,024/1,025 operations.
- All seven codecs, malformed bool/fixed widths/text and valid shortened bytes.
  Dynamic bytes are raw data: truncating bytes remains a valid encoding and is
  correctly accepted by both implementations.
- Trap categories of length 1/127/128, empty categories, boundary hyphens,
  uppercase and non-ASCII refusals.

For each accepted program, native Fairy validation also succeeds with exactly
the remaining measured 16 MiB work, rejects a one-short budget without exceeding
the bound, and rejects an already-over-bound ledger without modifying it. A
successful sealed factory retains a nonempty graph and independent layout.
These are detached validation tests; they do not execute a state transaction.

```powershell
go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateStateGraphIR$'
go vet -mod=readonly ./internal/cli
git diff --check
```

The native command passes Windows MSVC low/optimized in **41.787s**, with no
selected skips. Scoped vet and whitespace checks pass. Initial pre-native
attempts found missing fixture imports, attempted external opaque-resource
access and a deferred Result match; these were corrected using explicit import
closure, immutable accessors/assembly and immediate Result matching. A fixture
editing command also failed on the Windows default text decoder before its Go
source edit; the subsequent undefined-constructor failure was corrected with an
explicit UTF-8 edit. None of those failed attempts counts as native acceptance.

## Current-source integration check

A freshly built Go bootstrap checks an isolated copy of the complete current
example, including the already committed Linux observed-read change. Both
`check` and `check --locked` pass **704 source files / 8,750 MIR functions**.
The checked lock is copied back to the example. Local records are under
`.cache/state-graph-source-check-20261107/check-results.json`; the executable is
`.cache/fairyc-state-graph-integration.exe`. Generated records are evidence,
never source authority. Compiler-sized example test execution is not repeated.

All authored files are nonempty UTF-8/LF with final newlines. Stable CLI/SDK/
runtime ABI, dependencies, database/provider, ordinary compiler limits,
repository policy and CI/runners are unchanged. Native Linux, full Go/Fairy
comparisons, consecutive self-build, distribution/clean-machine and same-commit
hosted acceptance remain in the full implementation queue. No push or dispatch.
