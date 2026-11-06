# SDK Vec calls in user aggregate cleanup, 2026-11-07

This slice advances the SDK/user-aggregate item from
`candidate-atlas-sdk-vector-20261105.md`. The current checkout already contains
the five bounded scalar aggregate/SDK producer cases and command-leaf tests.
Those existing files, rather than older queue paragraphs, define the baseline.
This receipt does not close the whole queue or the final authored-main replay.
SDK/user-aggregate admission was already present before this patch; this change
does not newly enable that family. It establishes additional combined-source
acceptance and factors the concrete production path for direct inspection.

## Production path

`project_atlas_source::load` now owns the same source pipeline previously inside
`query`: locked workspace loading, exact-source aggregate mode selection,
`build_complete_aggregate_sources` or ordinary `build_complete`, source-bound
schema and instance projection, v3 production, current admission, and whole
workspace joining. It returns `Loaded` only after every package has passed.
`query` consumes that joined graph and continues the same bounded query ledger
through selection, reverse-call exploration, and rendering. Package selection,
SDK authentication, compiler-image identity, failure stages, and limits remain
in this production path.

Selection is still exact after the split: `join::join` moves its input Ready
vector unchanged into `Packages.values`; only the separate `order` index is
sorted. `join::instance(g, p)` reads the original values slot. Thus the selected
producer owner index continues to designate the same package instance, rather
than a position in sorted traversal order. Native query checks cover both root
and SDK selection.

The new native probe inspects the maps held by this production loader, then
invokes the actual `experimental atlas-impact` command in the same executable.
It does not rewrite a probe to force aggregate mode, replace Complete, supply a
prebuilt Go map to Fairy, or launch a Go child. PATH is empty during native
execution. The independent Go source oracle supplies the expected facts.

## Exact source case

`atlasSDKAggregateCleanupSource` contains:

- `Cell<T>` with one `T value` field; `cell<T>` returns `Cell<T>` from a concrete
  source constructor. Main reaches `cell<i32>` through ordinary `leaf`.
- `Check<T>` with one `T value` field, instantiated as `Check<bool>` in `leaf`.
  Its nonempty cleanup creates a Vec and calls `empty<i32>`, `push<i32>`, and
  `at<i32>`. These are calls from the concrete user cleanup, not only calls in
  a neighboring ordinary function.
- An authenticated `std::vec` import. The SDK retains the five fixed scalar
  `Vec`/`take_slot` pairs plus `empty<i32>`, `push<i32>`, `at<i32>`, and
  `write_slot<i32>`: fourteen SDK implementations. User implementations are
  exactly `Cell<i32>`, `Check<bool>`, and `cell<i32>`, each once.

The Go oracle verifies every cleanup call's written source bytes, source span,
original SDK target, and specialization endpoint. It compares the SDK's original
declarations, generic interface hash, source identity, and `sdk` provenance with
the unused-import control. Usage must not publish SDK clones as declarations or
alter pinned SDK ownership. The original SDK noneditable declaration policy is
unchanged; this slice does not alter editor admission.

Go source inventory observed on Windows:

| Package | Provenance | Declarations | Ordinary functions | Implementations | Relations |
| --- | --- | ---: | ---: | ---: | ---: |
| example.session.app | local | 5 | 2 | 3 | 8 |
| example.session.answer | local | 1 | 1 | 0 | 0 |
| fairyc.std | sdk | 278 | 156 | 14 | 516 |

Native map comparisons cover original declarations and symbols, both interface
hashes, specialization identities and arguments, physical aggregate fields,
cleanup presence, exact calls and all collected trap/drop/unsafe/move/allocation/
error/borrow facts, and relation counts. Completeness remains explicit: reads,
test-calls, and writes are unavailable; generic template bodies remain unavailable
until specialization. The map remains partial. The two actual impact subjects
are original SDK `at` and original user resource `Check`, compared with Go
including source contexts and the current native executable hash.

## Refusals and bounded verification

`TestAtlasSDKAggregateCleanupRefusals` verifies the combined source still rejects
client private storage intrinsics with E0101 and nominal user aggregate arguments
with E4502. No broader type-argument or SDK source authority is granted.

Executed Windows command:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 1m -v ./internal/cli -run '^TestAtlasSDKAggregateCleanup(Oracle|Refusals)$'
```

PASS, 0.823s, no skips. Source inventory and exact specialization/span/authority
assertions pass; both refusal controls pass.

Executed native command:

```powershell
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasSDKAggregateCleanupSource$'
```

PASS, 570.128s, no skips. The native test took 569.93s: low 464.39s and
optimized 82.04s, with initial source lowering outside those profile durations.
Both profiles pass the full admitted-map comparisons and both actual impact
queries. Each native invocation retains the existing three-minute bound.
Workspace trees remain byte-identical before and after each profile. The source
loader SHA-256 in the compiled snapshot matches the checkout:
`59fed7a15d0453f38e725fe927a7bd5f75bef95c38019cbc2a970a07ec4d8068`.
Native query contexts were checked against their own executable SHA-256, not
against the Go compiler's identity.

That native command ran before adding the existing private integration opt-in
to this new nine-minute case. Current replay requires
`$env:FAIRYC_RUN_PARITY_INTEGRATION='1'` before the same command. Default test
runs retain the fast source/refusal oracles and explicitly skip this expensive
native replay. No native assertion, timeout, job or runner is changed; the
recorded low/optimized acceptance is reused for the unchanged product inputs.

The coordinator reruns the fast source/refusal checks plus the default native
selection: PASS in 1.793s, with the heavy case explicitly skipped as intended.
Scoped CLI vet and staged encoding/diff checks pass. An isolated staged-source
snapshot passes reference `check` and `check --locked` at 701 source files /
8,704 MIR functions and supplies this slice's lock without the parallel inline
worker's source. The production loader's SHA-256 remains the native-tested value.

`go vet -mod=readonly ./internal/cli` passed. `gofmt -l` on both new Go files
returned no paths; the scoped tracked diff check and all four owned files'
trailing-whitespace checks passed. No new Go build was attempted during the
other worker's inline-file recovery; the native run used its already compiled
Go test binary and previously copied Fairy sources.

An initial probe build rejected the newly introduced accessor named `graph`
because it collided with the existing module alias (E4501). The accessor is now
`admitted_graph`. This failed attempt is not counted as a native pass.

An earlier broad existing producer/oracle invocation passed the five Go source
oracle families, then was deliberately stopped during its first native profile
to keep this slice bounded. That interrupted producer run is not a pass.

## Remaining gates and ownership

Native Linux execution is deferred. Full repository tests, the remaining source
combination matrix, refreshed whole authored-main replay, editor/MCP regressions
for this new cleanup-call source, consecutive self-build, distribution, and
same-commit hosted gates are not established here. The original unsupported
nominal/composite argument families remain refused.

No owning grammar, public CLI/schema, SDK/runtime ABI, DB, dependency, runner,
shared queue/index, lock, or ADR change. No push, dispatch, release, deployment,
staging, or commit: the parent owns integration and lock refresh. Other agents'
state graph and inline edits are outside this slice.
