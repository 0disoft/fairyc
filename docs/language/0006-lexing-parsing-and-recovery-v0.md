# Lexing, Parsing, and Recovery v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision sources: `docs/adr/0009-stage-0-grammar-and-parser-contract.md`,
  `docs/adr/0322-no-textual-preprocessing.md`
- Normative grammar: `spec/grammar/stage0.ebnf`
- Parser fixtures: `spec/fixtures/parser/manifest-v0.json`

## Parser boundary

The Stage 0 parser is a batch compiler parser. Its input is the original source byte sequence; its outputs are a trivia-preserving token stream, a compact lossless CST, structured diagnostics, and—only when lexing and parsing succeed—a lowered AST.

No preprocessing phase runs before this boundary. Textual includes, macro expansion, token pasting,
ambient definitions, and conditional token removal are not Fairy source operations. A `#` outside a
literal or comment is retained as invalid input and reported as `E0001`, not interpreted as a
directive.

Every original byte belongs to exactly one raw token, trivia token, invalid-input token, or error node. Synthetic missing tokens have zero width and record their insertion point and repair identity. The parser consumes the entire file or reports why it could not.

Incremental parsing, streaming input, formatter-grade immutable green trees, and editor snapshots are deferred. Their future outputs must converge with a fresh batch parse of identical bytes.

## Source bytes and positions

- Source files end in `.fy` and are decoded as UTF-8.
- An optional UTF-8 BOM is accepted only at byte offset zero and is preserved as trivia.
- Invalid UTF-8 is retained as an invalid-input token and reported as `E0001`; replacement characters are not inserted into source truth.
- LF and CRLF are accepted. CRLF is one logical newline but remains two source bytes. A lone CR outside a literal is invalid source text.
- A literal NUL byte outside an escape is invalid source text.
- Canonical spans are zero-based, half-open byte ranges `[startByte, endByte)` over the original file.
- Line, Unicode scalar, UTF-16, grapheme, and display-column positions are derived explicitly. They cannot be compared to or stored as byte offsets.
- Identifiers use ASCII letters, digits, and underscore, cannot begin with a digit, and are case-sensitive. Source text is not Unicode-normalized.

Line comments begin with `//`. Block comments begin with `/*`, end with `*/`, and may nest. Comments
and whitespace are trivia, not discarded input. An unterminated block comment is a lexer error and
consumes through EOF so lexing still advances.

A physical-line-leading line comment whose first three bytes are `///` is documentation trivia. It
remains an ordinary grammar-skipped trivia token, but its subtype is retained in the lossless stream.
ADR 0199 attaches only consecutive `///` physical lines immediately above a supported top-level
declaration, with exactly one LF or CRLF plus optional ASCII indentation between lines and before
the declaration's first attribute, modifier, or keyword. A blank line, ordinary line comment,
block comment including `/**`, code before `///` on the same physical line, or another token breaks
attachment. The exact attachment span retains markers, intermediate line endings, and intermediate
indentation; no marker removal, trimming, normalization, or Markdown interpretation occurs.

## Tokenization and progress

The lexer uses longest match within one lexical class, then the following fixed priorities:

1. comment openers before `/` as an operator;
2. `b"` and `c"` string-family openers before an identifier followed by a text string;
3. a floating candidate before an integer when a required decimal point or exponent marker is present;
4. multi-character operators before their one-character prefixes;
5. a complete identifier before exact keyword classification.

A numeric candidate consumes the complete radix, digit, separator, decimal-point, and exponent-shaped run before validation. Invalid text such as `1__0`, `0xG`, or `1e+` is one `E0004` token rather than several valid tokens. `1.field` remains an integer, dot, and identifier because a floating decimal point requires a following decimal digit.

Every lexer step reaches EOF, consumes at least one source byte, or emits one declared zero-width synthetic token while changing recovery state. Invalid UTF-8, escapes, comments, and numeric candidates retain their raw bytes and cannot be silently dropped or replaced.

The lexer preserves longest-match `>>` as one raw token. In an expression it remains the right-shift
operator. In a type-argument closing position the parser consumes its two source bytes as two
adjacent `>` delimiters, so `Option<Option<T>>` needs no separating whitespace. The lossless token
stream keeps the original `>>` spelling, while nested type-node spans may end at the byte boundary
between its two characters.

## Keywords and contextual words

The reserved keywords are:

```text
as assert break case cleanup const continue discard else enum extern false fn for
if import in let module mut null ptr pub ref region resource return
struct switch true unsafe var while
```

Primitive type names, `state`, `cell`, `map`, `this`, and the `C` in `extern C` are contextual names
rather than globally reserved keywords. The state words gain meaning only at module-item start or
inside an already recognized state declaration. `this` gains receiver or cleanup meaning only in the corresponding
owner-qualified function or resource cleanup block. A contextual name is interpreted specially only
in its owning grammar position.

## Literals

### Integer literals

Decimal, hexadecimal, and binary integers are accepted:

```fy
0
1_000_000
0xDEAD_BEEF
0b1010_0101
```

An underscore may appear only between two digits of the active radix. Octal spelling, a trailing underscore, doubled underscores, and type suffixes are rejected. A leading `-` or `+` is a prefix operator and is not part of the literal token.

The lexer stores the raw spelling and an arbitrary-precision mathematical value. Type context selects an integer type; absent a constraining context, the default is `i32`. Range checking happens after type selection and never wraps silently.

### Floating literals

Floating literals are decimal and require either a decimal point with digits on both sides or an exponent:

```fy
1.0
1_024.5e-2
2e10
```

Hexadecimal floats, `.5`, `1.`, `NaN`, infinity spellings, and type suffixes are excluded. The lexer preserves an exact decimal representation. Type context selects `f32` or `f64`; absent a constraining context, the default is `f64`.

### String-family literals

`"..."` produces an immutable `ref str` with `Static` borrow origin. Its decoded contents must be valid Unicode scalar values and are encoded as UTF-8.

`b"..."` produces an immutable `ref u8[]` with `Static` origin. `c"..."` produces an immutable
`ref CStr` with `Static` origin; the compiler appends exactly one trailing NUL, and a decoded
interior NUL is rejected as `E0006`.

All three forms support `\\`, `\"`, `\n`, `\r`, `\t`, `\0`, and `\xNN`. Text strings additionally support `\u{H}` through `\u{HHHHHH}` for a valid non-surrogate Unicode scalar. Byte and C strings reject `\u{...}`; non-ASCII source characters are encoded as their UTF-8 bytes. Unknown, truncated, out-of-range, or surrogate escapes are `E0005`. A text literal whose complete decoded byte sequence is not valid UTF-8, including invalid combinations of otherwise well-formed `\xNN` escapes, is also one `E0005` with no automatic repair.

Raw strings, character literals, multiline strings, implicit concatenation, locale conversion, and interpolation are excluded. Adjacent literals remain separate tokens and require an explicit operation.

### Boolean and null literals

`true` and `false` are `bool`. `null` is contextually valid only for a raw pointer type. Safe
references use `Option<ref T>` and cannot be initialized with `null`.

## File-to-module identity

The compiler driver supplies exactly one package source root. A `.fy` file's relative path maps directly to its declared module:

```text
src/main.fy       -> module main;
src/net/http.fy   -> module net::http;
src/a/b/c.fy      -> module a::b::c;
```

Every directory segment and file stem must be a valid Stage 0 identifier. The extension is removed, path separators are normalized, and no package name is prepended. `main.fy`, `lib.fy`, and `mod.fy` have no special mapping. A symlink or normalized path that escapes the source root is rejected before module registration.

The comparison is case-sensitive on every host. Two paths or module declarations that differ only by ASCII case are distinct identities but are rejected together when their coexistence cannot be represented safely on the host filesystem. A declaration mismatch is `E0002`.

## Operator table

Higher binding power groups first. The Pratt parser uses this table as executable data rather than
duplicating precedence in control flow.

| Binding power | Form | Operators | Associativity or policy |
| ---: | --- | --- | --- |
| 140 | Postfix | call `()`, index `[]`, field `.`, propagate `?` | Repeating left chain |
| 130 | Prefix | `-`, `+`, `!`, `~`, `ref`, `ref mut`, `*` | Right |
| 120 | Cast | `as Type` | Repeating left chain |
| 110 | Multiplicative | `*`, `/`, `%` | Left |
| 100 | Additive | `+`, `-` | Left |
| 90 | Shift | `<<`, `>>` | Left |
| 80 | Bitwise AND | `&` | Left |
| 70 | Bitwise XOR | `^` | Left |
| 60 | Bitwise OR | `|` | Left |
| 50 | Comparison | `==`, `!=`, `<`, `<=`, `>`, `>=` | Non-associative |
| 40 | Logical AND | `&&` | Left, short-circuit |
| 30 | Logical OR | `||` | Left, short-circuit |

Assignment is not an expression and has no binding power. A second comparison operator without parentheses is `E0009`. Parentheses create an explicit group node retained through CST-to-AST lowering.

Postfix forms may chain in source order. `?` terminates no chain by itself, so `value()?.field` parses; later type and ownership checks decide whether the operation is legal. Call arguments and operands evaluate left to right independently of precedence grouping.

`if`, `while`, and `switch` always require a parenthesized condition or scrutinee. The closing `)`
therefore separates a record literal inside the header from the following control-flow block without
lexer or name-resolution guesses.

## Recovery contract

Recovery produces a diagnostic CST, never a valid program. Any `E00xx` lexer or parser diagnostic blocks AST-to-HIR handoff.

At a failure site the parser must do exactly one of the following:

1. insert one grammar-declared missing token at a zero-width span and emit `E0007`;
2. consume at least one unexpected real token into an error node and emit `E0008`;
3. synchronize to a token in the current grammar-owned set without crossing an unmatched outer delimiter; or
4. exit the current production.

The parser may not loop while keeping the same token, delimiter state, and repair state. Every repair receives an identity so diagnostics caused only by that repair can be suppressed without hiding independent errors.

Synchronization sets are contextual:

| Context | Synchronization boundary |
| --- | --- |
| Source item | `;`, `}`, EOF, attribute or item starter |
| Block statement | `;`, `}`, EOF, statement starter |
| Parameter or argument | `,`, `)`, `]`, `}`, EOF |
| Record field | `,`, `}`, EOF |
| Switch case | `case`, `}`, EOF |

Missing closing delimiters are inserted only when the current production owns the opener. An unexpected `}` is never deleted merely to continue an inner production. Expected-token lists are deduplicated and sorted by canonical token spelling for deterministic diagnostics.

Exact input-size, token-count, node-count, nesting, repair, diagnostic, and work-unit limits remain UNDECIDED until implementation measurements exist. Before parsing untrusted source, one `ParseBudget` must charge all these dimensions and return `E0010` on exhaustion; wall-clock timeout alone is insufficient.

## Fixture contract

`spec/fixtures/parser/manifest-v0.json` lists syntax-pass and syntax-fail mini-packages. Each case owns one source root, so the module declaration is checked against the actual fixture path. Expectations record normalized expression trees or literal facts for successful cases and stable code, byte span, and repair kind for failures.

Syntax-pass means only that lexing, parsing, recovery, and module-path checks succeed. It does not
claim name resolution, type checking, ownership checking, or execution success. The parser fixture
runner compares every expectation against a fresh full parse and rejects unlisted diagnostics.
