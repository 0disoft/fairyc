# Command Contract

- Status: Draft
- Repository type: cli-tool
- Runtime implementation: Go Stage 0

This document owns planned `fairyc` command semantics. It does not claim that a command is implemented. Implemented commands must be marked with tests and a language revision before release.

## Implemented diagnostic explanation surface

`fairyc explain <diagnostic-code>` is implemented as a provisional, source-independent lookup over
the reserved Stage 0 catalog. It accepts exactly one canonical uppercase `Edddd`, `Adddd`, or
`Vdddd` code. Success writes the exact catalog meaning, namespace category, and
`fairyc.diagnostic-catalog/v0 (provisional Stage 0)` to stdout. The command reads no source or
configuration, performs no directory scan, process execution, toolchain discovery, target probe,
network request, or write, and does not require the source root to exist.

Missing, extra, malformed, and unknown codes are invocation failures with prototype code `2` and no
stdout. An explanation reserves a prototype meaning; it does not claim that the rule is implemented,
fixture-backed, released, or stable. JSON output, listing, fuzzy search, localization, and repair
guidance remain unimplemented. Phase 3 still owns release stabilization of this command and the
broader CLI.

## Implemented target maturity surface

`fairyc targets` is implemented as a provisional, source-independent human report over the
validated values returned by `target.All()`. It accepts no arguments. For every target in canonical
ID order it writes the TargetSpec schema, admitted emission subset, independent `emittable`,
`conformant`, and `supported` yes/no states, and the sorted missing-evidence markers.

The command performs no source or configuration read, directory scan, executable lookup, toolchain,
SDK, sysroot, or WSL probe, process execution, network request, or write. It reports declared target
contracts rather than local host capabilities. The `x86_64-windows-msvc` entry therefore reports
LLP64 facts and missing evidence independently from the target-neutral MSVC host smoke. JSON,
filtering, target selection, and release-stable output remain unimplemented; Phase 3 still owns
stabilization.

## Implemented experimental compiler surface

The current Go prototype implements twelve deliberately unstable developer commands:

| Command | Purpose | Output |
| --- | --- | --- |
| `fairyc experimental parse <file.fy>` | Parse one source-root-relative file and verify lossless syntax plus error-free AST lowering | `fairyc.experimental.parse/v0` JSON on stdout, defined by `spec/cli/experimental-parse-v0.schema.json` |
| `fairyc experimental check-syntax <file.fy>` | Check lexing, parsing, recovery, module identity, lossless reconstruction, and error-free AST lowering | Success line on stdout or human diagnostics on stderr |
| `fairyc experimental check-semantics <file.fy> [<file.fy>...]` | Resolve names, type-check declared effects and direct-call supersets, and analyze local safe-borrow, whole-resource, Region, and complete Stage 0 Result must-use Flow for an explicit module set in the implemented subset | Typed-HIR module and function counts on stdout, human source diagnostics on stderr, or an explicit unsupported-subset error |
| `fairyc experimental target-spec <x86_64-linux-gnu\|x86_64-windows-msvc>` | Inspect one known host-independent target description without resolving a local toolchain | Deterministic `fairyc.target-spec/v0` JSON on stdout, defined by `spec/cli/experimental-target-spec-v0.schema.json` |
| `fairyc experimental emit-c <entry.fy> [--source <file.fy>]...` | Lower one explicit FY module closure in the admitted constrained-C subset through verified MIR and emit the provisional transport without invoking a toolchain | Exactly one deterministic C11 translation unit, with an explicit POSIX.1-2008 typedef profile only when required, on stdout; source-set diagnostics or capability errors on stderr |
| `fairyc experimental build <entry.fy> [--source <file.fy>]... --cc <clang\|gcc> --profile <low\|optimized> --output <build/path>` | Compile and link one explicitly closed verified program in that constrained subset for the host `x86_64-linux-gnu` profile with an explicit C compiler, optimization profile, and ignored output path | A no-overwrite executable below `build/` plus one relative-path receipt, or source-set, capability, toolchain, link, or artifact diagnostics |
| `fairyc experimental build-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low\|optimized> --output <build/path.exe>` | Emit one explicitly closed FY module set through the provisional Windows MSVC C profile, automatically link exact proven SystemHeap, Buffer, Arena, owned-text, CStr, and UTF-8 validation dependency closures when called, resolve local x64 MSVC, and retain one target-labelled PE | A no-overwrite `.exe` below `build/` plus one `x86_64-windows-msvc` receipt, or source-set, target-subset, support-link, toolchain, compile, link, or artifact diagnostics |
| `fairyc experimental run-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low\|optimized>` | Compile the same explicitly closed Windows target module set, validate the temporary PE and Microsoft x64 unwind structure, execute it with bounded captured output, and remove it | `native result: <i32> (x86_64-windows-msvc, msvc <version>, <profile>)`, a canonical trap, or source-set, target-subset, support-link, toolchain, compile, link, structure, or execution diagnostics |
| `fairyc experimental exec-target <entry.fy> [--source <file.fy>]... --target x86_64-windows-msvc --profile <low\|optimized>` | Compile the same explicitly closed Windows target module set, execute it with separated bounded streams, and behave as the program process | Exact program stdout and stderr plus the program's unsigned-preserved 32-bit Windows exit status; compiler or execution failures still use CLI diagnostic codes |
| `fairyc experimental build-host <entry.fy> [--source <file.fy>]... --profile <low\|optimized> --output <build/path.exe>` | On Windows/amd64, compile one explicit target-neutral FY module closure with locally discovered x64 MSVC and retain its PE without declaring a Windows target ABI | A no-overwrite `.exe` below `build/` plus a host-smoke receipt, or source-set, capability, toolchain, compile, or artifact diagnostics |
| `fairyc experimental run-host <entry.fy> [--source <file.fy>]... --profile <low\|optimized>` | On Windows/amd64, compile and execute one explicit target-neutral FY module closure with locally discovered x64 MSVC without declaring a Windows target ABI | `native result: <i32> (windows/amd64 host-smoke, msvc <version>, <profile>)`, a canonical trap, or source-set, capability, toolchain, compile, or execution diagnostics |
| `fairyc experimental run <entry.fy> [--source <file.fy>]...` | Run one explicitly closed verified-MIR module set in the scalar, tagged, whole-resource, and implemented Region subset whose entry module declares `main() -> i32` | `result: <i32>` on stdout, a canonical trap category on stderr, source-set diagnostics, or an explicit unsupported-subset error |

These commands treat the current directory as their temporary source root. `parse` and
`check-syntax` accept exactly one relative regular file; `check-semantics` accepts one or more
explicit relative regular files under bounded file-count and aggregate-byte budgets. All reject root
escape including escaping symbolic links and read no more than the current implementation budget.
Their spelling and numeric exit codes are prototype-only.
Every program command accepts one positional entry plus repeatable `--source` values under the same
64-file and aggregate-byte ceilings. They analyze only that explicit FY set, require
source-root-relative module identity, and perform no source discovery. The flag never denotes a C,
object, archive, library, or linker input.
The JSON report includes relative path, byte length, module identity, token kinds, CST node kinds,
half-open byte spans, child indexes, and structured diagnostics. It excludes source text, token
text, cooked literal values, and absolute local paths. When bounded loading detects more bytes than
the implementation limit, `truncated` is true and `byteLength` is a lower bound rather than a
claimed full-file size.

`experimental target-spec` accepts exactly one known target ID and performs no source loading,
directory scan, executable lookup, SDK discovery, sysroot probing, or network request. Its JSON
contains target-semantic identity, layouts and classifier identities plus explicit maturity
evidence. It contains no local tool path. Unknown, missing, or extra IDs are invocation failures
and emit no partial JSON. The reported constrained-C emission subset does not make the target
`emittable`, `conformant`, or `supported`.

`check-semantics` never scans directories or follows an import to discover another source file. Each
imported module must be included explicitly. Duplicate paths and input sets beyond the prototype
budgets are rejected before semantic analysis. Syntax outside the implemented semantic subset
returns a capability error and never counts as a successful check.
Resource declarations preserve ordered resolved fields in typed HIR. A pure nonempty user `drop`
body becomes a compiler-private typed function linked from its resource; an effect-bearing body
remains rejected until resource drop effects propagate through implicit cleanup.

`experimental run` does not invoke a backend, linker, runtime, or host subprocess. It runs one
positional entry plus zero or more repeated `--source` files through parser, semantic analysis, Flow,
MIR lowering, the MIR verifier, and the bounded MIR interpreter. The closure uses the same 64-file,
per-file, aggregate-byte, duplicate, containment, and no-discovery rules as the Windows target
commands. The entry is the positional file module's zero-argument `main() -> i32`; boolean
`if`, `!`, comparisons, source-level `&&` and `||` short-circuiting, scalar `while`, scalar tagged
values, resource construction and field reads, whole-root and tagged-payload moves, resource-bearing
built-in `Option` and `Result` values, ordered resource reassignment, consuming-close ownership,
nearest-loop `break` and `continue`, lexical Region placement and reset on fallthrough, direct
return, Result propagation, nested, break, and continue exits from a direct `&mut` Arena local,
cleanup drop events, checked `i32` negation, addition,
subtraction, multiplication, division, remainder, matching-type bitwise operations, and signed or
unsigned right shift are
executable. The bounded interpreter reports its stable `step-limit` capability rather than hanging
or reclassifying it as a language trap. Interpreter-fixture byte hooks are not available from this
command and matching function names do not activate them. User drop-body effects, I/O, and arguments
remain outside the current interpreter subset. Matching-type left shift reaches the interpreter
only through verified width and representability checks and reports `invalid-shift` before
`integer-overflow` when both conditions could apply.

`experimental emit-c` shares the same explicit source closure, semantics, Flow, MIR-lowering,
verifier, and zero-argument entry-module `main() -> i32` path, then invokes only the constrained C emitter.
It writes raw C11 to stdout with no banner or metadata and writes no repository file. It does not
parse the generated C, invoke Clang or GCC, link the test runtime, execute a program, or establish an
`emittable`, `conformant`, or `supported` target. `Option<i32>` and `Result<i32, i32>` use private
compact tagged structs. Other built-in tags whose payloads are canonical Fairy copy scalars or
admitted inert structs use target-laid-out private numeric structs. Direct nonempty all-`i32`
resources over Fairy copy scalars or raw pointers with empty or pure nonempty drop bodies and built-in
resource-bearing tags whose payloads are exactly `i32` or direct admitted resources use private
numeric structs whose size, alignment, and field offsets are asserted against target-layout results.
Restricted user enums use private numeric tags and private union payload storage. Payloadless and
named-field variants, exact exhaustive shallow matching, and admitted scalar/raw-pointer/flat-struct
payload fields are supported; nested user enums and built-in tags inside their payloads fail closed.
Nonempty source `@repr(C)` structs over `u8`, `i32`, `u32`, `usize`, exact C scalars, admitted raw
pointers, and acyclic nested `@repr(C)` records preserve their public C tag and field names, emit
dependencies before users, and receive target-computed layout assertions. The same records may be
foreign results by value; foreign aggregate parameters remain rejected. An exact direct-
local `&T -> *T`, `&mut T -> *T`, or `&mut T -> *mut T` cast inside `unsafe` lowers to verified
raw-address MIR and generated `&local`; the exact `&mut *mut void` output slot is included. Nested
tags, empty or non-`i32` resources, partial
initialization, user drop effects, Region, retained safe borrows, raw dereference or arithmetic,
arbitrary casts, by-value aggregate parameters, complete runtime, and general standard-library lowering fail with the provisional capability
code. The closed native fixture `drop-trace-v0` profile is not selectable through the CLI and its
observer declaration, calls, site metadata, and stderr protocol never appear in this output. The
native runner's `address-undefined-o1` sanitizer mode is likewise validation-only and cannot be
selected by `emit-c` or `build`.

`experimental build` consumes that same C unit, binds the embedded scalar runtime plus the minimal
dependency-closed provisional standard-library C source selected by actual called link names, and uses
the shared constrained-C driver. It requires Linux/amd64 with glibc and the explicitly selected
`clang` or `gcc` from `PATH`; `low` fixes `-O0` and `optimized` fixes `-O2`. There is no compiler,
profile, target, or output default. The output must be a canonical forward-slash path below the
ignored source-root `build/` directory. Absolute, drive-relative, UNC, traversal, mixed-separator,
overlong, platform-reserved, and escaping symbolic-link paths fail closed. The command stages and
flushes the executable under the destination directory, promotes it without replacing an existing
path, and removes ordinary staging residue. It emits no `.c` file, object, cache entry, manifest,
package state, or network request. The provisional executable is not a supported target artifact or
a stable `fairyc build` result. It is uninstrumented and does not link the native differential
drop recorder or sanitizer runtimes.

`experimental run-host` and `experimental build-host` are separate Windows/amd64 developer smokes,
not target selections. They discover x64 MSVC and a complete Windows SDK locally and compile with C11
warnings-as-errors plus a static C runtime. `run-host` executes and removes a temporary PE;
`build-host` requires a canonical `.exe` path below the ignored source-root `build/` directory and
uses the same no-overwrite staging and promotion boundary as the Linux experimental build. Both
accept only explicit FY source inputs and reject foreign symbols, provisional standard-library
linkage, `@repr(C)`, exact `c::*` scalar use,
and foreign calls or results so this target-neutral lane cannot silently borrow the distinct
`x86_64-windows-msvc` target profile, its provisional standard-library slice, or its partial
layout/ABI evidence. Neither
uses WSL or inherits a Developer Command Prompt, and a retained host-smoke PE is
not a supported distribution artifact or Windows conformance evidence.

`experimental build-target` is the first command that consumes the declared Windows TargetSpec
instead of borrowing the Linux emission profile. The current closed subset admits ordinary scalar
programs and target C-scalar declarations that the Windows profile actually provides. Emitted
`c::long` assertions require the four-byte LLP64 layout while `c::ssize_t` fails as unavailable
before C is emitted. Exact `fy_std_system_heap_allocate_v0`,
`fy_std_system_heap_deallocate_v0`, `fy_std_buffer_initialize_v0`,
`fy_std_buffer_resize_zeroed_v0`, `fy_std_buffer_drop_v0`,
`fy_std_arena_initialize_v0`, `fy_std_region_begin_v0`,
`fy_std_region_allocate_zeroed_v0`, `fy_std_region_reset_v0`, `fy_std_arena_drop_v0`, and
`fy_std_cstr_validate_v0`, `fy_std_string_copy_v0`, `fy_std_string_as_view_v0`,
`fy_std_string_append_v0`, `fy_std_string_drop_v0`, `fy_std_cstring_from_text_v0`,
`fy_std_cstring_as_view_v0`, `fy_std_cstring_drop_v0`, and `fy_std_text_utf8_valid_v0` calls select
project-owned SystemHeap/Buffer/Arena/text-view/owned-text C components and scalar runtime. Buffer,
Arena, and owned text close over their declared dependencies and every component is emitted once in
deterministic order. Windows x64 SystemHeap admits fundamental
alignment through 16 bytes. Every other foreign symbol and unproven standard-library component
fails before toolchain discovery. One positional entry and zero or more repeated `--source` FY files
form an explicit closure of at most 64 files under the current source root and existing aggregate
byte limit. Duplicate normalized paths, source-root escape, module/path drift, unresolved imports,
and a missing entry fail before emission; no source discovery occurs. Companion objects, user linker inputs, debug/PDB output
contracts, and distribution packaging remain closed. The command accepts only the exact Windows
target ID, low or optimized profile, and a canonical no-overwrite `.exe` path under `build/`; it
does not use WSL. The checked-in target example observes the Arena generation/active/pointer
lifecycle but cannot dereference the returned storage, so it does not independently prove zeroed
payload bytes, copied placement, failure handling, or safe lexical Region lowering. It also executes
String and CString success lifecycles plus direct CStr validation, but observes only statuses,
lengths, alignment, pointers, and normalized drops; returned bytes, terminator contents, rollback,
and safe ownership remain outside the command contract.

`experimental run-target` accepts the same explicit FY source closure but no output path, foreign companion input, linker option, ambient target,
or compiler override. It shares `build-target` lowering, exact standard-library resolution, local
MSVC discovery, object inspection, PE inspection, and unwind inspection before execution. The
temporary executable must produce no stdout or stderr; ordinary results preserve signed `i32` exit
bits, while reserved product-trap statuses map to canonical categories and exit code `5`. Temporary
files are removed by the driver. This is direct target execution, not target support, installation,
distribution, or self-hosting.

`experimental exec-target` accepts the same required source, target, and profile values and the same
closed compiler/linker inputs, but it is the program-facing sibling rather than an inspection
receipt. It forwards captured stdout and stderr to their matching CLI streams without adding a
banner, preserves the program's full 32-bit Windows process status as its own status, and emits no
native-result or trap receipt. The two streams share one 64 KiB capture limit and have no declared
cross-stream ordering. Toolchain, compilation, structure, timeout, capture-limit, and CLI output-
write failures remain compiler failures rather than program statuses. The current source-callable
slice exposes only unsafe `fy_std_windows_stdout_write_all_v0`; stdin, stderr writes, arguments,
Unicode console policy, and a safe public `std::io` wrapper remain absent.
The checked-in output example confines that raw call to a safe local `&str` helper, supplies the
length through the view's `usize` byte-count `.len`, and verifies status plus completed progress.
The raw declaration, wrapper, and `main` propagate `@effects(console)`; `unsafe` remains an
independent support-module boundary. `main` contains no raw pointer, foreign call, or `unsafe`, and a safe `&[u8]` helper separately
proves byte-view internal-call transport.

For the admitted foreign-call slice, `emit-c` writes a plain external prototype and call using the
declaration name by default or its single validated `@link_name` string. Compatible mappings to one
external symbol coalesce and conflicting signatures fail closed.
The exact C-scalar slices emit C `int`, LP64-sensitive `long`, `<stddef.h>` `size_t`, or target-specific POSIX.1-2008
`<sys/types.h>` `ssize_t`, target assertions, and a typed local between foreign producer and
consumer calls. `ssize_t` additionally fixes `_POSIX_C_SOURCE 200809L` before headers and preserves
negative values. They do not accept literals or operators, do not
accept raw pointers to C scalars, and do not alias the types to coincidentally sized Fairy integers.
The exact internal SystemHeap, Buffer, Arena/Region, text-view, CStr, String, CString, and Linux File
open/read/write-all/sync/close symbols
exercised by native source probes form a closed project-owned allowlist; the `fy_` prefix does not
become generally callable. The backend records sorted unique link names used by call terminators;
declarations without calls select no library component. `build` resolves the exact project-owned
subset through one shared symbol/component contract and links only its dependency closure. It still
supplies no user companion source, object, library, search path, or arbitrary linker argument, so
every other foreign symbol must resolve from the selected driver's ordinary closed environment or
the build fails. `experimental run` never binds host symbols and reports the interpreter capability
boundary.

`fairyc check` remains unimplemented. It must not be aliased to `check-syntax` or the current
effect-aware `check-semantics`, because the planned command below still requires later semantic
groups beyond the current Flow prototype.

## Stage 0 surface

| Command | Purpose | Code generation |
| --- | --- | --- |
| `fairyc check [target]` | Parse, resolve, type-check, and run resource, borrow, region, and effect analysis | No |
| `fairyc build [target]` | Compile and link the selected target with the configured backend | Yes |
| `fairyc run [target] -- [args...]` | Build and run a program while preserving the program exit status | Yes |
| `fairyc test [target]` | Build and run semantic test targets | Yes |
| `fairyc emit <ast|hir|mir|c> [target]` | Emit a requested compiler representation for inspection | Requested form only |
| `fairyc explain <diagnostic-code>` | Explain a stable diagnostic code | No |
| `fairyc targets` | Report known target profiles, status, evidence, and unavailable requirements | No |

Stable positional syntax, package selection, workspace selection, target triples, feature flags,
and configuration precedence remain UNDECIDED. The implemented `experimental` spellings do not
settle those stable decisions.

## General requirements

- Commands are non-interactive by default unless an explicit future subcommand states otherwise.
- Input, output, and cache paths are resolved under documented roots and reject unsafe traversal.
- Human diagnostics go to stderr; requested artifacts and machine-readable results go to stdout or an explicit output path.
- `--json` output uses a versioned schema and contains no ANSI escapes.
- Machine-readable build and target receipts identify the applicable compiler, language, MIR, target-schema, ABI-profile, runtime, standard-library, diagnostic, package, and Atlas compatibility contracts rather than collapsing them into one compiler version.
- Deterministic commands preserve ordering across identical inputs.
- `check` must not invoke a backend or linker.
- `emit` must not silently choose a different semantic profile from `build`.
- `run` separates compiler arguments from program arguments with `--`.
- Cache inspection and cleanup are distinct from project output cleanup.
- `targets` reports `emittable`, `conformant`, and `supported` independently and cannot promote a profile while a required fixture is skipped or absent.

## Planned profile flags

The stable spelling is UNDECIDED, but the semantic profiles are fixed:

- Stage 0 development: C reference backend, low optimization, bounded reuse, and requested debug information;
- Stage 0 release: C reference backend, ordinary optimization, stripped artifact, and no default LTO;
- later development and production: backend choice is enabled only by a separate measured contract;
- release LTO: explicit ThinLTO request;
- profiling: explicit instrumentation generation and profile use;
- unsafe optimization: source-local unsafe or unchecked operations, never a project-wide silent semantic switch.

## Exit categories

Numeric codes remain UNDECIDED. The CLI must distinguish success, invalid invocation or configuration, source diagnostics, backend or linker failure, toolchain or target unavailability, and executed-program exit status.

## Review blockers

- Help, documentation, JSON schema, examples, tests, and exit behavior disagree.
- A normal development command enables expensive global optimization.
- A build option changes safe-language semantics.
- Output embeds absolute local paths without an explicit diagnostic or debug contract.
- A machine-readable result includes source contents, secrets, or unbounded compiler logs by default.
