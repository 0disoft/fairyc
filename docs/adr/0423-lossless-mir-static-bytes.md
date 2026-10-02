# ADR 0423: Lossless MIR static-byte serialization

- Status: Implemented
- Date: 2026-10-03
- Depends on: ADR 0149, ADR 0154, ADR 0171, ADR 0352, ADR 0422

## Problem and decision

MIR `Operand.StaticData` stores exact compiler-owned bytes in a comparable Go
string. The previous JSON representation treated that string as Unicode text.
Encoding an invalid UTF-8 byte replaced it with U+FFFD; decoding then returned
different bytes while canonical encoding and MIR verification still passed.
This affected whole-program MIR and the function bodies embedded in package
artifacts and caches, independently of the new `bytes::compare` helper.

The minimal source in [ADR 0422](0422-sdk-immutable-byte-view-comparison.md)
compares `b"\x80"` with an array containing byte 128. Previously direct MIR
interpretation passed, but the codec round trip changed `80` to `EF BF BD` and
the assertion trapped. That reproduction is now a passing execution regression.

Keep the in-memory string and all source literal semantics unchanged. Serialize
every operand through one shared [JSON boundary](../../internal/mir/operand_json.go).
Replace wire field `StaticData` with required string field `StaticDataBase64`.
It contains standard padded canonical Base64 of the exact raw bytes, including
NUL, invalid UTF-8 and C-string terminators. Empty static data is `""`, not null.
The remaining scalar/type fields retain their names and representations,
including the existing omission of zero `I64`.

Decode with unknown-field rejection, strict Base64 padding-bit validation and
an exact re-encoding check. Reject missing/null data, malformed Base64 and
noncanonical whitespace. Update the destination only after successful decoding.
The enclosing codecs still enforce their caller-supplied byte limits, one JSON
value, exact versions, canonical complete payloads and MIR/package verification.
Text and C-string validity remain verifier responsibilities; Base64 does not
weaken them. Base64 expansion counts toward the existing artifact size budgets.

## Version and cache boundary

| Compiler-private format | Previous version | Current version |
| --- | --- | --- |
| Whole-program MIR, including storage-transfer envelope | `fairyc-mir/1` | `fairyc-mir/2` |
| Stable function body | `fairyc-package-function-body/3` | `fairyc-package-function-body/4` |
| Stable package MIR | `fairyc-stable-package-mir/4` | `fairyc-stable-package-mir/5` |
| Stable storage-transfer package MIR | `fairyc-stable-package-storage-mir/1` | `fairyc-stable-package-storage-mir/2` |
| Stable package compilation | `fairyc-stable-package-compilation/2` | `fairyc-stable-package-compilation/3` |
| Process-local package MIR | `fairyc-package-mir-process/2` | `fairyc-package-mir-process/3` |
| Process-local storage-transfer package MIR | `fairyc-package-mir-storage-process/0` | `fairyc-package-mir-storage-process/1` |

The body schema is now `fairyc.package-function-body/v4`; the ordinary process
schema is `fairyc.package-mir-process/v3`. Other envelope schemas are unchanged
and use the explicit new versions above. Existing cache-key construction binds
MIR and artifact versions, so old persisted entries cannot be executable hits.
No decoder migration guesses the original bytes of a previously corrupted
string. Old artifacts are rejected and disposable caches regenerate from source.
Historical reproduction files, compiler pins and receipts remain preserved.

SDK stage0.78, language `/20`, runtime ABI `/0`, grammar, public APIs and target
maturity are unchanged. The compiler-private wire change requires new artifact
versions rather than an SDK API version bump.

## Verification

- Operand tests preserve all 256 byte values, empty data, binary C strings with
  their terminating NUL, Unicode text and scalar extremes. They reject old
  fields, invalid/padding-bit/noncanonical Base64, missing/null data and unknown
  fields without partially changing the destination.
- Whole-program ordinary and storage-transfer codecs round-trip all 256 literal
  bytes exactly, execute the decoded program and reject the prior MIR version.
- Package/body codecs retain canonicality, byte-limit, ownership and rebinding
  tests. Explicit previous-version rejection is checked for ordinary process,
  body, stable MIR and compilation payloads. The deterministic compilation
  digest fixture is updated for the deliberate new wire representation.
- [Literal execution tests](../../internal/cli/bytes_literal_codec_test.go)
  check the minimal equality/comparison reproduction and all 256 literal bytes
  through checked MIR, codec, interpreter, Linux/Windows C generation and
  Windows MSVC low/optimized native execution. `.cache/bytes-literal-codec/`
  retains source, compiler/SDK identity, EXE SHA256 and raw UTF-8 execution
  receipts. Memory cold/warm reuse and independent disk sessions execute the
  same byte checks and require real cache hits on reuse.
- Comparison testing now includes high-byte literal inputs in its 256-byte
  pivot and 56 ordered triples. Equality native execution is also rechecked.
  Existing package artifact, workspace compiler and package cache tests pass.

Linux native execution and performance experiments are not run. This is a
compiler-byte correctness fix, not a new format interoperability or target
support claim.
