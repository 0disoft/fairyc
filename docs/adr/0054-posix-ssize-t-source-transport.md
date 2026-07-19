# POSIX `c::ssize_t` Source Transport

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Target source: `internal/target/spec.go`
- Runtime precedent: `internal/cstdlib/stage0_linux_file.c`

## Context

`ssize_t` is not an ISO C11 type and cannot be modeled as an `isize` or `intptr_t` alias. The
provisional Linux File implementation already relies on the POSIX typedef and preserves negative
kernel results before converting successful counts to `size_t`, but the source FFI and target JSON
had no way to state which header and availability profile owned that name.

## Decision

1. Target scalar facts gain optional `header` and `availability` metadata. Language scalars cannot
   carry it; a C typedef must provide both. The current closed availability values are `iso-c11`
   and `posix.1-2008`.
2. Existing `c.size_t` declares `stddef.h`/`iso-c11`. New `c.ssize_t` declares
   `sys/types.h`/`posix.1-2008`, signed 8-byte size, and 8-byte alignment for
   `x86_64-linux-gnu`. This is target data, not a portable POSIX claim.
3. Source `c::ssize_t` remains distinct from `isize`. Generated C defines
   `_POSIX_C_SOURCE 200809L` before headers, includes `<sys/types.h>`, spells `ssize_t` exactly, and
   asserts target size, alignment, and signedness.
4. The executable slice is transport-only: a foreign producer may return a negative `ssize_t`, a
   typed MIR local retains it, and a foreign consumer receives it. Fairy literals, operators,
   casts, raw C-scalar pointers, aggregate fields, and interpreter host bindings remain closed.
5. The layout manifest adds `pass.c-ssize-t`, raising the current total to sixteen cases and the
   target-specific native probe to eleven pass cases. That probe uses the same POSIX feature macro
   and header.
6. A hand-written C producer returns `-41`; its consumer requires the negative value and returns
   `42`. `fairyc.native-c-report/v18` adds `foreignCSSizeTCallExecutions`, expected to be four only
   after both compiler directions and profiles execute successfully.

## Consequences

- The target description can distinguish C keywords from ISO typedefs and target-library typedefs
  without embedding a host path or installed libc version.
- Generated translation units using only ISO C families remain extension-free; POSIX exposure is
  explicit and source-feature driven.
- This does not yet expose Linux `read` or `write` as public Fairy APIs. Buffer pointers, partial
  counts, errno conversion, effects, and resource ownership still need their own wrapper slice.
