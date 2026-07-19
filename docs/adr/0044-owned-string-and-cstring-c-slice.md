# Owned String and CString C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation source: `internal/cstdlib/`

## Context

Owned text must preserve two different invariants: `String` owns valid UTF-8 while `CString` owns a
bounded byte sequence with exactly one trailing NUL and no interior NUL, without promising UTF-8 to
C consumers. A consuming Buffer conversion must move storage only after validation succeeds.

## Decision

1. Provisional `String` and `CString` each own a Buffer. Their C representations are private
   standard-library transport, not stable language ABI.
2. String copy accepts a validated text view, allocates exact initial storage, and copies bytes.
   Append accepts only valid text and delegates overlap-safe failure-atomic growth to Buffer.
3. Buffer-to-String conversion validates only the initialized prefix. On invalid UTF-8, both the
   input Buffer and destination record remain unchanged. On success, the Buffer is moved and the
   input is replaced with the canonical empty state.
4. String exposes only an immutable text view and drop. No mutable byte view or raw-parts API is
   added.
5. CStr validation receives a bounded storage length, requires the last byte to be NUL, rejects any
   earlier NUL, and exposes a length excluding the terminator. It makes no UTF-8 or locale claim.
6. CString conversion from text rejects interior NUL, checks terminator-length overflow before
   scanning, copies the text, and appends exactly one terminator. Failure leaves the destination
   unchanged and releases any private partial replacement.
7. Owned-text operations return a closed result containing a high-level status and the underlying
   allocation status where applicable. This is an internal transport for later `Result` mapping,
   not a source-visible error representation.
8. The native probe covers String copy/append/view/drop, self-source append, invalid Buffer rollback,
   successful consuming Buffer move, CString creation/view/drop, interior-NUL rollback, invalid CStr
   rejection, terminator presence, and empty CString.
9. `fairyc.native-c-report/v10` adds `ownedTextExecutions`, expected to be four only after a
   successful hosted run. No hosted receipt has been observed.
10. ADR 0060 later supersedes only the fully source-disconnected CStr-validation boundary: an
    internal generated-Fairy caller now crosses the exact view record into the real bounded C
    validator. Owned String/CString construction and result mapping remain source-disconnected.

## Consequences

- The source wrapper must turn failed consuming conversion into an error resource that still owns
  the unchanged Buffer; the C helper alone does not define source Result layout.
- String and CString capacity remain unobservable and may differ from logical length.
- Unicode normalization, case mapping, graphemes, locale conversion, and arbitrary raw C-string
  adoption remain excluded.
