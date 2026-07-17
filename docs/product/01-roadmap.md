# Roadmap

- Status: Draft
- Owner: UNASSIGNED
- Scheduling: UNDECIDED

This roadmap defines capability gates, not calendar promises. Exact durations, fixture totals, latency targets, and storage limits remain absent until an executable harness and source of truth exist.

## Phase 0: freeze the executable contracts

- Freeze Stage 0 surface syntax through `spec/grammar/stage0.ebnf`, parser fixtures, semantic fixtures, resource, Region, Result Flow, verified-MIR, and resolved-package-graph fixtures, one-file modules, alias imports, restricted enums, ownership, borrow origins, regions, `Result`, evaluation order, traps, unsafe boundaries, and exclusions.
- Freeze the first planned conformance target, `TargetSpec`, `ToolchainSpec`, C ABI v0, runtime and standard-library layer ownership, and the compatibility vector.
- Reserve diagnostic categories and codes for rejected syntax, ownership, region, FFI, ABI, and compatibility paths.
- Define Flow CFG, MIR v0, drop elaboration, the MIR verifier, generated-C safety rules, and compile-pass, compile-fail, run, trap, drop-trace, runtime, and bidirectional C ABI fixture formats.

Exit gate: the written contracts have no contradictory syntax, initialization, ownership, cleanup, package identity, visibility, target, runtime, or compatibility rule, and every mandatory rule has an observable fixture class or an explicit dependency-gated deferral. Parser, semantic, resource, Region, Result, MIR, and package manifest cases must pass their future full runners with no unlisted independent diagnostics or observation drift.

## Phase 1: first interpreted semantic slice

- Implement source management, lexer, trivia-preserving tokens and spans, compact recoverable syntax, deterministic diagnostics, one-file modules, alias imports, and non-generic typed HIR in Go.
- Implement restricted inert enums, whole-resource moves, local last-use loans, call-bounded mutable reborrows, single-origin borrow returns, plain regions, `Option`, `Result`, shallow `match`, and explicit `?` and `discard`.
- Implement Flow CFG, mutable-local MIR, drop elaboration, the verifier, target-independent layouts, and the compatibility-vector metadata model.
- Build a small MIR interpreter covering scalar and tagged behavior, resource payloads, drop traces, region allocation, partial I/O test intrinsics, and traps.

Exit gate: positive and negative fixtures agree with the source contracts, resource-bearing built-in variants have correct active-payload traces, and mutation tests demonstrate every MIR verifier invariant.

## Phase 2: first hosted executable slice

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
