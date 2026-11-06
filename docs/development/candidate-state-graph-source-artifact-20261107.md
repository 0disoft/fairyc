# Exact current-source graph artifact binding, 2026-11-07

This private integration closes the exact-source transport gate for the current
single-module authored root/helper producer. It does not close ordinary
Complete publication, package/cache admission, wider source lowering, command
selection, execution or persistence.

## Ownership and restoration

`module_state_graph_source_artifact` exposes:

```fairy
pub fn Result<wire::Payload,u32> encode(ref source::Produced current,ref mut usize work)
pub fn Result<Restored,u32> restore(ref u8[] saved,ref u8[] actual,ref mut usize work)
pub fn ref source::Produced current(ref Restored p)
pub fn ref program::Verified graph(ref Restored p)
```

`FYSGSRC0` contains a length-framed exact source and length-framed `FYSGRPH0`
artifact, with no trailing bytes. Encoding takes the source producer's sealed
owner and encodes its retained independently verified whole program through
`source::program`, including helper graphs, names and transitive effects. It
uses the canonical detached codec without rebuilding a no-helper wrapper. Neither an arbitrary
Parsed/layout/graph nor a claimed hash can create a source envelope.

Restoration checks all framing before taking slices, compares the saved source
with the actual current bytes, independently produces the current root/helper program,
regenerates its canonical graph artifact, compares the complete saved graph
payload, and re-enters detached decode/verification. This guards both required
relationships: current source versus saved source, and source-derived graph
versus independently valid saved graph. A module-only edit can preserve every
detached graph byte and still invalidates the exact-source envelope.

`Restored` retains independently owned current source and decoded graph/layout
proofs. Its constructor is private and both accessors return immutable borrows.
After restoration, mutating the caller's original request bytes changes neither
owner. There is no public mutable getter, unchecked proof constructor, ordinary
Complete accessor or execution/cache-hit result.

Immutable locals, typed aliases, found/dynamic operands, root borrow/view/copy,
structured if/else joins and helpers are carried through this same source-bound
transport. The producer permits at most 32 helpers, 64 blocks per graph and
1,024 combined root/helper operations.

The producer's private 65,536-byte source ceiling and existing 8 MiB envelope,
16 MiB work and graph limits apply. Size/range checks precede additions, slices
and allocation. Copying, comparisons, rederivation, encoding and decoding charge
the supplied ledger; scanner/parser retain their own producer-owned bounds.
Errors remain 1001 malformed/drift, 1002 bound/work and 1003 allocation.

## Previous Windows verification

The previous literal-focused command passed in **21.633s**:

```powershell
go test -mod=readonly -count=1 -timeout 3m -v ./internal/cli -run '^TestCandidateStateGraphSourceArtifact(Envelope|Owner)?$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_state_graph_source_artifact_test.go
git diff --check
```

Both Windows low/optimized profiles pass **40 native requests each / 80 total**:

- Seven envelopes are independently encoded, retaining exact source and graph
  bytes. A separate Go envelope writer checks the complete outer bytes.
- Four roundtrip requests cover original source, retained comments, reject and
  trap. Each request also tests independent re-encoding, encode/restore at exact
  work and one unit short, initially exhausted ledgers, and caller mutation.
- Six source edits change literal, module, contract UID, field, effects or exit.
  Each old envelope is refused against new source, and each freshly paired
  envelope restores. Five independently valid changed graph payloads are also
  refused when spliced next to the original source. The same-width module edit
  retains its detached graph bytes; the identical payload restores for original
  source, while its stale source envelope is still refused for the new module.
- Eleven corrupt/bound requests cover cross-family marker, bare detached graph,
  empty/truncated source or graph, trailing data, source 65,537, oversized graph
  framing, graph marker corruption and literal mutation.

Three additional semantic refusals cover the private constructor and mutable
current-source/graph access. Three Go envelope-only byte-preservation checks
include empty and binary source data; these are transport checks, not source
admission passes. There are no selected skips in this suite.

Initial preparation failed on missing switch-statement semicolons in the test
probe; these were fixed. A subsequent import-closure attempt reported missing
`scalar_modules.fy` while the producer was being finalized; a coherent later
snapshot passed. One package compilation failed while the parallel hash test
temporarily referenced an unfinished helper definition. Those commands remain
failures, not passes. The final run uses complete, stable authored files.

## Previous whole-source integration and retained results

At that earlier snapshot, a fresh Go bootstrap checked an isolated complete
example. Both `check`
and `check --locked` pass **712 source files / 8,866 MIR functions**, with empty
stderr. Only the successful lock is copied back. Every `.fy` file in that snapshot was
compared byte-for-byte with the checked copy after worker completion; there
are no differences. Local records:
`.cache/state-graph-source-integrated-check-20261107/check-results.json` and
`.cache/fairyc-state-graph-source-integration.exe`. These ignored outputs are
execution evidence, never source authority.

The following totals belong to that earlier snapshot; they are not the current
expanded validation total:

| Slice | Final passing test command time | Windows native processes | Additional ownership refusals |
| --- | ---: | ---: | ---: |
| Original graph + borrow + helpers | 50.456s | 464 | 6 |
| Detached graph artifact | 29.455s | 416 | 2 |
| Authored source producer | 29.699s | 166 | 5 |
| Go-compatible body identity | 21.735s | 254 | 4 |
| This source-bound restoration | 21.633s | 80 | 3 |
| Total | Independent focused commands | 1,380 | 20 |

Five helper identity/effect-order oracle cases are unrepresentable in each
native helper profile and are explicitly skipped there; they pass Go and are
not counted in native totals. No other selected suite skips are included.
Final scoped vet, formatting, UTF-8/LF/final-newline and diff checks pass.

Linux native execution remains batched. Actual allocation-failure injection,
full current-source Go/Fairy equality, ordinary Complete/cache/CLI selection,
native transactions/persistence, consecutive self-build, Windows distribution
and hosted acceptance remain separate gates. Remaining source families and package/provider/target choices are not implied
by these private owners. The full implementation objective remains active.

Normative additions are ADRs 0512-0517 with their topical index entries; existing
public CLI/SDK/runtime ABI, DB/provider, dependencies, compiler defaults,
repository policy and CI/runners are unchanged. No push, dispatch or release.

## Integrated authored-source validation, 2026-11-07

The parent independently compared the actual source producer and restored
root/helper program with Go: 68 authored cases / 136 Windows low/optimized
processes, package 24.792s. Retained source and artifact regressions bring this
slice to 374 native processes and eight ownership refusals. Fresh bootstrap
`check` and `check --locked` pass 716 files / 8,917 MIR functions; scoped vet
and source hygiene pass. The separate operation-limit suite is not included.

Exact commands, work/ownership controls, failed-attempt accounting and full
integration records are in
[the source receipt](candidate-state-graph-source-20261107.md#integrated-authored-source-validation-2026-11-07).
The earlier module-specific checks above remain historical evidence.
Linux execution and allocator-failure injection remain deferred; ordinary
Complete/package/cache/command/execution admission remains separate.
