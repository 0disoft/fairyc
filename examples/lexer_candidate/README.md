# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

## Owned consumer API

`scanner::collect` consumes an input `bytes::Buffer` and the same explicit revision
and byte/token/diagnostic/work limits as recognition. It returns
`Result<scanner::Stream, i32>` without console output. Stream owns the original
source, token storage, cooked bytes and diagnostics; input is consumed on both
success and failure. Status 3/4 means allocation/storage failure and publishes no
Stream. Reference-budget exhaustion instead returns a complete lossless Stream
containing `E0010`.

`scanner::release(stream)` explicitly consumes and destroys the result. Ordinary
scope exit also destroys it; the API does not use `discard` as an early-release
operation.

`token_count`, `token_info`, `token_raw`, `token_cooked`, `source_view`,
`diagnostic_count`, `diagnostic_info` and `diagnostic_token` provide read-only
access. Metadata is copied; returned byte views borrow the Stream and cannot
outlive it or coexist with its move/drop. Indexed access retains the vector bounds
trap. Diagnostic access uses recognition order, while the legacy report sorts for
display. Numeric kinds/codes and cooked encoding remain experimental implementation
contracts, not a stable SDK or public ABI. Native consumers exercise transfer and
budget exhaustion; compile-fail tests cover local escape and use after/drop during
a live borrow.

## Recognition and limits

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
diagnostic subset is `E0001`/`E0004`/`E0005`/`E0006`/`E0010`, including comment and string
`exit` repairs and expected closing tokens. The separate diagnostic store uses
stable source/code/repair/token ordering and preserves duplicate records.

Input is bounded to 16384 bytes, each materialized decimal expansion to 16384 bytes
and cumulative retained cooked data to 65536 bytes. Status 0 means a
complete supported token/diagnostic stream (not syntactically valid code);
2 means unsupported input, 3 allocation failure, and 4 candidate-limit exhaustion.
Scanning finishes before output starts, so scan failure produces no partial stream.
The default diagnostic budget admits 256 ordinary diagnostics. The next attempted
diagnostic stops recognition before allocating that diagnostic and adds one
uncharged `E0010`; the complete lossless exhaustion stream is printed normally.
Output failures still use the existing assertion/trap boundary.

`scanner::report_limits` accepts caller-selected source-byte and token limits
within the candidate's independent storage guards. Exhaustion emits `E0010` once,
retains the denied token and remaining bytes as invalid records, and always emits
an uncharged EOF. Byte limits are checked before recognition; token limits count
trivia and the leading BOM. Native tests compare zero, exact and exceeded limits
with the Go reference, including malformed-token diagnostic ordering.
`scanner::report_budget` also accepts a diagnostic limit. The same counter follows
code, numeric and string recognition, including escapes, invalid UTF-8, raw NUL,
newline and final literal validation. Tests compare both the failure position and
the retained diagnostics, including zero allowance and the default 256/257 boundary.
Independent storage guards still apply if callers request more than 256 ordinary
diagnostics. `src/budget.fy` supplies the work meter as an immutable value returned
by each charge. Native tests compare charge sequences against the Go budget and
check exact-limit acceptance, sticky exhaustion and arithmetic at the maximum `usize` value.
`scanner::report_work` connects the meter to BOM, whitespace, comments,
identifiers, punctuation and non-ASCII code bytes. Native comparisons vary work
limits at every source-byte boundary together with token limits, including the
reference's one-byte progress guard when a multi-byte step cannot start.
Numeric recognition also charges source advancement and complete pre-trimming
decimal expansion before materializing output. Exact expansion boundaries,
cumulative numbers, signed 64-bit exponent endpoints and out-of-range exponents
are compared against Go. Independent expansion/storage caps remain in force.
String recognition carries the same work meter through prefixed openers, raw
UTF-8, escapes, malformed-escape recovery and closing quotes. A multi-byte charge
either advances in full or leaves the cursor unchanged, matching the reference.
Native work-boundary tests also combine token and diagnostic limits and mixed
numeric/string input. All recognition paths now participate in work charging;
independent candidate storage caps and the separate compiler-component promotion
gate remain in force.
Status 2 remains reserved for unsupported extensions;
it must not be interpreted as a successful empty token stream.
The Go lexer stays authoritative. The start evidence and separate promotion gate
are in `docs/architecture/self-hosting-readiness-v0.md`.

`TestWindowsFairyLexerCandidate` compares native low/optimized observations against
the Go reference and separately tests unsupported and limit outcomes. The harness
supplies only raw bytes, not precomputed candidate tokens or recognition tables.

## Complete checked-in source corpus

`TestWindowsFairyLexerCorpus` reads every `.fy` file below `spec/fixtures`, without
filtering by pass/fail status. It compares current and immediately previous lexer
revision modes and deterministic NUL insertion, first-byte deletion and invalid-byte
replacement. Every source/mode/mutation case runs with five budget settings:
defaults, half the source-byte allowance, one token, no diagnostics, and half the
source length in work units. The current corpus has 303 fixture source files plus
the real `std_bytes_quickstart` and `std_io_scalar_quickstart` main sources, both
larger than the former 4096-byte limit. Identifier lengths around 4096 and 16384,
a 6000-byte string, and closed/unterminated 12000-byte comments add large-input
boundaries. All 7670 framed observations match Go under low and optimized Windows MSVC.
Comparison retains exact `E0010` positions, invalid ranges, preceding diagnostics
and EOF for exhausted runs. Newly added oversized fixtures
fail the lane rather than silently disappearing. Embedded Go-only seeds and
arbitrary inputs outside the stated candidate limits are not a full conformance claim.

The maintained `src/main.fy` driver reads private binary stdin frames: one mode
byte and a two-byte little-endian source length. Modes 0/1 use default budgets;
modes 2/3 add four unsigned 32-bit little-endian limits in byte, token, diagnostic,
work order. Odd modes use the previous revision. Exact source bytes follow the
header. EOF is valid only between frames. Invalid mode, truncated framing or more
than 2048 frames exits 8; oversized source exits 4. The harness checks invalid
mode, every truncated extended-header length, missing body and a 16385-byte length
at both profiles.

The harness compiles once per profile through the ordinary workspace compiler,
then reuses that artifact for output-bounded batches of at most 2048 frames. Each batch stays below 60KB
of expected output and retains the existing native driver's 64KB output limit and
trap classification. Fixture coverage remains complete; the two project sources
are a bounded workload sample, not all project files. A single observation that
exceeds the output cap fails explicitly, so 16KB source admission does not promise
that every such source fits this private reporting transport. This is exact record
comparison, not a token hash. Hosted CI for the large-input expansion is pending.
