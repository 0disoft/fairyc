# Command Contract

- Status: Draft
- Repository type: cli-tool
- Runtime implementation: Go Stage 0

This document owns planned `fairyc` command semantics. It does not claim that a command is implemented. Implemented commands must be marked with tests and a language revision before release.

## Stable starter project creation v4

`fairyc new <package-name> --uid <package-uid>` creates one starter project as a direct child of the
current process source root. The name must be one portable lowercase kebab-case manifest name and
directory segment. The UID is mandatory manifest-v1 identity with at least two lowercase
dot-separated segments; it is never synthesized from the movable name.

The target must not already exist as a file, directory, or symbolic link. The command accepts no
destination path, current-directory mode, template, version, language, force, merge, target,
  toolchain, config, environment, network, stdin, or prompt surface. It writes exactly `.gitignore`,
  `README.md`, `fairy.toml`, `src/main.fy`, and `tests/main.fy` under the new directory. The manifest
  fixes version `0.1.0`, language `fairyc-language/8`, and source root `src`; public `answer()` returns
  42, private `main()` returns `answer() - 42`, and the semantic test imports and checks `answer`.
  `.gitignore` excludes only `/build/`.

The generated README is exact compiler-owned starter content, not a selectable template. It labels
its commands as manual steps, explains first-check lock publication and stable run v2's stream and
process-status behavior, distinguishes fast semantic testing from explicit isolated Windows native
testing, substitutes the validated package name into the retained Windows executable path, and
includes one complete safe `std::io::write` source replacement. The source handles both
Result variants and returns zero only after the complete message was written. It directs the user to
run an ordinary check before locked release execution because changing source makes the old lock stale.

No `fairy.lock` is generated: the first ordinary workspace command owns complete graph validation
and lock publication. Files are exclusive creates under an opened root and synced before success.
An ordinary post-create failure attempts contained rollback. A process or host crash may leave a
partial directory, which a later invocation still refuses to merge or overwrite.

Success is exactly `created Fairy C package "<name>" at <name>`. Codes are 0 for success, 2 for
invocation or identity rejection, 4 for internal/output failure, and 7 for parent, target,
permission, filesystem, or rollback failure. Output failure occurs after creation and cannot undo
the project. No absolute parent path or raw operating-system error is part of stable output.

## Stable diagnostic explanation v1

`fairyc explain <diagnostic-code>` is a stable, source-independent lookup over
the reserved Stage 0 catalog. It accepts exactly one canonical uppercase `Edddd`, `Adddd`, or
`Vdddd` code. Success writes the exact catalog meaning, namespace category, and
`fairyc.diagnostic-catalog/v0 (provisional Stage 0)` to stdout. The command reads no source or
configuration, performs no directory scan, process execution, toolchain discovery, target probe,
network request, or write, and does not require the source root to exist.

Missing, extra, malformed, and unknown codes are invocation failures with stable code `2` and no
stdout. An explanation reserves a prototype meaning; it does not claim that the rule is implemented,
fixture-backed, or released. The exact three-line shape, code/category/catalog identities, and codes
0, 2, and 4 are stable; human meaning prose may improve without reassigning a code. JSON output,
listing, fuzzy search, localization, and repair guidance remain unimplemented.

## Stable target maturity report v1

`fairyc targets` is a stable, source-independent human report over the
validated values returned by `target.All()`. It accepts no arguments. For every target in canonical
ID order it writes the TargetSpec schema, admitted emission subset, independent `emittable`,
`conformant`, and `supported` yes/no states, and the sorted missing-evidence markers.

The command performs no source or configuration read, directory scan, executable lookup, toolchain,
SDK, sysroot, or WSL probe, process execution, network request, or write. It reports declared target
contracts rather than local host capabilities. The `x86_64-windows-msvc` entry therefore reports
LLP64 facts and missing evidence independently from the target-neutral MSVC host smoke. The block
grammar, canonical ordering, atomic output, final newline, and codes 0, 2, and 4 are stable. Target
rows and maturity evidence evolve under TargetSpec compatibility. JSON, filtering, target selection,
and local-toolchain availability remain outside v1.

## Stable workspace C emission v1

`fairyc emit c --target <target-id> [--locked]` emits one target-selected application translation
unit from the current manifest-owned workspace. `c` is the only stable form and `--target` is
mandatory; the accepted current IDs are `x86_64-linux-gnu` and `x86_64-windows-msvc`. Host target
inference, default targets, AST/HIR/MIR output, JSON, source maps, headers, objects, and executables
are outside v1.

The command uses the stable workspace/lock boundary and requires root `main::main fn() -> i32`.
Ordinary mode atomically refreshes a missing or stale canonical `fairy.lock`; `--locked` verifies it
without writing. Success writes exactly one deterministic, ANSI-free ISO C11 application unit to
stdout with no banner, receipt, JSON envelope, absolute path, or FY source text. No partial unit is
written for compilation or emission failures. An external stdout writer may accept a prefix before
returning an output error; that transport failure is code 4 and the byte stream cannot be rolled
back.

The output is the target-selected application unit consumed by the backend, but excludes the
separate runtime and dependency-closed standard-library support unit. It may contain external
references and is not a standalone build artifact. The command writes no source, header, object,
executable, cache, or build directory and performs no compiler, linker, SDK, sysroot, WSL, process,
or network probe. Whole-unit bytes and private symbols are deterministic for one exact compiler,
input, and target but are disposable transport rather than a cross-release C API.

Stable codes are 0 for success, 2 for invocation, entry, target, or backend-capability rejection, 3
for source diagnostics, 4 for internal or output failure, and 7 for manifest, workspace, graph, or
lock failure. An output failure may follow ordinary lock maintenance; `--locked` is the no-write
form.

## Implemented experimental compiler surface

The current Go prototype implements twenty-three deliberately unstable developer commands:

| Command | Purpose | Output |
| --- | --- | --- |
| `fairyc experimental parse <file.fy>` | Parse one source-root-relative file and verify lossless syntax plus error-free AST lowering | `fairyc.experimental.parse/v0` JSON on stdout, defined by `spec/cli/experimental-parse-v0.schema.json` |
| `fairyc experimental check-syntax <file.fy>` | Check lexing, parsing, recovery, module identity, lossless reconstruction, and error-free AST lowering | Success line on stdout or human diagnostics on stderr |
| `fairyc experimental check-semantics <file.fy> [<file.fy>...]` | Resolve names, type-check declared effects and direct-call supersets, and analyze local safe-borrow, whole-resource, Region, and complete Stage 0 Result must-use Flow for an explicit module set in the implemented subset | Typed-HIR module and function counts on stdout, human source diagnostics on stderr, or an explicit unsupported-subset error |
| `fairyc experimental check <primary.fy> [--source <file.fy>]...` | Run syntax, semantic, Flow, MIR-lowering, and MIR-verifier stages over one explicit source closure without requiring an executable entry | One verified-MIR function and foreign-declaration count line, source diagnostics, or an explicit unsupported-subset error |
| `fairyc experimental check-workspace [--locked]` | Load the current directory's strict `fairy.toml`, explicit local members and sources, validate or refresh canonical `fairy.lock`, then run package-aware semantic and verified MIR | Deterministic instance/source/MIR counts and lock state, diagnostics, or prototype package failure code `7` |
| `fairyc experimental atlas-pack overview --package-uid <uid> --max-records <count> --max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace, select one exact unique package UID, and run the bounded current Atlas overview query | Exact canonical `fairyc.context-pack-json/v1` bytes with no trailing newline, free text, source body, host path, map-store write, or hidden numeric limit |
| `fairyc experimental atlas-pack show --package-uid <uid> --symbol <symbol-id> --direction <outgoing\|incoming\|both> --relation <kind> [--relation <kind>]... --max-records <count> --max-relations <count> --max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace, select one exact package and visible/full SymbolId, then retrieve bounded subject facts and filtered relations | Exact canonical `fairyc.context-pack-json/v1` bytes with the same no-newline, compiler-fact-only, no-store, and explicit-limit boundary |
| `fairyc experimental atlas-pack around --package-uid <uid> --symbol <symbol-id> --direction <outgoing\|incoming\|both> --relation <kind> [--relation <kind>]... --max-records <count> --max-relations <count> --max-depth <depth> --max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Run deterministic package-local breadth-first traversal from one exact visible/full SymbolId under explicit direction, relation, depth, and size policy | Exact canonical `fairyc.context-pack-json/v1` bytes with explicit incompleteness and no inferred names, dependency joins, free text, or hidden limit |
| `fairyc experimental atlas-symbols list --package-uid <uid> --kind <kind> [--kind <kind>]... [--module <module>] [--name <identifier>] [--after <full-symbol-id>] --max-symbols <count> --max-selection-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace and page exact current declarations in full-SymbolId order with optional exact module/name filters | One canonical checksummed `fairyc.symbol-catalog-json/v1` value with no newline, source body, fuzzy rank, guessed unique match, map-store path, or hidden limit |
| `fairyc experimental atlas-calls list --package-uid <uid> --symbol <symbol-id> --direction <outgoing\|incoming\|both> --max-packages <count> --max-calls <count> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace once, atomically join every exact-current direct call under explicit construction limits, then select one exact package-qualified symbol and direction | One canonical checksummed `fairyc.workspace-call-catalog-json/v0` value with no newline, source body, host path, hidden limit, partial join, traversal, or impact claim |
| `fairyc experimental atlas-text source --package-uid <uid> --symbol <symbol-id> --origin workspace-authored --max-fragments <count> --max-fragment-bytes <bytes> --max-text-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace, resolve one exact current visible/full SymbolId, and select its workspace-authored declaration bytes under explicit count and byte policy | One canonical checksummed `fairyc.attributed-text-json/v0` value with no newline or host path; a text-budget exclusion is an explicit successful omission, while final-output overflow is atomic failure |
| `fairyc experimental atlas-text documentation --package-uid <uid> --symbol <symbol-id> --origin workspace-authored --max-fragments <count> --max-fragment-bytes <bytes> --max-text-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the same workspace, resolve one exact current declaration plus its immutable frontend attachment sidecar, and select only exact marker-bearing `///` bytes | The same canonical attributed-text envelope with `declaration-documentation`, or a successful checksummed `kind-unavailable`, safety, or text-budget omission; no source scan, declaration body, map-format change, or host path |
| `fairyc experimental atlas-evidence show --package-uid <uid> --symbol <symbol-id> --direction <outgoing\|incoming\|both> --relation <kind> [--relation <kind>]... --max-records <count> --max-relations <count> --max-selection-bytes <bytes> --max-fragment-bytes <bytes> --max-text-bytes <bytes> --max-facts-bytes <bytes> --max-text-document-bytes <bytes> --max-output-bytes <bytes> [--locked]` | Compile the manifest-owned workspace once and compose exact-symbol compiler facts, declaration source, and attached documentation from the same retained exact-current state | One canonical checksummed `fairyc.atlas-evidence-json/v0` value whose unchanged independently checksummed children retain `compiler-fact` versus `untrusted-text` authority and explicit component/final bounds |
| `fairyc experimental check-workspace-session --target <target-id> --max-requests <count> --cache-max-entries <count> --cache-max-bytes <bytes> --cache-max-artifact-bytes <bytes> [--locked]` | Keep one bounded process-local verified-MIR Session while a caller submits sequential versioned NDJSON check requests on stdin | One `fairyc.experimental.workspace-check-session/v0` JSON response per completed request, human diagnostics on stderr, and the last completed request's prototype exit code |
| `fairyc experimental check-workspace-cache --target <target-id> --cache-dir <build/path> --cache-max-entries <count> --cache-max-bytes <bytes> --cache-max-record-bytes <bytes> --cache-max-artifact-bytes <bytes> --cache-temp-max-age-seconds <seconds> [--locked]` | Run one package-aware semantic and verified-MIR check with explicit bounded cross-process stable disk reuse keyed by the exact executable image | One `fairyc.experimental.workspace-cache-check/v0` JSON object, human diagnostics on stderr, or prototype invocation/package/internal failure |
| `fairyc experimental build-workspace --target x86_64-windows-msvc --profile <low\|optimized> --output <build/path.exe> [--locked]` | Compile the current local-exact workspace's root `main::main` through package-aware verified MIR and the closed Windows target pipeline | A no-overwrite `.exe`, target/toolchain/profile/lock receipt, or package, source, capability, toolchain, link, and artifact diagnostics |
| `fairyc experimental run-workspace --target x86_64-windows-msvc --profile <low\|optimized> [--locked]` | Compile and execute the same package-aware root entry as a temporary inspected Windows PE | `native workspace result: <i32>` with target/toolchain/profile/lock receipt, a canonical trap, or package/build diagnostics |
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
Explicit source-closure program commands accept one positional entry plus repeatable `--source`
values under the same 64-file and aggregate-byte ceilings. They analyze only that explicit FY set,
require source-root-relative module identity, and perform no source discovery. The flag never
denotes a C, object, archive, library, or linker input. Workspace commands instead accept no source
arguments and use the bounded manifest-owned source closure described below.

`experimental check-workspace` accepts only optional `--locked`. It loads only the current source
root, explicit members, and their complete sources. Ordinary mode replaces `fairy.lock` only when
missing, invalid, or byte-different; `--locked` requires an exact match and never writes. It selects
the exact compiler-bundled SDK only for actual `std::*` imports but selects no target, backend,
toolchain, persistent cache, or executable.

`experimental atlas-pack overview` uses the same manifest-owned workspace and lock boundary, then
selects exactly one package by its unique manifest-v1 UID. Package UID, maximum records, query
selection bytes, and final output bytes are all mandatory; only `--locked` is optional. Canonical
positive decimals are capped at 65,536 records and 1,073,741,824 bytes for each byte class.
Overview fixes its unused relation and depth budgets to zero. The command hashes the running
compiler image, produces current package maps, and writes exactly one canonical compiler-fact
Context Pack to stdout without a newline. It writes no map store, source, artifact, cache, or default
path and starts no toolchain, process, watcher, daemon, network request, or model call. Free text and
pack decoding remain unavailable.

`experimental atlas-pack show` and `experimental atlas-pack around` reuse that exact workspace,
compiler-identity, current-partition, package-UID, output, lock, and failure boundary. Both require a
canonical visible or full SymbolId, one explicit outgoing, incoming, or bidirectional direction, one
or more implemented relation kinds, positive record/relation/selection/output limits, and no hidden
defaults. Relation filters reject duplicates and are canonicalized. Counts are capped at 65,536 and
byte classes at 1,073,741,824. `show` fixes depth to zero and rejects `--max-depth`; `around` requires
a canonical decimal depth from zero through 64. Full IDs are accepted for collision-checking lookup,
while the canonical pack records the matched visible ID. These package-local commands retain filter,
dependency-map, incoming-package-map, and caller-budget omissions. They perform no name or fuzzy
search, dependency-map join, workspace-wide reverse indexing, source-text import, or safe-change
inference. Callers obtain exact IDs separately through `experimental atlas-symbols list`.

`experimental atlas-symbols list` shares that exact workspace/compiler/currentness preparation. It
requires one or more exact declaration kinds and explicit symbol, selection-byte, and final-output-
byte limits. Optional module matching is exact. Optional `--name` accepts one ASCII Stage 0
identifier of at most 512 bytes and matches declaration names byte-for-byte and case-sensitively.
Same-name declarations in different modules or kinds remain separate records; zero matches are a
successful complete empty catalog. Continuation accepts only a canonical full SymbolId returned by
the same module/name/kind-filtered current catalog. The command writes no `.fymap`, cache, or catalog
file and performs no normalization, prefix, substring, or fuzzy search. Ordinary mode may refresh
`fairy.lock` before a later filter or budget failure; locked mode is exact and no-write.

`experimental atlas-calls list` prepares that workspace once but consumes the separate exact-current
`WorkspaceCallIndex`, not the package-local Context Pack query. Package UID, canonical visible or
full SymbolId, outgoing/incoming/both direction, positive package and call construction limits, and
a positive final-output limit are mandatory; only `--locked` is optional. Counts are capped at
65,536 and final bytes at 1,073,741,824. The complete supplied package set must close every direct
dependency and fit both construction limits before selection; exhaustion is code `2` with no partial
catalog. A valid subject with no selected calls succeeds with `calls: []` and `complete: true`.
Output is one canonical no-newline `fairyc.workspace-call-catalog-json/v0` envelope containing only
package-qualified symbols, exact relative span identity, compiler certainty/provenance, construction
budget and usage, selected calls, and an integrity checksum. The command writes no map store,
catalog file, source, cache, artifact, or default path and performs no traversal, indirect-call
recovery, other relation join, impact analysis, name search, toolchain, process, network request, or
model call. Ordinary mode may refresh the lock before a later join, selector, or output failure;
locked mode is exact and no-write.

`experimental atlas-text source` and `documentation` share the exact workspace/compiler/currentness
preparation but emit the separate attributed-text artifact, never a Context Pack. Package UID,
canonical visible or full SymbolId, exact origin `workspace-authored`, maximum fragment count,
per-fragment bytes, aggregate text bytes, and final output bytes are mandatory; only `--locked` is
optional. There are no inferred defaults, name search, SDK-text admission, or redundant kind flag:
action `source` fixes `source-excerpt`, while `documentation` fixes
`declaration-documentation`. Count is capped at 4,096, fragment bytes at 8,388,608, aggregate
text bytes at 67,108,864, and final output bytes at 1,073,741,824; fragment bytes cannot exceed
aggregate text bytes. A selected fragment that exceeds a text budget produces a successful
checksummed `fragment-byte-limit` omission with no text. A declaration without attached
documentation similarly produces successful `kind-unavailable` with zero text bytes. An unknown
exact symbol or a final envelope that exceeds its output budget is code `2` with no stdout. Ordinary
mode may maintain the lock before such a later failure; locked mode verifies it without writing.
The commands reopen no
source path, accept no encoded input or output path, and start no toolchain, process, network
request, model call, or persistent store.

`experimental atlas-evidence show` reuses that same exact workspace preparation once, then invokes
the owning Context Pack v1 source, attributed-text source, and attributed-text documentation encoders
for one exact symbol. It requires package UID, visible/full SymbolId, direction, at least one exact
relation kind, positive record/relation/selection limits, positive fragment/text limits, positive
facts and per-attributed-document limits, and a positive final bundle limit; only `--locked` is
optional. Counts are capped at 65,536, fragment bytes at 8,388,608, aggregate text bytes at
67,108,864, and all document/final byte classes at 1,073,741,824. The fixed one-fragment text budget
and `--max-text-document-bytes` apply independently to source and documentation. The compact wrapper
labels itself `none`, facts `compiler-fact`, and both text children `untrusted-text`; it grants no
permission or authenticity. Successful text omissions remain inside their child, while query,
component-document, and final-envelope overflow reject the whole output. There is no `overview`,
`around`, default, name search, decoder, bundle store, dependency/generated text, token estimate,
model call, or Context Pack text mutation.

`experimental check-workspace-session` exposes that workspace check through one foreground
process-local Session. It requires an explicit known target and caller-selected positive cache entry,
retained-byte, artifact-byte, and request-count limits. The command provides no cache defaults; its
hard safety ceilings are 256 requests, 65,536 entries, 1,073,741,824 retained bytes, and
1,073,741,824 artifact bytes, and one artifact may not exceed the total byte limit. It starts no
watcher, daemon, socket, network request, toolchain, backend, or concurrent worker.

Each stdin line is one UTF-8 object from
`spec/cli/experimental-workspace-check-session-v0.schema.json` with exact
`schemaVersion: "fairyc.experimental.workspace-check-session/v0"`, `kind: "check"`, and a contiguous
`sequence` beginning at one. Lines are limited to 1,024 bytes. Duplicate, unknown, missing, mistyped,
out-of-order, malformed, trailing, and oversized input terminates before that request with code `2`.
Every accepted request reloads the current manifest-owned workspace and performs ordinary lock
refresh or exact no-write `--locked` verification before checking through the shared Session.

One compact `kind: "check-result"` response line reports the same sequence, status and per-request
exit code, lock state, workspace/MIR/diagnostic counts, cache path counts, retained entries and bytes,
and cumulative verified hits. It includes no source text, source path, artifact, or host tool path.
Source diagnostics and package failures remain recoverable requests and the next input may repair
them. Internal, stdin-read, and stdout-write failures terminate. EOF or the explicit request bound
ends a nonempty session with its last response's exit code; EOF before any request is code `2`.
This is an unstable sequential compiler-process protocol, not a stable JSON mode, LSP, watcher,
concurrent service, persistent cache, or performance claim.

`experimental check-workspace-cache` is a separate one-shot process contract. It accepts no source
arguments or stdin protocol. Target, ignored `build/` cache directory, entry count, retained bytes,
record bytes, artifact bytes, and abandoned-temp age in seconds are all explicit. Numeric values are
canonical positive decimals; the hard ceilings are 65,536 entries, 1,073,741,824 bytes for each byte
class, and 31,536,000 seconds. Artifact bytes cannot exceed record bytes; record bytes cannot exceed
retained bytes. The executable's complete SHA-256 bytes supply the compiler-build key member.

Success writes exactly one object from
`spec/cli/experimental-workspace-cache-check-v0.schema.json`. It includes the relative cache path,
target, executable identity, lock/workspace/MIR counts, verified cache outcome, retained entries and
bytes, record hit/miss/rejection/eviction counters, verified hits, and distinct skipped versus failed
stores. It includes no source text, absolute path, artifact bytes, or toolchain path. Any miss,
corruption, incompatible schema/build/package/interface, or incomplete stable package set falls back
to ordinary lowering. A verified result survives a failed cache write and reports `storeFailed`;
failure to initialize the explicitly requested cache is a package failure. The command starts no
daemon, watcher, network request, toolchain, backend, or concurrent worker and makes no speed claim.

`experimental build-workspace` and `experimental run-workspace` reuse that exact workspace, SDK,
lock, semantic, Flow, MIR, and diagnostic path. They accept no positional entry or repeated source:
the entry is the root package's exact build-local `main::main`, and it must be `fn() -> i32`.
Both require the declared `x86_64-windows-msvc` target plus an explicit low or optimized profile.
Build additionally requires a canonical no-overwrite `.exe` path below the current root's ignored
`build/` directory. Run retains no artifact, rejects program output, and consumes the same
compiler-private trap token as stable workspace run so every normal `i32` remains distinct from a
trap. Both use ordinary lock
refresh unless `--locked` requires a valid exact no-write match. They perform no registry access,
version solving, source download, environment expansion, WSL invocation, persistent cache write, or
user linker-input discovery. The checked-in `examples/workspace_quickstart` proves one direct local
dependency and native result 42; configurable entries, multiple binaries, program stream forwarding,
and multiple binaries remain deferred; stable check/build/run have separate closed spellings.

`experimental check` uses the same repeated-`--source` closure syntax and limits, but its first file
is a primary source rather than an executable entry. It requires no `main`, runs no interpreter or
backend, selects no target, resolves no toolchain, and writes no artifact. It accepts verified
foreign calls because it proves MIR structure rather than host executability. Its success counts are
provisional compiler-IR observations and may include compiler-private functions.
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
Resource declarations preserve ordered resolved fields in typed HIR. A nonempty user `drop` body
becomes a compiler-private typed function linked from its resource. Its declared effects are checked
inside the glue and at every compiler-scheduled cleanup edge; moving the complete resource transfers
the obligation without charging the sender.

`experimental run` does not invoke a backend, linker, runtime, or host subprocess. It runs one
positional entry plus zero or more repeated `--source` files through parser, semantic analysis, Flow,
MIR lowering, the MIR verifier, and the bounded MIR interpreter. The closure uses the same 64-file,
per-file, aggregate-byte, duplicate, containment, and no-discovery rules as the Windows target
commands. The entry is the positional file module's zero-argument `main() -> i32`; boolean
`if`, `!`, comparisons, source-level `&&` and `||` short-circuiting, scalar `while`, scalar tagged
values, resource construction and field reads, whole-root and tagged-payload moves, resource-bearing
built-in `Option` and `Result` values, ordered resource reassignment, consuming-close ownership,
nearest-loop `break` and `continue`, lexical Region placement and reset on fallthrough, direct
return, Result propagation, nested, break, and continue exits from a direct `ref mut` Arena local,
cleanup drop events, checked `i32` negation, addition,
subtraction, multiplication, division, remainder, matching-type bitwise operations, and signed or
unsigned right shift are
executable. Effect-bearing drop contracts are accepted, but the interpreter does not emulate their
host filesystem, console, network, process, clock, environment, or random operations. The bounded interpreter reports its stable `step-limit` capability rather than hanging
or reclassifying it as a language trap. Interpreter-fixture byte hooks are not available from this
command and matching function names do not activate them. Host I/O and arguments remain outside the
current interpreter subset. Matching-type left shift reaches the interpreter
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
named-field variants, exact exhaustive shallow switching, and admitted scalar/raw-pointer/flat-struct
payload fields are supported; nested user enums and built-in tags inside their payloads fail closed.
Nonempty source `@repr(C)` structs over `u8`, `i32`, `u32`, `usize`, exact C scalars, admitted raw
pointers, and acyclic nested `@repr(C)` records preserve their public C tag and field names, emit
dependencies before users, and receive target-computed layout assertions. The same records may be
foreign results by value; foreign aggregate parameters remain rejected. An exact direct-
local `ref T -> ptr T`, `ref mut T -> ptr T`, or `ref mut T -> ptr mut T` cast inside `unsafe` lowers to verified
raw-address MIR and generated C `&local`; the exact `ref mut ptr mut void` output slot is included. Nested
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
`fy_std_buffer_reserve_v0`, `fy_std_buffer_append_v0`,
`fy_std_buffer_resize_zeroed_v0`, `fy_std_buffer_truncate_v0`,
`fy_std_buffer_clear_v0`, `fy_std_buffer_drop_v0`,
`fy_std_arena_initialize_v0`, `fy_std_region_begin_v0`,
`fy_std_region_allocate_zeroed_v0`, `fy_std_region_copy_bytes_v0`,
`fy_std_region_reset_v0`, `fy_std_arena_drop_v0`, and
`fy_std_cstr_validate_v0`, `fy_std_string_copy_v0`, `fy_std_string_as_view_v0`,
`fy_std_string_append_v0`, `fy_std_string_drop_v0`, `fy_std_cstring_from_text_v0`,
`fy_std_cstring_as_view_v0`, `fy_std_cstring_drop_v0`,
`fy_std_string_copy_storage_v0`, `fy_std_string_append_storage_v0`,
`fy_std_string_drop_storage_v0`, `fy_std_cstring_from_text_storage_v0`,
`fy_std_cstring_drop_storage_v0`, and `fy_std_text_utf8_valid_v0` calls select
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
and safe ownership remain outside that raw target example. A separate explicit two-source
`examples/owned_text/` build uses only safe entry calls over the storage bridges, executes at both
profiles in the configured Windows smoke, and still does not establish stable `std::text`, layout,
or borrowed-view contracts.

`experimental run-target` accepts the same explicit FY source closure but no output path, foreign companion input, linker option, ambient target,
or compiler override. It shares `build-target` lowering, exact standard-library resolution, local
MSVC discovery, object inspection, PE inspection, and unwind inspection before execution. The
temporary executable must produce no stdout or stderr; ordinary results outside the reserved
product-trap status range preserve signed `i32` exit bits, while that reserved range maps to
canonical categories and exit code `5`. This experimental command cannot distinguish equal normal
results from traps. Temporary
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
and Unicode console policy remain absent. Compiler-bundled safe `std::io` belongs to manifest-owned
workspace build/run and is not injected into this explicit-source target command.
The checked-in output example confines that raw call to a safe local `ref str` helper, supplies the
length through the view's `usize` byte-count `.len`, and verifies status plus completed progress.
The raw declaration, wrapper, and `main` propagate `@effects(console)`; `unsafe` remains an
independent support-module boundary. `main` contains no raw pointer, foreign call, or `unsafe`, and a safe `ref u8[]` helper separately
proves byte-view internal-call transport.

For the admitted foreign-call slice, `emit-c` writes a plain external prototype and call using the
declaration name by default or its single validated `@link_name` string. Compatible mappings to one
external symbol coalesce and conflicting signatures fail closed.
The exact C-scalar slices emit C `int`, LP64-sensitive `long`, `<stddef.h>` `size_t`, or target-specific POSIX.1-2008
`<sys/types.h>` `ssize_t`, target assertions, and a typed local between foreign producer and
consumer calls. `ssize_t` additionally fixes `_POSIX_C_SOURCE 200809L` before headers and preserves
negative values. They do not accept literals or operators, do not
accept raw pointers to C scalars, and do not alias the types to coincidentally sized Fairy integers.
The exact internal SystemHeap, Buffer, Arena/Region, text-view, CStr, String, CString, Linux File
open/read/write-all/sync/close, and Windows File open/read/write-all/sync/close/drop symbols
exercised by native source probes form a closed project-owned allowlist; the `fy_` prefix does not
become generally callable. The backend records sorted unique link names used by call terminators;
declarations without calls select no library component. `build` resolves the exact project-owned
subset through one shared symbol/component contract and links only its dependency closure. It still
supplies no user companion source, object, library, search path, or arbitrary linker argument, so
every other foreign symbol must resolve from the selected driver's ordinary closed environment or
the build fails. Windows File symbols are target-bound and resolve to an independent inline-handle
C component only for `x86_64-windows-msvc`. ADR 0264 exposes the same closed symbol set through the
compiler-bundled public `std::fs` module without opening linker inputs. `experimental run`
never binds host symbols and reports the interpreter capability boundary.

## Stable workspace check v1

`fairyc check [--locked]` is the stable current-workspace verification command. It accepts no
positional sources and loads the strict `fairy.toml` closure under the process source root. Ordinary
mode creates or refreshes canonical `fairy.lock`; `--locked` requires an exact existing lock and
writes nothing. The loader selects the bundled SDK only for actual `std::*` imports, and the compiler
runs package-aware syntax, semantic, Flow, MIR lowering, and MIR verification without requiring an
entry point or invoking a target, backend, toolchain, cache, network, or child process.

Success is exactly `check ok: <instances> instance(s), <sources> source file(s), <functions> MIR
function(s), lock <updated|unchanged|verified>`. The command freezes codes 0, 2, 3, 4, and 7 for
success, invalid invocation, source diagnostics, internal/output failure, and package/manifest/lock
failure. Stable JSON, target-specific validation, feature selection flags, persistent-cache defaults,
and configuration precedence remain outside v1.

## Stable workspace build v1

`fairyc build [--release] [--locked]` is the stable native build for the current workspace. It
accepts no positional source, output path, target, compiler, linker, profile value, cache, JSON,
configuration, network, or stdin input. The validated root package name selects
`build/<package-name>.exe`. Default mode uses the low-optimization development profile;
`--release` selects the ordinary optimized profile without LTO.

The command applies the stable check lock policy, requires root `main::main` with type
`fn() -> i32`, and reuses the closed package-aware `x86_64-windows-msvc` C11, project-owned support,
MSVC, PE, and unwind pipeline. The complete new image is compiled and inspected before a synced
staging file replaces the previous default artifact. Failures before successful replacement retain
the previous artifact and remove staging state.

Success is exactly `built: <path> (x86_64-windows-msvc, <development|release>, lock
<updated|unchanged|verified>)`. The command freezes codes 0, 2, 3, 4, 6, and 7. This command
stability does not promote the target's independent `emittable`, `conformant`, or `supported`
status and does not select persistent-cache defaults.

## Stable workspace run v4

`fairyc run [--release] [--locked] [-- <program-args...>]` compiles and executes the current
workspace through the same manifest, lock, root `main::main`, development/release profile, and
closed local Windows pipeline as stable build. The complete PE and unwind image is inspected before
execution, exists only in compiler-owned temporary storage, and is removed afterward. It does not
create, replace, read, or execute the retained `build/<package-name>.exe` artifact.

The command accepts no positional source, output, target, compiler, linker, profile value, cache,
JSON, configuration, or network input. The first `--` separates compiler options from program
arguments: every argument after it becomes the temporary program's process argument in order, the
delimiter itself is excluded, and `build` and other stable commands still reject the delimiter.
`run -- --locked` passes `--locked` to the program, while `run --locked -- x` keeps the compiler
option and passes `x`; an unknown flag before the delimiter is the existing invocation error, and a
NUL byte in any program argument is rejected before compilation. The generated C entry remains
`int main(void)`; the program reads its arguments through the effect-free `std::process` surface
against the UCRT wide argv, which the stdlib adapter materializes once with
`_configure_wide_argv(_crt_argv_unexpanded_arguments)` before strict UTF-8 conversion. After
compilation, linking, and PE/unwind inspection succeed, the caller's stdin is attached directly to
the temporary program without
compiler-side pre-reading, decoding, retention, logging, replay, or a separate byte ceiling.
Pre-launch failures consume no program input. Bounded program stdout and stderr are forwarded to
their matching compiler streams without a success banner. Capture uses separate OS pipes under one
shared 64 KiB ceiling, so stream identity is preserved but cross-stream order is unspecified.
Input liveness remains bounded by the existing process timeout.

A normal return across the full `i32` range becomes the `fairyc` process status. After launch, a
status numerically equal to 2, 3, 4, 5, 6, or 7 is still program-owned. A compiler-private run-only
trap suffix distinguishes reserved trap statuses from equal normal results: the suffix alone is
removed, preceding program stderr is forwarded, `trap: <category>` is appended, and the command
returns 5. The receipt is not public runtime ABI, authentication, or hostile-code isolation.
Pre-execution invocation, source, internal/output, build/execution, and package failures retain codes
2, 3, 4, 6, and 7 respectively without promoting target maturity or persistent-cache defaults.
An input transport failure after launch is an execution failure and cannot roll back program effects.
The embedding helper `cli.Run` still supplies EOF; callers use `cli.RunWithInput` to provide a stream.

Ordinary mode may update `fairy.lock` after a valid graph is loaded even when later compilation,
execution, a nonzero program status, or a program trap follows. `--locked` performs no write. A
compiler-output write failure after program execution returns code 4 but cannot undo earlier bytes
or program effects; no executable is retained.

## Stable workspace test v3

`fairyc test [--native [--release]] [--exact <test-identity>]... [--locked]` first loads and locks the ordinary production workspace, then overlays only
the root package's optional exact-lowercase `tests/` tree in memory. Ordinary check, build, run,
emit, lock, interface, and cache inputs never include those files. The tree may not be a symlink or
overlap the production source or a workspace member, shares the workspace source budgets, and maps
`tests/path.fy` to `module tests::path;`. An absent or empty tree is valid. Member test trees are not
recursively discovered.

Argument-free `@test` selects exactly private safe zero-argument free functions returning `void`;
invalid placements or shapes are `E9001`. Other functions in test modules are ordinary private
helpers. After removing only validated harness markers from the in-memory AST, the command runs the
package-aware semantic, Flow, MIR lowering, and verifier pipeline. Default execution runs each test
by canonical `<module>::<function>` order in a fresh deterministic interpreter state. `--native`
instead clones the verified MIR per test, appends and verifies one private i32 harness that calls
the selected void entry, emits only its reachable Windows closure, compiles and inspects a temporary
PE, and runs it with EOF stdin and no arguments in a fresh process and temporary directory.
Development uses low MSVC; `--native --release` uses optimized MSVC. A trap fails that test and later
tests continue. Interpreter or backend capability rejection aborts as code 2 rather than becoming a
false test failure.

Repeatable `--exact` values select canonical `<module>::<function>` identities byte-for-byte. The
selector order does not affect canonical execution/report order. Duplicate selectors, missing
values, and identities absent from the complete discovered set are code-2 invocation errors with no
report. Selection happens after test declaration validation and limits execution only: syntax,
semantic, Flow, MIR, and verifier admission still cover the complete test overlay.

Completed lines are `test <identity> ... ok` or
`test <identity> ... FAILED (<trap-category>)`. The final line is
`test result: <ok|FAILED>. <passed> passed; <failed> failed; lock <updated|unchanged|verified>`.
Zero tests succeed with only the final line. Codes are 0 for all
passing, 2 for invocation/capability/native-program-output rejection, 3 for source or declaration
diagnostics, 4 for internal/output failure, 5 for one or more trapped tests, 6 for native
toolchain/build/inspection/execution failure, and 7 for workspace/test-source/lock failure. Default
mode invokes no target backend, toolchain, process, network, or host I/O. Native mode requires local
Windows x64 MSVC but never WSL or network and retains no artifact. A report writer may accept a
prefix before code 4; ordinary lock maintenance and earlier native test effects may precede later
failure, while `--locked` writes nothing.

## Stage 0 surface

| Command | Purpose | Code generation |
| --- | --- | --- |
| `fairyc new <package-name> --uid <package-uid>` | Create one exact no-overwrite starter project below the current directory | Fairy C source only |
| `fairyc check [--locked]` | Check the current manifest-owned workspace through verified MIR | No |
| `fairyc build [--release] [--locked]` | Build or replace the current root package's one Windows executable | Yes |
| `fairyc run [--release] [--locked] [-- <program-args...>]` | Build and run the current workspace as one temporary Windows image, forwarding arguments after `--` to the program | Yes |
| `fairyc test [--native [--release]] [--exact <test-identity>]... [--locked]` | Run all or exact canonical root-package `@test fn void` entries through the semantic interpreter or one temporary Windows MSVC image per selected test | Optional temporary native images |
| `fairyc emit c --target <target-id> [--locked]` | Emit the current executable workspace's target-selected C11 application unit | C only, stdout |
| `fairyc explain <diagnostic-code>` | Explain a stable diagnostic code | No |
| `fairyc targets` | Report known target profiles, status, evidence, and unavailable requirements | No |

Alternate package/workspace selection, fuzzy/tag/file test filtering, portable native-test targets, target triples beyond
  the declared emit profiles and build/run's fixed local Windows path,
additional program-input routing beyond stable run,
feature flags, and configuration precedence
remain UNDECIDED. The implemented `experimental` spellings do not settle those decisions.

## General requirements

- Compiler-owned prompts remain absent. Stable run may attach caller stdin to the launched program;
  the compiler does not interpret those bytes as command interaction.
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
