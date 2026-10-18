# Prototype Usage

- Status: Draft

Manual developer examples moved from `README.md` unchanged. They drive the product commands and
experimental surfaces directly from a checkout with `go run`. They are examples, not an
authority over command behavior; the stable command surface is owned by
`docs/cli/stable-cli-contract-v0.md`.

Product commands stay under `cmd/`: `cmd/fairyc` and `cmd/fairyc-lsp`. Commands that were
developer utilities now live under `tools/dev/` or `tools/candidate/`, and the examples below use
the relocated paths.
## Try the current prototype

The following commands are manual developer examples, not agent command authority:

```text
go run ./cmd/fairyc explain E1101
go run ./cmd/fairyc targets
go run ./cmd/fairyc experimental run-host examples/run/modules/main.fy --source examples/run/modules/math.fy --profile optimized
go run ./cmd/fairyc experimental build-host examples/run/modules/main.fy --source examples/run/modules/math.fy --profile optimized --output build/fairyc-windows-smoke.exe
go run ./cmd/fairyc experimental build-target examples/windows_target/main.fy --target x86_64-windows-msvc --profile optimized --output build/fairyc-windows-target.exe
go run ./cmd/fairyc experimental run-target examples/windows_target/main.fy --target x86_64-windows-msvc --profile optimized
go run ./cmd/fairyc experimental exec-target examples/windows_output/main.fy --source examples/windows_output/windows_console.fy --target x86_64-windows-msvc --profile optimized
go run ./cmd/fairyc experimental check-syntax examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/semantic/main.fy examples/semantic/math.fy
go run ./cmd/fairyc experimental check-semantics examples/resources/main.fy
Push-Location examples/regions
go run ../../cmd/fairyc check
Pop-Location
go run ./cmd/fairyc experimental check-semantics examples/results/main.fy
go run ./cmd/fairyc experimental check examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental check-workspace
go run ./cmd/fairyc experimental check-workspace --locked
Push-Location examples/workspace_quickstart
go run ../../cmd/fairyc check
go run ../../cmd/fairyc check --locked
go run ../../cmd/fairyc build
go run ../../cmd/fairyc build --release --locked
& ./build/workspace-quickstart.exe
go run ../../cmd/fairyc run --release --locked
go run ../../cmd/fairyc test --locked
go run ../../cmd/fairyc test --native --locked
go run ../../cmd/fairyc test --native --release --locked
go run ../../cmd/fairyc experimental emit-workspace-c --target x86_64-windows-msvc --locked
go run ../../cmd/fairyc experimental emit-workspace-c --target x86_64-linux-gnu --locked
go run ../../cmd/fairyc experimental check-workspace
go run ../../cmd/fairyc experimental run-workspace --locked --target x86_64-windows-msvc --profile optimized
Pop-Location
go run ./cmd/fairyc experimental target-spec x86_64-linux-gnu
go run ./cmd/fairyc experimental target-spec x86_64-windows-msvc
go run ./cmd/fairyc experimental emit-c examples/c/main.fy
go run ./cmd/fairyc experimental emit-c examples/c/tagged.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource_tagged.fy
go run ./cmd/fairyc experimental emit-c examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental build examples/c/tagged.fy --cc clang --profile optimized --output build/fairyc-example
go run ./cmd/fairyc experimental build-target examples/owned_text/main.fy --source examples/owned_text/owned_text.fy --target x86_64-windows-msvc --profile optimized --output build/owned-text.exe
go run ./cmd/fairyc experimental run examples/run/main.fy
go run ./cmd/fairyc experimental run examples/run/resource_option.fy
go run ./cmd/fairyc experimental run examples/run/modules/main.fy --source examples/run/modules/math.fy
go run ./cmd/fairyc experimental parse examples/syntax/main.fy
```

To exercise the bounded process-local workspace cache from PowerShell without WSL:

```powershell
Push-Location examples/workspace_quickstart
@(
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":1}'
  '{"schemaVersion":"fairyc.experimental.workspace-check-session/v0","kind":"check","sequence":2}'
) | go run ../../cmd/fairyc experimental check-workspace-session --locked --target x86_64-windows-msvc --max-requests 2 --cache-max-entries 8 --cache-max-bytes 33554432 --cache-max-artifact-bytes 8388608
Pop-Location
```

The first response must report `fallback` and the second `all-hit`. These numeric limits are only
explicit example inputs; they are not product defaults or tuning recommendations.

For repeated Windows use, `DEVELOPMENT.md` gives the WSL-free manual bootstrap that creates ignored
`build/fairyc.exe`. The configured Windows smoke separately builds a temporary standalone compiler
process, requires it to create a no-overwrite starter, check it, run its semantic and native tests, and compile
its generated source through MSVC with conventional program status zero. The same executable runs
`examples/windows_quickstart/main.fy` through MSVC with
native result 42 by iterating two bytes with checked `usize` arithmetic and safely converting each
indexed `u8` value to `i32`,
then execute `examples/windows_output/main.fy` with exact stdout and process status
42 while explicitly supplying its separate `windows_console.fy` module, and finally run the
two-package `examples/workspace_quickstart` graph through an exact generated lock with native result
42, and runs the safe `std::io` workspace with exact text and byte-view lines on separated
stdout/stderr and status zero. The same standalone
compiler runs the isolated semantic test, a real native trap followed by a passing native test, and
the checked-in native test at both MSVC profiles without changing the retained executable. It also
emits locked Windows and Linux C11 application units, then measures nine fresh-cache cold/warm pairs with one new compiler process per sample,
sends eight editor-shaped workspace checks to one compiler process, validates the cold, no-op,
private-root, private-dependency, and public-dependency cache paths, and records Windows
process-memory peaks. Raw paired timings are local observations with zero retries, not a speedup or
default-cache claim. The compiler binary is
still built from Go; only the input programs and resulting temporary
target executables are Fairy C.

The stable `explain` and `targets` commands do not read the current directory or source tree. The
former marks its diagnostic catalog as provisional; the latter reports declared TargetSpec evidence
without probing local MSVC, SDK, WSL, or any other toolchain. The experimental source commands treat the current directory as the temporary source
root. `parse` emits versioned,
deterministic JSON containing token and CST kinds and byte spans, but not source text, token text,
literal values, or absolute paths. `check-semantics` accepts an explicit bounded list of source files
for the currently implemented scalar, safe-borrow, whole-resource, Region-parameter, and Result subset. It rejects duplicate paths, root
escapes, oversized input sets, and unsupported syntax rather than claiming success. The official
`experimental check` continues that explicit closure through MIR lowering and verification without
requiring `main`, running code, generating C, probing MSVC or WSL, or writing an artifact. It can
therefore validate an admitted FFI wrapper that the interpreter cannot execute. The official
`experimental run` accepts one entry plus a bounded explicit repeated-`--source` module closure whose
entry module contains `main() -> i32`, then executes its
verified MIR in the bounded interpreter, including boolean `if`, short-circuit control, scalar
`while`, classic scalar `for`, checked compound assignment,
resource construction, field reads, whole-root call moves, resource-bearing built-in tagged values,
ordered resource reassignment, consuming-close ownership patterns, cleanup drop events, Region
fallthrough, direct-return, Result-propagation, nested, break, and continue reset paths, checked
addition, subtraction, multiplication, negation, matching-type bitwise operations, and checked
signed or unsigned right shift. It performs no source
discovery and needs no C toolchain, MSVC, WSL, or host subprocess, and reports either `result: <i32>` or a canonical
trap. The official
stable `fairyc check [--locked]` command now checks the current `fairy.toml` workspace through
verified MIR and maintains or verifies `fairy.lock`. `experimental check` remains the explicit-file
developer surface; stable machine-readable target JSON and broader configuration are still separate work.

The private `fairyc experimental emit-workspace-c --target <target-id> [--locked]` command consumes that same manifest-owned
workspace and root `main::main fn() -> i32`, then writes only the selected Linux GNU or Windows MSVC
ISO C11 application translation unit to stdout. Target selection is mandatory because LP64 and
LLP64 are different contracts; the command never guesses from this host. It invokes no C compiler,
linker, SDK, WSL, subprocess, or network and creates no build artifact. The separate runtime and
standard-library support unit is intentionally absent, so redirected output is inspection/transport
input rather than a promised standalone C package. Ordinary mode maintains `fairy.lock`;
`--locked` verifies it without writing. Generated C is compiler bootstrap transport rather than a
cross-release source product; the removed stable `fairyc emit c ...` spelling returns code 2 with
the experimental migration direction.

The stable `fairyc test [--json] [--locations] [--native [--release] [--stdin | --stdin-file <portable-workspace-file>] [--timeout-seconds <1..60>] [--capture-output] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` command keeps tests out of ordinary compilation. It overlays
the root package's optional lowercase `tests/` directory after the production lock is handled, or
exactly one root-reachable local member when its manifest UID is selected with `--package`;
`tests/answer.fy` therefore declares `module tests::answer;`. Mark a private safe zero-argument
`void` function with argument-free `@test`. Without `--native`, each discovered test runs in
canonical name order through a fresh verified-MIR interpreter state without a toolchain or artifact.
With `--native`, the compiler appends one private verified `fn() -> i32` harness per test, emits its
reachable closure, compiles and inspects one temporary Windows MSVC image, and executes it with EOF
stdin by default. Explicit `--stdin` requires exactly one selected test after complete compiler
admission and then attaches caller input directly without pre-reading, buffering, or replay; values
after the first `--` become that process's arguments, and `--release` selects
optimized MSVC. Repeatable `--exact` values select
canonical `<module>::<function>` identities byte-for-byte while retaining canonical report order;
repeatable `--file` instead selects exact portable workspace-relative test-source paths, including
member workspace prefixes and known files with zero tests. Unknown, duplicate, host-separated, or
case-mismatched paths fail; `--file` and `--exact` do not combine. Repeatable `--dir` recursively
selects exact admitted directory descendants using the same portable path spelling. Known directories
come only from admitted test-source ancestors, so empty filesystem directories are not selection
targets; a directory containing only zero-test sources is valid. Unknown, duplicate, host-separated,
case-mismatched, dot-segment, and trailing-slash spellings fail, and `--dir` does not combine with
`--file` or `--exact`. The complete selected-package test overlay is still compiled.
`fairyc test --list [--package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]` performs that complete
compiler admission and prints the canonical identities without running the interpreter, backend,
toolchain, or test process. Listing with `--json` emits the stable `fairyc.test-list/v1` schema with
the canonical identity array, count, and lock state. Execution with `--json` emits the stable
`fairyc.test-report/v1` schema with canonical outcomes, engine/target/profile identity, totals,
aggregate status, and lock state. A trap fails an executed test and later selected
tests still run in fresh processes. `--locations` adds an exact-current source anchor to text and
selects `fairyc.test-list/v3` or `fairyc.test-report/v3` for JSON. Each anchor contains package UID,
portable workspace-relative path, 1-based line and byte column, exact name byte span, and matching
`fyfile1`/`fylm1` identities without exposing an absolute path or source text. Native JSON
`--capture-output` uses `fairyc.test-report/v4` and adds required canonical base64 stdout/stderr to
every outcome, preserving binary and pre-trap bytes while the private trap control stays off-stream.
Without that flag, native output remains rejected and legacy report bytes do not change. Shared
multi-test input, live output streaming,
portable targets, benchmarks, fuzzy/tag/glob filters, catalog-wide unreachable aggregation, and
feature-instance selection remain separate contracts.

Phase 4 now has its first executable boundary: `internal/semanticsnapshot` turns successful typed
HIR, semantic facts, verified MIR, and exact package metadata into detached, path-free compiler
facts. It is invariant under build-local package renumbering, keeps external C calls explicitly
opaque without degrading unrelated local facts, and publishes no source text or authority-bearing
documentation. `internal/symbolidentity` now turns those declarations into provisional
source-instance-independent SymbolIds with canonical signature hashing and mandatory visible/full
collision checks. Exact per-file content and LF-derived line-map identities now travel from the
validated workspace into the immutable snapshot without source bodies or host paths, closing the
stale-span metadata prerequisite. `internal/atlasmap` now validates the complete workspace input and
builds one immutable, deterministic logical partition per exact package. Each header carries the
exact compiler build, language, package, source, interface, file, line-map, and SymbolId algorithms,
and incomplete relations become explicit stable omissions. The same package now encodes each
partition as caller-bounded canonical `fairyc.fymap-json/v1`, binds it with the domain-separated
`fymap1-sha256` checksum, rejects unknown or noncanonical JSON, revalidates every identity and
record relationship, and rederives every SymbolId from the stored declaration and exact package
namespace references. The opt-in `MapStore` now publishes checksum-named immutable generations under
an opened root with mandatory retention bounds, recovers abandoned temporaries, rejects corrupt or
conflicting generations, and requires exact current workspace package/file/line-map plus compiler
and interface identities before returning spans. The internal query index repeats that freshness
check, accepts only exact SymbolIds, and implements deterministic caller-bounded `overview`, `show`,
and package-local `around` queries with explicit filter, depth, byte, dependency-join, and incoming-
map omissions. A separate bounded `WorkspaceCallIndex` now revalidates the complete caller-supplied
package set, joins exact compiler-observed calls to dependency SymbolIds, and exposes detached local
and cross-package outgoing/incoming ledgers without changing those query results. The default-free
`experimental atlas-calls list` command selects one exact package-qualified visible/full SymbolId
and direction under explicit package, call, and output limits, then emits the separate checksummed
`fairyc.workspace-call-catalog-json/v0` direct-call catalog. It adds no traversal or impact claim.
`internal/contextpack` now runs the package-local queries directly and emits caller-bounded,
checksummed canonical `fairyc.context-pack-json/v1` with exact package/compatibility/query/budget/
usage/completeness/fact/relation/omission fields and visibly empty text sections. The first
`experimental atlas-pack overview` command exposes one explicitly selected package UID with explicit
record/selection/output limits and no map-store default. Exact SymbolIds obtained from the separate
catalog command can now drive `experimental atlas-pack show` and `around` with explicit direction,
relation, record, relation, depth, selection-byte, and output-byte policy. Those commands add no
default store policy, workspace call-ledger consumption, source-excerpt output, pack decoder, or text
importer. The internal
query index now also provides exact full-SymbolId-ordered catalog pages with exact module/name/kind
filters, explicit symbol/byte limits, and full-ID-only continuation. Exact-name matches retain every
same-name declaration across modules rather than guessing one winner. Symbol Catalog v1 exposes
those pages through `experimental atlas-symbols list` as checksummed
`fairyc.symbol-catalog-json/v1`, with every filter and output limit explicit. The Windows
host smoke now also records nine raw whole-package canonical rewrite samples after planning, checks
byte identity, and reports fixed source/record/encoded-byte denominators without choosing chunking,
compression, incremental updates, or a performance budget.

The Windows host smoke now also drives that exact root `main` call catalog through nine fresh locked
standalone compiler processes after one untimed warmup. Its path-free
`fairyc.atlas-call-process-measurement/v0` report preserves raw caller wall time, whole-process peak
working-set and pagefile bytes, fixed source/call/output denominators, and the explicit construction
bounds. This is a tiny local control with uncontrolled OS caches, not a reference budget, speedup,
Atlas-only heap measurement, representative repository result, traversal decision, or impact claim.

A separate in-process scaling control generates fixed star workspaces with 2 packages and 8 calls,
8 packages and 64 calls, and 32 packages and 512 calls. After generation, loading, compilation, map
planning, and symbol discovery, it records nine raw `WorkspaceCallIndex` construction plus outgoing
catalog-encode observations per shape, including process-wide Go `TotalAlloc` and `Mallocs` deltas
and canonical output bytes. The three shapes and 64 MiB fail-closed output ceiling are harness
policy, not representative workloads, complexity evidence, retained-memory attribution, product
defaults, traversal policy, impact evidence, or regression thresholds.

The Go correctness suite separately generates complete chain, diamond, and fan-in `.fy` workspaces.
It compiles them to verified MIR, prepares exact-current maps, checks complete cross-package call
usage plus outgoing/incoming/both selections, and requires repeated canonical catalog bytes. These
synthetic non-star controls remove one structural blind spot from the star measurement, but they are
still not representative repositories, indirect-call recovery, traversal, impact, or performance
evidence.

ADR 0195 closes the attributed-text policy, ADR 0196 implements its exact retained-source gate, and
ADR 0197 adds canonical bounded `fairyc.attributed-text-json/v0` output with explicit selection,
exact subject identity, independent document integrity, and no caller-result forgery. ADR 0198
exposes it as `experimental atlas-text source` with exact package/SymbolId selection, explicit
workspace origin and count/text/output budgets, and successful checksummed size omissions. `.fymap`
v1 stays body-free and Context Pack v1 keeps its text arrays empty. ADR 0199 separately classifies
physical-line-leading `///` trivia and carries the exact attached span for supported top-level
declarations through HIR. ADR 0200 preserves that fact in an in-memory-only snapshot sidecar and
exposes exact marker-bearing bytes as `experimental atlas-text documentation`, including explicit
`kind-unavailable`, safety, and size omissions. Existing source output and `.fymap` v1 bytes remain
unchanged. No decoder, text store, stable attributed-text CLI, rendered documentation model,
dependency text, file reopening, network fetch, model call, or agent authority exists.

ADR 0202 adds `experimental atlas-evidence show` for one exact SymbolId. It prepares the workspace
once and nests an unchanged compiler-fact Context Pack v1 plus unchanged source and documentation
attributed-text v0 documents in canonical checksummed `fairyc.atlas-evidence-json/v0`. The wrapper
has authority `none`, text remains `untrusted-text`, component and final limits are mandatory, and no
decoder, default, store, token estimate, model call, or Context Pack text population is introduced.

`experimental target-spec` accepts exactly one known Linux GNU or Windows MSVC ID and writes
deterministic `fairyc.target-spec/v0` JSON. The result separates target semantics from local
toolchain discovery and lists the profile-specific missing evidence that keeps all three target
maturity claims false.

`experimental emit-c` accepts the same explicit entry-plus-`--source` closure and prints the verified
scalar plus copy-scalar/inert-struct tags, flat all-`i32` resources, and resource-bearing built-in
tags as a C11 translation unit directly to stdout.
Every used C scalar receives target size/alignment assertions; signed and unsigned integer families
also receive signedness assertions. Target `c::ssize_t` alone adds the declared POSIX.1-2008 feature
macro and provider header. The command does not write a `.c` file or invoke a native toolchain.

On Linux/amd64 with glibc and the explicitly selected Clang or GCC installed, `experimental build`
uses that same verified unit and the embedded runtime to produce a no-overwrite executable
below the ignored `build/` directory. Compiler, `low` or `optimized` profile, and output path are
mandatory. This Windows host cannot execute that command, and the command remains narrower than the
interpreter: scalar tags, direct flat nonempty resources with empty or pure nonempty drop bodies, and
`Option<R>` or `Result<S, E>` where each payload is exactly `i32` or such a direct resource work.
Nested tags, partial initialization, user drop effects, general unsafe operations, arbitrary
aggregate parameters, callbacks, and public standard-library programs still fail closed before
native compilation. `unsafe extern C fn` calls, optionally mapped through one validated
`@link_name`, admit only the closed scalar, raw-pointer, and selected `@repr(C)` result transports;
`experimental build` succeeds only when the selected C driver can resolve the
external symbol from its ordinary link environment because the product CLI has no companion-source
or arbitrary link-input option.
