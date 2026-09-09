# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

Current recognition: leading UTF-8 BOM, ASCII identifiers and exact current
keywords, longest-match punctuation, horizontal whitespace/LF/CRLF, line and
documentation comments, nested block comments, arbitrary-width 2/10/16-base
integers, decimal fractions and exponents. `src/numbers.fy` follows the reference
numeric recovery envelope and emits `E0004` for malformed numeric tokens.
`src/strings.fy` recognizes ordinary, byte and C strings, simple/hex/Unicode
escapes, exact hexadecimal cooked bytes and C terminators. Invalid escape,
UTF-8, NUL, newline and EOF recovery preserve the reference diagnostic spans.
Invalid code bytes, non-ASCII code points, lone CR and unterminated block comments
retain lossless tokens with `E0001`. `src/utf8.fy` rejects overlong encodings,
surrogates, out-of-range scalars and truncated sequences without skipping bytes.
Token observations are
`kind trivia startByte endByte cookedLength` numeric fields followed by each cooked
byte as a decimal number, separated by spaces. `true`, `false` and `null` retain
their cooked spelling; numbers retain their canonical decimal bytes. EOF appears exactly once.
These numbers are private test observations tied to `internal/lexer/token.go`.
After tokens, a diagnostic count precedes each diagnostic's length-prefixed code,
repair and token bytes followed by its half-open start/end span. The current
diagnostic subset is `E0001`/`E0004`/`E0005`/`E0006`, including comment and string
`exit` repairs and expected closing tokens. The separate diagnostic store uses
stable source/code/repair/token ordering and preserves duplicate records.

Input is bounded to 4096 bytes, each materialized decimal expansion to 16384 bytes
and cumulative retained cooked data to 65536 bytes. Status 0 means a
complete supported token/diagnostic stream (not syntactically valid code);
2 means unsupported input, 3 allocation failure, and 4 candidate-limit exhaustion.
Scanning finishes before output starts, so scan failure produces no partial stream.
At most 256 diagnostics are published. A 257th insertion triggers status 4 and is
dropped with the failed scan's owner; it is never sorted or printed.
Output failures still use the existing assertion/trap boundary.

Reference-budget parity, previous-revision mode and the complete differential
corpus remain unfinished. Status 2 remains reserved for unsupported extensions;
it must not be interpreted as a successful empty token stream.
The Go lexer stays authoritative. The start evidence and separate promotion gate
are in `docs/architecture/self-hosting-readiness-v0.md`.

`TestWindowsFairyLexerCandidate` compares native low/optimized observations against
the Go reference and separately tests unsupported and limit outcomes. The harness
supplies only raw bytes, not precomputed candidate tokens or recognition tables.
