# Canonical UTF-8 Text View C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation source: `internal/cstdlib/`

## Context

`&str` cannot be a byte slice with a nicer name. Validation must reject non-shortest encodings,
UTF-16 surrogate values, values above U+10FFFF, obsolete five-byte leaders, stray continuation
bytes, and truncated sequences. Slicing must also reject byte offsets inside a code point while
preserving the input view on failure.

## Decision

1. The provisional text view is an immutable byte pointer and length whose constructor performs a
   complete allocation-free UTF-8 validation pass.
2. ASCII, including NUL, is valid `str` content. NUL restrictions belong only to C string types.
3. Two-byte sequences start at C2; E0 and F0 use raised second-byte minima; ED excludes surrogate
   encodings; F4 caps the second byte at 8F; leaders above F4 are rejected.
4. Validation accepts null storage only for zero length and normalizes a successful empty view to
   null storage. Failure never mutates the output record.
5. Text slicing requires ordered in-range byte offsets and rejects a start or end positioned on a
   continuation byte. It revalidates the input record defensively but allocates and copies nothing.
6. Empty-view pointer identity is not observable. Borrow origin remains compiler analysis data, not
   a pointer-identity promise in the C helper.
7. A closed native probe covers mixed ASCII/NUL/two/three/four-byte text, valid slicing, interior
   boundary rejection, reversed and out-of-range bounds, empty input, stray continuations,
   overlong encodings, surrogates, values above U+10FFFF, truncation, obsolete leaders, invalid
   pointer/length pairs, and unchanged failure outputs.
8. `fairyc.native-c-report/v9` adds `textViewExecutions`, expected to be four only after a successful
   hosted run. No hosted receipt has been observed.
9. The helper is not a compiler intrinsic, source-visible API, owned `String`, locale conversion,
   normalization, grapheme operation, or Unicode database.
10. ADR 0048 later exposes only the canonical validator as an internal ABI-major transport and
    links its empty-input path from Fairy source. That does not create the public text wrapper.

## Consequences

- Validation and defensive slicing are linear in byte length; a validated-source fast path may be
  added later only with a representation proof and measurements.
- Code-point indexing, normalization, case conversion, and grapheme segmentation remain excluded.
- Owned text must build on Buffer without exposing mutable bytes that could violate this invariant.
