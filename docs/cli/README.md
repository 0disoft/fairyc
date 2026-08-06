# `fairyc` CLI

- Status: Draft
- Repository type: cli-tool
- Command contract: `docs/cli/command-contract.md`

`fairyc` is the canonical compiler and toolchain executable. The CLI must expose fast development checks separately from expensive production optimization and must make emitted intermediate forms explicit.

## Planned command families

- project: `new`, `build`, `run`, `test`, `check`;
- source: `fmt`, `lint`, `doc`;
- compiler inspection: `emit`, `explain`, `targets`;
- packages and reproducibility: `add`, `vendor`, `cache`;
- interoperability: `bindgen`, `abi-check`;
- semantic navigation: `atlas`.

Stable `new`, `check`, the minimal Windows `build` and `run`, compiler inspection, C emission, and
semantic or isolated Windows-native test execution now have Stage 0 contracts. The remaining command families are product
commitments only after their own contracts and implementations exist.

New starter v4 projects keep the same manifest and Fairy sources as v3, while their exact generated
README distinguishes fast `fairyc test --locked` from isolated Windows `fairyc test --native --locked`.

## Current prototype

The Go frontend currently exposes stable starter creation
`fairyc new <package-name> --uid <package-uid>`, stable current-workspace commands
`fairyc check [--locked]`, `fairyc build [--release] [--locked]`, and
`fairyc run [--release] [--locked]`, the stable semantic-by-default or Windows-native
`fairyc test [--json] [--locations] [--native [--release] [--stdin] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` and execution-free
`fairyc test --list [--json] [--locations] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]`, plus the stable source-independent
`fairyc explain <diagnostic-code>` lookup and stable host-independent `fairyc targets` maturity
report, and stable target-explicit
`fairyc emit c --target <target-id> [--locked]`, plus `fairyc experimental parse <file.fy>`,
`fairyc experimental check-syntax <file.fy>`, and
`fairyc experimental check-semantics <file.fy> [<file.fy>...]`. The complete implemented
non-executing pipeline is `fairyc experimental check <primary.fy> [--source <file.fy>]...`.
The package-aware sibling is `fairyc experimental check-workspace [--locked]`; it consumes the
current directory's `fairy.toml`, explicit members, and canonical `fairy.lock`.
`fairyc experimental check-workspace-session --target <target-id> --max-requests <count>
--cache-max-entries <count> --cache-max-bytes <bytes> --cache-max-artifact-bytes <bytes> [--locked]`
keeps one bounded verified-MIR Session while a caller submits sequential versioned NDJSON checks.
It is a foreground experimental process, not a daemon, watcher, persistent cache, or stable editor
protocol; every limit is mandatory and caller-selected.
`fairyc experimental check-workspace-cache --target <target-id> --cache-dir <build/path>
--cache-max-entries <count> --cache-max-bytes <bytes> --cache-max-record-bytes <bytes>
--cache-max-artifact-bytes <bytes> --cache-temp-max-age-seconds <seconds> [--locked]`
is the separate one-shot persistent proof. It identifies the exact running executable, reads and
writes only the explicit ignored workspace cache directory, and reports versioned JSON.

`fairyc experimental atlas-pack overview --package-uid <uid> --max-records <count>
--max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` compiles the manifest-owned
workspace, selects one exact package UID, and writes one canonical compiler-fact Context Pack to
stdout without a trailing newline. Every size limit is explicit. The command neither reads nor
writes a `.fymap` store, imports source or documentation text, nor discovers symbols by name.

`fairyc experimental atlas-symbols list --package-uid <uid> --kind <kind> [--kind <kind>]...
[--module <module>] [--name <identifier>] [--after <full-symbol-id>] --max-symbols <count>
--max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` is that exact discovery path.
It emits a checksummed Symbol Catalog v1 page in full-SymbolId order. Optional module and name
filters are exact; same-name declarations remain separate and there is no fuzzy search or hidden
default.

`fairyc experimental atlas-calls list --package-uid <uid> --symbol <symbol-id>
--direction <outgoing|incoming|both> --max-packages <count> --max-calls <count>
--max-output-bytes <bytes> [--locked]` consumes the separate exact-current workspace call join. It
atomically validates the complete manifest workspace under the caller's construction limits, then
emits one checksummed workspace call catalog v0 for that exact package-qualified symbol and
direction. It has no default limit, partial join, source text, host path, graph traversal, indirect
call, or impact claim.

The resulting visible or full ID can drive `fairyc experimental atlas-pack show` or `around`.
Both require explicit direction, one or more exact relation kinds, record/relation/selection/output
limits, and an exact package UID; `around` additionally requires a depth from zero through 64. They
emit the same checksummed compiler-fact Context Pack version and preserve incomplete dependency or
incoming-package joins as omissions. They do not search names, read or write a map store, import
source/documentation text, or infer that a change is safe.

The same exact ID can drive `fairyc experimental atlas-text source --package-uid <uid>
--symbol <symbol-id> --origin workspace-authored --max-fragments <count>
--max-fragment-bytes <bytes> --max-text-bytes <bytes> --max-output-bytes <bytes> [--locked]`.
This separate command returns one canonical checksummed untrusted source-excerpt document or a
checksummed text-budget omission. Replacing action `source` with `documentation` selects only the
exact frontend-attached marker-bearing `///` block and returns `kind-unavailable` when none exists.
Neither action has default limits, name search, SDK or dependency text, output path, decoder, model
call, or permission authority, and neither populates Context Pack v1.

`fairyc experimental atlas-evidence show --package-uid <uid> --symbol <symbol-id> --direction
<outgoing|incoming|both> --relation <kind> [--relation <kind>]... --max-records <count>
--max-relations <count> --max-selection-bytes <bytes> --max-fragment-bytes <bytes>
--max-text-bytes <bytes> --max-facts-bytes <bytes> --max-text-document-bytes <bytes>
--max-output-bytes <bytes> [--locked]` compiles that exact workspace once and nests the complete
unchanged Context Pack, source attributed-text, and documentation attributed-text documents in one
checksummed Evidence Bundle v0. The wrapper has no authority; facts remain compiler facts and both
text children remain untrusted. Every component and final bound is explicit, and the command adds no
decoder, defaults, store, token estimate, model call, or Context Pack text population.

The persistent command is deliberately not another session mode. Every invocation is a fresh
process and hashes its executable before constructing a stable disk Session. The cache path must be
canonical, use forward slashes, and remain under ignored `build/`. All entry, retained-byte,
record-byte, artifact-byte, and abandoned-temp-age values are mandatory canonical positive decimals;
artifact bytes cannot exceed record bytes and record bytes cannot exceed retained bytes. There is no
config file, environment override, remote cache, implicit source discovery, default, watcher, or
daemon. Its output schema is
`spec/cli/experimental-workspace-cache-check-v0.schema.json`. A raw record is never an executable
hit: the public interface, package instance, stable codec, complete package set, fresh identity
rebind, and ordinary MIR verifier must all succeed. Corruption and incompatibility fall back to
lowering. Oversized admission and failed disk writes are reported separately and cannot turn cache
state into source truth.
`fairyc experimental build-workspace --target x86_64-windows-msvc --profile <low|optimized> --output <build/path.exe> [--locked]`
retains a package-aware Windows PE, while
`fairyc experimental run-workspace --target x86_64-windows-msvc --profile <low|optimized> [--locked]`
compiles and executes the root package's exact `main::main` without retaining an artifact.
The compiler prototype also exposes
`fairyc experimental target-spec <x86_64-linux-gnu|x86_64-windows-msvc>` for deterministic
host-independent target inspection,
`fairyc experimental emit-c <entry.fy> [--source <file.fy>]...` for deterministic scalar, explicitly link-mapped foreign-call,
flat-resource, and resource-tagged C11 inspection and
`fairyc experimental build <entry.fy> [--source <file.fy>]... --cc <clang|gcc> --profile <low|optimized> --output <build/path>`
for an explicit host-Linux executable in that subset,
`fairyc experimental build-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low|optimized> --output <build/path.exe>`
for the first declared Windows target C-emission subset,
`fairyc experimental run-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low|optimized>`
for direct temporary execution of that declared Windows target subset,
`fairyc experimental exec-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low|optimized>`
for program-facing Windows execution with exact stdout/stderr forwarding and process status,
`fairyc experimental build-host <entry.fy> [--source <file.fy>]... --profile <low|optimized> --output <build/path.exe>` for a
retained target-neutral Windows/amd64 MSVC PE,
`fairyc experimental run-host <entry.fy> [--source <file.fy>]... --profile <low|optimized>` for a temporary target-neutral
Windows/amd64 MSVC host smoke, and
`fairyc experimental run <entry.fy> [--source <file.fy>]...` for the fixture-backed scalar MIR interpreter subset. These are developer commands whose
spelling and numeric exit codes may change. `check-semantics` resolves names and creates typed HIR
for an explicit bounded module set in the implemented scalar, safe-borrow, whole-resource, Region, and Result subset.
It checks local last-use loans, borrow conflicts, borrow return origins, whole moves, cleanup states,
branch joins, loop back-edges and control exits, Region parameter origins, `place`, and lexical Region escape when its
Arena provider is included explicitly. It also checks Result return, discard, propagation, local
and parameter transfer, branch and loop obligations, overwrite and early-return rejection, and
error compatibility.

`experimental check` uses the same bounded explicit closure transport as `run`, but the primary file
need not define `main`. It stops after MIR verification and reports function plus foreign-declaration
counts. It performs no target selection, interpreter execution, C emission, toolchain discovery,
process execution, network access, or artifact write, so verified FFI wrappers can pass even though
`experimental run` cannot execute their foreign calls.

`experimental run` analyzes only its explicit entry-plus-`--source` closure and executes the selected
module's zero-argument `main() -> i32` through verified MIR, including boolean `if`,
short-circuit control, scalar `while`, whole resources, copy-only and resource-bearing built-in
tags, ordered
resource reassignment, consuming-close ownership, Region fallthrough, return, propagation, nested,
break, and continue reset paths, and source-linked drop traces, and reports an
exact scalar result or canonical division, integer-overflow, or invalid-shift trap. Checked
negation, matching-type addition, subtraction, multiplication, division, remainder, bitwise
operations, and checked left or right shift are executable. Invalid shift amounts trap before a
left-shift overflow check; an unrepresentable left-shift result traps as `integer-overflow`;
loop control without an enclosing loop is rejected as `E1012`.
It does not discover files, execute user drop bodies, implement effects, or provide host I/O.
It also does not execute foreign calls: those fail with the interpreter capability boundary even
when the same verified MIR is admissible to `emit-c`.
`experimental emit-c` runs the same explicit verified-MIR module closure and writes one raw constrained C11
translation unit to stdout. It does not write an artifact, invoke a C toolchain, link, or run native
code. The native fixture lane's `drop-trace-v0` observer profile is not available through this
command and never appears in its output.
The admitted foreign declaration form currently supports `i32`, `u8`, `u32`, `usize`, all sixteen
exact transport-only TargetSpec C scalars, raw pointers to
the fixed-width Fairy scalar set or opaque `void`, and raw pointers to admitted nonempty acyclic
`@repr(C)` structs whose fields may include exact C scalars or nested `@repr(C)` records. Those
records may also be returned by value from a foreign C function. One
pointer-to-opaque-pointer output slot such as `ptr mut ptr mut void` is admitted without general pointer
nesting. It uses its source
identifier as the C symbol by default or accepts one nonempty `@link_name("symbol")`, and requires
an explicit unsafe boundary. Raw pointers can be null,
returned by another admitted foreign call, or produced by an exact direct-local borrow-to-pointer
cast inside `unsafe`. Dereference, arithmetic, integer conversion, mutability strengthening, and
arbitrary casts remain unavailable. No callback, C-scalar value operation, array or enum aggregate
field, by-value aggregate parameter, export, or inferred C attribute is implemented.
`experimental build` sends that exact constrained-C unit through the shared bounded C driver and embedded
runtime. It has no implicit compiler, optimization, or output choice; requires Linux/amd64 with
glibc; accepts only Clang or GCC and the closed `low` or `optimized` profiles; and creates a new
executable only under the source root's ignored `build/` directory. It neither overwrites an
  existing path nor implements empty, nested, partially initialized, or effect-bearing-drop
  resources; nested tags or resource tags outside the admitted direct-resource payload gate; the broader interpreter subset; native execution;
package builds; cross-compilation; installation; or a supported target.
It automatically supplies only the minimal project-owned standard-library C dependency closure for
exact admitted symbols used by actual calls. It accepts no user companion source, object, library,
search path, or arbitrary linker argument, so every other admitted foreign symbol must already
resolve through the selected driver's ordinary link environment.
The resulting product artifact is uninstrumented; the native fixture recorder is not linked.
`experimental run-host` and `experimental build-host` use the same WSL-free Windows developer-smoke
compiler path. They auto-discover the installed x64 MSVC and Windows SDK, admit only target-neutral
private C transport, accept only explicit FY `--source` inputs, and reject FFI, provisional standard-library calls, `@repr(C)`, and exact
`c::*` values. `run-host` removes the temporary PE after execution; `build-host` retains one
no-overwrite `.exe` under `build/`. Their receipts are host-smoke evidence, not Windows target, ABI,
distribution, or self-hosting claims.
`experimental build-target` is separate from those smokes. It selects the Windows MSVC emission
profile, emits target-labelled C11, requires Windows LLP64 C-scalar providers, rejects unavailable
`c::ssize_t`, and builds a no-overwrite PE through locally resolved x64 MSVC. One positional entry
plus repeatable `--source` values form an explicit closure of at most 64 FY files under the current
source root and existing aggregate byte limit. Duplicate paths, module/path drift, unresolved
imports, and a missing entry fail; no directory, manifest, or transitive source discovery occurs. The current command
permits target declarations such as unused `c::long` and automatically links exact proven
move-only resource transport over Fairy copy scalars or raw pointers, plus
SystemHeap allocate/deallocate, Buffer initialize/zeroed-resize/drop, Arena initialize/Region
begin/zeroed-allocation/reset/Arena-drop, CStr validation, String copy/view/append/drop, CString
from-text/view/drop, the five opaque String/CString storage bridges, and
`fy_std_text_utf8_valid_v0` project-owned components when called. Buffer,
Arena, and owned text pull in their declared dependencies once;
the Windows x64 SystemHeap ceiling is 16-byte fundamental alignment. Every other foreign symbol or
unproven standard-library component fails before toolchain discovery; foreign companion inputs and user
linker options remain absent. Its receipt is target-emission and one-way closed-FFI evidence, not
complete ABI/runtime conformance, zeroed Arena-byte or owned-text byte/terminator observation,
owned-text failure behavior, hosted support, a distributable
artifact promise, or self-hosting.
The checked-in `examples/owned_text/` pair remains independent evidence for those storage bridges.
ADR 0247 now exposes the same audited ownership and owner-bound view shape through compiler-bundled
`std::text`; it does not stabilize resource layout or a foreign owned-to-borrowed view ABI.
`experimental run-target` shares that exact target compilation and support-link contract, validates
the temporary PE and x64 unwind structure, executes it with bounded captured output, and removes it.
Success writes `native result: <i32> (x86_64-windows-msvc, msvc <version>, <profile>)`; a product
trap writes its canonical category and exits with the trap code. Any program output is currently a
build/execution failure because this command reserves a zero-output result-receipt contract. It retains no artifact
and does not use WSL.
`experimental exec-target` uses the same validated temporary target path but adds no result receipt:
program stdout and stderr stay on their matching streams and the program's 32-bit Windows status
becomes the CLI status. The shared 64 KiB capture ceiling remains fail-closed, and cross-stream order
is unspecified. The checked-in output example reaches only the unsafe internal Windows stdout
write-all symbol, but confines it to a checked local function accepting `ref str`. Its `main` passes
borrowed text and byte views only through safe internal calls and contains no raw pointer or
`unsafe`. The entry and helper are separate modules passed through one explicit `--source`; this
legacy target-only path is not the public package API. ADR 0205 separately adds compiler-bundled
`std::io::write` for manifest-owned package builds; `examples/std_io_quickstart` uses it without
unsafe code or a platform symbol. ADR 0207 makes stable `fairyc run` the ordinary package-aware
program-facing surface; the experimental command remains only the explicit-source target prototype.
`experimental target-spec` emits `fairyc.target-spec/v0` JSON without probing the host or exposing
local tool paths. Linux reports its current scalar, flat-resource, and resource-tagged emission
subset. Windows reports `scalar-and-llp64-c-scalar-c11`. Both keep `emittable`, `conformant`, and `supported` false and list
their own missing evidence.
The interpreter fixture's in-memory byte hooks require manifest options that this CLI never accepts;
declaring a similarly named Fairy C function does not grant output authority.
Infinite execution is stopped by the bounded interpreter and reported as an unsupported-capability
result rather than a language trap. It is separate from stable `fairyc check`, which never executes
the program.

`fairyc new <package-name> --uid <package-uid>` creates one direct child project and never touches
an existing file, directory, or symbolic link. The explicit manifest UID is not guessed from the
package name. The exact starter contains `.gitignore`, `README.md`, `fairy.toml`, `src/main.fy`, and
`tests/main.fy`; it omits `fairy.lock` so the first ordinary workspace command remains the owner of
graph validation and lock publication. The generated guide labels its commands as manual, preserves
stable run v2's stream and process-status boundary, and includes one checked safe `std::io::write`
source replacement plus the package-specific retained Windows executable path. Public `answer()`
returns 42 while private `main()` returns zero when that answer is correct. The command accepts no arbitrary destination, merge,
current-directory, force, template, prompt, stdin, network, or configuration surface. Ordinary
post-create failures attempt rollback; a crash can leave a partial directory that still requires
explicit user inspection and removal.

`fairyc check [--locked]` loads the current directory's strict `fairy.toml`, its explicit local
members, complete source closure, and compiler-bundled SDK package when imported. It runs the
package-aware frontend through verified MIR without requiring `main`, selecting a target, invoking a
backend or toolchain, starting a subprocess, using a cache, or accessing the network. Ordinary mode
creates or updates canonical `fairy.lock` only after a valid workspace exists; `--locked` requires an
exact existing lock and writes nothing. No positional source, target, profile, JSON, cache, or config
flag is accepted in v1.

`fairyc build [--release] [--locked]` uses the same workspace and lock boundary, requires root
`main::main` as `fn() -> i32`, and retains exactly `build/<package-name>.exe`. Default mode selects
the low-optimization development profile; `--release` selects ordinary optimization without LTO.
It accepts no source, target, output, compiler, linker, profile-value, cache, JSON, configuration,
network, or stdin input. A complete inspected image replaces the prior artifact only after successful
compilation; a failed build leaves the previous executable intact. Build v1 currently requires
Windows/amd64 and local x64 MSVC. That stable spelling does not promote target support.

`fairyc run [--release] [--locked]` uses the same workspace, entry, lock, profiles, and native
pipeline, but executes an inspected compiler-temporary image and retains no executable. V3 attaches
the caller's stdin only after successful compilation and inspection, without pre-reading, retaining,
or imposing a separate input byte ceiling. It forwards captured stdout and stderr to matching
streams with no success banner, then preserves every ordinary
`i32` return as the 32-bit process status. A run-only internal suffix keeps equal reserved normal
statuses distinct from recognized language traps; only that suffix is removed, preceding program
stderr is preserved, and the canonical trap line uses code 5. The two captured streams share a
64 KiB ceiling and retain no cross-stream ordering promise. Program arguments, a `--` delimiter,
retained build output, input forwarding for other native commands, and cache policy remain absent.

`fairyc test [--json] [--locations] [--native [--release] [--stdin] [-- <program-args>...]] [--workspace | --package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--exact <test-identity>]... [--locked]` overlays the root package's optional exact-lowercase `tests/` tree
after maintaining or verifying the production lock. `--package` instead selects exactly one
root-reachable local package by its manifest UID and overlays only that package. Paths keep their `tests/` prefix for module
identity, and argument-free `@test` selects private safe zero-argument `void` functions. Default
execution keeps the canonical fresh verified-MIR interpreter runtime. `--native` instead appends a
private reverified i32 harness for each test, compiles its reachable closure as one temporary
`x86_64-windows-msvc` PE, inspects PE/unwind structure, and executes it in a fresh process and
temporary directory; `--release` selects optimized MSVC. Repeatable `--exact` values select known
canonical test identities byte-for-byte and never reorder them; unknown and duplicate selectors
fail instead of broadening or silently selecting nothing. Traps are reported per test and do not
stop later selected tests. Repeatable `--file` selects exact portable workspace-relative test paths
from the same admitted catalog as `--locations`; member paths include their workspace prefix.
Unknown, duplicate, host-separated, or case-mismatched paths fail, while a known zero-test file
succeeds. Repeatable `--dir` recursively selects the descendants of exact admitted test-source
directories. The catalog includes only `tests`-root ancestors backed by admitted `.fy` sources;
it never reopens a user-supplied path. Empty filesystem directories are unknown, while a directory
containing admitted zero-test sources is valid. Matching is exact, case-sensitive, `/`-only, and
rejects alternate dot-segment or trailing-slash spellings. `--dir`, `--file`, and `--exact` are
mutually exclusive, and none hides complete-overlay compiler failures. Only native execution accepts
the first `--`; every later value is passed unchanged to each isolated test process, while semantic
execution and listing reject the delimiter and embedded NUL fails before workspace loading.
Native stdin remains EOF unless explicit `--stdin` is supplied. That flag is native-only, requires
exactly one selected test after complete compiler admission, and attaches caller input directly only
after compilation and PE inspection without pre-reading, buffering, replay, retention, or an extra
byte ceiling. Zero or multiple selections fail without consuming input or launching a process.
`fairyc test --list [--package <package-uid>] [--dir <portable-test-directory>]... [--file <portable-test-path>]... [--locked]` runs the same complete compiler admission,
then reports every canonical identity without invoking an execution engine or target toolchain;
listing `--json` selects the compact versioned `fairyc.test-list/v1` report, while execution
`--json` selects `fairyc.test-report/v1` with engine/target/profile identity, canonical outcomes,
totals, status, and lock. `--native`, `--release`, `--stdin`, and `--exact` do not combine with listing. Shared multi-test input, native output streams, catalog-unreachable member aggregation,
fuzzy/tag/glob selection, setup/teardown, benchmarks, and portable targets remain outside this contract.
`--locations` is valid in listing or execution and preserves canonical identity and selection
semantics. Text lines gain `at <workspace-relative-path>:<line>:<byte-column>` only when requested.
JSON switches to `fairyc.test-list/v3` or `fairyc.test-report/v3` and includes package UID, portable
path, exact function-name byte span, and exact `fyfile1`/`fylm1` identities. It includes no source
text, host root, timestamp, or reopened-path observation.

`fairyc emit c --target <target-id> [--locked]` accepts exactly the `c` form and one mandatory
`x86_64-linux-gnu` or `x86_64-windows-msvc` target. It loads the current manifest-owned workspace,
maintains or verifies the canonical lock, requires root `main::main fn() -> i32`, and writes exactly
one raw deterministic ISO C11 application translation unit to stdout. It emits no banner or
metadata, never infers a target from the host, creates no `.c` or build artifact, and invokes no
toolchain, SDK, WSL, subprocess, or network. The separate runtime and dependency-closed
standard-library support unit is not included. Whole-unit bytes and private names may change with
the compiler even though identical compiler/input/target requests are deterministic; public C ABI
contracts remain independently versioned. AST, HIR, MIR, JSON, source maps, headers, objects, and
executables are not stable emit forms.

`fairyc explain` accepts one exact uppercase `Edddd`, `Adddd`, or `Vdddd` code and writes its
reserved meaning, category, and catalog identity without reading source, configuration, or the
current directory. It performs no toolchain discovery, target probe, process execution, network
request, or write. The output explicitly marks the catalog provisional; a successful lookup does
not claim that the diagnostic is implemented, fixture-backed, or released. Listing, fuzzy
search, JSON, localization, and repair guidance remain unimplemented.

`fairyc targets` accepts no arguments and deterministically lists every declared TargetSpec with
its schema, admitted emission subset, independent `emittable`, `conformant`, and `supported` states,
and sorted missing evidence. It reads no source or configuration and performs no host, toolchain,
SDK, sysroot, WSL, process, network, or write operation. The declared `x86_64-windows-msvc`
TargetSpec is host-independent; only `build-target` later resolves a local toolchain after selecting
the declared Windows profile. It remains separate from the target-neutral Windows MSVC smoke.
The human block grammar, canonical ordering, atomic rendering, and codes 0, 2, and 4 are stable.
Target rows and evidence evolve under TargetSpec compatibility. JSON, filters, target selection, and
local-toolchain availability remain outside this command.

Human developers can run the examples shown in the repository `README.md`. Agent verification uses
the configured `fairyc_go_build`, `fairyc_go_vet`, and `fairyc_go_test` intents; the manual `go run`
examples do not grant agent command authority.
`DEVELOPMENT.md` also documents the manual WSL-free build of ignored `build/fairyc.exe` for repeated
Windows use. The configured Windows smoke independently builds and removes a temporary compiler
executable, verifies its help surface, drives the scalar quickstart through that separate process,
executes the checked-in isolated semantic test, native trap-isolation test, and both native MSVC profiles without changing the retained build artifact,
emits locked Windows and Linux C11 application units without changing the retained executable,
invokes the persistent cache command in nine fresh cold/warm process pairs and preserves raw durations,
and sends eight editor-shaped workspace checks through one stdin/stdout session while recording
Windows process-memory peaks; none of those paths is self-hosting, a distribution promise, a
reference-machine baseline, or a performance budget.

The stable human `targets` command distinguishes `emittable`, `conformant`, and `supported` without
turning the experimental single-target JSON into a stable machine contract. The planned
`x86_64-linux-gnu` profile cannot be promoted before its runtime, standard-library, ABI, toolchain,
and clean-CI fixtures pass without skips.

## Profile principle

- Stage 0 development and release both use the C reference backend with different explicit optimization profiles; `check` uses no backend.
- A later fast or production backend changes this profile only after a measured adoption decision.
- ThinLTO, PGO, full debug information, and cross-target builds remain explicit product requests.
  The current ASan+UBSan mode is validation-only and is deliberately not a CLI profile.

## Naming

The executable is `fairyc`. `fc` is not used. A `fyc` alias may be considered only after packaging and shell compatibility are specified.
