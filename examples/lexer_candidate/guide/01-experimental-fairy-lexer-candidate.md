[Candidate guide index](../README.md)

# Experimental Fairy lexer candidate

This is the first Fairy-written recognition slice, built by the Go stage0
compiler for Windows AMD64 MSVC. It is not a compiler replacement or stable CLI.
`src/scanner.fy` recognizes original bytes directly; Go does not supply its tokens.

Version 0.61.0 adds `--c-modules-file <source-list>` for bounded multi-file nominal
programs. See [module program progress](../../../docs/ops/candidate-module-progress-20261008.md)
for the source-list rules, execution evidence and remaining SDK/package limits.

The scanner can also be tested after the Fairy candidate emits its C transport:

```sh
python examples/lexer_candidate/verify_scanner_corpus.py --images /path/to/candidate-dev /path/to/candidate-release --receipt /path/to/corpus-receipt.json
```

Run from the repository root on Windows AMD64 with MSVC, Bun and Go. The observer
compiles original scanner/SDK sources plus a test-only console framing fixture.
It reuses the checked-in Go lexer corpus for current/previous recognition,
malformed sources, mutations and four budget variations. Each C native profile
builds once per candidate and reuses that image across bounded batches. The Go
reference supplies no recognition results to the tested native process. This
checks the scanner component; project commands and compiler promotion remain
separate gates.

Example version 0.19.0 adds C11 emission and native execution verification for
same-file nominal structs (see [Owned nominal C](04-generic-declarations-and-sdk-observations.md#owned-nominal-c)). The candidate
owns recognition, typing, HIR, MIR and C generation; the native C compiler and
runtime link the resulting unit. This remains an experimental compiler slice.

Unit 22 connects same-file resource roots to owned HIR/MIR drop definitions
and native C execution at ordinary scope fallthrough. Live roots drop in reverse
declaration order; whole-value moves and consuming parameters transfer one
obligation. Empty cleanup needs no runtime call; nonempty cleanup calls an exact
private typed function whose receiver cannot move or recursively drop itself.
Direct scalar receiver updates preserve that same execution view. Resource
loop exits, overwrite, owned fields/tagged values and borrow expansion remain
the following numbered units. Native evidence is in
`docs/ops/evidence/candidate-resource-exit-20261007.json`.

Unit 23 adds direct/nested return cleanup, resource-value return transfer and
scalar Result `?` cleanup. Return values are frozen before remaining roots drop;
the error-return chain does not alter the continuing success path. Resource
factory/call/if/switch returns and terminal short-circuit paths execute. Evidence
is `docs/ops/evidence/candidate-resource-return-20261007.json`; owned tagged
payloads remain unit 27.

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

## Regular file token dump CLI (`--tokens-file`)

A new front end adds a regular file mode without changing the binary stdin protocol:
the no-argument frame reader stays exactly as described below, and
`--tokens-file <path>`, `--declarations-file <path>`,
`--check-names-file <path>`, `--bindings-file <path>`, `--check-types-file <path>`,
`--check-modules-file <manifest>`, `--hir-file <path>`, `--mir-file <path>`, `--c-file <path>`, and `--help`
are the file-mode arguments.

```sh
lexer-candidate --tokens-file path/to/source.fy
```

A checked-in `samples/token-demo.fy` demonstrates it:

```powershell
.\build\lexer-candidate.exe --tokens-file samples/token-demo.fy
```

The supplied path is UTF-8. `src/file_cli.fy` opens it read-only and reads at most
32768 source bytes plus one sentinel byte, so any 32769th byte trips the existing
byte limit. It collects current-revision tokens only, through the same
`scanner::collect` entry point and the default byte/token/diagnostic/work budgets
used by the binary stdin driver; it does not accept the previous revision and does
not add scanner rules. It accepts any supplied regular file name, with no `.fy` suffix
requirement, and the path argument is bounded at 4095 bytes.

`src/token_output.fy` writes one TSV row per token, including trivia and the single
EOF token. Each row has six tab-separated columns: a human kind label, a human
trivia label, the half-open start and end byte offsets into the original source, and
the lowercase hexadecimal bytes of the raw source slice and of the scanner's cooked
representation. Dumping reuses the existing read-only Stream access
(`token_count`, `token_info`, `token_raw`, `token_cooked`, `source_view`) and
releases the Stream afterwards; no candidate tokens are precomputed and Go supplies
none. The dump is capped at 1 MiB: each row's conservative bound (96 plus twice the
raw length plus twice the cooked length) is checked before append.

Any scan diagnostic makes the dump fail closed: stdout stays empty, the process
exits 2, and stderr carries one line per diagnostic in recognition order:

```text
path:LINE:COLUMN: error: E000x (bytes START..END)
```

`path` is the path exactly as supplied, `LINE` and `COLUMN` are 1-based byte
positions over the original source, and the half-open `START..END` bytes are that
diagnostic's span. CLI I/O, usage, allocation and candidate-limit failures print
plain messages without the path/line/column prefix, and `--help` prints the usage.

Scanner rules, the compiler and the SDK are unchanged; this is a read-only reporting
front end over the existing candidate. The default diagnostic budget and the
`E0010` exhaustion contract described above are unchanged.

Portable verification uses a test-only Go reference fixture producer plus
`examples/lexer_candidate/verify.py` (Python standard library); both commands run from
the repository root:

```sh
go run ./tools/dev/fairyc-lexer-file-oracle .cache/lexer-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/lexer-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The `go run` form is the portable default; an optional workspace wrapper is:

```sh
bun tools/run-go.ts run ./tools/dev/fairyc-lexer-file-oracle .cache/lexer-file-cli/reference.json
```

The file CLI passes on Windows development and release: 29 Go reference files per
profile with their 29 original hashes preserved across 82 CLI executions, covering the
file, I/O, usage, `--help` and legacy-protocol cases. Accepted tokens are compared
exactly for kind, trivia, byte range, raw and cooked bytes, and byte reconstruction;
diagnostics are compared as a multiset of code, span and byte line/column, because the
candidate reports in recognition order while Go sorts. An output-limit fixture
alternates identifiers and spaces to 32768 source bytes, for which the Go oracle
expects more than 1 MiB of TSV while the native binary exits 2 with the output-limit
message on stderr and empty stdout. The narrow `TestLexerFileCLIWorkspace` test, the
CLI vet, the Go oracle vet, and the Linux and Windows C emission checks pass. Actual
Linux execution remains deferred to the owner's consolidated batch. This section makes
no performance claim, and allocation-failure or OS I/O fault injection was not run.
Evidence is `docs/ops/evidence/lexer-file-cli-20261005.json`.

## Regular file declarations CLI (`--declarations-file`)

`--declarations-file <path>` reads the same bounded regular file and reuses
`scanner::collect` plus the byte-only `syntax::parse(stream, 512)`. It is a standalone
structure dump, not a project check, so there is deliberately no basename, module or
path identity validation.

On success it writes one TSV row per declaration in source order with six columns:

```text
kind	name	start	end	name_start	name_end
```

`kind` is one of `module`, `import` (the alias is the name), `function`, `struct`,
`resource`, `const`, `static-assert`, `foreign-function`, `state`, or `enum`. An
unnamed `static-assert` or `state` has an empty `name` and `0..0` name spans; method
name spans keep the member spelling.

Lexical diagnostics keep the existing `--tokens-file` code format. Any parser problem
returns 2 with empty stdout and one located line per problem:

```text
path:LINE:COLUMN: error: expected TEXT (bytes START..END)
```

`syntax::Problem.code == 4` with `expected == 0` gives unsupported syntax; a parse
error 4 gives plain parser-limit. Go parser diagnostic IDs are not fabricated, and
allocation failures stay plain. The same 32768-byte file cap, 4095-byte path bound and
1 MiB output cap apply, and the 512-declaration parser limit counts module and import
items while the existing parser node, field and depth limits are retained.

Portable source invocation:

```sh
lexer-candidate --declarations-file samples/token-demo.fy
```

Optional Windows executable form:

```powershell
.\build\lexer-candidate.exe --declarations-file samples/token-demo.fy
```

`src/declaration_output.fy` writes the dump. Reference comparison runs from the
repository root and needs both references:

```sh
go run ./tools/dev/fairyc-lexer-file-oracle .cache/declarations-file-cli/tokens-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --declarations .cache/declarations-file-cli/declarations-reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/declarations-file-cli/tokens-reference.json --declarations-reference .cache/declarations-file-cli/declarations-reference.json --receipt /path/to/evidence/receipt.json
```

The Windows development and release profiles pass 156 CLI executions (78 per profile),
with 29 token fixtures and 29 declaration fixtures per profile; the per-profile
declaration set is 15 accepted, 9 syntax, 4 lexical and 1 limit. Declaration output is
compared exactly for kind, name, start, end, nameStart and nameEnd against Go
`syntax.Parse` for accepted inputs, and negative inputs are checked against authored
diagnostic goldens; original source hashes are preserved. `TestLexerFileCLIWorkspace`,
`go vet ./internal/cli ./tools/dev/fairyc-lexer-file-oracle`, and the Linux and Windows
`--locked` C emission checks pass, and the 619 compiler inputs match the manifest with
the compiler EXE hash unchanged. The example package is version 0.2.0 with no compiler
or SDK change. Evidence is `docs/ops/evidence/declarations-file-cli-20261005.json`; the
earlier token-dump receipt above remains the historical record for that mode.

## Regular file name checking (`--check-names-file`)

```sh
lexer-candidate --check-names-file samples/names-demo.fy
```

This private name-only candidate reads the same bounded file and reuses the scanner
and parser. `src/name_resolution.fy` collects ordinary function names before checking
parameters and source-ordered local bindings. The rules follow
`docs/language/0007-name-resolution-and-typing-v0.md` and the Go resolver in
`internal/semantic/resolve.go`. Functions can reference later functions. Parameters
occupy their own scope; the function body and each nested block open child scopes.
Same-scope duplicates fail, nested shadowing is allowed, and an initializer resolves
before its new local is registered. A local is invisible after its owning block ends.

The admitted subset is a module plus non-generic ordinary functions with optional
`pub`, scalar written types (`i32`, `i64`, `u8`, `u32`, `usize`, `bool`, result-only
`void`), inferred locals, integer/bool literals, simple names, arithmetic/comparison/
boolean expressions, ordinary positional calls, parenthesized expressions, nested
blocks, `let`/`var`, returns and simple-name assignment targets. Version 0.6.0
also admits the control-flow name scopes described below. This checks names only: call arity, types, mutability, return obligations, overflow and ownership are
unchecked. For example a named but mistyped return or assignment to a `let` can pass
this command. Those are later type/flow checks, not valid-program claims.

Imports, constants, composite types, fields, references, labeled calls, generics,
attributes, methods, unsafe forms, `switch` patterns and the `source_location` intrinsic
are rejected as unsupported. Output is `names ok` plus newline only after all checks
succeed. The first failure returns 2 with empty stdout and:

```text
path:LINE:COLUMN: error: MESSAGE (bytes START..END)
```

Messages are `unresolved name`, `duplicate name in scope`,
`unsupported name-check syntax`, and `name-check limit`. These are private candidate
messages, not new public Go diagnostic codes. Scanner/parser errors retain their
existing file-mode format. Limits are 128 module functions, 128 simultaneously active
parameters/locals, and 64 active walker levels, in addition to existing file/parser
limits. Sibling blocks release their active-name slots. One error is reported; this
candidate does not provide full diagnostic recovery or bound HIR/MIR.

From the repository root, generate the three test references and run the portable
harness against the desired native compiler:

```sh
go run ./tools/dev/fairyc-lexer-file-oracle .cache/names-file-cli/tokens-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --declarations .cache/names-file-cli/declarations-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --names .cache/names-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/names-file-cli/tokens-reference.json --declarations-reference .cache/names-file-cli/declarations-reference.json --names-reference .cache/names-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

Windows dev/release pass 276 CLI executions (138/profile). Each profile includes
50 name fixtures: 19 accepted, 8 unresolved, 3 duplicate, 17 unsupported and 3 limits;
accepted names and the exact unresolved/duplicate spans are independently checked
against Go semantic analysis. The 29 token and 29 declaration fixtures per profile,
malformed input, CLI/file errors and framed stdin smoke also pass. Source hashes are
unchanged before/after every fixture execution. Existing historical receipts remain
separate. Additional target/dependency checks are recorded in
`docs/ops/evidence/names-file-cli-20261005.json`. Linux native execution is deferred to
the owner's batch; allocation/I/O fault injection and performance testing were not
run. This changes the example package to 0.3.0; compiler, SDK and language contracts
are unchanged.

## Retained declaration-use links (`--bindings-file`)

```sh
lexer-candidate --bindings-file samples/names-demo.fy
```

The name resolver now retains its exact `syntax::Parsed` owner together with symbol
and reference vectors in `name_resolution::Resolved`. `resolve(parsed)` consumes
that syntax owner; `problem`, `parsed`, `source_view`, `symbol_count`, `symbol_at`,
`reference_count`, and `reference_at` are shared queries. A symbol ID is a one-based
index local to that result, not a durable cross-file/compiler identity. Every name
expression retains its expression ID, use span and actual target symbol ID.
Symbols retain kind, name span, unique owning scope, function owner, written type
root and syntax declaration/entry coordinates for the next type phase. Locals have
independent IDs even when sibling blocks reuse an active-name slot. Function symbols
are collected first, then parameter/local symbols in traversal order. Failed results
retain source for diagnostics, but table queries assert success and the CLI emits no
partial table. Builders own separate vectors while collecting; the completed resource
owns and cleans them alongside syntax, without mutable resource-field reborrows.

The command has the same admitted syntax and first-error behavior as name checking.
Success emits a TSV table with columns:

```text
record	id	kind	start	end	scope	owner	target
```

A `symbol` row uses `function`, `parameter`, `local-let` or `local-var`; its target is
zero. A `reference` row has kind `name`, its ID is the syntax expression ID, scope and
owner are zero, and target is the symbol ID. Spans identify source bytes; function
symbols use scope/owner zero. Output is fully buffered before publication and bounded
at 1 MiB. Parser bounds also cap storage at 128 module functions plus 1024 entries and
at most 4160 expression-use links.

The portable verification command in the preceding section accepts the regenerated
`--names` reference, now including Go AST symbol inventories and exact Go semantic
use-to-declaration resolutions. The Windows dev/release harness passes 396 CLI executions (198/profile).
Both name and binding file modes are checked for all 50 name fixtures, file/CLI errors, malformed syntax and invalid bytes. Per-profile
accepted inventories and 26 reference targets are compared independently with Go, including
forward calls, shadow initializer targets, restored outer names and sibling scopes.
`tests/bindings.fy::retained_symbol_links` reads the moved source owner and verifies
symbol/type/entry and expression links in both native profiles. Interpreter execution
is unavailable because this component uses the native-only `std::vec` SDK path.
Evidence is `docs/ops/evidence/bindings-file-cli-20261005.json`. The example version is
0.4.0; compiler, SDK and language contracts are unchanged. Linux native execution is
still deferred to the batch; no allocation-failure or OS I/O injection was run.

## Scalar type checking (`--check-types-file`)

```sh
lexer-candidate --check-types-file samples/types-demo.fy
```

Success emits `types ok` and exit 0 after lexing, parsing, name resolution and
checking the complete file. Failure emits one source/line/byte-range diagnostic
to stderr, exit 2 and no stdout. The input file is read only. This mode checks
`i32`, `i64`, `u8`, `u32`, `usize`, `bool` and `void` function results, typed or
inferred locals, arithmetic/bitwise/shift/comparison/boolean operators, immutable
assignment, direct function calls, argument count/types and return types. It
applies ADR 0417's fixed-peer contextual integer rules without implicit width
conversion, checks each literal's range, and preserves the direct-token signed
minimum exception in decimal, hexadecimal and binary.

The reusable `type_check::Typed` resource owns the resolved source and a bounded
SDK vector of scalar type slots. `problem`, `source_view`, `resolved`,
`symbol_type`, `expression_type` and `signed_minimum` expose the retained facts;
type-table queries require success. Private type codes are 0 absent/linkage,
1 i32, 2 i64, 3 u8, 4 u32, 5 usize, 6 bool, 7 void and 8 callable. Function
symbol types describe their results; a callee expression uses code 8. Postorder
linkage nodes carry no value type. Only a direct negative minimum sets the
signed-minimum flag, including its separately retained magnitude child type.

Slots use bounded heap storage, avoiding large fixed-array copies on the Windows
native stack. Allocation failure rejects the file. Additional limits are 4096
syntax expression nodes, 1024 entries, 1152 symbols and 64 active checker levels;
existing source/parser/name limits also apply. Statements after a return are
still checked. The return gate is extended by the control-flow section below;
ownership/initialization Flow remains subsequent work. Imports, composite types,
casts and generics are currently rejected by the preceding name phase.

Generate independent test data and run the portable harness from the repository
root (no PowerShell runtime is required):

```sh
go run ./tools/dev/fairyc-lexer-file-oracle .cache/types-file-cli/tokens-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --declarations .cache/types-file-cli/declarations-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --names .cache/types-file-cli/names-reference.json
go run ./tools/dev/fairyc-lexer-file-oracle --types .cache/types-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/types-file-cli/tokens-reference.json --declarations-reference .cache/types-file-cli/declarations-reference.json --names-reference .cache/types-file-cli/names-reference.json --types-reference .cache/types-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The type fixtures compare pass/fail and the selected error span against Go
semantic analysis, while unsupported syntax and the candidate's storage cap have
explicit rejection checks. CLI/file errors, malformed syntax/UTF-8, source
preservation, existing modes and the framed stdin protocol run in both Windows
native profiles. `tests/types.fy` checks retained symbol/expression types and
minimum flags. Evidence is `docs/ops/evidence/types-file-cli-20261005.json`;
the example is version 0.5.0, with compiler/SDK inputs unchanged. Linux C emission
is checked; Linux native execution remains deferred to the owner's batch.
Allocation/OS fault injection and performance experiments were not run.

## Control-flow name scopes

Version 0.6.0 extends `--check-names-file` and `--bindings-file` to `if/else`,
`else if`, `while`, classic `for`, `assert`, `discard`, `defer`, `break` and
`continue`. Branches and loop bodies have independent lexical scopes. A `for`
initializer has a separate scope visible to its condition, post clause and body;
the post clause is resolved before body-local declarations, and the initializer
binding disappears after the loop. Shadow initializers still see the outer name.

```sh
lexer-candidate --check-names-file samples/control-names-demo.fy
lexer-candidate --bindings-file samples/control-names-demo.fy
```

This is name resolution only: condition/operator types, loop-control validity,
return coverage and defer-call restrictions belong to the following type-phase
unit, connected in version 0.7.0 below. `switch` patterns require the later enum/Option/
Result type component. File, name-slot, recursion, first-error and no-partial-
output policies remain the same. Native `tests/name_control.fy` checks the exact
use targets across outer parameters, loop initializers, body shadows and branches.
Evidence: `docs/ops/evidence/control-names-cli-20261005.json`.

## Control-flow type checking

Version 0.7.0 connects `--check-types-file` to the control-flow name scopes above.
`if/else` and `else if` require bool conditions, propagate the destination type to
branch values and require matching branch results. `while`, classic scalar `for`
and `assert` require bool conditions. `break`/`continue` require an enclosing loop,
and nested scopes preserve the loop context until that loop's body finishes.

Classic `for` admits optional initializer, condition and post clauses. Its header
uses the current scalar subset and rejects calls, blocks and other effects under
ADR 0213; body expressions retain ordinary call semantics. `defer` admits direct
internal void calls with scalar arguments under ADR 0217. Both argument types and
arity are checked. `discard` checks its operand. The candidate performs static
analysis only: it never executes a loop, assertion, deferred call or input program.

The function return gate follows the Go reference: direct return, an exact
`assert(false)`, or complete nested `if/else` coverage satisfies the gate. A loop
by itself is not return proof, and a plain standalone block's nested return is not
promoted. All source branches and statements after a return are still checked.
Ownership/initialization Flow and code generation remain later components.
`switch` requires the later composite-type pattern implementation; it stays
explicitly unsupported here.

```sh
lexer-candidate --check-types-file samples/control-names-demo.fy
```

The portable `--types` oracle now adds normal and rejected control cases from
`tools/dev/fairyc-lexer-file-oracle/control_types.go`; the existing verification command
uses freshly regenerated references. `tests/type_control.fy` reads actual retained
branch and comparison types, contextual u8 branch values, and return/loop results
in both native profiles. Evidence: `docs/ops/evidence/control-types-cli-20261005.json`.

## Multi-file scalar checking (`--check-modules-file`)

Version 0.8.0 connects explicit imports to the existing name/type/control-flow
walkers. Run from this example directory:

```text
lexer-candidate --check-modules-file samples/modules.txt
```

The UTF-8 manifest contains one unquoted, untrimmed relative filename per line.
Paths resolve against the process working directory, including `..`; this is
explicit read-only input selection. LF, CRLF and no final newline are accepted.
Empty lines, BOM, control bytes, absolute entries and colon/ADS spelling are
rejected. Spaces and Korean filenames are accepted. The manifest filename itself
uses the existing CLI UTF-8 path reader. Limits are 8192 manifest bytes, 4095
bytes per path, 32 files and 32768 bytes per source, with the existing parser and
name/type budgets. Duplicate source selection fails as a duplicate module.

`module_registry.fy` builds module identities, import aliases and scalar function
signatures from the actual parser-owned syntax. Token normalization ignores
comments/trivia in qualified paths without rewriting the source. All files are
registered before imports resolve, so manifest order, forward imports and import
cycles do not prevent lookup. Each import alias and ordinary function name shares
the module declaration namespace. Only an explicit `alias::function` reaches a
public item in its target module; imported aliases do not re-export items. Private
functions remain usable within their own module.

`name_resolution::resolve_modules` retains each qualified expression's declaration
link and copied scalar signature. `type_check` applies the real imported parameter
and result types to the existing contextual integer and call checks. Every source
body is checked, including private functions and modules not called by another
file. Local variable names may shadow an alias for unqualified references without
changing the qualified import lookup. `module_check.fy` and `module_cli.fy` publish
`modules ok` and exit 0 only after the whole set succeeds. Failures exit 2 with
empty stdout and an input or source-location diagnostic on stderr. Inputs are
never modified. Single-file modes retain their existing import rejection.

This mode receives an explicit source set, not a resolved workspace/package graph:
it does not discover files, validate filename/module correspondence, parse
`fairy.toml` or locks, select SDK dependencies, or implement cross-package identity.
Reserved `source_location` item/alias bindings fail closed. Constants, composite
values, generic functions, attributes, effects and ownership
remain outside this scalar component. Unsupported constructs fail closed.

Generate Go semantic reference observations and verify both native profiles:

```text
go run ./tools/dev/fairyc-lexer-file-oracle --modules .cache/modules-file-cli/reference.json
python examples/lexer_candidate/verify.py --compiler /absolute/path/to/fairyc --reference .cache/modules-file-cli/tokens-reference.json --declarations-reference .cache/modules-file-cli/declarations-reference.json --names-reference .cache/modules-file-cli/names-reference.json --types-reference .cache/modules-file-cli/types-reference.json --modules-reference .cache/modules-file-cli/reference.json --receipt /path/to/evidence/receipt.json
```

The portable Python harness prepares files only; registry/resolution/typing run in
the Fairy executable. Semantic rejection categories are compared to Go diagnostics,
and candidate byte ranges are checked against the selected source. Candidate
messages are private, not the stable Go diagnostic schema. Manifest/IO/syntax
controls check exact status/output and input preservation. Native `tests/modules.fy`
checks retained import signatures and resolution/type facts. Evidence:
`docs/ops/evidence/modules-file-cli-20261005.json`. Windows native execution is
checked locally; Linux native execution remains deferred to the owner's batch.
