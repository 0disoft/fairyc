# Configured source-byte limit in the C scanner reference

Date: 2026-09-25. Baseline child HEAD: `b4bc136c`.
The independent C scanner budget images already rejected sources longer
than the candidate's hard 32,768-byte cap. They did not model the smaller
caller-supplied `max_bytes` value accepted by the Fairy scanner. The C
reference now has an optional configured byte limit: an unset limit keeps
the existing default behavior, while an explicit limit below the input
length publishes terminal `E0010`, the unscanned invalid token and EOF.
The hard candidate cap still returns status 4 before this path.

The [configured-limit native test](../../internal/compiler/vector_lexer_c_byte_limit_test.go)
checks 13 empty, accepted and rejected inputs, including BOM, invalid UTF-8,
decimal, string and CRLF boundaries. Both C storage modes and both Fairy
source forms match the Go lexer under low and optimized Windows MSVC.
The existing 38-input lexical-parity test also passes with the new C budget
define enabled but the configured limit unset. These checks establish only
the listed boundaries, not every combined limit or allocation failure.

The optimized four-image reporter and 1,000-round checksum smoke passed.
Its [receipt](evidence/scanner-configured-byte-limit-smoke-20260925.json)
retains the C source hash, image hashes and measured sizes.

| Image | Previous EXE bytes | Current EXE bytes | Current source bytes |
| --- | ---: | ---: | ---: |
| Independent C | 184,320 | 184,832 | 52,988 |
| Geometric C | 185,344 | 185,344 | 53,069 |
| Explicit-index Fairy | 201,728 | 201,728 | 395,860 |
| Private vector-loop Fairy | 201,728 | 201,728 | 396,700 |

The smaller equivalent C image now gives a Fairy/C EXE ratio of 1.09141.
The 1.08 ceiling is 199,618 whole bytes, leaving 2,110 bytes of the Fairy
image above it. The C image's 512-byte increase reflects a newly matched
failure behavior, not a reduction in Fairy code. One-thousand-round QPC
readings are diagnostic only. Stable runtime, native build time, other
combined limits and later allocator failures remain open; the vector source
form stays compiler-private.
