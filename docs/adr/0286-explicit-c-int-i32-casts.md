# ADR 0286: Admit explicit `c::int` and `i32` casts

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0007-name-resolution-and-typing-v0.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Depends on: ADR 0030, ADR 0052, ADR 0073, ADR 0098

## Context

Fairy C preserves `c::int` as a target-owned type distinct from `i32`, so imported C values cannot
silently inherit Fairy arithmetic or C promotion rules. Both current executable targets nevertheless
declare `c.int` as a signed 32-bit scalar with the exact `i32` range. Without an explicit conversion,
even a thin checked FFI wrapper cannot inspect a C integer with ordinary Fairy operations or return a
computed `i32` value to a C consumer without another hand-written shim.

Opening every C scalar would require target-specific range, signedness, floating-point, and narrowing
policy. This slice needs only the one representation equality already established by both target
specifications.

## Decision

1. Admit safe explicit `i32 as c::int` and `c::int as i32` expressions. Keep the two types nominally
   distinct in HIR, MIR, foreign signatures, locals, and generated C.
2. Require the selected target to define `c.int` as signed 32-bit with the exact `i32` range. The
   current Linux GNU and Windows MSVC targets satisfy that contract. A future target that does not
   satisfy it must reject the cast rather than truncate, wrap, or reinterpret a value.
3. Lower each direction through the existing verified `scalar-cast` operation. The MIR verifier
   checks the exact source/result pair and operand kind; it does not infer conversion from storage
   width alone.
4. Emit an explicit C cast to `int32_t` or `int` and retain the target-derived `c.int` size,
   alignment, and signedness assertions whenever the feature is used.
5. Execute pure cast round trips in the MIR interpreter. Foreign calls themselves remain
   unavailable to the interpreter.
6. Prove the boundary with one generated-Fairy caller linked to a separate hand-written C producer
   and consumer. The caller round-trips 42 through `c::int -> i32 -> c::int`; run the crossing under
   low and optimized x64 MSVC profiles and require result 42 without unrelated runtime helpers.
7. Cut the language revision from `fairyc-language/9` to `fairyc-language/10`. Keep SDK source and
   public API version `0.0.0-stage0.26` and runtime ABI `fairyc-runtime/0`; update the SDK language
   requirement, current manifests, lock expectations, and whole-package cache vectors.

## Consequences

- Fairy wrappers can inspect and compute with the most common exact C integer without erasing its
  boundary identity or adding an allocation, runtime check, or helper call.
- Existing `/9` programs remain source-compatible after updating their exact manifest pin.
- There is no implicit conversion, contextual `c::int` literal, C arithmetic, comparison, truthiness,
  default argument promotion, or C operator behavior.
- `c::uint`, `c::long`, `c::size_t`, every floating C scalar, pointer/integer casts, narrowing, and
  signedness-changing conversion remain rejected pending their own evidence and policy.
- The language identity, SDK package instance, dependent locks, and package-cache keys change. The
  runtime ABI, target schema, MIR version, generated-C dialect, and public native ABI do not.

## Evidence

- `internal/hir/hir.go` owns the closed admitted cast-pair table.
- `internal/semantic/semantic_test.go` owns positive typing and the negative non-`c::int` boundary.
- `internal/mir/verify.go` and `internal/mir/lower_test.go` own operand/result verification.
- `internal/interpreter/interpreter.go` owns pure execution.
- `internal/cbackend/emit.go` owns exact C emission and `c.int` target-feature recording.
- `spec/fixtures/semantic/cases/pass/c-int-explicit-cast/` and
  `spec/fixtures/c-emission/cases/pass/c-int-explicit-cast/` own stable stage fixtures.
- `cmd/fairyc-windows-host-smoke/c_int_cast_probe.go` owns separate-translation-unit low/optimized
  MSVC execution.
