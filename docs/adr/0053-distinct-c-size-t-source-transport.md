# Distinct `c::size_t` Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Target source: `internal/target/spec.go`
- C backend source: `docs/mir/c-backend-safety-v0.md`

## Context

`usize` and `uintptr_t` happen to share the first target's width with C `size_t`, but they are not
the same source or ABI type. The target table previously lacked a `size_t` entry, so accepting the
source spelling without a target-owned layout and required-header contract would only move the
aliasing bug into generated C.

## Decision

1. The source type `c::size_t` is distinct from `usize` and maps to target key `c.size_t`. For
   `x86_64-linux-gnu`, that key is an unsigned 8-byte scalar with 8-byte alignment.
2. Generated C includes `<stddef.h>` only when another admitted feature or `c::size_t` needs it,
   spells the type exactly `size_t`, and emits target-derived size/alignment plus unsignedness
   assertions.
3. Like the first `c::int` slice, this is transport-only. A foreign call may produce `c::size_t`, a
   typed local may retain it, and a later foreign call may consume it. Literals, operators, casts,
   interpreter execution, raw pointers to C scalars, and aggregate fields remain deferred.
4. The layout manifest adds `pass.c-size-t`, raising the current total to fifteen cases and the
   native-probe input to ten validated pass cases. The recursive C11 probe uses exact `size_t`.
5. A dedicated hand-written C producer/consumer probe expects four Clang/GCC low/optimized
   executions. `fairyc.native-c-report/v17` adds `foreignCSizeTCallExecutions` without changing the
   eighteen production emission cases or their 72/36 counts.
6. `c::ssize_t` remains excluded by this decision. ADR 0054 later adds it as a distinct POSIX
   target-library type requiring `<sys/types.h>` and signed return/error transport, not as the
   signed counterpart of an ISO C `size_t` language type.
7. ADR 0282 later makes the shared header rule explicit: structs, enums, resources, and generic
   tagged records also require `<stddef.h>` whenever they emit `offsetof` assertions. The
   `c::size_t` mapping and transport contract remain unchanged.

## Consequences

- FFI wrappers can transport a real C byte/count type without silently converting it to Fairy
  pointer-width arithmetic.
- Adding one target scalar changes target JSON data and layout evidence but does not promote target
  maturity or claim broad source `c.*` support.
- The earlier accidental `*c::int` backend acceptance is closed; pointer forms need a separate
  mutation/aliasing contract and native probe.
