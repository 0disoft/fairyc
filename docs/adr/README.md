# Architecture Decisions

- Status: Draft
- Owner: UNASSIGNED

## Purpose

This document captures the durable design contract for Architecture Decisions.
It is intentionally a scaffold and should be filled with project-specific decisions as they become known.

## Current decisions

- `0001-initial-architecture-boundaries.md`: Go Stage 0 compiler, explicit MIR, C reference backend, and deferred native backends.
- `0002-contract-source-of-truth.md`: authority of product, specification, MIR, target, CLI, and generated surfaces.
- `0003-stage-0-safety-boundary.md`: deliberately restricted ownership, borrowing, regions, traps, and FFI.
- `0004-atlas-semantic-snapshot-boundary.md`: one-way compiler-fact export and trust separation.
- `0005-stage-0-surface-and-module-boundary.md`: minimal syntax, one-file modules, alias imports, and restricted enums.
- `0006-c-abi-v0-and-first-conformance-target.md`: no native Fairy ABI and `x86_64-linux-gnu` as the first planned conformance target.
- `0007-runtime-stdlib-and-allocator-boundary.md`: compiler, thin-runtime, standard-library, and allocation ownership.
- `0008-versioned-contract-boundaries.md`: independent compatibility vector and fail-closed mismatch behavior.
- `0009-stage-0-grammar-and-parser-contract.md`: `.fy`, literal spelling, precedence, module identity, parser architecture, and bounded recovery.
- `0010-stage-0-semantic-fixture-contract.md`: phase-owned semantic fixtures, canonical assertions, byte spans, and diagnostic cascade policy.
- `0011-stage-0-resource-flow-fixture-contract.md`: whole-root resource states, cleanup obligations, moves, joins, and loop mismatch fixtures.
- `0012-stage-0-region-and-result-fixture-contracts.md`: Region origins and restrictions plus Result handling and propagation fixture boundaries.
- `0013-stage-0-mir-fixture-contract.md`: source-linked verified-MIR observations without freezing internal graph identity.
- `0014-resolved-package-graph-fixture-contract.md`: package-instance equality, direct dependency visibility, and explicit standard-library injection without public manifest syntax.
- `0015-mir-verifier-mutation-fixture-contract.md`: verified named seeds and isolated mutation recipes covering every current `Vxxxx` invariant.
- `0016-stage-0-interpreter-fixture-contract.md`: verified-MIR scalar returns and canonical traps as the first executable semantics oracle.
- `0017-stage-0-result-constructors-and-match.md`: qualified compiler-known Result construction and exhaustive shallow Result matching.
- `0018-stage-0-option-constructors-and-match.md`: qualified compiler-known Option construction and exhaustive shallow Option matching.
- `0019-stage-0-result-must-use-dataflow.md`: local, parameter, branch, loop, overwrite, and early-exit Result obligations.
- `0020-stage-0-region-mir-and-interpreter.md`: normal lexical Region placement, inner cleanup, reset trace, and `V1403`.
- `0021-stage-0-region-early-exit-cleanup.md`: scope-indexed Region reset for `return`, Result `?`, and nested exits.
- `0022-controlled-interpreter-io-harness.md`: opt-in in-memory stdout/stderr byte hooks with a shared output limit.
- `0023-explicit-checked-arithmetic-mir.md`: verified overflow and right-shift checks, exact trap edges, and the historical left-shift restriction later replaced by ADR 0103.
- `0024-stage-0-loop-control-and-cleanup.md`: nearest-loop `break` and `continue`, Flow obligations, and scope-indexed Region cleanup.
- `0025-verified-scalar-c-emission.md`: fail-closed verified-MIR emission for the provisional scalar ISO C11 transport.
- `0026-linux-scalar-c-differential-lane.md`: fail-closed Clang/GCC low/optimized native execution on Linux/amd64 with glibc.
- `0027-experimental-c-emission-cli.md`: deterministic raw C11 inspection from the canonical unstable CLI without file or toolchain side effects.
- `0028-experimental-linux-scalar-build-cli.md`: explicit Linux constrained-C executable builds through the shared C driver and embedded runtime.
- `0029-scalar-option-result-c-transport.md`: private ISO C11 structs and typed MIR flow for `Option<i32>` and `Result<i32, i32>`.
- `0030-executable-x86-64-linux-gnu-target-spec.md`: host-independent target facts, fail-closed maturity evidence, and deterministic experimental inspection.
- `0031-target-layout-engine-and-fixtures.md`: checked scalar, pointer, array, enum, nested-struct, offset, and padding layout execution without host assumptions.
- `0032-resource-definition-preservation-and-empty-drop-gate.md`: ordered resource definitions across AST, HIR, and MIR with fail-closed empty-drop execution.
- `0033-flat-i32-resource-c-transport.md`: target-laid-out private C11 transport for nonempty all-`i32` resources with empty drop bodies.
- `0034-resource-tagged-c-transport.md`: private target-laid-out C11 transport for resource-bearing built-in tags and typed Result propagation.
- `0035-native-drop-trace-differential.md`: closed source-linked drop instrumentation for exact interpreter-to-native C cleanup comparison without product-build observers.
- `0036-native-c-layout-probe.md`: deterministic private C11 probes for the nine valid target-layout cases across Clang/GCC and low/optimized profiles.
- `0037-sanitizer-assisted-c-backend-checks.md`: closed Clang/GCC ASan+UBSan execution of every admitted production C unit without exposing a product build profile.
- `0038-sysv-amd64-classifier-fixtures.md`: host-independent INTEGER/SSE/MEMORY classification, hidden returns, register exhaustion, and whole-aggregate rollback fixtures.
- `0039-bidirectional-cross-compiler-c-abi-probes.md`: separate Clang/GCC caller and callee objects in both directions for 56 low/optimized ABI crossings.
- `0040-provisional-system-heap-c-slice.md`: standard-library-owned fallible SystemHeap C transport and four-profile native probe without source API claims.
- `0041-failure-atomic-overlap-safe-buffer-c-slice.md`: initialized-byte Buffer growth, overlap append rebinding, overflow rollback, and zeroed resize C probe.
- `0042-single-active-region-arena-c-slice.md`: single-active Region generations, failure-atomic byte placement, reset invalidation, and Arena drop C probe.
- `0043-canonical-utf8-text-view-c-slice.md`: allocation-free canonical UTF-8 validation and code-point-boundary text slicing C probe.
- `0044-owned-string-and-cstring-c-slice.md`: failure-atomic Buffer-to-String ownership, immutable text views, and exact CString NUL invariants.
- `0045-linux-file-consuming-close-c-slice.md`: partial Linux I/O, write-all progress, sync, and never-retried consuming close.
- `0046-scalar-source-foreign-c-call.md`: exact-name scalar source FFI with unsafe provenance, fail-closed interpretation, and four-way native crossing.
- `0047-raw-pointer-target-scalar-foreign-call.md`: typed-null raw-pointer plus fixed-width and pointer-width source FFI transport with a separate four-way native crossing.
- `0048-source-callable-utf8-validation-transport.md`: exact allowlisted FY-to-real-stdlib-C empty-input UTF-8 validation crossing without a public wrapper claim.
- `0049-repr-c-output-record-source-transport.md`: flat source `@repr(C)` records, verified direct-local raw addresses, and a real stdlib C output mutation crossing.
- `0050-restricted-inert-enum-execution.md`: qualified restricted user-enum construction, exhaustive named-field matching, verified MIR, interpreter execution, and private C tag transport.
- `0051-source-link-name-transport.md`: distinct Fairy declaration and external C symbol identities through AST, typed HIR, verified MIR, generated C, and existing native probes.
- `0052-distinct-c-int-source-transport.md`: exact source `c::int` identity across foreign return/local/argument transport, target assertions, and a separate native crossing.
- `0053-distinct-c-size-t-source-transport.md`: exact ISO C `size_t` identity, target/header assertions, layout evidence, and foreign producer/consumer transport.
- `0054-posix-ssize-t-source-transport.md`: target-owned POSIX typedef metadata, exact negative `ssize_t` transport, and a separate native crossing.
- `0055-distinct-c-long-source-transport.md`: exact LP64-sensitive C `long` identity, signed target assertions, and a separate negative-value native crossing.
- `0056-complete-c-scalar-source-transport.md`: all sixteen target C scalar identities, one drift-checked backend table, and one twelve-type native family crossing.
- `0057-c-scalar-repr-c-system-heap-source-transport.md`: exact C-scalar fields in flat `@repr(C)` records, opaque `void` pointers, and a generated Fairy caller crossing the real SystemHeap C unit.
- `0058-nested-repr-c-buffer-source-transport.md`: dependency-ordered nested `@repr(C)` records and a generated Fairy caller crossing the real Buffer C unit.
- `0059-arena-region-source-transport.md`: exact self-linked Arena/Region records, one opaque `void **` output slot, and a generated Fairy caller crossing the real Arena C unit.
- `0060-cstr-validation-source-transport.md`: exact CStr view transport, nonempty bounded storage, failure-atomic output, and a generated Fairy caller crossing the real owned-text C unit.
- `0061-repr-c-struct-foreign-result-transport.md`: nonempty `@repr(C)` foreign results with generated-Fairy register and hidden-memory return crossings while aggregate parameters stay closed.
- `0062-owned-text-source-transport.md`: exact nested String/CString lifecycle calls from generated Fairy source with real by-value results, byte/terminator observers, and normalized drop.
- `0063-linux-file-source-transport.md`: exact Linux File/open/result records and generated Fairy calls through real write-all, sync, consuming close, reopen, partial-read, and EOF behavior.
- `0064-used-symbol-standard-library-auto-link.md`: one shared exact-symbol contract, called-symbol tracking, minimal dependency closure, and product-build linkage of provisional C standard-library units.
- `0065-compiler-owned-c-string-static-storage.md`: verified `Static` C-string literals, exact terminated storage, unsafe immutable data-address extraction, and native evidence counters.
- `0066-compiler-owned-byte-string-slice-storage.md`: exact byte-string storage, private non-null slice views, empty sentinels, typed-local data extraction, and native evidence counters.
- `0067-compiler-owned-utf8-string-static-storage.md`: decoded UTF-8 validation, distinct text operands, private non-null string views, typed-local data extraction, and native evidence counters.
- `0068-windows-msvc-target-neutral-host-smoke.md`: WSL-free target-neutral `.fy` compilation and temporary execution through a locally discovered x64 MSVC toolchain without a Windows ABI claim.
- `0069-provisional-diagnostic-explain-command.md`: exact prototype diagnostic lookup without source, toolchain, target, or release-stability claims.
- `0070-provisional-target-maturity-list.md`: deterministic known-target maturity and missing-evidence reporting without host probing or target promotion.
- `0071-checked-division-c-materialization.md`: volatile private divisor transport after verified MIR checks prevents host C compilers from rejecting unreachable constant-undefined division and remainder.
- `0072-defined-checked-arithmetic-c-construction.md`: checked negation, addition, subtraction, and multiplication construct results through defined unsigned C operations after the verified trap edge.
- `0073-executable-x86-64-windows-msvc-target-spec.md`: host-independent LLP64, Microsoft x64 stack, PE/COFF, unwind, and fail-closed maturity facts without promoting the Windows host smoke.
- `0074-windows-msvc-llp64-layout-fixtures.md`: separate host-independent Windows scalar, pointer, unavailable-typedef, aggregate-offset, and padding observations without claiming a native MSVC probe.
- `0075-windows-msvc-native-layout-probe.md`: deterministic non-POSIX C11 assertions for the eight valid Windows layout cases compiled and executed by local x64 MSVC at low and optimized profiles without maturity promotion.
- `0076-microsoft-x64-classifier-fixtures.md`: target-bound four-slot Microsoft x64 argument, aggregate-indirection, and hidden-result plans across fifteen host-independent cases without native-call promotion.
- `0077-windows-msvc-native-abi-probe.md`: same-version local MSVC caller/callee crossings for the twelve validated Microsoft x64 pass cases without source-level, independent, hosted, object, or unwind promotion.
- `0078-windows-pe-executable-structure-inspection.md`: bounded AMD64 PE32+ executable-header, section-range, and entry-point validation for five local MSVC images without COFF-object, unwind, debug, or distribution promotion.
- `0079-windows-msvc-coff-object-inspection.md`: bounded raw AMD64 COFF section, symbol, and relocation-structure validation for every local MSVC object without exact object-fixture, unwind, debug, hosted, or maturity promotion.
- `0080-windows-x64-unwind-structure-inspection.md`: static `.pdata`, runtime-function, version-1 prolog, version-2 epilog-boundary, handler-tail, and chained-tail inspection for five local PE images without semantic unwinder or maturity promotion.
- `0081-windows-x64-native-unwind-execution.md`: low/optimized native SEH dispatch, captured-context virtual walk, and abnormal termination-handler execution across three separately compiled MSVC C frames without adding Fairy C exception semantics or promoting target maturity.
- `0082-windows-msvc-target-c-emission.md`: explicit pre-emission Windows target selection, LLP64-aware scalar C output, and a checked-in FY-to-MSVC executable path without source-level Windows FFI or maturity promotion.
- `0083-windows-msvc-closed-stdlib-crossing.md`: one exact FY-to-project-C UTF-8 validation call through target-specific minimal linkage, with all other foreign symbols remaining fail-closed.
- `0084-windows-msvc-system-heap-crossing.md`: closed FY-to-SystemHeap allocation and release through the Windows x64 16-byte `malloc` alignment contract without opening broader runtime or linker inputs.
- `0085-windows-msvc-buffer-dependency-closure.md`: closed FY-to-Buffer construction, resize metadata, and drop with deterministic single SystemHeap dependency linkage at both MSVC profiles.
- `0086-windows-msvc-arena-region-lifecycle.md`: closed FY-to-Arena initialization, one Region allocation/reset, and final drop with deterministic SystemHeap dependency linkage at both MSVC profiles.
- `0087-windows-msvc-owned-text-lifecycle.md`: closed compiler-owned text to String/CString/CStr success lifecycles with deterministic SystemHeap, Buffer, and text-view dependency linkage at both MSVC profiles.
- `0088-experimental-windows-target-run-command.md`: direct temporary execution of the declared Windows target path with shared compilation, PE/unwind validation, exact result/trap reporting, and no WSL.
- `0089-standalone-windows-compiler-smoke.md`: temporary Go-built `fairyc.exe` process validation that compiles and executes a minimal FY Windows target without distribution or self-hosting claims.
- `0090-bounded-windows-process-stream-capture.md`: separate stdout/stderr capture under one synchronized output budget without inventing cross-stream order or enabling source-visible console I/O.
- `0091-windows-stdout-exec-target.md`: program-facing Windows target execution with separated streams, preserved process status, and one progress-reporting unsafe stdout write-all symbol.
- `0092-borrowed-view-byte-length.md`: zero-cost byte-length access for borrowed UTF-8 and byte-slice views through typed HIR, verified MIR, the interpreter, and private C transport.
- `0093-borrowed-view-internal-call-transport.md`: exact internal-function transport for borrowed UTF-8 and byte-slice views so narrow unsafe operations can be contained behind safe support-module helpers without inventing a public standard library.
- `0094-explicit-multi-source-windows-target-commands.md`: bounded entry-plus-repeated-source module closures for Windows target build, inspection-run, and program-facing execution without automatic discovery or linker-input widening.
- `0095-explicit-multi-source-interpreter-run.md`: the same bounded explicit FY module closure for fast verified-MIR interpreter execution without a backend, host process, or source discovery.
- `0096-explicit-multi-source-prototype-program-commands.md`: entry-plus-repeated-source closures for C inspection, Linux build, and target-neutral Windows host build/run without widening foreign or linker inputs.
- `0097-checked-immutable-byte-view-indexing.md`: read-only `&[u8]` indexing through contextual `usize`, verified dominating bounds checks, interpreter traps, and private C byte loads without opening slicing or text indexing.
- `0098-value-preserving-integer-casts.md`: four target-independent widening integer `as` pairs through explicit MIR, interpreter values, generated C casts, and a WSL-free Windows quickstart that returns its indexed byte.
- `0099-typed-scalar-comparisons.md`: exact matching-type equality and integer ordering through typed MIR, the interpreter, generated C, and the WSL-free Windows quickstart length guard.
- `0100-checked-unsigned-arithmetic.md`: matching-type trapping `u8`, `u32`, and `usize` addition, subtraction, and multiplication through typed MIR, interpreter, C11, and WSL-free Windows execution.
- `0101-checked-unsigned-division.md`: matching-type `u8`, `u32`, and `usize` division and remainder with typed zero-divisor traps and exact C divisor materialization.
- `0102-typed-bitwise-and-checked-right-shift.md`: typed unary complement, matching-type binary bitwise operations, and exact-width checked arithmetic/logical right shift for `i32`, `u8`, `u32`, and `usize` through MIR, interpreter, C11, and WSL-free Windows execution.
- `0103-checked-left-shift.md`: matching-type exact-width left shift with stable invalid-amount-before-overflow precedence and defined C11 construction.
- `0104-borrowed-view-emptiness.md`: zero-cost boolean emptiness metadata for borrowed UTF-8 and byte views through interpreter, C11, and Windows execution.
- `0105-checked-immutable-byte-view-slicing.md`: end-exclusive zero-copy `&[u8]` subviews with preserved borrow origin and dominating `bounds` checks before pointer derivation.
- `0106-always-checked-assert.md`: profile-independent `assert(condition);` with exact bool typing, verified failure edges, and stable `assertion-failed` traps.
- `0107-omitted-immutable-byte-view-slice-bounds.md`: prefix, suffix, and whole immutable byte-view slices normalized to the existing checked MIR path without receiver re-evaluation.
- `0108-integer-literal-range-validation.md`: radix-correct range validation at the semantic boundary with stable `E1013` rejection before MIR.
- `0109-direct-i32-minimum-literal.md`: direct decimal, hexadecimal, or binary `i32` minimum folding plus allocation-limited semantic magnitude comparison.
- `0110-scalar-fixed-array-execution.md`: provisionally bounded copy-scalar fixed arrays with nonempty list construction, internal value calls, checked indexing, and private C wrappers.
- `0111-scalar-fixed-array-length.md`: exact zero-cost `usize` length observation for admitted fixed arrays without opening layout reflection or slicing.
- `0112-fixed-u8-array-byte-view-borrow.md`: contextual shared borrowing of admitted fixed `u8` arrays as immutable byte views without opening mutable slices or public ABI.
- `0113-branch-aware-safe-borrow-reassignment.md`: provenance-carrying safe-borrow reassignment with reachable branch joins, loan generations, and loop-carried reassignment kept fail-closed.
- `0114-interprocedural-borrow-return-summaries.md`: fixed-point internal return-origin summaries that map one exact parameter, region parameter, or static source through borrow-returning calls.
- `0115-loop-carried-safe-borrow-provenance.md`: canonical loan-site generations plus loop-header, `continue`, and `break` environment joins for safe-borrow reassignment in `while`.
- `0116-stage-0-source-effects.md`: closed external-capability declarations, canonical typed-HIR sets, caller-superset checks, and unsafe-independent effect diagnostics.
- `0117-effect-annotated-windows-console-wrapper.md`: end-to-end `console` authority from the raw Windows stdout declaration through safe FY wrappers and `main`.
- `0118-copy-tagged-transport-and-typed-console-result.md`: target-laid-out copy-only built-in tags and a progress-preserving typed Windows console result.
- `0119-flat-copy-resource-c-transport.md`: move-only target-laid-out resource transport over Fairy copy scalars and raw pointers, with schema-feature drift enforcement.
- `0120-verified-nonempty-resource-drop-glue.md`: compiler-private typed drop functions linked through HIR, MIR, interpreter, and constrained C.
- `0121-resource-drop-effects.md`: canonical resource-drop effects propagated through implicit cleanup edges while complete moves remain effect-free.
- `0122-windows-file-consuming-close-c-slice.md`: target-bound Win32 File state, partial I/O, progress-preserving write-all, sync, consuming close, and best-effort drop through an inspected native MSVC probe.
- `0123-mutable-parameter-and-opaque-borrow-cast.md`: callee-local `mut` parameters, verifier-retained opaque owner addresses, and borrowed CStr parameter data extraction for checked raw wrappers.
- `0124-safe-windows-file-fy-lifecycle.md`: effect-checked move-only FY File wrappers with typed errors, bounded safe buffer I/O, consuming close, and native implicit-drop proof on Windows.
- `0125-mutable-fixed-u8-array-byte-view.md`: contextual exclusive fixed-array byte views with private mutable C transport and a length-generic safe Windows File read wrapper.
- `0126-checked-mutable-indexed-assignment.md`: bounds-before-value fixed-array and mutable-byte-view stores with verified alias and value-copy semantics.
- `0127-experimental-verified-mir-check.md`: explicit source-closure validation through MIR verification without execution, backend, toolchain, or artifact writes.

Product scope remains owned by `docs/product/02-spec.md`. Validation names are owned by `VALIDATION.md`.

## Review Blockers

- The change invents a product domain without a source.
- The change weakens validation or skips required evidence.
- The change relies on generated, cache, or build output as source truth.
