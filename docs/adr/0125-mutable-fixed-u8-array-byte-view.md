# ADR 0125: Borrow a mutable fixed `u8` array as a byte view

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Language sources: `docs/language/0002-borrows-and-return-origins.md`, `docs/language/0005-surface-syntax-v0.md`
- MIR sources: `docs/mir/mir-v0.md`, `docs/mir/c-backend-safety-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`

## Context

ADR 0112 connected fixed `[u8; N]` storage to immutable `&[u8]`. ADR 0124 then proved a safe FY
Windows File lifecycle, but its first read wrapper was tied to `[u8; 16]`. That avoided inventing a
mutable slice before its ownership and backend transport existed, at the cost of one wrapper per
buffer length.

The next connection needs to pass a mutable fixed array with its exact length, preserve one
exclusive loan through the call, and let a checked wrapper derive a writable raw byte pointer. It
must not silently open mutable indexing, slicing, resizing, temporary-array borrows, foreign safe
references, retained pointers, or a public slice ABI.

## Decision

1. When `&mut [u8]` is already the expected destination or internal-call argument type, a direct
   `&mut` borrow of one mutable admitted `[u8; N]` place contextually produces that view. A bare
   array value, shared or immutable place, literal, call result, index result, other element type,
   or inferred borrow does not convert.
2. The view retains the fixed array's hidden origin and exact canonical `N`. Flow treats it as the
   existing exclusive whole-owner loan: the array cannot be read, assigned, moved, or borrowed
   incompatibly while the view is live, and a callee-local view cannot escape.
3. This slice admits internal parameters and zero-cost `.len` observation. It does not add mutable
   indexing, indexed assignment, slicing or subviews, iteration, array resizing, return of a local
   view, safe foreign-reference parameters, or implicit raw-pointer conversion.
4. Typed HIR uses `array-view-borrow`. MIR uses the existing `array-borrow-view` operation with one
   local `[u8; N]` owner, result `&mut [u8]`, and operator `&mut`. The verifier admits only the exact
   shared pair `&`/`&[u8]` or mutable pair `&mut`/`&mut [u8]`.
5. Generated C represents the mutable view privately as
   `fy_mut_byte_slice { uint8_t *data; uintptr_t length; }`. Construction takes the verified fixed
   array local's data address and canonical `N` without copying or allocating. The immutable
   descriptor remains `const`-qualified and distinct.
6. Inside an explicit unsafe block, `byte-slice-data-address` may derive `*mut u8` only from an
   `&mut [u8]` owner. A shared view may derive only immutable `*u8`; no operation strengthens
   mutability. Foreign declarations continue to reject safe references, so wrappers pass only the
   checked pointer and length.
7. The interpreter carries the view's bytes and metadata for the currently observable internal
   `.len` behavior. It exposes no mutation primitive. Native mutation evidence belongs to the
   generated-C target path, where the Windows File read call is synchronous and does not retain the
   pointer.
8. The example-level Windows File `read` wrapper now accepts `&mut [u8]`, passes `buffer.len` and
   `buffer as *mut u8`, and lets callers use any admitted mutable fixed `[u8; N]`. The raw C entry
   accepts `uint8_t *`; the general safe API, Unicode paths, and portable File surface remain later
   work.

## Consequences

- Safe FY code can reuse one buffer-taking wrapper across admitted fixed byte-array lengths.
- The first mutable byte view is deliberately transport-first: it proves exclusivity, length, and
  writable pointer derivation without pretending that safe source mutation operations exist.
- The private descriptor, the current `1..=4096` fixed-array admission ceiling, and the Windows
  example are pre-1 implementation contracts, not a stable Fairy or C ABI.

ADR 0126 amends the transport-only limitation in Decision 3 and Decision 7 by adding checked `u8`
reads and indexed stores through `&mut [u8]`. It preserves this ADR's origin, exclusivity, private
descriptor, raw-pointer, no-slicing, no-resizing, and no-public-ABI boundaries.

## Verification

- semantic, MIR, interpreter, and constrained-C tests cover contextual construction, exact
  mutability pairs, length observation, and forbidden strengthening;
- semantic, interpreter, and C-emission manifests contain a positive mutable-view case;
- the Windows host smoke compiles the real `.fy` wrapper through the Go Stage 0 compiler, emits C,
  builds with local x64 MSVC, fills the caller's fixed array, observes EOF, and returns 42 at both
  profiles without WSL;
- Linux-only Clang/GCC, glibc, ASan, and UBSan execution remains unobserved on this Windows host.
