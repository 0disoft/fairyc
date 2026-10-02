# Checked byte encoding

This example uses SDK `0.0.0-stage0.76` decimal parsing and
`std::num::u8_from_i64` to contrast
exact byte conversion with unsigned little-endian field encoding.
From this directory, with a compiler bundling that SDK:

```powershell
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

Success prints lowercase two-digit hex bytes separated by single spaces and
terminated by LF, and exits 0. No stdout is written until complete input and
encoding validation succeeds. Input/field failures print a `checked-byte-codec:`
diagnostic to stderr and exit 1; usage errors exit 2. Output or internal failure
exits 3. A short or failed console write may have emitted partial output;
console output is not transactional. No files are read or written by the app.

[ADR 0419](../../docs/adr/0419-sdk-checked-byte-conversion.md) owns byte conversion;
[ADR 0420](../../docs/adr/0420-sdk-checked-i64-decimal-parsing.md) owns decimal parsing.
`src/codec.fy` owns this example's encoding policy. The
[adoption receipt](../../docs/ops/evidence/checked-parser-adoption-codec-20261002.json)
records exact argv/cwd/stdout/stderr/exit and source/SDK/compiler/EXE identities.
`internal/cli/checked_byte_codec_example_test.go` checks 40 CLI cases in each
Windows MSVC low and optimized profile. A separate interpreter check exercises
seven exact codec failures with sentinel buffers and successful suffix preservation.
The [original receipt](../../docs/ops/evidence/checked-byte-codec-example-20261002.json)
retains the earlier 38-case evidence and historical compiler identity.

The compiler identity is the actual in-process CLI Go test executable, not a
standalone `fairyc.exe`. Linux native execution, performance measurements,
physical persistence, and console failure injection are outside this example's
validation. SDK/language/runtime versions are unchanged by this adoption;
example package `1.0.1` replaces the local accumulator without changing CLI behavior.
External round4 sources and receipts
are preserved.
