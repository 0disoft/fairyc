# Configured diagnostic limit in the C scanner reference

Date: 2026-09-25. Baseline child HEAD: `ce923394`.
The independent C scanner budget images previously fixed the ordinary
diagnostic limit at 256. The Fairy scanner and Go lexer also accept a
caller-selected smaller limit. The C reference now uses an optional configured
limit while retaining 256 when it is unset. Terminal `E0010` remains outside
the ordinary diagnostic count.

The [configured-limit native test](../../internal/compiler/vector_lexer_c_configured_diagnostic_limit_test.go)
checks nine empty, valid, invalid-byte and unclosed-comment inputs at limits
zero, one and two. Both C storage modes and both Fairy loop forms match the Go
lexer under low and optimized Windows AMD64 MSVC. The existing lexical-parity
and combined-limit tests also pass, covering the previous default path. These
cases do not prove every diagnostic kind, competing limit or allocation failure.

The optimized four-image reporter and 1,000-round checksum smoke passed. Its
[receipt](evidence/scanner-configured-diagnostic-limit-smoke-20260925.json)
preserves source and image hashes and the measured sizes.

| Image | Previous EXE bytes | Current EXE bytes |
| --- | ---: | ---: |
| Independent C | 184,832 | 184,832 |
| Geometric C | 185,344 | 185,344 |
| Explicit-index Fairy | 201,728 | 201,728 |
| Private vector-loop Fairy | 201,728 | 201,728 |

The smaller C image leaves the Fairy/C EXE ratio at 1.09141, or 2,110 bytes
above the cumulative 8% ceiling. No size improvement is claimed. The smoke's
single QPC reading per image is diagnostic only. Stable runtime, other limit
interactions, later allocator failures and unused-feature overhead remain open;
the vector source form stays compiler-private.
