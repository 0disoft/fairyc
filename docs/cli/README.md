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

Only `check`, a minimal `build`, compiler inspection, and semantic test execution are Stage 0 candidates. The remaining commands are product commitments only after their contracts and implementations exist.

## Current prototype

The Go frontend currently exposes the source-independent
`fairyc explain <diagnostic-code>` lookup, the host-independent `fairyc targets` maturity report,
plus `fairyc experimental parse <file.fy>`,
`fairyc experimental check-syntax <file.fy>`, and
`fairyc experimental check-semantics <file.fy> [<file.fy>...]`. The compiler prototype also exposes
`fairyc experimental target-spec <x86_64-linux-gnu|x86_64-windows-msvc>` for deterministic
host-independent target inspection,
`fairyc experimental emit-c <file.fy>` for deterministic scalar, explicitly link-mapped foreign-call,
flat-resource, and resource-tagged C11 inspection and
`fairyc experimental build <file.fy> --cc <clang|gcc> --profile <low|optimized> --output <build/path>`
for an explicit host-Linux executable in that subset,
`fairyc experimental build-target <file.fy> --target x86_64-windows-msvc --profile <low|optimized> --output <build/path.exe>`
for the first declared Windows target C-emission subset,
`fairyc experimental build-host <file.fy> --profile <low|optimized> --output <build/path.exe>` for a
retained target-neutral Windows/amd64 MSVC PE,
`fairyc experimental run-host <file.fy> --profile <low|optimized>` for a temporary target-neutral
Windows/amd64 MSVC host smoke, and
`fairyc experimental run <file.fy>` for the fixture-backed scalar MIR interpreter subset. These are developer commands whose
spelling and numeric exit codes may change. `check-semantics` resolves names and creates typed HIR
for an explicit bounded module set in the implemented scalar, safe-borrow, whole-resource, Region, and Result subset.
It checks local last-use loans, borrow conflicts, borrow return origins, whole moves, cleanup states,
branch joins, loop back-edges and control exits, Region parameter origins, `place`, and lexical Region escape when its
Arena provider is included explicitly. It also checks Result return, discard, propagation, local
and parameter transfer, branch and loop obligations, overwrite and early-return rejection, and
error compatibility. `experimental run` executes the selected
module's zero-argument `main() -> i32` through verified MIR, including boolean `if`,
short-circuit control, scalar `while`, whole resources, resource-bearing built-in tags, ordered
resource reassignment, consuming-close ownership, Region fallthrough, return, propagation, nested,
break, and continue reset paths, and source-linked drop traces, and reports an
exact scalar result or canonical division, integer-overflow, or invalid-shift trap. Checked
negation, addition, subtraction, multiplication, and signed right shift are executable; signed
left shift is rejected as `E1011`; loop control without an enclosing `while` is rejected as `E1012`.
It does not discover files, execute user drop bodies, implement effects, or provide host I/O.
It also does not execute foreign calls: those fail with the interpreter capability boundary even
when the same verified MIR is admissible to `emit-c`.
`experimental emit-c` runs the same verified-MIR entry pipeline and writes one raw constrained C11
translation unit to stdout. It does not write an artifact, invoke a C toolchain, link, or run native
code. The native fixture lane's `drop-trace-v0` observer profile is not available through this
command and never appears in its output.
The admitted foreign declaration form currently supports `i32`, `u8`, `u32`, `usize`, all sixteen
exact transport-only TargetSpec C scalars, raw pointers to
the fixed-width Fairy scalar set or opaque `void`, and raw pointers to admitted nonempty acyclic
`@repr(C)` structs whose fields may include exact C scalars or nested `@repr(C)` records. Those
records may also be returned by value from a foreign C function. One
pointer-to-opaque-pointer output slot such as `*mut *mut void` is admitted without general pointer
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
existing path nor implements empty, non-`i32`, nested, partially initialized, or nonempty-drop
resources; nested tags or resource tags outside the exact `i32`/direct-resource payload gate; the broader interpreter subset; native execution;
package builds; cross-compilation; installation; or a supported target.
It automatically supplies only the minimal project-owned standard-library C dependency closure for
exact admitted symbols used by actual calls. It accepts no user companion source, object, library,
search path, or arbitrary linker argument, so every other admitted foreign symbol must already
resolve through the selected driver's ordinary link environment.
The resulting product artifact is uninstrumented; the native fixture recorder is not linked.
`experimental run-host` and `experimental build-host` use the same WSL-free Windows developer-smoke
compiler path. They auto-discover the installed x64 MSVC and Windows SDK, admit only target-neutral
private C transport, and reject FFI, provisional standard-library calls, `@repr(C)`, and exact
`c::*` values. `run-host` removes the temporary PE after execution; `build-host` retains one
no-overwrite `.exe` under `build/`. Their receipts are host-smoke evidence, not Windows target, ABI,
distribution, or self-hosting claims.
`experimental build-target` is separate from those smokes. It selects the Windows MSVC emission
profile, emits target-labelled C11, requires Windows LLP64 C-scalar providers, rejects unavailable
`c::ssize_t`, and builds a no-overwrite PE through locally resolved x64 MSVC. The current command
permits target declarations such as unused `c::long`, but rejects every called foreign symbol
because companion-link and Windows standard-library contracts are still absent. Its receipt is
target-emission evidence, not ABI conformance, hosted support, a distributable artifact promise, or
self-hosting.
`experimental target-spec` emits `fairyc.target-spec/v0` JSON without probing the host or exposing
local tool paths. Linux reports its current scalar, flat-resource, and resource-tagged emission
subset. Windows reports `scalar-and-llp64-c-scalar-c11`. Both keep `emittable`, `conformant`, and `supported` false and list
their own missing evidence.
The interpreter fixture's in-memory byte hooks require manifest options that this CLI never accepts;
declaring a similarly named Fairy C function does not grant output authority.
Infinite execution is stopped by the bounded interpreter and reported as an unsupported-capability result rather than a language trap. In
particular, it does not satisfy the planned `fairyc check` contract.

`fairyc explain` accepts one exact uppercase `Edddd`, `Adddd`, or `Vdddd` code and writes its
prototype meaning, category, and catalog identity without reading source, configuration, or the
current directory. It performs no toolchain discovery, target probe, process execution, network
request, or write. The output explicitly marks the catalog provisional; a successful lookup does
not claim that the diagnostic is implemented, fixture-backed, released, or stable. Listing, fuzzy
search, JSON, localization, and repair guidance remain unimplemented.

`fairyc targets` accepts no arguments and deterministically lists every declared TargetSpec with
its schema, admitted emission subset, independent `emittable`, `conformant`, and `supported` states,
and sorted missing evidence. It reads no source or configuration and performs no host, toolchain,
SDK, sysroot, WSL, process, network, or write operation. The declared `x86_64-windows-msvc`
TargetSpec is host-independent; only `build-target` later resolves a local toolchain after selecting
the declared Windows profile. It remains separate from the target-neutral Windows MSVC smoke.
JSON, filters, target selection, and release-stable output remain unimplemented.

Human developers can run the examples shown in the repository `README.md`. Agent verification uses
the configured `fairyc_go_build`, `fairyc_go_vet`, and `fairyc_go_test` intents; the manual `go run`
examples do not grant agent command authority.

The future stable `targets` command must distinguish `emittable`, `conformant`, and `supported`.
The experimental single-target JSON does not settle its stable list or filtering syntax. The
planned `x86_64-linux-gnu` profile cannot be promoted before its runtime, standard-library, ABI,
toolchain, and clean-CI fixtures pass without skips.

## Profile principle

- Stage 0 development and release both use the C reference backend with different explicit optimization profiles; `check` uses no backend.
- A later fast or production backend changes this profile only after a measured adoption decision.
- ThinLTO, PGO, full debug information, and cross-target builds remain explicit product requests.
  The current ASan+UBSan mode is validation-only and is deliberately not a CLI profile.

## Naming

The executable is `fairyc`. `fc` is not used. A `fyc` alias may be considered only after packaging and shell compatibility are specified.
