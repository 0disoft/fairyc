# Checked byte encoding and compact hex decoding

This example uses SDK `0.0.0-stage0.86` decimal parsing and
`std::num::u8_from_i64` to contrast
exact byte conversion with unsigned little-endian field encoding, and adds
compact hex decoding back to i64. Its inline
scalar array field, direct `ref mut` field view and direct field iteration also
require the compiler admissions in
[ADR 0468](../../docs/adr/0468-inline-scalar-array-fields.md),
[ADR 0469](../../docs/adr/0469-direct-array-field-views.md) and
[ADR 0470](../../docs/adr/0470-direct-array-field-iteration.md). From this directory,
with a compiler implementing those admissions and bundling that SDK:

```sh
fairyc check
fairyc run -- byte 255
fairyc run -- le 256 2
fairyc run --release -- le 9223372036854775807 8
fairyc run -- decode-le 0001
fairyc run --release -- decode-le ffffffffffffff7f
```

The five runs print, respectively:

```text
ff
00 01
ff ff ff ff ff ff ff 7f
256
9223372036854775807
```

`byte VALUE` requires exactly one signed decimal argument and converts the full
value. `byte 256` fails with `VALUE: outside byte range 0..255`.
`le VALUE WIDTH` requires exactly two decimal arguments. WIDTH must be 1..8;
VALUE must be nonnegative and fit that unsigned field. `le 256 1` fails rather
than truncating. `le 0 8` produces eight zero bytes. This example accepts values
only up to i64 maximum; it is not a general u64 or signed two's-complement codec.

`decode-le HEX` requires exactly one argument of 2..16 ASCII hex digits, read as
little-endian byte pairs and printed as a nonnegative decimal i64 followed by LF.
Upper, lower and mixed case digits are accepted; a prefix, sign, whitespace and
separators are not. Validation runs empty, then longer than 16 digits, then odd
length, then every hex digit, then numeric range, so the first failure wins.
`decode-le 0001` prints `256` and `decode-le ffffffffffffff7f` prints
`9223372036854775807`. Full-width input such as `ffffffffffffffff` reports
`HEX: value is outside nonnegative i64 range` without trapping, because the range
precheck runs before each multiply and add. The encoder renders hex with spaces
between bytes; remove those spaces when using its output as the `HEX` argument.

Each numeric argument is at most 20 UTF-8 bytes and follows `-?[0-9]+`.
Leading zeros and `-0` are accepted. Empty values, a bare minus, plus signs,
whitespace, decimal points and values outside i64 are rejected. `numbers::parse`
delegates to `std::num::parse_i64_decimal` and maps its errors to the existing
caller errors. The SDK handles i64 minimum without negating it. The 20-byte
argv limit remains example policy. Competing failures retain first-failure
ordering: `9223372036854775808x` reports range, while `922337203685477580x8`
reports syntax.

For `byte` and `le`, CLI validation order is mode/arity, VALUE parsing, then (for `le`) WIDTH parsing
and range, then codec policy. Thus `le -1 0` reports the width error. `main` reads
the mode into a 9-byte buffer and matches `byte`, `le` or `decode-le`; the
`byte`/`le` numeric parsing, error ordering and output stay unchanged, and only
the usage text for errors expands to
`usage: checked-byte-codec byte VALUE | le VALUE WIDTH | decode-le HEX`. The CLI
hex argument reader has a 16-byte capacity and rejects longer input as the
`TooLong` failure.

The reusable
example-local `codec::encode_le` checks sign, width, capacity and excess high
bits before writing. It extracts `remaining % 256`, converts only that digit,
then divides by 256. Invalid fields leave the caller's buffer unchanged;
successful fields leave all bytes beyond WIDTH unchanged. The SDK conversion
still does its own conversion work; no speed claim is made.

`codec::decode_le` takes a shared raw byte view of length 1..8 and returns
`Result<i64, DecodeError>`; `codec::decode_hex_le` takes a shared ASCII hex view
and returns `Result<i64, DecodeError>`. Decoding reads from the most significant byte and
checks `value > (i64 maximum - digit) / 256` before each multiply and add, so
high-bit input never wraps or traps. `decode_hex_le` constructs a local
`Packet` containing an eight-byte array with `length = size`, fills it through a direct `ref mut` field view
using `?` on each nibble, and hands it to `decode_le`; the caller's buffers are
never mutated. The existing `encode_le` API is unchanged.

`codec::Packet { u8[8] bytes; usize length; }` carries the encoded bytes and their
length. `main` initializes it as `{ .bytes = [0; 8], .length = 0 }` and sets
`.length` only after a successful encode, so a failed `le` leaves the packet
untouched. `encode_le` is unchanged and is called with `ref mut packet.bytes[..width]`,
using a `width` local captured before that exclusive owner loan. The renderer takes
`ref codec::Packet` and iterates `packet.bytes[..packet.length]` with a shared
`for (let u8 byte in ...)` loop, rendering into the existing 24-byte scratch output.

Successful `byte` and `le` commands print lowercase two-digit hex bytes separated by single spaces and
terminated by LF, and exit 0. No stdout is written until complete input and
encoding validation succeeds. Input/field failures print a `checked-byte-codec:`
diagnostic to stderr and exit 1; usage errors exit 2. Output or internal failure
exits 3. A short or failed console write may have emitted partial output;
console output is not transactional. No files are read or written by the app.

A bad hex input or out-of-range decode prints the same diagnostic form to stderr,
writes no stdout and exits 1; wrong arity exits 2; a console write failure exits
3. Example package `1.1.0` adds this mode to the `1.0.1` interface.

[ADR 0419](../../docs/adr/0419-sdk-checked-byte-conversion.md) owns byte conversion;
[ADR 0420](../../docs/adr/0420-sdk-checked-i64-decimal-parsing.md) owns decimal parsing.
`src/codec.fy` owns this example's encoding and decoding policy. The previous
[Packet adoption receipt](../../docs/ops/evidence/packet-codec-adoption-20261005.json)
records the `codec::Packet` adoption: the Windows MSVC native receipt, the standalone
`fairyc check`/`build`/`run` dev/release representatives, and
compiler/source/SDK/image digests. The
[decode receipt](../../docs/ops/evidence/byte-codec-decode-20261005.json) records the
`decode-le` addition with its standalone CLI and native companions and their exact identities.
`internal/cli/checked_byte_codec_example_test.go` checks 64 CLI cases in each
Windows MSVC low and optimized profile (40 existing plus 24 new) for 128 native process
executions; the selected Go suite reports 131 passing entries with 0 skipped or failed.
A separate interpreter check exercises twenty-four decoder round trips across
widths 1..8 for zero, one and the width-specific maximum, plus raw empty, nine-byte
and high-bit rejects with input state unchanged and hex error ordering, alongside
the existing seven exact encoder failures that preserve every Packet sentinel byte
and length.
The [20261002 adoption receipt](../../docs/ops/evidence/checked-parser-adoption-codec-20261002.json)
and [20261002 original receipt](../../docs/ops/evidence/checked-byte-codec-example-20261002.json)
retain the historical 40-case and 38-case runs, respectively, with their earlier
compiler identities, and the previous Packet adoption also recorded 40 cases.

Policy checks cover the interpreter and Windows/Linux C emission. The Windows native
matrix records the actual in-process CLI Go test executable identity; the
standalone representatives separately record their `fairyc.exe` identity, verified
separately for dev and release. Linux execution stays deferred to the owner's batch;
performance measurements, physical persistence, and console failure injection are
outside this example's validation. The decode addition adds no SDK API,
SDK/language/runtime version, runner or script; the SDK stays `0.0.0-stage0.86`
with language `20` and runtime `0`, so no compiler, SDK API, PowerShell runner or
CI change is involved. External round4 sources and receipts are preserved.
