# ADR 0248: Public allocation-free UTF-8 validation

- Status: Implemented local compiler, SDK, and native-smoke gate
- Date: 2026-07-31
- SDK source: `internal/sdkstdlib/sources/std/text.fy`
- C backend: `internal/cbackend/`
- Runtime source: `internal/cstdlib/stage0_text_view.c`
- Example: `examples/std_text_quickstart/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0043, ADR 0048, ADR 0067, ADR 0092, ADR 0093, ADR 0147, ADR 0247

## Context

The target text-view component already performed complete UTF-8 validation without allocation, and
the compiler already represented immutable byte and text views with retained borrow origins. The
public SDK nevertheless accepted only compiler-known `ref str` input. A safe caller receiving bytes
from a file, console, protocol, or embedded record could not prove those bytes as text without
declaring the private C predicate or copying an example wrapper.

## Decision

1. SDK `0.0.0-stage0.9` adds
   `std::text::validate_utf8(ref u8[]) -> Result<ref str, TextError>`.
2. Success returns an immutable text view over the exact input bytes. It performs no allocation,
   copy, normalization, terminator insertion, locale conversion, or replacement of invalid input.
3. The returned view retains the input slice's shared borrow origin. Safe code cannot mutate or end
   the backing storage while the result or a bound success payload remains live.
4. Validation calls the existing complete `fy_std_text_utf8_valid_v0` predicate. Invalid input
   returns `TextError` with text status `2` and allocation status `0`; the input bytes remain
   unchanged. ASCII NUL is valid UTF-8 and is not treated as C-string termination.
5. Byte slices do not convert implicitly to text. Callers must name validation and handle both
   `Result` variants.
6. The constrained C backend admits `ref str` only as a non-owning payload of copy-tagged
   `Option`/`Result` transport. Its layout is the existing private `fy_str { data, length }` pair;
   this does not admit arbitrary borrow payloads or establish a public C ABI.
7. The language revision remains `fairyc-language/7` and the runtime ABI remains
   `fairyc-runtime/0`. The SDK source digest, package identity, public-interface hash, generated
   locks, dependent identities, and whole-package cache keys change.

## Evidence and limits

The backend regression executes a `Result<ref str, TextError>` success payload through the MIR
interpreter and requires the exact target layout and generated C payload type. The public
quickstart validates nonempty multibyte input, observes its exact eight-byte length, rejects one
invalid byte with status `2`, observes allocation status `0`, and then exercises the existing owned
String and CString surface. Windows development and release builds execute the safe quickstart;
the configured MSVC smoke inspects and executes both PE images.

Semantic regressions separately prove that the returned `Result` keeps the input owner borrowed
while either the wrapper or its `.ok` payload binding remains live, releases the loan after the last
use, rejects a local view hidden inside `.ok`, and preserves the same origin through `?`. These are
Flow guarantees; successful native transport alone is not treated as lifetime proof.

This decision does not add text indexing or slicing, Unicode scalar or grapheme counting,
normalization, case conversion, lossy decoding, streaming validation, mutable text, implicit
conversion, public view layout, or a foreign borrowed-view ABI. The native evidence is local; it
does not prove hosted same-commit CI or promote another target.
