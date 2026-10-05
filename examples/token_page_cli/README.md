# Token page CLI (fixed-array field storage)

Windows AMD64 and Linux GNU amd64 example. It requires a compiler with ADR 0472 support;
the SDK stage0.86 version alone does not identify that support. Product logic is Fairy C in
`src/main.fy`, `src/tokens.fy`, `src/evaluator.fy`, and `src/arithmetic.fy`.

A bounded ASCII lexer stores every scanned token directly into the fixed array field of
a plain struct instead of a `Vec`:

```fairy
struct Span { usize start; usize end; }
struct Token { u32 kind; Span position; i64 value; }
struct TokenPage { Token[128] tokens; usize used; }
```

The command is `token-page-cli [ASCII-source]` for the default analysis mode, and
`token-page-cli --eval <ASCII-source>` for the evaluation mode described below. `--help`
prints both forms on stdout:

```text
usage: token-page-cli [ASCII-source]
       token-page-cli --eval <ASCII-source>
```

Argument-count errors keep the single-line stderr message. With no argument it analyzes the
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
tokens are admitted. The analysis lexer admits identifiers `[A-Za-z_][A-Za-z0-9_]*`,
unsigned decimal integers in `0..=i64 max`, the operators `+`, `=` and `;`, and the ASCII
whitespace SP, HT, LF and CR; `*`, `(` and `)` are not admitted in analysis mode. The
nonnegative i64 literal and the running number sum are overflow-checked. In the default
analysis mode the program lexes and sums only: it has no expression parser and does not
evaluate expressions. The separate `--eval` mode below adds a bounded assignment evaluator
over the evaluation scanner entry point `scan_eval`, which wraps the same scan
implementation with a flag that also admits `*`, `(` and `)`.

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

## Evaluation mode (`--eval`)

`token-page-cli --eval "<ASCII-source>"` evaluates a bounded assignment program instead of
analyzing it. It reuses the 4095-byte input limit and 128-token cap, over the evaluation
scanner entry point `scan_eval`, which wraps the same scan implementation with a flag that
also admits `*`, `(` and `)`; the analysis scan path is unchanged.

```sh
fairyc run --locked -- --eval "price = 4000000000 + 7; total = price + 3;"
```

```powershell
.\build\token-page-cli.exe --eval "price = 4000000000 + 7; total = price + 3;"
```

Output is one `name<TAB>value` row per assigned name, in the order the names were first
defined, with a newline after each row:

```text
price	4000000007
total	4000000010
```

Precedence and parentheses follow:

```sh
fairyc run --locked -- --eval "a=2+3*4; b=(2+3)*4;"
```

```text
a	14
b	20
```

The admitted program is a sequence of assignments, each terminated by `;`:

```text
program = (identifier '=' expr ';')* ;
expr    = product ('+' product)* ;
product = primary ('*' primary)* ;
primary = unsigned-i64-literal | earlier-defined-identifier | '(' expr ')' ;
```

Every `;` is required. `*` binds tighter than `+`, and both are left-associative.
Primaries are unsigned decimal i64 literals still capped at i64 max by the lexer, identifiers
defined earlier, or a parenthesized `expr`. Unary `+` and `-`, comments, subtraction,
division, and every other expression form are rejected, so a value is built only from
nonnegative literals, defined names, `+` and `*`. An empty or all-whitespace source
succeeds and writes no output. Parentheses nest at most 16 levels deep; a 17th level fails
with `error: parenthesis limit (16)`, and malformed, empty, or unmatched-parenthesis
programs report `error: invalid evaluation syntax`. Intermediate operations are checked,
including inside a parenthesized group even when the group is later multiplied by zero, so a
value above i64 max fails with `error: evaluation overflow`. The lexer scans the whole input
before evaluation begins.

A name may be reassigned: the later assignment updates the existing slot, and the first
definition order is kept in the output. Each right-hand side is evaluated against the
bindings that existed before its own statement, so `x = x + 1;` fails with
`error: undefined variable` when `x` is new, while an already-defined `x` may reference
itself. At most 16 distinct names are admitted; a 17th new name fails with
`error: variable limit (16)`.

No stdout is written until every statement has been evaluated and the whole output buffer is
complete; that buffer is then written to stdout once. Evaluation uses the unrebased original
token spans from the lexer, and it reduces iteratively with a fixed value stack and operator
stack of 128 slots each rather than a recursive parser; stack saturation reports
`error: invalid evaluation syntax`, and exceeding 16 parenthesis levels reports
`error: parenthesis limit (16)`.

Evaluation-mode failures exit 2 with exactly one of these lines on stderr:

```text
error: invalid evaluation syntax
error: undefined variable
error: evaluation overflow
error: variable limit (16)
error: parenthesis limit (16)
```

The lexical, integer, token-limit, input-limit, allocation, and stdout-write errors above
still apply. `--eval` requires exactly one source argument after the flag; any other
argument count exits 2 with `usage: token-page-cli [ASCII-source]` on stderr. `--eval
--help` treats `--help` as the source and fails with `error: invalid ASCII input`, because
in this mode the second argument is always the source.

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
  TSV rows and handles the argument, help, limit and failure paths.
- `tests/page.fy` is a source `@test` (`normalized_spans_and_reuse`): for `' \talpha=7;'` it
  asserts the four tokens' original offsets `2..10` normalize to `0..8` with exact per-span
  values and number `7`, then reuses the same page to show an empty input keeps `used = 0`
  and a `;` scan leaves `used = 1`, so no stale prefix is exposed. It passes under the
  interpreter and as a native development and release case.
- `src/evaluator.fy` holds the evaluation mode: an iterative precedence parser over the
  `scan_eval` token stream, name lookup, and a `Binding[16]` `Environment` that records
  first-seen names in definition order; `src/main.fy` formats the result as
  `name<TAB>value` rows for one stdout write.
- `src/arithmetic.fy` owns the evaluator's arithmetic: a fixed value stack and a fixed
  operator stack of 128 slots each, the `+`/`*` reduction with precedence and
  left-associativity, and the checked add and multiply that raise the overflow result.
- `tests/evaluation.fy` is a source `@test` (`bindings_and_reuse`): for `'  x=4000000000;
  x=x+7; total=x+3;'` it asserts two bindings whose values `4000000007` and `4000000010`
  exceed i32, that the reassigned name keeps its original unrebased source span `2..3`,
  that a failed `x = missing;` right-hand side returns the undefined-variable error and
  leaves the earlier binding intact (`used = 1`, value `1`) with no CLI output, and that
  reusing the environment for an empty input resets `used` to `0`. It passes under the
  interpreter and as a native development and release case.
- `tests/arithmetic.fy` is a source `@test` covering `*` precedence over `+`, both
  left-associative, parenthesized expressions within the depth limit, the
  `parenthesis limit (16)` and `invalid evaluation syntax` failures, and checked multiply
  and add overflow. It passes under the interpreter and as a native development and release
  case.

Token storage uses the fixed array field, so scanning, rebasing and analysis neither
allocate per token nor copy the whole token array; constructing the page returns a
`TokenPage` value (for example `tokens::empty()`), and the input and output buffers may
allocate. This example does not claim that the whole CLI is allocation-free, and it makes
no performance claim.

Portable verification (Python 3.10 or later, standard library only) requires the compiler
executable and runs `check`, `check --locked`, the locked development and release builds with
their actual EXEs, the interpreter source tests, and the whole-stdout, error and boundary
cases for the analysis, evaluation, and arithmetic modes. Evaluation outputs are checked
against an independent recursive-descent oracle written in Python, including 16-level
parenthesis nesting and mixed-operator stack pressure. The suite is described by what it
exercises rather than by a fixed case count. The optional `--receipt PATH` option writes a
summary plus the development and release raw JSON and retained images (execute permission
preserved) to that path; without it no extra evidence logs or image copies are saved, while
the ordinary `check` and `build` outputs (the lock file and the executable) are still
written:

```sh
python verify.py --compiler /absolute/path/to/fairyc
python verify.py --compiler /absolute/path/to/fairyc --receipt /path/to/evidence/receipt.json
python verify.py --compiler C:\path\to\fairyc.exe --receipt C:\path\to\evidence\receipt.json
```

The full suite passes on Windows amd64 in both analysis and evaluation modes, including the arithmetic extension,
and the fresh Linux and Windows C emission checks pass. The focused `TestTokenPageCLIStorage`
and `go vet` checks pass as well; `TestTokenPageCLIStorage` confirms the tokens module's
authored workspace MIR has no fixed-array `FieldRead` or `ArrayIndex` copy, exercising the
descriptor views and stores instead. The recorded runs used the cached compiler
`.cache/nominal-array-field-view-compiler/fairyc.exe`, with no compiler or SDK changes.
Actual Linux execution remains deferred to the owner's consolidated batch, so Linux is
verified by C emission only. The analysis baseline evidence is
`docs/ops/evidence/token-page-cli-20261005.json`, the evaluation-mode evidence is
`docs/ops/evidence/token-page-eval-cli-20261005.json`, and the arithmetic extension evidence
is `docs/ops/evidence/token-page-arithmetic-cli-20261005.json`; each records `.dev.json`
and `.release.json` raw logs against the final source hashes. There is no PowerShell harness
and no performance claim. See
[ADR 0472](../../docs/adr/0472-direct-plain-struct-array-field-views-and-iteration.md).
