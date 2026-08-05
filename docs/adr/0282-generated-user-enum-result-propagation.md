# ADR 0282: Execute generated user-enum Result propagation

- Status: Accepted
- Date: 2026-08-06
- Product source: `docs/product/02-spec.md`
- MIR source: `docs/mir/c-backend-safety-v0.md`
- Depends on: ADR 0019, ADR 0029, ADR 0280

## Context

ADR 0280 admits restricted inert user enums as private built-in Result payloads and proves direct
construction, extraction, and switching. MIR already lowers `input?` by branching once, extracting
the exact error payload, reconstructing the enclosing function's declared Result type, running exit
cleanup, and returning that reconstructed value. Existing native fixtures cover scalar and resource
errors but did not prove that an enum error survives both tagged records and the propagation return.

A compile-only fixture would not distinguish correct enum transport from a wrong tag, payload copy,
or optimized-build miscompile. The first native execution also exposed a hidden preamble dependency:
enum and generic tagged layout assertions use `offsetof`, but `<stddef.h>` was previously included only
when another struct, resource, `c::size_t`, or Region happened to require it. The missing evidence
therefore spans interpreter semantics, deterministic C emission, self-contained C headers,
compilation, linking, and execution at both Windows optimization levels.

## Decision

1. Add one source-to-C fixture with `Result<i32, Failure>` where `Failure` is a restricted inert enum
   containing both payloadless and named-payload variants.
2. Execute `.ok(19)` through `?` and return `.ok(20)`. Separately execute
   `.err(.Code { .status = 22 })` through `?`, reconstruct the enclosing Result, extract the enum
   payload, and switch exhaustively. The combined observable result is 42.
3. Run the same generated-Fairy program under low and optimized Windows MSVC. Both profiles must
   return 42 after real C compilation and linking.
4. Keep MIR lowering and propagation transport unchanged. Make C preamble ownership explicit by
   including `<stddef.h>` whenever emitted structs, enums, resources, or generic tagged records use
   `offsetof`, independent of unrelated feature presence.
5. Add a unit regression requiring the header for an enum-only generic Result unit.
6. Language remains `fairyc-language/8`, SDK remains `0.0.0-stage0.25`, runtime ABI remains
   `fairyc-runtime/0`, and no public or private native symbol changes.

## Consequences

- `Result<T, UserEnum>` now has direct success and propagated-error evidence across interpreter,
  generated C, and both Windows native optimization profiles.
- Propagation preserves the enum's private tag and active named payload instead of mapping it to an
  integer code or target-specific constant.
- Enum-only and generic-tag-only generated units no longer depend on an unrelated aggregate to make
  `offsetof` available.
- ADR 0283 later admits restricted user enums inside `Option`; nested enums, resource-bearing user
  enums, and mismatched propagation error types remain outside the admitted boundary.
- The Windows smoke source constants move to a focused Go file so this proof does not add another
  full embedded Fairy program to the existing oversized runner entry file.

## Evidence

- `spec/fixtures/c-emission/cases/pass/copy-result-enum-propagation/src/main.fy` owns the interpreter
  and deterministic source-to-C case.
- `internal/cbackend/emit.go` owns the self-contained `<stddef.h>` preamble condition.
- `cmd/fairyc-windows-host-smoke/enum_result_sources.go` owns the bounded native source inputs.
- `fairyc_windows_host_smoke` compiles, inspects, links, and executes both direct and propagation
  user-enum Result programs under low and optimized MSVC.
