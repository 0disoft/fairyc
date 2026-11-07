# Current Windows authored compiler acceptance route, 2026-11-07

This is the integration plan and bounded evidence record. The current-source
normal release self-build now passes; see the latest observation below. The parent owns the final isolated source
check, lock refresh, fresh Go-emitted seed, actual Fairy emission and native
fixed-point execution. No Linux, network, remote or CI operation belongs here.

## Source identity and admission

The inspected `file_cli` import closure has 646 inputs including a generated
`main`; its existing module files total approximately 4.53 MB before that entry
and the generated version replacement. Recompute identities after integration.
`candidate_limits.fy` now admits at most 1024 original source files and 8 MiB
including separators in the compilation-unit planner. Catalog retention counts
raw source bytes without separators. Both are bounded independently of:

- 65536 bytes per source and per manifest; paths at most 4095 UTF-8 bytes;
- 1048576 bytes per joined owner and for the catalog name pool;
- 512 declarations per joined owner and the other existing AST/IR arena limits.

`module_cli::paths`, `module_catalog::build`, `module_units::build`,
`module_ast_join::join_range`, registry admission and the C frame index consume
the shared file-count limit. Their source/path storage is `Vec`/`Buffer`, not a
512-slot source array. The catalog byte check and unit-planner separator check
consume `source_set_bytes`; neither requires a fixed byte-array enlargement.
1024 sources do not promise that every possible 4095-byte path fits the separate
64 KiB manifest or that every source fits in one joined owner.

The source admission owners are described by
`docs/architecture/candidate-compilation-units-v0.md` and
`docs/architecture/candidate-source-join-v0.md`. Their current aggregate/file-count
policies are synchronized with these bounds; historical measurements stay dated.
Do not reinterpret unrelated 512 package, declaration or parser caps as this
file-count domain.

The existing `tests/source_limits.fy` now uses decimal module names beyond 676
owners, checks two independent 512-declaration units, rejects a combined join,
and exercises 1024/1025 files and exact/one-over byte bounds. Its unchanged
independent joined-owner limits remain enforced. Both its native profiles pass;
the low execution uses a test-only 90-second allowance.

## Exact input and companion closure

`build-candidate-native-probe.py` discovers `module` declarations from
`examples/lexer_candidate/src/**/*.fy` and
`internal/sdkstdlib/sources/**/*.fy`, recursively follows imports from its
generated entry, and writes `sources.list`: entry first, then module-name order.
The full entry calls `file_cli::run` with filesystem, console, environment,
network and clock effects. `public_cli_data` is replaced with the generated
current version module. `candidate_build_version.py` builds a fresh Go version
oracle and checks CLI metadata; it does not use the Go frontend on Fairy input.

For comparison, the inspected private compiler-only closure has 490 inputs,
check/build/run-only 459, and authored companion 36. Their success cannot stand
in for full-dispatcher acceptance. All snapshots include exact source hashes;
the full build also binds `fairy.lock`, tool hashes and generated version data.

The companion builder emits a separate `native_build_cli` import closure and
packs the 23 original C/inc files in
`tools/package-candidate-runtime-assets.py::SOURCES`. The generated runtime pin
must match those originals. Keep each companion executable and its matching
`fairyc-runtime-assets.pack` together. Old receipts are evidence, not source.

## Integration commands

Run from the repository root, without Python `-O`, using fresh output paths.
After the parent finishes current-source admission and lock handling, prepare
a new seed; the old baseline directory must remain untouched:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION = '1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS = '.cache/current-source-whole-cli-20261107'
go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli -run '^TestCandidateInlineLocalsFullCLI$/^low$'
go build -mod=readonly -o .cache/current-bootstrap-builder.exe ./tools/candidate/fairyc-candidate-build
python -B tools/build-candidate-native-probe.py --candidate .cache/current-source-whole-cli-20261107/low/fairyc.exe --builder .cache/current-bootstrap-builder.exe --output .cache/current-selfbuild-20261107 --profile release
python -B tools/verify-candidate-native-fixed-point.py --build-receipt .cache/current-selfbuild-20261107/receipt.json --output .cache/current-selfbuild-fixed-20261107
```

The probe automatically builds the authored companion unless an authenticated
current `--authored-driver-receipt` is supplied. Each compiler image is built
with the explicit Go bridge argv:

```text
<builder> --packet <packet> --packet-sha256 <sha256> --output <image> --compiler-host [--release] [--stack-reserve-bytes <bytes>]
```

The fixed-point verifier authenticates precisely that Stage1 release argv and
uses the same flags for Stage2. It freezes the same manifest/version bytes and
compares complete C packets and complete optimized PE images. It rejects
compiler-only, commands-only and reused-frontend receipts as full-CLI proof.

`--compiler-host` selects the Go bridge's header-only 8 MiB admission and private
compiler image builder together. Literals and each unit remain limited to
2 MiB. Ordinary target decoding/building stays unchanged; a 16 MiB stack alone
does not widen admission. See
`docs/development/current-compiler-host-bridge-20261107.md` for the parent-owned
implementation and executed bridge checks.

The installed Fairy companion still consumes ordinary runtime-target packets
with a 2 MiB part cap. This stage uses the Go C-only bridge for compiler images;
it does not give the authored companion an 8 MiB compiler-host mode. Its own
separate `verify-candidate-companion-fixed-point.py` gate must reproduce and
rebuild the companion packet using ordinary companion admission. A future
authored compiler-host opt-in is a separate change.

## Behavior acceptance

Use a fresh current Go reference from the parent, and the new probe receipt:

```text
python -B tools/compare-candidate-go-behavior.py --reference <current-Go.exe> --candidate-build-receipt <new-receipt.json> --suite <check|run|cli> --output <fresh-.cache-directory>
python -B tools/compare-candidate-go-workspaces.py --reference <current-Go.exe> --candidate-build-receipt <new-receipt.json> --suite <examples|cli> --output <fresh-.cache-directory>
```

Repeat suites as needed; the default compiler profile is release. Inspect JSON
`parity`, per-case comparisons, `skipped` and native contract failures: process
exit zero only means the comparison ran. Package-graph fixture adapters remain
explicit skips, and the unfiltered examples suite intentionally skips compiler
self-source. Report those gaps rather than claiming universal parity.

## Baseline and stale artifacts

The clean `b1ce7d5f` public ZIP snapshot is recorded separately in
`docs/ops/windows-preview-readiness-20261107.md`. It ships the public Go compiler
and does not prove authored self-build.

The parent-reported clean authored CLI low baseline passed in 252.092 seconds;
its evidence is `.cache/current-whole-cli-low.log` and
`.cache/current-whole-cli-20261107/low/`. Its recorded Source digest is
`8e0b83e9b2e004dcdfc6081839859fb1a20b2b9fd83627ebdba2160e353e20eb`, image
`56022dfb1f5e4a027a03e9764037d0cad1f0aa0d86826700b5a6addb9c46c390`.
It predates these capacity changes and must not be called a current-source seed.

The older `.cache/inline-locals-full-cli-20261107/{low,optimized}/host.json`
images match their recorded image digests. Those helpers emit through Go and
build a Go companion; their records contain C/profile and image digests, not a
source manifest or proof of authored self-build. `Source` hashes profile name,
header, runtime support and C units, excluding builder implementation, MSVC
identity and compiler options. Use a fresh directory after backend changes.

The inspected historical full authored set is
`.cache/completion-20261023/self-hosting/candidate-regions-v4/release/`:
`fairyc.exe`, `fairyc-candidate-build.exe`, `fairyc-runtime-assets.pack`.
Its artifacts match its receipt, but 178 of its 512 source rows were stale
against the inspected checkout. Its connected
`native-driver-long-path-v2/receipt.json` had 28 stale source/runtime rows.
They are historical evidence only; none is a current acceptance pass.

## Checks executed for this change

```text
# Repository root:
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateSourceAdmissionBounds$/^low$'
# tools/:
python -B -m unittest test_candidate_fixed_point test_candidate_native_probe test_candidate_build_version
```

The authored native low probe passed in 42.20 seconds. It exercises 1024/1025
manifest and catalog file counts, independent two-unit grouping versus single
join rejection, exact raw-catalog 8 MiB, separator-inclusive 8 MiB and one-byte
overflow. Its 90-second process allowance is private test policy, not a product
timeout change. The initial shared 30-second runner did not complete this case.
The optimized profile is not repeated for this bounded source-storage change.

All 13 Python tests passed. Native subprocesses in the probe argv tests are
mocked; those tests establish the full/commands-only/compiler-only builder argv
agreement, effect declaration, source bounds and stale-input rejection, not
native self-build. The existing version-generation regressions also passed.
`git diff --check` passed. The integrated observations below supersede the
worker's pending source-check/seed gates. The two source-admission architecture
documents are updated; public API/SDK ABI, DB, dependencies, runners and CI
configuration are unchanged. No remote operation was made.

## Integrated current-source Windows observations

- Complete isolated `check` and `check --locked` after metadata refresh pass in 23.861s / 19.794s:
  755 source files, 9500 MIR functions, empty stderr. All 1225 authored files,
  including the maintained source-limit fixture, match both the final snapshot
  and current source. Successful lock only is copied back. Evidence:
  `.cache/current-source-20261107-all-check.json` and
  `.cache/current-source-check-metadata-20261107.log`.
- Before the metadata follow-up, a fresh Go-emitted whole-CLI low image with the new capacity passes the 205-unit
  native build and all 14 CLI/cache/trap processes in 329.459s. Evidence:
  `.cache/current-whole-cli-host-limits-20261107.log`; image directory
  `.cache/current-whole-cli-host-limits-20261107/low/`.
  Source digest: `0780eeb46866e3f2d335c34f3bc5352b9202c29abbe6ea5b9dcc3dfc5000704e`;
  image: `e0c52e3c3cd242032900285d486ba722a3f71e6b05db5c37ce26bea9c8d737a8`.
- This seed actually emits the 36-source authored companion (2037965-byte packet,
  37.580s). The Go C-only bridge builds low/optimized in 2.488s / 22.716s.
  Its authenticated runtime pack is 169685 bytes from 23 original assets.
  Evidence: `.cache/current-authored-driver-20261107/receipt.json` and
  `invocations.json`. This proves companion generation/build, not complete CLI
  self-build or consecutive-generation equality.
- The maintained authored boundary fixture passes native low in 43.521s
  (37.537s execution) and optimized in 10.50s. The combined initial command
  failed: its default 30-second low allowance expired, and the unrelated parser
  fixture check reports `missing-visible-module`. The narrowed lexer check
  passes; no complete fixture-suite pass is claimed. Raw logs:
  `.cache/source-limits-20261107/windows-native.log` and `windows-low-90s.log`.

The source-capacity integration and seed/companion prerequisites are complete.
Current-source whole-CLI authored emission, optimized self-build, fixed point and
broader Go/Fairy parity remain open. Native Linux, hosted CI and publication are
still deferred. Scoped `go vet` for the bridge and CLI and diff hygiene pass.

## CLI diagnostic metadata follow-up

The first full probe after `fd98503c` stopped before Fairy frontend invocation:
the embedded CLI metadata omitted the current Go catalog's `E0011` explanation.
The owning `fairyc-candidate-cli-embed --write` generator adds that single branch;
its exact `--check` passes for all 35239 generated bytes. Help, target and version
data are unchanged. The failed attempt is retained at
`.cache/current-selfbuild-fd98503c.log`; it is not a successful self-build.

The capacity-enabled image above remains an authenticated bootstrap seed from
before this metadata refresh. Its earlier native results do not validate the
updated explanation. Fresh authored emissions consume the updated source;
subsequent CLI/fixed-point receipts must bind those bytes and their frozen version
oracle rather than claim that the older seed itself contains the new metadata.

## Global import admission follow-up

The next real 646-input emission reaches the Fairy frontend and fails after
14.957s: `project_edit_preview.fy:10`, phase 1/stage 0/code 10, span 347..391.
The exact manifest contains 5052 imports; `module_catalog` still limited its
shared import table to 4096. This is a reproduced catalog admission failure,
not a native C compiler timeout. Evidence:
`.cache/current-selfbuild-metadata-20261107/invocations.json`.

`candidate_limits::project_imports` now bounds imports and scoped provider
bindings at 8192. The catalog checks only the table selected by the incoming
declaration kind, so a full import table no longer rejects an ordinary function
that has capacity in the declaration table. Per-owner 128 items, source and
joined-owner limits are unchanged. The manifest diagnostic now reports 1024
files consistently with the file-count owner.

The native regression admits exactly 8192 imports and matching scoped bindings,
including an ordinary provider function after the import table fills; it rejects
8193 imports with code 10 and 8193 bindings with the existing code 9. Windows
low/optimized pass in 6.204s / 9.047s. Full isolated check/locked after this change
pass in 14.424s / 14.883s, 755 files / 9501 MIR functions; all 1225 authored files
match the checked snapshot. Logs: `.cache/current-import-admission-20261107.log`,
`.cache/current-import-admission-optimized-20261107.log`, and
`.cache/current-source-check-imports-20261107.log`.

An earlier image with the old catalog limit cannot validate full 5052-import
input. The next bootstrap stage must contain the new catalog implementation;
no earlier full-CLI native pass is counted as a new self-build pass.

The separately discovered parser fixture failure is also repaired: its main
uses the actual syntax API, and its helper follows actual module dependencies.
Both fixture checks and five borrow-boundary cases pass in 12.973s at
`.cache/lexer-fixture-repair-20261107/after.log`. The earlier failed log remains
retained; it is superseded only for these seven repaired cases.

## Source-cardinality consumers follow-up

The optimized import-enabled seed passed its full CLI regression (205 C parts,
14 independent CLI processes, 213.942s), then rejected the actual 646-source
emission in 1.705s with phase 2/stage 3/code 9. This reaches the region-demand
source-selection gate, which still used 512 instead of the shared 1024-source
bound. It is a frontend rejection; no full self-build packet was emitted.

Source-selection and checked-body stamps, generic bindings/context, public type
stamps, provenance and cache source mappings now share source cardinality.
Source-index masks/mappings own bounded buffers/vectors. Package-owner admission
remains 512, and detached/artifact import admission remains 32768 independently
of active project imports. Details and boundary evidence are in
`candidate-region-source-bounds-20261107.md`,
`candidate-source-stamp-bounds-20261107.md` and
`candidate-source-artifact-bounds-20261107.md` in this directory.

The integrated full-source snapshot passes check/locked in 14.316s / 13.330s:
755 checked files, 9503 MIR functions; all 1225 authored files match the checked
snapshot. Its lock is copied only after both successful commands and that hash
comparison. Evidence: `.cache/current-source-check-sourcecards-20261107.log`.
The fresh seed and authored self-build remain separate acceptance steps; these
source and narrow native checks do not establish a fixed point or full parity.

The source-cardinality-enabled optimized Go seed now passes the same full CLI
regression: 205 C parts, 14 independent CLI processes, four cold/warm trap pairs
and six Go-matching test reports in 185.300s. Its C/profile source digest is
`863653e671bad05457fed030bbc29a60aa306bb9ad326fc446958e0a5442cf17`
and its PE digest is
`b792dea1742d2dcc04206be021990fb27385111284da36d69ea2baf4a39db388`.
Evidence: `.cache/current-whole-cli-sourcecards-20261107.log` and the
`optimized/host.json` under that artifact directory. Durations are single-run
observations with concurrent host work, not a speed comparison. The full authored
emission is a subsequent step and is not included in this pass.

## Provider value-work follow-up

The source-cardinality-enabled seed reaches provider analysis, then fails in
91.448s at phase 2/stage 7/code 10 while checking `module_unit_interfaces.fy`:
limit 1048576, attempted 1048581. Stage 7 propagates the value checker failure
through origin input construction: `origins -> input_factory -> provider ->
values::build`. The failing charge aggregates completed body work. This is not
the interface-construction stage or a timeout. The failed invocation remains in
`.cache/current-selfbuild-sourcecards-20261107/invocations.json`.

Nominal field lookup now indexes canonical type IDs instead of repeatedly
scanning all shapes. The index is owned by the immutable Interfaces result;
first-match/field order, permission errors and existing work/type caps remain.
Index construction charges its actual bounded setup work. The real-builder
regression checks 975 queries against the old ordered lookup in both native
profiles; a last-field lookup needs 11 probes instead of 172. See
`candidate-interface-shape-index-20261107.md` for exact scope and hashes.

The integrated source check/locked pass in 23.341s / 19.488s, with 755 checked
files and 9504 MIR functions; all 1225 authored files match the checked snapshot.
The optimized Go seed also passes 205 C parts and 14 independent CLI processes
in 247.553s. Concurrent host work makes these single-run times unsuitable for a
wall-clock speedup claim. Logs: `.cache/current-source-check-shape-index-fixed-20261107.log`
and `.cache/current-whole-cli-shape-index-20261107.log`. The initial source-check
attempt exposed a Fairy statement/expression ambiguity and failed; only the
corrected source hash in the regression record has these passing results.

The full authored emission is being rechecked separately; neither the reduced
probe counts nor the Go-seeded CLI regression establishes self-build acceptance.

The new seed's complete-input retry still fails after 115.495s with phase
2/stage 7/code 10 at `module_unit_interfaces.fy`, now without the earlier rich
expected/actual pair. A source-owner relocation trial preserves all 975 lookup
results in both profiles and passes full-source check/locked, but its complete
retry also fails the same stage after 98.604s. The relocation and its lock were
reverted to `329f611b`; no self-build improvement is claimed from that trial.
Its scope and hashes remain in the shape-index regression record. The three
failed full retries emit no packet and cannot establish C linking, self-build,
consecutive-generation equality or current-source parity.

Further work is diagnostic: split factory, value/AST, liveness, return inference,
publication and borrow-capture work before changing another capacity or loop.
Generated-C instrumentation under `.cache/current-origin-trace-20261107` is a
diagnostic image only. It cannot substitute for a normal authored build receipt.

## Selected-row body AST projection

The split diagnostic identifies the exhausted ledger inside
`module_unit_body_ast::build`: `scope=7 work=1048576 amount=1`. The diagnostic
run takes 91.141s, emits no packet and exits 2. Its image SHA-256 is
`d6f264415b252f04b03077c5d90d085093745ac65ff4c85f10809b3fc2d45852`;
the manifest SHA-256 is
`b0ceb6b8888273154131e2cfc4bca88859d42c074f8849ec5c08cba872ee2f08`.

Commit `7a2ac590` records selected original arena IDs while building dense lookup
maps. The second pass visits only those selected rows. For N arena rows and K
selected rows the charged projection work is N+K instead of 2N, with K<=N.
Original source IDs, arena order, remapped links and the shared work cap remain.
The actual public-builder regression performs 80 projections in a shared ledger:
961520 work versus the old loops' analytic lower bound of 1920640. Both Windows
profiles also cover exact-cap success/refusal, dense input, prepared closure reach
and empty/nonempty cleanup. Details are in
`candidate-body-ast-projection-20261107.md`.

Full isolated source check/locked pass in 18.807s/17.246s (755 files, 9504 MIR
functions, all 1225 authored source hashes matching). The optimized Go-seeded CLI
regression passes in 238.220s with 205 C parts and 14 independent CLI processes.
These timings are observations under concurrent host load, not speedup claims.
The complete authored emission is a separate acceptance step; fixed-point and
parity tools must consume a successful authenticated full-CLI build receipt.

The normal complete-input retry still fails at phase 2/stage 7/code 10 after
100.365s, with zero packet bytes. The recorded source version is clean
`7a2ac59033db1e6578559980cc90bb2dcafa5a42`; evidence is
`.cache/current-selfbuild-ast-projection-20261107/invocations.json` and its
source snapshot. The selected-row optimization is verified independently but
does not by itself resolve full self-build. The next diagnostic uses a fresh
current-source instrumented image and records failed AST declaration/source
identity plus work before/after, rather than raising a capacity.

### Current selected-row diagnostic and ordinary AST reuse

The fresh generated-C diagnostic reproduces the remaining failure against the
646-input manifest from `.cache/current-selfbuild-ast-projection-20261107/sources.list`.
Its export/build/run take 23.568s / 126.113s / 98.305s. The run exits 2 without a
timeout or packet bytes. The final failure chain is:

```text
enter returns -> factory -> values
failed-charge scope=7 work=1048576 amount=1
ast-failure decl=45 before=1045958 after=1048576 expressions=2920 statements=788 entries=264 types=529 sourceBytes=20122
exit values tag=1 -> factory tag=1 -> returns tag=1
```

The Parsed source identifies `module module_unit_interfaces`; declaration 45
matches `field_measured` at source line 100 by written declaration order. This
identifies the body being projected, not execution of that function's field
lookup. Its original arenas contain N=4501 rows. Only 2618 work remains at AST
entry, less than even the first 2920-row expression-mapping scan. Selected-row
copying cannot remove this earlier full-arena scan. The exhausted ledger is the
provider value phase's AST projection, not the factory-to-summary history charge.

The diagnostic is pinned to `7a2ac59033db1e6578559980cc90bb2dcafa5a42`.
Recorded SHA-256 identities are:

- Diagnostic image: `b07416db6060815e916b7920fa6f7704ce6a36d1fe69661b6a52ea06b27965a7`.
- Input manifest: `9387926174655e8541e29fd810889bee3a2242e7cf8a853d8c55ca7dfd6fe3bd`.
- `module_unit_interfaces.fy`: `45df26b072fed2ec69d16414b7b9f34d25ec76286705af0e5620903e610f45e2`.
- Trace stderr: `c24c01343ea15e71312eb5b0f4e2cc0d5150ff73535ade77a2eb9afb5d8664fc`.

Full evidence is retained under `.cache/current-origin-trace-ast-20261107/`:
`diagnostic-receipt.json`, `input-receipt.json`, `run.json`, `trace.stderr`,
`export.log`, `build.log`, and the cache-only overlay/instrumentation sources.
All captured export and manifest input hashes remained unchanged during this
diagnostic. It is diagnostic-only generated C, not acceptance evidence, and does
not validate the subsequent ordinary-AST reuse change.

The current two-file production follow-up retains the ordinary projection in
`module_unit_ordinary_names::Function` and borrows it from `module_unit_values`
instead of projecting the same body again. Read-only review found no concrete
regression: source/stamp and type-prefix validation still precede value use;
the same retained projection supplies name resolution, original IDs and values;
generic and cleanup AST paths remain unchanged. Retained projections remain
within the existing 512-function, symbol/reference and shared-work admission
boundaries; source bytes and temporary reach/mapping buffers are not retained.
This is a static audit, not a native execution or peak-memory measurement.

The parent's isolated source check/locked log records 14.040s / 12.623s PASS,
755 checked files, 9505 MIR functions and all 1225 source hashes matching, with
the lock copied after success:
`.cache/current-source-check-ordinary-ast-reuse-20261107.log`.
The optimized Go-seeded full CLI regression passes in 199.816s: 205 C parts,
14 independent CLI processes, four cold/warm trap pairs and six exact Go-matching
test reports. `TestCandidateOrdinarySourceIdentities` and
`TestCandidateGenericAggregateValues` both pass in low/optimized (102.558s for
the package); they cover original identities, stale source/work refusals and
generic/cleanup/ordinary value routes. Evidence is
`.cache/current-whole-cli-ordinary-ast-reuse-20261107.log` and
`.cache/current-ordinary-ast-reuse-integration-20261107.log`.
These are single-run durations under concurrent load. The permanent reuse probe
is recorded separately in `candidate-ordinary-ast-reuse-20261107.md`.
Full authored self-build, fixed point and current-source parity remain separate
pending gates; no pass is inferred from the seed or narrow tests.

The permanent ordinary-AST reuse regression also passes in both Windows
profiles (117.767s package total, four native subtests). Its 80-function small
and large inputs grow the source arena from 1044 to 11844 rows. Name-analysis
work changes from 86083 to 950083, while value-analysis work stays exactly 4149;
the removed second AST projection would cost 84560 and 948560 respectively.
Original IDs, all remapped links, value/source coordinates and retained-owner
readability are checked. Existing generic/cleanup, stale-source and 128-active-
local boundary cases pass too. Evidence is
`.cache/ordinary-ast-reuse-20261107/windows-rerun.log`. The first fixture run
fails two harness assumptions and is retained separately; only the corrected
fixture has these passing results. Production hashes stay unchanged throughout.
Scoped vet and diff checks pass; native Linux remains deferred.

## Clean ordinary-AST reuse retry

At clean commit `6270e3673e8a5551a927422550a853c3cbb109e9`, the normal
646-source authored retry proceeds beyond the preceding interface/provider AST
failure, then exits 2 after 306.889s with no packet:

```text
native_cli.fy:1:1: project phase 2 stage 4 code 10
limit 1048576 attempted 1118812
```

Stage 4 is Values, not names. The rich pair comes from adding a completed body's
work to the unit's value-work ledger. Evidence is
`.cache/current-selfbuild-ordinary-ast-reuse-20261107/invocations.json` and its
source snapshot. This is another capacity failure, not a timeout or self-build
pass. A current-source, cache-only value-category diagnostic is being prepared
before selecting another optimization; the product cap stays unchanged.

`TestCandidateCopiedClosureCalls` independently passes at that clean commit in
38.721s with no failures/skips and no source drift (2960 inputs authenticated
before/after). Evidence is
`.cache/ordinary-ast-reuse-20261107/copied-closure-calls-result.json` and its log.
Public Go Windows ZIP and source-only Linux handoff validation are recorded in
their own readiness documents and do not attest authored self-build. Fixed-point
and full current-source parity remain pending because no normal build receipt
has been produced.

## Reference type lookup index

The completed value-category diagnostic preserves all 646 inputs and the exact
failure above (389.337s, exit 2, no packet). The failed last ordinary body is
module 391, declaration 44: 73822 work, including 58162 in 26 place-borrow calls.
Across the failing Values build, place borrow accounts for 726613 charged work
and builtin canonicalization another 252369. These are accounting categories,
not CPU-time percentages. Full trace/hash evidence is in
`.cache/current-value-trace-20261107/analysis.json` and `run.json`.

An independent focused probe keeps the complete source/metadata closure but
selects only the native_cli unit. It reproduces the identical 1118812 attempted
work in 2.568s. This accelerates diagnosis; it is not a smaller-source acceptance
substitute. Both focused receipts authenticate source snapshot SHA-256
`95efe98ad91d455badcc7074fadee07a7e4a08f4bf3b5893d31124898708ed8c`
and manifest SHA-256
`9c4928f4c13298f480f706cfdc01facf67f6afa4eb1a33c7652d934681969586`.

The new Interfaces index maps each original target type ID and shared/mutable
reference kind to its first canonical ID in the validated immutable type prefix.
Scalar borrowing uses that index; the legacy signature/validation and linear
fallback remain. For prefix hits, O(Q*G) worst-case searches become O(G+Q),
including charged setup. The complete focused Values run now succeeds with
606196 work and 216 contexts in 2.337s, with no input drift. Single-run times
under concurrent host load are not compiler speedup claims.

The actual builder/oracle regression passes 310 queries and budget, permission,
fallback and 4096-ValueGraph boundaries in both Windows profiles (36.530s).
Full source check/locked, scoped vet and diff checks pass. See
`candidate-borrow-type-index-20261107.md` for exact tested hashes, extra setup
cost and reserved-memory bounds. A new seed and normal full authored self-build
remain required; no fixed point or full parity pass is claimed yet.

## Integrated parallel follow-up

The new reference-index seed passes optimized full CLI in 244.666s (14 independent
processes). The three ordinary-source, generic-aggregate and shape-index regressions
also pass both Windows profiles in 174.788s. The normal 646-source authored build
then exits 2 after 342.601s at imported project_source_modules cleanup, phase 2
stage 3 code 10. No packet or normal build receipt exists.

The selected-unit diagnosis identifies caller project_atlas_source and work
exhaustion while copying an imported cleanup provider's tokens. An experiment
retaining only source text for empty-cleanup-only providers passes with the same
646 inputs; the complete production retry remains pending. Details and corrected
diagnostic labels are in
`candidate-empty-cleanup-sources-20261107.md`.

Parallel closure/view/vector work is integrated in four separate commits. Its
bounded Windows generated-C and artifact-restoration executions pass; the
integrated source checks pass with 755 files and 9515 MIR functions. These checks
do not replace normal self-build or current-source parity. The fixed-point tools
also reject empty source receipts and recheck the compiler lock at entry/exit;
eight focused Python regressions pass. Native Linux remains batched separately.

The empty-cleanup follow-up is committed as `83452419`, with four source-owner
and fallback scenarios passing both Windows profiles. Its fresh optimized full
CLI passes in 185.451s. The next normal build exits 2 after 372.375s, at owned IR
stage 6 code 1002, before any C packet. A source-bound focused probe separates
the cleanup field cap from work exhaustion: 1024 fields, work 647221. The unit
recipe policy is explicitly extended to the imported interface's 2048 fields;
raw-source and linked-project bounds remain separate. See
`candidate-cleanup-recipe-fields-20261107.md` for the original and frozen-input
receipts, source checks and native boundary results. No normal build, fixed
point or latest-source parity pass follows merely from this change.

## Cleanup recipe lookup follow-up

The `121035cc` normal retry exits 2 after 427.558s at phase 2 stage 6 code 1002,
Vec bytes 104..122, with zero output. Commit `cd72aaef` integrates the cleanup
index (`df15a0a2`): the unchanged 646-input unit-471 comparison reduces cleanup
work 948429 to 798289 and passes 197 owned-IR frames. This is diagnostic evidence.
Windows low/optimized index and field-boundary replays pass (129.03s / 131.79s);
integrated source check/locked pass (755 files, 9527 MIR, 18.419s / 19.598s).
A fresh optimized build-only seed/help passes in 140.587s. The normal 646-source
retry reaches linking, then fails after 551.636s at phase 3/stage 0/code 1002
(bytes 0..0, zero output). Its specific linker limit remains under diagnosis.
Normal self-build, fixed points and latest-source parity remain pending; Linux is batched.
Contracts and work/field caps are unchanged. See
`candidate-cleanup-recipe-index-20261107.md` for receipts, semantics and bounds.

## Project type prefix follow-up

Commit `acd2cda3` integrates `2bfd4575`: validated shared-prefix IDs map directly,
retaining only suffix maps. Windows both-profile prefix/existing-type package
replays pass in 23.350s / 61.337s; source check/locked pass (755 files, 9528 MIR).
The prior normal retry fails linking after 551.636s with zero output. The 157-unit
diagnostic's 21157 input frames include duplicates; the exact limit is unresolved.
The 256-input cap is not the cause; no 4096-type hit or work-cap cause is established.
The frozen-input A/B finishes in 577.746s without drift: the old full-map
type plan hits work 1048576; the prefix plan succeeds at work 373752, with
5940 retained maps and 3359 canonical types. The next refusal is metadata
preparation (diagnostic 33002); its specific internal limit is unresolved.
A fresh optimized build-only seed/help passes in 132.660s. The new normal
646-source retry exits 2 after 561.174s at phase 3/stage 0/code 1002,
with zero output.
Normal acceptance, fixed points and current parity remain pending; Linux is batched.
Contracts, ABI, limits and source validation stay unchanged. See
`candidate-project-type-prefix-20261107.md` for evidence and comparison boundaries.

## Project metadata field map follow-up

Commit `8c2d5acb` integrates worker `2ca0cb9b`: ordered field visits append their
target directly; holes and unused tails keep charged zero initialization. The
completed dense map, duplicate-target checks and all limits remain unchanged.
Windows low/optimized native regression passes in 113.530s; an 8192-cell ordered
map reduces work 16384 to 8192, and real 8-unit IR reduces 14312 to 12264.
Integrated source check/locked pass (755 files, 9530 MIR, 13.413s / 13.338s).
A fresh optimized full CLI seed and help smoke pass in 131.862s (205 C parts).
The new normal retry exits 2 after 515.570s in phase 3/stage 0/code 1002,
with zero output. The frozen-input diagnosis isolates the independent 1024-layout
cap at exactly 1024; metadata work exhaustion is not the cause. It saves all
157 checked IR units with authenticated frozen source/image/packet hashes.
A 24.075s detached replay with an isolated 2048 cap prepares 1116 unique layouts
(5708 fields, 5772 callables, 17685 parameters, metadata work 266376), then
refuses in cleanup assembly (44002). This is diagnostic evidence, not normal
acceptance or a source-bound cache hit. A bounded layout policy follow-up is in
an isolated worker worktree. Fixed points, current-source full parity and Linux
native remain pending. See
`candidate-project-metadata-map-20261107.md` for evidence and scope.

## Linked layout and drop capacity follow-up

Commits `88d59a6e` and `830f4d14` integrate a shared 2048 compiled-layout limit,
including public-view projection. Boundary/codec tests pass Windows low and
optimized in 27.800s; existing generic aggregate public-interface regressions
pass both profiles in 109.742s. Raw/parser/AST limits remain separate.
Commit `c5b3646c` admits 32768 cleanup drops for linked projects while units
retain 16384. Its Windows both-profile tests pass in 19.298s, with independent
golden bytes, transport/structural checks and selected/unselected remapping.

Frozen-input diagnosis identifies the prior 44002 as the selected-drop 16384
cap (44402). A bounded count-only experiment reaches 2950 selected operands and
23100 selected drops in 24.565s; it intentionally stops before assembly.
An isolated replay with both proposed caps passes full linking in 25.848s,
retaining 10292 frames, with unchanged input/archive/image hashes and empty
stderr. Evidence is `.cache/current-project-link-cap-replay-20261107/run.json`.
This is detached diagnostic evidence from the original 646-source snapshot,
not a current-source normal build, Complete owner, cache hit or fixed point.

Integrated source check/locked pass with 755 files and 9532 MIR functions in
13.400s / 12.965s; their source hashes agree and the lock is refreshed. Evidence:
`.cache/current-project-capacity-source-check-20261107.log`. A fresh seed and
normal current-source retry follow separately. Native Linux remains batched;
CI, push, fixed points and full current-source parity are unexecuted here.

The integrated capacity seed/help passes in 130.092s (205 C parts). A fresh
36-source authored companion also builds development and release, with receipt
`passed=true`, at `.cache/current-authored-driver-capacity-20261107/receipt.json`.
The old companion receipt is not reused across the candidate-limits source
change. Evidence for the seed is
`.cache/current-project-capacity-seed-20261107/build.log`.

The normal current-source retry now passes linking and entry selection, then
exits 2 after 527.361s at **phase 5/stage 0/code 1002**, bytes 0..0, with zero
output. This is a new C-emission failure from `module_unit_c_emit::partition_entry`
(128 frames per initial partition), not the prior layout/drop link refusal.
Its exact internal guard is unresolved; no work, output-byte, CFG, SDK or
partition-count limit is asserted as the cause. The invocation is recorded in
`.cache/current-selfbuild-capacity-20261107/invocations.json`, with adjacent
source snapshot and stderr in `.cache/current-selfbuild-capacity-20261107.log`.
There is no successful normal build receipt, current-source fixed point or
full parity result. The authenticated detached IR archive remains available
for bounded emitter diagnosis without repeating the original frontend pass.

## C reachability drop projection follow-up

The test-only profiler isolates the next failure: reachability exhausts its
1048576 work ledger after 8299 frames, with 637322 CFG-block visits and 278010
target-lookup work. A specialization-only lookup change did not remove it.
The checked CFG now records drop-block IDs in its existing block-validation
walk; reachability reads that immutable projection in the same order, retaining
per-site/mark charges and all independent graph checks and existing bounds.

Source check/locked pass with 755 files and 9536 MIR functions in
19.681s / 15.707s. A fresh optimized diagnostic probe builds in 68.861s.
The original authenticated 157-unit IR replay then links and fully emits in
40.790s: 10292 frames, 81 C units, header 1842785 bytes, project work 9989821,
empty stderr and unchanged input/archive/image/changed-source hashes. Evidence:
`.cache/current-project-reach-drop-replay-20261107/run.json`. This remains
detached diagnostic evidence from the frozen 646-source snapshot, not a normal
current-source build, cache hit, executable publication or fixed point.

The focused `TestCandidateReachabilityWork` passes Windows low and optimized
(2/2), including independent reachability sets, ordered summaries and exact
work-bound refusal. Its checked production-file hashes match this checkout.
A fresh full CLI seed/normal retry follows separately.
Linux native, CI and push remain unexecuted. See
`candidate-reachability-drop-projection-20261107.md` for the projection contract
and precise acceptance boundary.

The seed/help passes in 239.176s. The normal build now succeeds at revision
`b4008e0e`: `.cache/current-selfbuild-reach-drop-20261107/receipt.json` has
`passed=true`, full authored CLI scope, both development/release images and no
Go frontend/runtime builder requirement. Its fresh 646-source C emission takes
522.053s (55072792 bytes), followed by development/release native builds in
48.709s / 663.291s and frozen-version checks. This serial, one-CL run is not a
parallel compilation benchmark. Fixed point and full behavior parity remain
unexecuted; this is not a later-revision acceptance claim.

## Shared enum/switch layout lookup follow-up

The already charged and duplicate-checked SDK layout index is now retained and
shared by enum values and switch emission. Standalone scanner entry points,
member checks, C bytes, ordering and all bounds remain unchanged. The focused
2048-layout fixture reduces 4194304 scanner visits to 2048 index queries and
passes Windows low/optimized separately in 18.070s / 23.337s.

The combined drop-projection/layout-lookup tree passes source check/locked
(755 files, 9541 MIR, 14.270s / 13.959s) and the both-profile native regression
in 32.064s. Its exact production source and refreshed lock are integrated.
The prior successful normal build belongs to `b4008e0e`; a new combined-source
normal build, fixed points and full parity remain pending. See
`candidate-c-enum-layout-lookup-20261107.md` for evidence and restrictions.
Linux native stays batched; no CI, push, publishing or runner changes occur.

## Latest-source timeout and parallel investigation

The combined 7237dd67 source is attempted with both the b4008e0e Fairy-emitted
native image and the Go-emitted C seed. Both exceed the unchanged 900s frontend
watchdog before publishing a C packet or successful receipt. Their 646 source
byte sequences, order, lock and frozen version match; only generated wrapper
paths differ. These are failed attempts, not current-source acceptance.

The subsequent host CPU samples are 91-100% while unrelated compilation is
active. Contention has not been isolated as the cause, nor has a code-generation
slowdown been established. A bounded 24-observation microbenchmark has matching
checksums/exits and no major scalar/Vec runtime regression; its POD case is
noisy and cannot explain the complete workload. Existing opaque copy barriers
remain intact. No third expensive retry starts under the observed load.

Latest-source normal acceptance, fixed points and full parity remain pending.
See `candidate-parallel-acceptance-20261107.md` for complete attempt boundaries,
static findings, microbenchmark evidence and the distinction from the successful
b4008e0e build. Linux remains batched; default time/work bounds are unchanged.

## Body-phase isolation

An exact-input, test-only optimized Windows probe reaches all 646 inputs in
153 checked units after 88 capacity retries. Parsing/signatures take 3.416s,
body checks and IR construction 860.413s, and linking 27.505s. C emission enters
at 891.334s; the unchanged 900s diagnostic watchdog terminates it before a
packet is published. The native process uses 786.453s CPU, all input/image
hashes remain unchanged, and no successful normal receipt exists. The runner's
exit 0 only records that it saved the failed diagnostic and verified hashes.

A separate three-call constant-plan probe measures only 1.6228ms median
removable computation, with 5.2553ms source/owner validation retained. Provider
recurrence is unmeasured; project-wide constant caching is not implemented on
that evidence. A single slow checked range is the next bounded measurement.
The native-build helper now preserves timeout invocation/partial-output evidence;
nine focused tests and an actual two-second Windows child timeout pass.

See `candidate-body-phase-diagnosis-20261107.md` for phase markers, failed-probe
boundaries and remaining acceptance. This is not normal CLI, fixed-point or
full-parity evidence. Product limits and MSVC aliasing barriers remain intact.

The subsequent single-range [445,450) probe succeeds with unchanged source/image
hashes: 22.886s checking/lowering, of which 21.870s (95.56%) is origin analysis
including 35 provider value-plan builds. Primary closure takes 853.152ms;
primary values take 45.575ms. Provider preparation and the remaining origin
factory/solver costs are not yet separated. No provider cache is implemented;
this is a bounded diagnostic, not full compiler acceptance.

## Current-source normal release success

Revision `d0b84329` passes a fresh optimized full-CLI seed regression (205 C
parts, 14 CLI processes) and a normal release self-build. The authored frontend
reads 646 sources and emits a 55063498-byte C packet in 512.243s; release native
compile/link takes 633.650s. Both retain their existing 900s stage watchdogs.
Version equality, source/tool/image hashes and the inherited 16MiB PE stack
pass. The complete wrapper takes 1151.378s, not less than 900s.

The successful full-dispatcher receipt is
`.cache/current-selfbuild-d0b84329-optimized-20261107/receipt.json`.
It has `passed=true`, `goFrontendInvoked=false`, no frontend reuse, no narrowed
entry scope, no runtime Go builder and no product timeout change. The installed
Fairy companion/assets reuse is authenticated against all 59 source hashes.
The failed entry-lookup optimization was discarded before this build.

This establishes normal release construction. The separate fixed-point and
comparison gates are recorded below, including their exclusions.
See `candidate-body-phase-diagnosis-20261107.md` for the seed identity, packet
and image hashes, invocation timings and varying host CPU samples. Linux stays
batched; CI, push and release/publishing are unexecuted.

## Parallel follow-up verification

The normal release receipt above feeds three independent jobs with disjoint
directories under `.cache/current-post-selfbuild-d0b84329-20261107`: native fixed
point, authored-companion fixed point, and the five serial comparison suites.
Each native builder retains the existing two-CL admission limit. No product
source, compiler option, deadline, public API, SDK ABI or runner is changed.

The companion verifier passes in 51.409s: the current compiler reproduces the
exact 2037976-byte C packet, and three 401920-byte companion generations have
the same SHA-256 `11b0a82552e785529d015bdb5479aec2bae395478b35749e942308d8e79fe960`.
`packetBytesEqual`, `optimizedImageBytesEqual` and `scratchDirectoryRemoved`
are true; no image bytes are excluded, no bootstrap builder or Go frontend is
invoked. Evidence: `companion-fixed/receipt.json`.

| Release Go/Fairy suite | Executed / exact matches | Excluded | Seconds |
| --- | --- | --- | --- |
| Behavior check | 173 / 173 | 1 | 56.115 |
| Behavior run | 54 / 54 | 2 | 167.343 |
| Behavior CLI | 7 / 7 | 0 | 4.130 |
| Workspace CLI | 30 / 30 | 0 | 10.973 |
| Shipped examples | 78 / 78 | 1 | 414.328 |

All 342 executed comparisons match acceptance, exit status, diagnostics, stdout
and stderr. Both workspace suites have `nativeContractFailures=[]`. The two
CLI suite receipts have `parity=true`; check/run/examples retain `parity=false`
because their exclusions are not silently converted into passing cases:

- `fail.flow.local-region-escape` uses a direct fixture packageGraph with a
  synthetic standard-library provider. CLI workspaces reject local `std::*`
  providers and inject the bundled SDK. Substituting its Arena would change
  provider source/identity and is not a graph-equivalent adapter.
- `pass.c-emission.fixed-array-bounds` and
  `pass.c-emission.mutable-fixed-array-byte-view-bounds` do not meet this run
  suite's passing `main::main` entry condition.
- `lexer_candidate` self-source belongs to the separate normal/fixed-point
  gates, not the shipped-example workspace adapter.

Receipts and every observation remain in each suite's separate output folder.
The reference is the authenticated, frozen-version `source-version.exe` from
the normal build. The synthetic-stdlib fixture remains an original-fixture
coverage gap; an adapted real-SDK check must be labelled separately.

The native fixed point also passes: frontend 821.298s, fresh Stage2 release
compile/link 632.728s and version execution 0.048s, all exit 0 with empty stderr.
The complete verifier takes 1455.800s. Each frontend/build stage retains its
default 900s watchdog. Both complete 55063498-byte C packets have the original
packet hash, and both complete 7788544-byte PE images have the original
`8a5e2e0d0522dc12a6c0c90c54c5d0b5efd85ce50131bf20aa9050a93128cc40`
hash. No image byte is excluded. `native-fixed/receipt.json` has `passed=true`,
`packetBytesEqual=true` and `optimizedImageBytesEqual=true`, with the inherited
16MiB stack and no Go frontend or product timeout change.

All follow-up commands have exited, and no observed verification/child PID
remains. This completes the current Windows release construction/fixed-point
slice and the five comparison runs; it does not turn four excluded comparisons
into passes or establish development-profile self-build, Linux, clean-machine
distribution, complete `go test ./...` or CI acceptance. Linux remains batched;
no push, publishing, dependency, repository-policy or runner change occurs.
