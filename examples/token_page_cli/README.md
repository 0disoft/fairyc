# Token page CLI (fixed-array field storage)

Windows AMD64 and Linux GNU amd64 example. It requires a compiler with ADR 0472 support;
the SDK stage0.86 version alone does not identify that support. Product logic is Fairy C in
`src/main.fy` and `src/tokens.fy`.

A bounded ASCII lexer stores every scanned token directly into the fixed array field of
a plain struct instead of a `Vec`:

```fairy
struct Span { usize start; usize end; }
struct Token { u32 kind; Span position; i64 value; }
struct TokenPage { Token[128] tokens; usize used; }
```

The command is `token-page-cli [ASCII-source]`, and `--help` prints
`usage: token-page-cli [ASCII-source]` on stdout. With no argument it analyzes the
built-in source `price = 4000000000 + fee; other = 7;`.

```sh
fairyc check
fairyc build
fairyc build --release --locked
fairyc run --locked
fairyc run --release --locked
fairyc run --locked -- "price = 4000000000 + fee; other = 7;"
```

```powershell
fairyc build
.\build\token-page-cli.exe "price = 4000000000 + fee; other = 7;"
```

The optional argument is the ASCII source to analyze. Its byte length is limited to 4095
(`length > 4095` fails; the SDK argument buffer can hold 4096 bytes), and at most 128
tokens are admitted. The admitted grammar is identifiers `[A-Za-z_][A-Za-z0-9_]*`,
unsigned decimal integers in `0..=i64 max`, the operators `+`, `=` and `;`, and the ASCII
whitespace SP, HT, LF and CR. The nonnegative i64 literal and the running number sum are
overflow-checked. The program lexes and sums only: it has no expression parser and does not
evaluate expressions.

Output is eight deterministic TSV rows in this order:

```text
tokens<TAB>count of all tokens
identifiers<TAB>count of identifiers
numbers<TAB>count of decimal numbers
sum<TAB>checked sum of all numbers
first_tokens<TAB>token count through the first ';'
first_identifiers<TAB>identifier count through the first ';'
first_numbers<TAB>number count through the first ';'
first_sum<TAB>checked sum through the first ';'
```

The first segment ends at the first `;`, inclusive; if there is no `;` the segment is the
whole input. The built-in demo prints:

```text
tokens	10
identifiers	3
numbers	2
sum	4000000007
first_tokens	6
first_identifiers	2
first_numbers	1
first_sum	4000000000
```

No stdout is written before validation completes and both the all-token and first-segment
summaries are ready, and lexical or sum errors produce no stdout. The token-capacity check
runs before parsing the next significant token, so the token-limit diagnostic wins when the
129th token begins.

Failure cases exit 2 with exactly one of these lines on stderr:

```text
error: invalid ASCII input
error: integer overflow
error: token limit (128)
error: sum overflow
error: buffer allocation failed
error: input limit (4095 UTF-8 bytes)
error: stdout write failed
```

A usage error exits 2 with `usage: token-page-cli [ASCII-source]` on stderr.

Feature locations:

- `src/tokens.fy` scans the source and stores each token with a whole-element overwrite
  into the fixed array field. The slot is read before the store, because indexing by
  `page.used` directly is a competing loan, so it stores `page.tokens[slot]` for a local
  `slot = page.used` and then advances `used`.
- Normalization subtracts the segment origin from `page.tokens[0].position.start` and
  `.end` directly, then rebases the rest with `for (let ref mut Token token in
  page.tokens[1..used])`.
- `tokens::valid(ref page)` returns `ref page.tokens[..used]`, and analysis iterates
  `for (let ref Token token in segment)` over that passed segment, not the page directly.
  The first-segment end is found with a shared `ref` scan up to and including the first `;`.
- `src/main.fy` initializes the `TokenPage`, selects the first-segment range, formats the
  TSV rows and handles the argument, limit and failure paths.
- `tests/page.fy` is a source `@test` (`normalized_spans_and_reuse`): for `' \talpha=7;'` it
  asserts the four tokens' original offsets `2..10` normalize to `0..8` with exact per-span
  values and number `7`, then reuses the same page to show an empty input keeps `used = 0`
  and a `;` scan leaves `used = 1`, so no stale prefix is exposed. It passes under the
  interpreter and as a native development and release case.

Token storage uses the fixed array field, so scanning, rebasing and analysis neither
allocate per token nor copy the whole token array; constructing the page returns a
`TokenPage` value (for example `tokens::empty()`), and the input and output buffers may
allocate. This example does not claim that the whole CLI is allocation-free, and it makes
no performance claim.

Portable verification (Python 3.10 or later, standard library only) requires the compiler
executable and runs `check`, `check --locked`, the locked development and release builds with
their actual EXEs, the interpreter source test, and the whole-stdout, error and boundary cases.
The current run drives one interpreter test and 62 native results: 30 CLI cases per profile, 60
development and release executions, plus the normalized-span test in both profiles. The
`--receipt PATH` option is optional and writes a summary plus the development and release raw
JSON and retained images (execute permission preserved) to that path; without it no extra
evidence logs or image copies are saved, while the ordinary `check` and `build` outputs (the
lock file and the executable) are still written:

```sh
python verify.py --compiler /absolute/path/to/fairyc
python verify.py --compiler /absolute/path/to/fairyc --receipt /path/to/evidence/receipt.json
python verify.py --compiler C:\path\to\fairyc.exe --receipt C:\path\to\evidence\receipt.json
```

Windows amd64 native execution and Linux C emission both pass; actual Linux execution
remains deferred to the owner's consolidated batch, so Linux is verified by C emission
only. The run used the cached compiler `.cache/nominal-array-field-view-compiler/fairyc.exe`
from the current source, and no compiler source changed. `TestTokenPageCLIStorage` checks
the authored workspace MIR of the tokens module and confirms no fixed-array `FieldRead` or
`ArrayIndex` copy, exercising the descriptor views and stores instead. Evidence is
`docs/ops/evidence/token-page-cli-20261005.json` with `.dev.json` and `.release.json` raw
logs. There is no PowerShell harness and no performance claim. See
[ADR 0472](../../docs/adr/0472-direct-plain-struct-array-field-views-and-iteration.md).
