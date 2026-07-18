# Roadmap

- Status: Draft
- Owner: UNASSIGNED
- Scheduling: UNDECIDED

This roadmap defines capability gates, not calendar promises. Exact durations, fixture totals, latency targets, and storage limits remain absent until an executable harness and source of truth exist.

## Phase 0: freeze the executable contracts

- Freeze Stage 0 surface syntax through `spec/grammar/stage0.ebnf`, parser fixtures, semantic fixtures, resource, Region, Result Flow, verified-MIR, and resolved-package-graph fixtures, one-file modules, alias imports, restricted enums, ownership, borrow origins, regions, `Result`, evaluation order, traps, unsafe boundaries, and exclusions.
- Freeze the first planned conformance target, `TargetSpec`, `ToolchainSpec`, C ABI v0, runtime and standard-library layer ownership, and the compatibility vector.
- Reserve diagnostic categories and codes for rejected syntax, ownership, region, FFI, ABI, and compatibility paths.
- Define Flow CFG, MIR v0, drop elaboration, the MIR verifier, isolated verifier-mutation recipes, generated-C safety rules, and compile-pass, compile-fail, run, trap, drop-trace, runtime, and bidirectional C ABI fixture formats.

Exit gate: the written contracts have no contradictory syntax, initialization, ownership, cleanup, package identity, visibility, target, runtime, or compatibility rule, and every mandatory rule has an observable fixture class or an explicit dependency-gated deferral. Parser, semantic, resource, Region, Result, MIR, and package manifest cases must pass their future full runners with no unlisted independent diagnostics or observation drift.

## Phase 1: first interpreted semantic slice

Current implementation status: the source, lexer, recoverable-syntax, parser-diagnostic, AST
lowering, and parser-fixture-runner portion of the first item exists. Name resolution and typed HIR
now cover the semantic fixture's scalar and safe-borrow subset. Local last-use loans, borrow
  conflicts, single-origin borrow returns, Stage 0 signed-left-shift rejection, and loop-context diagnostics execute across all 13 semantic v0 cases. General
  whole-root resource Flow now executes all 12 resource v0 cases, including branch, normal-loop,
  break-edge, and continue-edge state rejection. Result Flow executes all 24 result v0 cases, including qualified scalar construction,
exhaustive shallow matching, and local, parameter, branch, loop, overwrite, and early-return
must-use obligations. Mutable-local MIR lowering and verification
  execute all 11 MIR normalization v0 cases for the current scalar, resource, tagged-payload, Result, Region, loop-control cleanup, call,
  division, and checked-arithmetic trap subset. All 10 verifier baselines and 21 isolated mutation cases execute through
  the same MIR verifier. The first MIR interpreter executes all 56 interpreter v0 cases, including
scalar `Option` and `Result` construction, exhaustive matching, and both Result `?` paths, and is
exposed by the unstable `experimental run` command. Resource construction, field reads, whole-root
call moves, source-linked drop traces, resource-bearing `Option` and `Result` payload moves, ordered
resource reassignment, explicit consuming-close success/failure ownership, and normal lexical
Region placement, inner cleanup, normal reset, direct return, Result propagation, and nested
early-exit reset execute. Manifest-authorized in-memory stdout/stderr byte hooks prove output order
and bounded failure without granting host access. Checked `i32` addition, subtraction,
multiplication, negation, division, remainder, and signed right shift execute across trapping and
  non-trapping boundaries with exact overflow, division-by-zero, or invalid-shift
  categories; signed left shift is rejected as `E1011`. Nearest-loop `break` and `continue` now execute
  with resource and Region cleanup. Verified resource definitions now preserve ordered fields and
  empty-drop classification from AST through HIR into MIR; nonempty user drop bodies fail as
  unsupported before executable lowering. The C backend now emits deterministic ISO C11 for 11
  verified scalar and flat-resource cases under the provisional `x86_64-linux-gnu` profile,
  including `Option<i32>` and `Result<i32, i32>` construction, match, calls, and propagation plus
  all-`i32` resource construction, field reads, moves, and reassignment. The unstable
  `experimental emit-c` command exposes that exact verified translation unit on stdout
  without writing an artifact. On an eligible Linux/amd64 glibc host, `experimental build` can now
  compile and link that subset with explicit Clang or GCC and low or optimized profiles into
  a no-overwrite `build/` artifact. Real file and console I/O, effects, and every later Phase 1
  capability remain unimplemented.

- Implement source management, lexer, trivia-preserving tokens and spans, compact recoverable syntax, deterministic diagnostics, one-file modules, alias imports, and non-generic typed HIR in Go.
- Implement restricted inert enums, whole-resource moves, local last-use loans, call-bounded mutable reborrows, single-origin borrow returns, plain regions, `Option`, `Result`, shallow `match`, and explicit `?` and `discard`.
- Implement Flow CFG, mutable-local MIR, drop elaboration, the verifier, target-independent layouts, and the compatibility-vector metadata model.
- Build a small MIR interpreter covering scalar and tagged behavior, resource payloads, drop traces, region allocation, a controlled in-memory I/O test harness, and traps.

Exit gate: positive and negative fixtures agree with the source contracts, resource-bearing built-in variants have correct active-payload traces, and every named verifier seed passes before its isolated mutation demonstrates the intended single `Vxxxx` rejection.

## Phase 2: first hosted executable slice

Current implementation status: a fail-closed Linux/amd64 constrained-C differential command now
requires glibc, Clang, and GCC and is wired into a separate hosted Ubuntu CI job. It is designed to
run all 11 emission cases at low and optimized profiles for 44 process-status comparisons, then drive
the actual experimental build CLI with scalar Option and Result plus a flat all-`i32` resource
through 4 compiler/profile executable
smokes. The current Windows
host has no eligible Linux toolchain, and no hosted run receipt has been observed, so this is a
configured lane rather than native conformance evidence. A shared bounded C driver and embedded
scalar runtime now serve both that lane and the experimental single-file build command. This proves
the product wiring exists, not that runtime, ABI, or target-conformance gates pass.
The first executable `fairyc.target-spec/v0` now centralizes host-independent LP64 scalar and
pointer layouts, SysV AMD64 calling and stack identities, ELF/DWARF identities, C-layout feature
vocabulary, and explicit missing maturity evidence. The C backend and driver share its typed target
ID, while local toolchain paths remain outside the target contract. Host-independent scalar,
array, explicit-C-enum, pointer, and nested-aggregate layout execution now runs through 14 versioned
v0 cases. Native C layout probes, SysV call classification, bidirectional ABI fixtures, and hosted
evidence remain incomplete.

- Implement `x86_64-linux-gnu` `TargetSpec`, toolchain resolution, constrained generated-C transport, and C-driver linking.
- Implement the thin runtime and minimum `SystemHeap`, `Arena`, `Buffer`, text views, `String`, `CString`, and Linux `File` surface required by representative fixtures.
- Run interpreter-to-C differential fixtures and bidirectional C ABI fixtures with Clang and GCC at low and optimized profiles.
- Add sanitizer-assisted backend checks without treating sanitizers as the language oracle.

Exit gate: the target reaches `conformant`: interpreter and generated-C runs agree on values, status, trap category, active-payload and drop traces; all declared layout and C ABI fixtures pass in clean Linux CI with no skips. This gate does not by itself make the target `supported`.

## Phase 3: modules, package builds, and bounded caching

- Add public package manifests, lockfiles, module interfaces, concrete package-instance encoding, and public-interface hashing on top of the resolved-graph fixture contract.
- Start with whole-package reuse and a bounded content-addressed cache whose limits are derived from measurements.
- Stabilize the minimal `fairyc check`, `build`, `run`, `test`, `emit`, `explain`, and `targets` contracts.
- Move to function-level reuse only when measured package rebuild or storage evidence justifies the complexity.

Exit gate: clean and warm build evidence is reproducible on a declared machine and workload; invalidation fixtures distinguish public interface, inline body, and private implementation; incompatible caches are safe misses.

## Phase 4: Atlas fact export

- Produce an immutable `SemanticSnapshot` from analyzed HIR and verified MIR.
- Export `.fymap` with package UID and instance identity, compatibility vector, provenance, relation-specific completeness, and deterministic symbol identity.
- Implement bounded `overview`, `show`, and `around` queries and Context Pack output.
- Measure whole-package map rewrite cost before considering chunks or compression.

Exit gate: Atlas failure cannot change compilation; stale or incompatible maps fail closed; unknown graph edges remain explicit; text cannot grant authority.

## Later capability gates

- `x86_64-linux-musl`: requires a separate libc, sysroot, startup, loader, and static-distribution profile.
- `x86_64-windows-msvc`: requires LLP64 layout, Microsoft x64 calling, PE/COFF, UCRT/VCRuntime, and unwind fixtures.
- QBE or another fast backend: requires stable internal semantics, interpreter and C oracles, ABI coverage, and measured edit-build benefit.
- LLVM backend: requires representative workloads that justify optimization and distribution cost after simpler paths are exhausted.
- Generics and `Vec<T>`: require non-generic stability plus compile-time, code-size, active-payload, and diagnostics fixtures.
- Immutable green tree, incremental reparse, formatter, LSP, and SCIP: require an editor product slice and measured parser needs.
- `impact`, semantic `diff`, stable anchors, MCP, and structural edits: require `.fymap` identity and completeness evidence from real use.
- Custom allocators, portable paths, resource-aware arenas, closures, concurrency capabilities, async, sockets, and freestanding or embedded support each require a separate ownership and target contract.

## Self-hosting gate

Self-hosting starts only after modules, strings, containers, `Result`, resource moves, C FFI, package builds, source maps, and the test runner can support the compiler without freezing immature language decisions. The Go compiler remains a bootstrap and reference implementation.
