# Candidate state graph program/helper admission, 2026-11-07

The helper admission tests live in
`internal/cli/candidate_state_graph_helpers_test.go`. They cover the authored
`module_state_graph_program` API with independent, explicitly assigned admission
expectations and the existing Go transaction MIR verifier. This adds tests and
evidence only; it does not change source syntax, storage codecs, layout identity,
runtime execution, or current-source authority.

## API and test transport

The probe imports `module_state_graph_program as program` and exercises
`assemble`, `validate`, `verify`, `root`, `helper_count`, `helper`, `helper_name`,
and `layouts`. `Program` and `Verified` remain opaque resource owners. The narrow
native build explicitly includes `module_state_graph_program` as a fixture root
and copies only its authored import closure, state declarations, and packet
reader dependencies.

`FYSPTEST` is a private test envelope: helper count, name count, name byte count,
ordered name spans, length-prefixed root/helper graphs, then packed name bytes.
Embedded graphs reuse `stateGraphIRPacket` and the existing Fairy graph reader:
19 operation fields, storage kinds 1..6, borrow kinds 7..10, call kind 11, and
helper return terminator 6. Calls use global helper indexes 1..N; zero is invalid.
Each helper-as-Program receives **all** original helpers when encoding, including
when calling a helper earlier in lexical order. The packet test checks both
diamond and backward-index translations.

## Admission matrix

The final integrated Go matrix contains 75 independently expected cases. Each case checks
acceptance and, for rejection, the relevant Go diagnostic family. The native
suite first checks these expectations against Go before compiling its probe.

| Area | Cases and expected behavior |
| --- | --- |
| Valid calls | Leaf, repeated, transitive, diamond, root branch, helper branch, backward lexical calls, empty helper, and no-helper root are admitted. |
| Effects | Read-only, transitive read-only, write-only, and mixed read/write diamond calls are admitted. Root and helper missing, excessive, foreign, duplicate, or unsorted effects are rejected. Root extra storage effects remain allowed when there are no helpers, matching Go. |
| Names and closure | Canonical module-qualified names include underscores, nested modules, and a 128-character identifier. Empty, unqualified, malformed, non-ASCII, and NUL-containing names fail. Foreign callees, unused helpers, duplicate/unsorted names, self recursion, mutual recursion, and unreachable recursion fail. |
| Ownership and exits | Root return, helper commit, helper return with unrelated fields, zero/foreign capabilities, noncanonical operation/terminator cost, and call payload contamination fail. Helper return, reject, and canonical trap are admitted. All four borrow operation kinds are forbidden in helpers. |
| Data | Noncanonical root/helper u32 literals and Go layout hash drift fail. Native cases additionally corrupt operand spans, graph headers, selected layout indexes, and call metadata. |
| Root borrow/call interaction | Read-only calls preserve handles, writing calls invalidate handles, and ordinary values copied before writing calls survive. |
| Helper limit | Reachable chains of 31 and 32 helpers pass; 33 fails. |
| Combined operation limit | Root plus helper totals of 1023 and 1024 pass; 1025 fails even though each individual graph remains within its own limit. |

Five Go-only rejection cases have no lossless representation in the native test
packet: layout hash drift and duplicate/unsorted root/helper effect lists. The
packet represents a selected layout index and an effect bitmask. Those cases
are explicitly skipped in native replay; native layout-index and effect-mask
rejections are covered separately.

For each Windows profile, 70 representable matrix cases and 17 additional packet
mutations execute, for **174 native process checks** across low and optimized.
Malformed names include missing/extra name records, leading gaps, reversed or
out-of-range spans, and unused trailing name bytes. Header cases cover root and
helper capability mismatch, coherent foreign helper header/body capability,
zero helper capability, mismatched layout selection, and all graphs selecting
an out-of-range layout. Calls also reject out-of-range callee indexes and
borrow metadata; storage operations reject nonzero callee metadata.

Expected native results are exact: `0` for admission, `1001` for malformed
programs, and `1002` for helper/combined-operation bounds. Rejected decoded
programs must produce the same rejection from both `validate` and consuming
`verify`; a successful `verify` on those programs fails the probe.

## Work and immutable owner checks

For every admitted native program, the probe measures validation work, then
checks exact remaining work, one-short remaining work, and an already-exceeded
16,777,216 work counter. It separately measures consuming verification,
including the layout copy, and checks its exact and one-short budgets.
Exhaustion returns `1002` without advancing the counter beyond its ceiling;
an already-exceeded counter is unchanged.

After verification, replacing the caller's original layout with an empty plan
does not change either retained proof. Their layout copies remain valid and
nonempty; helper counts/names match the packet, root/helper data and operation
counts agree between independent proofs, and all helper headers retain the
root's nonzero capability and selected layout.

Six semantic rejection probes require diagnostics at the authored caller:
`E0103` for forging `Verified` or writing a private root header, and `E1008` for
attempting mutable access to the root, layout, helper graph, or helper name.

## Executed validation

All checks below passed on Windows amd64 on 2026-11-07:

```text
go test -mod=readonly -count=1 -timeout 3m ./internal/cli -run '^TestCandidateStateGraphHelpers(Oracle|Packet)$'
PASS: 0.116s (initial 69-case oracle and packet checks)

go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateStateGraphHelpers$'
PASS: 17.571s (initial low/optimized native matrix)

go test -mod=readonly -count=1 -timeout 2m ./internal/cli -run '^TestCandidateStateGraphHelpersSealedOwner$'
PASS: 1.171s (initial four ownership rejection probes)

go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateStateGraphHelpers(Oracle|Packet|SealedOwner)?$'
PASS: 18.985s (final 72-case oracle, packet checks, both native profiles, six ownership rejection probes)
```

The first native attempt found a missing switch statement semicolon in this
test's probe. It was fixed in the test; no parent-owned source change was
needed. The first ownership-test assertion expected `src/main.fy` while actual
diagnostics use `main.fy`; the final assertions require caller location and the
specific visibility/mutability diagnostic instead.

Native Linux execution is deferred. Allocation-error (`1003`) fault injection,
full compiler/CLI rebuilding, full repository suites, fuel/runtime execution,
and source rebind/cache authority are outside this narrow admission test.
The worker did not modify the lockfile, central documentation or authored source.
Parent integration owns the program module, ADR, central index, queue and commits.

## Parent integration

The final combined regression includes 75 independent helper expectations,
both native profiles, packet checks, six sealed-owner refusals, the original
56-case graph matrix and the 87-case borrow matrix:

```text
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^(TestCandidateStateGraphIR|TestCandidateStateGraphHelpers(Oracle|Packet|SealedOwner)?|TestCandidateStateGraphBorrowIR(Oracle)?)$'
PASS: 50.456s
```

The helper native part passes in 17.73s. The five documented Go-only cases skip
in each native profile because selected layout/effect masks cannot transport
their full identity/order distinction. They pass the Go oracle. Across all
three matrices, **464 native processes** execute (112 graph + 178 borrow + 174
helper); the six semantic ownership refusals are additional checks.

A fresh current Go bootstrap checks an isolated complete example with the two
new source modules. Both `check` and `check --locked` pass **706 source files /
8,781 MIR functions**. Only the successful lock is copied back. Records are
`.cache/state-graph-borrow-helpers-check-20261107/check-results.json`; the fresh
executable is `.cache/fairyc-state-graph-borrow-helpers-integration.exe`. These
generated records are execution evidence, never source authority. Scoped vet,
format, UTF-8/LF/final-newline and diff checks pass. Linux, source/Complete
binding, artifacts/cache, runtime execution and every final gate remain open.
There is no push or CI dispatch.
