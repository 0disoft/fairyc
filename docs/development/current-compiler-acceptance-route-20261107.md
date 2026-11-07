# Current Windows authored compiler acceptance route, 2026-11-07

This is the integration plan and bounded evidence record, not a claim that the
current compiler has self-built. The parent owns the final isolated source
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
