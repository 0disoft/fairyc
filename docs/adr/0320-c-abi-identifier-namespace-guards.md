# ADR 0320: C ABI identifier namespace guards

- Status: Accepted
- Date: 2026-08-08
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0049, ADR 0137

## Decision

1. Validate every verbatim `@repr(C)` tag, field, and external link symbol against its actual C
   namespace before emitting a translation unit. ASCII C11 spelling and keyword rejection remain
   necessary but are no longer sufficient.
2. Reject every leading underscore for file-scope tags and external symbols. For member names,
   reject double underscore and underscore followed by an uppercase letter while retaining the
   ordinary `_lowercase` member form.
3. Reject `<stdint.h>` and `<stddef.h>` macro names used by the generated unit, the backend-owned
   `FY_` macro prefix, generated scalar names, and the exact private record tags `fy_region`,
   `fy_byte_slice`, `fy_mut_byte_slice`, and `fy_str`.
4. Preserve the closed target-specific `fy_std_` external-symbol allowlist and existing public
   `fy_std_*` record tags. This decision does not open either prefix generally.
5. Keep language, SDK, and runtime ABI identities unchanged. This is a constrained C-emitter
   correctness fix: previously accepted colliding names could not reliably form valid C11.

## Evidence

Source-to-verified-MIR unit regressions require the C emitter to reject representative header macro,
implementation-reserved, backend tag, and backend macro collisions for records, fields, and link
symbols. The 45th C-emission fixture preserves a nearby valid public record and `_lowercase` member,
so the guard cannot collapse all underscore-prefixed members. The configured native fixture lane
remains the independent Clang/GCC syntax and execution oracle for accepted output.

## Consequences

Programs using a newly rejected verbatim C name now stop at the C-emitter capability boundary
instead of producing a translation unit that can be macro-expanded or redeclared incorrectly.
Fairy-private names remain mangled independently. This does not import C headers, promise arbitrary
foreign identifiers, add a target symbol, or change record layout and calling convention.
