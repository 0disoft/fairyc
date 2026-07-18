# Fairy C

- Status: Stage 0 implementation prototype
- Scope: general
- Repository type: cli-tool
- Addon: monorepo

Fairy C is an experimental systems programming language for new native software that needs C-like control, predictable costs, explicit C interoperability, and a substantially safer default resource model without a garbage collector or a heavy runtime.

The project is not trying to beat every hand-tuned C program or make every C ownership graph safe. Its intended advantage is that ordinary Fairy C code carries enough ownership, aliasing, bounds, and effect information for the compiler to reject common failures and generate competitive machine code without relying on build-mode-dependent undefined behavior.

## Current phase

The first Stage 0 frontend slice now exists in Go: immutable source bytes, a lossless lexer,
compact recoverable syntax, error-free AST lowering, deterministic `E00xx` diagnostics, and a
runner for all parser v0 fixtures. The next slice resolves names and produces typed HIR for the
  fixture-backed scalar and safe-borrow subset. Its semantic runner executes all 13 v0 resolve, type,
and local Flow cases, including last-use loans and borrow-return origins. Whole-root resource Flow
  also executes all 12 v0 cases for moves, cleanup obligations, branch joins, normal, break, and continue loop edges,
consuming tagged matches, and live/dead resource reassignment. The
Region runner executes all 6 v1 cases, including package-proven `std::alloc::Arena`, hidden
allocation origins, lexical escape, capability storage, and plain-region resource rejection. The
  Result runner executes all 24 v0 cases for qualified construction, exhaustive shallow matching,
  must-use return/discard/propagation, local and parameter transfer, branch joins, loop back-edges and control exits,
overwrite and early-return rejection, exactly-once classification, non-Result `?`, and propagated
error compatibility. The MIR runner executes all
  11 v0 normalization cases through mutable-local block lowering and the MIR verifier, covering
left-to-right call evaluation, complete-root moves, normal-exit cleanup order, `Result` error
  cleanup, nearest-loop control, explicit checked arithmetic, and non-unwinding trap paths. The verifier mutation runner constructs all 10 named
  baselines and proves all 21 isolated `V1001` through `V1701` rejection recipes. The first MIR
  interpreter slice executes all 56 v0 cases for literal return, function-call arithmetic,
boolean branches and logical not, source `&&` and `||` short-circuiting, scalar `while`, scalar
`Option` construction/matching, scalar `Result` construction/matching/propagation,
resource construction, field reads, call moves, resource-bearing `Option` and `Result` payload moves,
resource returns and propagation, ordered reassignment, consuming-close success/failure,
nested active-variant cleanup, deterministic drop traces,
  normal lexical Region placement plus direct-return, Result-propagation, nested, break, and continue reset traces,
division/remainder zero and signed-overflow, ordinary integer-overflow, and invalid-shift traps, and bounded step-limit
termination. Fixture-only, manifest-authorized byte sinks additionally prove exact in-memory
stdout/stderr bytes, cross-stream order, and a shared output limit without granting host I/O.
Resource declarations now retain ordered, resolved fields and drop-body classification from AST
through typed HIR and executable MIR. The current executable subset accepts only an empty user
`drop` body; any nonempty body fails explicitly before Flow or MIR instead of being silently ignored.
The interpreter constructs, reads, and recursively drops resources from the verified MIR definition,
but still does not execute user drop-body effects.
The constrained C backend emits deterministic ISO C11 for 11 verified fixtures across the scalar
payload surface and flat nonempty all-`i32` resources with empty drop bodies. The cases cover
resource construction, field reads, whole-root call moves, live reassignment, calls and control
flow, bitwise operations, checked division and shift, overflow
trapping, scalar `Option` Some and None, and scalar `Result` propagation success and failure. This
lane only produces C text; no C compiler, linker, native executable, complete runtime, target
conformance, or self-hosting is implied.
The first executable `fairyc.target-spec/v0` description now owns the host-independent
`x86_64-linux-gnu` identity, LP64 scalar and pointer layouts, SysV AMD64 calling and stack facts,
ELF/DWARF identities, and fail-closed maturity evidence. It deliberately contains no local
compiler, SDK, or sysroot path and still reports `emittable`, `conformant`, and `supported` as
false.
The host-independent layout engine now executes all 14 target-layout v0 cases for language and C
scalars, data and function pointers, explicitly based C enums, fixed arrays, nested structs, field
offsets, internal and tail padding, and fail-closed malformed or overflowing inputs. These are
target-mechanics fixtures, not source-language aggregate support or native C ABI proof.
A separate fail-closed Linux/amd64 command is now present to compile those cases with Clang and GCC
at `-O0` and `-O2`, link the embedded scalar trap runtime, and compare process outcomes. It cannot run on
this Windows host because no WSL distribution or C compiler is installed, and its hosted CI result
has not yet been observed. The target therefore remains unsupported.
An unstable `fairyc experimental` CLI can inspect syntax, syntax-check, semantic-check, emit the
scalar-and-flat-resource C11 translation unit, build that subset into a Linux executable on an eligible host,
inspect the declared target specification, or interpret one source-root-relative `.fy` file, while semantic checking also
accepts an explicit module set. There is no stable CLI, user drop-body execution, native execution
command, complete runtime, standard library, package manager, supported target, or stable language
revision yet.

## Try the current prototype

The following commands are manual developer examples, not agent command authority:

```text
go run ./cmd/fairyc experimental check-syntax examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/syntax/main.fy
go run ./cmd/fairyc experimental check-semantics examples/semantic/main.fy examples/semantic/math.fy
go run ./cmd/fairyc experimental check-semantics examples/resources/main.fy
go run ./cmd/fairyc experimental check-semantics examples/regions/main.fy
go run ./cmd/fairyc experimental check-semantics examples/results/main.fy
go run ./cmd/fairyc experimental target-spec x86_64-linux-gnu
go run ./cmd/fairyc experimental emit-c examples/c/main.fy
go run ./cmd/fairyc experimental emit-c examples/c/tagged.fy
go run ./cmd/fairyc experimental emit-c examples/c/resource.fy
go run ./cmd/fairyc experimental build examples/c/tagged.fy --cc clang --profile optimized --output build/fairyc-example
go run ./cmd/fairyc experimental run examples/run/main.fy
go run ./cmd/fairyc experimental run examples/run/resource_option.fy
go run ./cmd/fairyc experimental parse examples/syntax/main.fy
```

These commands treat the current directory as the temporary source root. `parse` emits versioned,
deterministic JSON containing token and CST kinds and byte spans, but not source text, token text,
literal values, or absolute paths. `check-semantics` accepts an explicit bounded list of source files
for the currently implemented scalar, safe-borrow, whole-resource, Region-parameter, and Result subset. It rejects duplicate paths, root
escapes, oversized input sets, and unsupported syntax rather than claiming success. The official
`experimental run` accepts one file whose declared module contains `main() -> i32`, executes its
verified MIR in the bounded interpreter, including boolean `if`, short-circuit control, scalar `while`,
resource construction, field reads, whole-root call moves, resource-bearing built-in tagged values,
ordered resource reassignment, consuming-close ownership patterns, cleanup drop events,
  and Region fallthrough, direct-return, Result-propagation, nested, break, and continue reset paths,
checked addition, subtraction, multiplication, negation, and signed right shift, and reports either `result: <i32>` or a canonical
trap. The official
`fairyc check` command remains unavailable until effects and the
remaining semantic groups exist.

`experimental target-spec` accepts exactly the known `x86_64-linux-gnu` ID and writes deterministic
`fairyc.target-spec/v0` JSON. The result separates target semantics from local toolchain discovery
and lists the missing evidence that keeps all three target maturity claims false.

`experimental emit-c` accepts the same single-file entry shape and prints the verified scalar plus
flat all-`i32` resource
ISO C11 translation unit directly to stdout. It does not write a `.c` file or invoke a native
toolchain.

On Linux/amd64 with glibc and the explicitly selected Clang or GCC installed, `experimental build`
uses that same verified unit and the embedded runtime to produce a no-overwrite executable
below the ignored `build/` directory. Compiler, `low` or `optimized` profile, and output path are
mandatory. This Windows host cannot execute that command, and the command remains narrower than the
interpreter: scalar `Option<i32>`, `Result<i32, i32>`, and direct flat nonempty all-`i32` resources
with empty drop bodies work. Resource-bearing or nested tags, non-`i32` or nested resource fields,
partial initialization, user drop effects, Region, unsafe, FFI, and standard-library programs still
fail closed before native compilation.

## Design anchors

- Product contract: `docs/product/02-spec.md`
- Compiler and repository boundary: `docs/architecture/00-system-boundary.md`
- Language concepts: `docs/architecture/01-domain-model.md`
- Compilation flow: `docs/architecture/02-runtime-flow.md`
- Quality targets: `docs/architecture/03-quality-attributes.md`
- Initial compiler decision: `docs/adr/0001-initial-architecture-boundaries.md`
- Contract ownership decision: `docs/adr/0002-contract-source-of-truth.md`
- Stage 0 safety boundary: `docs/adr/0003-stage-0-safety-boundary.md`
- Language semantics: `docs/language/`
- Normative Stage 0 grammar: `spec/grammar/stage0.ebnf`
- Parser acceptance fixtures: `spec/fixtures/parser/manifest-v0.json`
- Semantic acceptance fixtures: `spec/fixtures/semantic/manifest-v0.json`
- Resource Flow fixtures: `spec/fixtures/resource/manifest-v0.json`
- Region fixtures: `spec/fixtures/region/manifest-v1.json`
- Result fixtures: `spec/fixtures/result/manifest-v0.json`
- MIR normalization fixtures: `spec/fixtures/mir/manifest-v0.json`
- MIR verifier mutation fixtures: `spec/fixtures/mir-verifier/manifest-v0.json`
- MIR interpreter fixtures: `spec/fixtures/interpreter/manifest-v0.json`
- Constrained C emission fixtures: `spec/fixtures/c-emission/manifest-v0.json`
- Shared constrained-C driver and embedded scalar runtime: `internal/cdriver/`, `internal/cruntime/`
- Linux scalar C differential runner: `internal/cbackendnative/`
- Executable target description: `internal/target/`
- Experimental target JSON schema: `spec/cli/experimental-target-spec-v0.schema.json`
- Target layout engine and fixtures: `internal/layout/`, `spec/fixtures/layout/manifest-v0.json`
- Resolved package graph fixtures: `spec/fixtures/package/manifest-v0.json`
- MIR and C backend contracts: `docs/mir/`
- Package identity and visibility: `docs/package/package-graph-v0.md`
- Target and C ABI: `docs/abi/c-abi-v0.md`
- Runtime and standard library: `docs/runtime/runtime-stdlib-v0.md`
- Compatibility and versioning: `docs/compatibility/versioning-v0.md`
- Diagnostic catalog: `docs/diagnostics/catalog-v0.md`
- Atlas schemas: `docs/atlas/`
- CLI contract: `docs/cli/command-contract.md`
- Workspace ownership: `docs/monorepo/package-ownership.md`

## Working name and tools

- Language name: Fairy C
- Compiler command: `fairyc`
- Short alias: `fyc` is reserved but not committed
- Source extension: `.fy`
- `fc` is not used as the executable name

Repository hygiene and generated design-document ownership are governed by `AGENTS.md`, `VALIDATION.md`, and `.ssealed/manifest.json`.
