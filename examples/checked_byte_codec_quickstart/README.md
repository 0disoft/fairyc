# Checked byte encoding

This example uses SDK `0.0.0-stage0.86` decimal parsing and
`std::num::u8_from_i64` to contrast
exact byte conversion with unsigned little-endian field encoding. Its inline
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
```

The three runs print, respectively:

```text
ff
00 01
ff ff ff ff ff ff ff 7f
```

`byte VALUE` requires exactly one signed decimal argument and converts the full
value. `byte 256` fails with `VALUE: outside byte range 0..255`.
`le VALUE WIDTH` requires exactly two decimal arguments. WIDTH must be 1..8;
VALUE must be nonnegative and fit that unsigned field. `le 256 1` fails rather
than truncating. `le 0 8` produces eight zero bytes. This example accepts values
only up to i64 maximum; it is not a general u64 or signed two's-complement codec.

Each numeric argument is at most 20 UTF-8 bytes and follows `-?[0-9]+`.
Leading zeros and `-0` are accepted. Empty values, a bare minus, plus signs,
whitespace, decimal points and values outside i64 are rejected. `numbers::parse`
delegates to `std::num::parse_i64_decimal` and maps its errors to the existing
caller errors. The SDK handles i64 minimum without negating it. The 20-byte
argv limit remains example policy. Competing failures retain first-failure
ordering: `9223372036854775808x` reports range, while `922337203685477580x8`
reports syntax.

CLI validation order is mode/arity, VALUE parsing, then (for `le`) WIDTH parsing
and range, then codec policy. Thus `le -1 0` reports the width error. The reusable
example-local `codec::encode_le` checks sign, width, capacity and excess high
bits before writing. It extracts `remaining % 256`, converts only that digit,
then divides by 256. Invalid fields leave the caller's buffer unchanged;
successful fields leave all bytes beyond WIDTH unchanged. The SDK conversion
still performs its own conversion work; no speed claim is made.

`codec::Packet { u8[8] bytes; usize length; }` carries the encoded bytes and their
length. `main` initializes it as `{ .bytes = [0; 8], .length = 0 }` and sets
`.length` only after a successful encode, so a failed `le` leaves the packet
untouched. `encode_le` is unchanged and is called with `ref mut packet.bytes[..width]`,
using a `width` local captured before that exclusive owner loan. The renderer takes
`ref codec::Packet` and iterates `packet.bytes[..packet.length]` with a shared
`for (let u8 byte in ...)` loop, rendering into the existing 24-byte scratch output.

Success prints lowercase two-digit hex bytes separated by single spaces and
terminated by LF, and exits 0. No stdout is written until complete input and
encoding validation succeeds. Input/field failures print a `checked-byte-codec:`
diagnostic to stderr and exit 1; usage errors exit 2. Output or internal failure
exits 3. A short or failed console write may have emitted partial output;
console output is not transactional. No files are read or written by the app.

[ADR 0419](../../docs/adr/0419-sdk-checked-byte-conversion.md) owns byte conversion;
[ADR 0420](../../docs/adr/0420-sdk-checked-i64-decimal-parsing.md) owns decimal parsing.
`src/codec.fy` owns this example's encoding policy. The current
[Packet adoption receipt](../../docs/ops/evidence/packet-codec-adoption-20261005.json)
records the `codec::Packet` adoption: the Windows MSVC native receipt, the standalone
`fairyc check`/`build`/`run` dev/release representatives, and
compiler/source/SDK/image digests.
`internal/cli/checked_byte_codec_example_test.go` checks 40 CLI cases in each
Windows MSVC low and optimized profile. A separate interpreter check exercises
seven exact codec failures that preserve every Packet sentinel byte and length, and a
successful prefix and `[2..4]` offset that leave the outside bytes and length
unchanged until the caller updates them.
The [20261002 adoption receipt](../../docs/ops/evidence/checked-parser-adoption-codec-20261002.json)
and [20261002 original receipt](../../docs/ops/evidence/checked-byte-codec-example-20261002.json)
retain the historical 40-case and 38-case runs, respectively, with their earlier
compiler identities.

The native matrix records the actual in-process CLI Go test executable identity;
the standalone representatives separately record their `fairyc.exe` identity.
Linux execution stays deferred to the owner's batch;
performance measurements, physical persistence, and console failure injection are
outside this example's validation. The adoption adds no SDK API, SDK/language/runtime
version, runner or script, and example package `1.0.1` and the CLI behavior stay
unchanged. External round4 sources and receipts are preserved.
