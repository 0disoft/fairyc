# Private parser evidence — 2026-09-11

This extends the preceding thirteen-file closure milestone without widening the
stable grammar, compiler API, SDK, runtime or candidate input budgets. The Go
frontend and unchanged `spec/fixtures/parser/manifest-v0.json` remain the reference;
this report is an observation, not a language specification.

## Own-source declarations and integrity

All fifteen current candidate source files parse with zero problems under both
low and optimized Windows MSVC profiles. The native declaration list matches the
Go CST in kind, order, full byte span and name/alias byte span. These named files
are regression gates; newly discovered files are observations until admitted.

| Source | Bytes | Declarations |
| --- | ---: | ---: |
| attributes.fy | 4136 | 13 |
| bodies.fy | 28770 | 31 |
| budget.fy | 574 | 4 |
| diagnostics.fy | 3982 | 14 |
| integrity.fy | 7621 | 13 |
| main.fy | 2916 | 9 |
| numbers.fy | 9859 | 12 |
| pages.fy | 3949 | 14 |
| parse_tokens.fy | 1860 | 6 |
| records.fy | 3547 | 10 |
| scanner.fy | 20161 | 43 |
| strings.fy | 9355 | 15 |
| syntax.fy | 25858 | 66 |
| types.fy | 8149 | 18 |
| utf8.fy | 807 | 3 |

The private `integrity::valid` checker verifies kind-specific references, backward
same-arena links, source/token bounds, block link counts and declaration ranges.
The native corpus driver checks every published parse result, including partial
results. Synthetic copies exercise missing, out-of-range, self-referential and
wrong-kind links without mutating the published result.

These checks do not prove full expression-tree equivalence, semantic validity,
cross-arena graph acyclicity, unique ownership or full source coverage.

## Nonrecursive helper extraction

`parse_tokens.fy` owns trivia scanning, exact word matching and operator tables.
`bodies.fy` retains cursor mutation, arena appends and recursive expression,
statement and block parsing. Its size fell from 30070 to 28770 bytes; the new
helper is 1860 bytes. Neither the 32768-byte source cap nor recursive parsing
budgets changed. The two additional admitted files are the integrity checker and
token helpers, not two new language features.

## Existing parser fixture observations

`TestWindowsFairyParserFixtureCoverage` reads the existing manifest unchanged,
with at most 32 cases and 32768 bytes per case. All 22 current cases were observed
under both profiles, with identical native integer records. The Go reference
must still agree with each fixture's expected pass/fail status.

| Classification | Count | Meaning |
| --- | ---: | --- |
| supported | 4 | Valid syntax accepted; declarations also match Go |
| unsupported | 8 | Valid reference syntax not completely accepted by candidate |
| rejected | 9 | Invalid reference syntax rejected by candidate |
| context-gap | 1 | Candidate lacks file/module identity input |

No budget exhaustion or unexplained false acceptance occurred. Either fails the
lane, as do unexpected native statuses and regressions/removal of the four
currently supported fixtures. Rejection classification does not assert identical
diagnostics or rejection reasons.

The supported fixtures are `pass.module-path`, `pass.mutable-parameter`,
`pass.nested-type-close` and `pass.array-forms`.

| Unsupported valid fixture | First problem byte span | First blocker |
| --- | --- | --- |
| pass.precedence | 220..221 | `?` |
| pass.literals | 18..23 | `const` declaration |
| pass.postfix-chain | 114..115 | `?` |
| pass.classic-for | 57..60 | `for` |
| pass.lexical-defer | 114..119 | `defer` |
| pass.foreign-call | 51..57 | `unsafe extern` declaration |
| pass.compile-time-items | 28..33 | `const` |
| pass.package-state | 54..59 | `state` |

These are first blockers, not exhaustive feature diagnoses. In particular,
`pass.literals` stops at its declaration form; it does not establish that every
literal used in that fixture is unsupported.

The matching rejections are invalid numeric separator, invalid escape, invalid
UTF-8 string escape, C-string interior NUL, missing semicolon, comparison chain,
import without alias, missing delimiter and unexpected token.

`fail.module-mismatch` declares `module net::https` in `net/http.fy`. The Go
frontend receives the relative filename and rejects that mismatch. The candidate
currently receives source bytes only and accepts the syntax. This remains an
explicit input-contract gap, not a repaired behavior or general grammar exception.

## Verification and boundaries

Configured local lanes passed:

- `fairyc_go_format`.
- `fairyc_go_test_parser_surface`: diagnostics, postfix, own-source and structs.
- `fairyc_go_test_parser_operands`: arrays, casts, generic calls and references.
- `fairyc_go_test_parser_evidence`: integrity mutations and fixture observations.

Each native test uses low and optimized Windows MSVC profiles. The surface and
operand lanes passed after helper extraction; the evidence lane was rerun after
adding known-supported fixture guards. Scoped strict mustflow and whitespace
checks accompany delivery.

The full repository suite and remote CI were not run for this bounded private
candidate change. No child documentation validator is configured. Stable source
contracts, fixture expectations, API/DB/version files and public runtime behavior
were not changed. The parent runner gains a bounded evidence intent and exact
commit scopes; no server, background worker, deployment or push is involved.
