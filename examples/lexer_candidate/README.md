# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

Current recognition: leading UTF-8 BOM, ASCII identifiers and exact current
keywords, longest-match punctuation, horizontal whitespace/LF/CRLF, line and
documentation comments, and nested closed block comments. Token observations are
`kind trivia startByte endByte cookedLength` numeric fields followed by each cooked
byte as a decimal number, separated by spaces. `true`, `false` and `null` retain
their cooked spelling; other admitted tokens have empty cooked data. EOF appears exactly once.
These numbers are private test observations tied to `internal/lexer/token.go`.

Input is bounded to 4096 bytes. Status 0 means a complete supported token stream;
2 means unsupported input, 3 allocation failure, and 4 input-limit exhaustion.
Scanning finishes before output starts, so scan failure produces no partial stream.
Output failures still use the existing assertion/trap boundary.

Numbers, strings, malformed-input diagnostics, lone CR, invalid code bytes,
reference-budget parity and the complete differential corpus remain unfinished.
Unsupported input must not be interpreted as a successful empty token stream.
The Go lexer stays authoritative. The start evidence and separate promotion gate
are in `docs/architecture/self-hosting-readiness-v0.md`.

`TestWindowsFairyLexerCandidate` compares native low/optimized observations against
the Go reference and separately tests unsupported and limit outcomes. The harness
supplies only raw bytes, not precomputed candidate tokens or recognition tables.
