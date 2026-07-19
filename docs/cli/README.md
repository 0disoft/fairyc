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

The Go frontend currently exposes `fairyc experimental parse <file.fy>`,
`fairyc experimental check-syntax <file.fy>`, and
`fairyc experimental check-semantics <file.fy> [<file.fy>...]`. The compiler prototype also exposes
`fairyc experimental target-spec x86_64-linux-gnu` for deterministic host-independent target
inspection,
`fairyc experimental emit-c <file.fy>` for deterministic scalar, explicitly link-mapped foreign-call,
flat-resource, and resource-tagged C11 inspection and
`fairyc experimental build <file.fy> --cc <clang|gcc> --profile <low|optimized> --output <build/path>`
for an explicit host-Linux executable in that subset, and
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
It accepts no companion source, object, library, search path, or arbitrary linker argument, so an
admitted scalar foreign symbol must already resolve through the selected driver's ordinary link
environment.
The resulting product artifact is uninstrumented; the native fixture recorder is not linked.
`experimental target-spec` emits `fairyc.target-spec/v0` JSON without probing the host or exposing
local tool paths. Its current scalar, flat-resource, and resource-tagged emission subset is reported separately from the false
`emittable`, `conformant`, and `supported` target states and their missing evidence.
The interpreter fixture's in-memory byte hooks require manifest options that this CLI never accepts;
declaring a similarly named Fairy C function does not grant output authority.
Infinite execution is stopped by the bounded interpreter and reported as an unsupported-capability result rather than a language trap. In
particular, it does not satisfy the planned `fairyc check` contract.

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
