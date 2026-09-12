# Private parser control syntax — 2026-09-12

This extends the [2026-09-11 baseline](parser-evidence-2026-09-11.md).
The owning language grammar and existing parser fixture manifest are unchanged.
Only the private candidate and its native probes gain syntax coverage; there is
no stable compiler, SDK, runtime, API, DB or version change.

## Preserved constructs

| Construct | Private representation | Evidence |
| --- | --- | --- |
| Postfix `?` | Expression 30, one operand, question-mark token and byte span | Go AST shape/span comparison for repeated chains, grouping, prefix/infix precedence and call/field/index combinations; malformed suffix rollback |
| `const` | Declaration 7, declared type in `result_type`, initializer in `expression` | Go declaration/name spans, public/attribute forms, initializer links/type spans, malformed declaration rollback and 511/512 constant item boundary |
| `defer` | Statement 13, written operand and block position | Nested values, spans, required-operand mutation, malformed operand/terminator rollback and 2047/2048 statement boundary |
| Classic `for` | Statement 14, optional condition, body and kind-15 header | All eight clause-presence combinations, binding/assignment initialization, post assignment, nesting, invalid clauses, wrong-kind/self-link mutations and node/depth limits |

The for header has optional initialization in `first` and optional post
assignment in `second`. Zero means the clause was not written. The initializer
statement includes its semicolon; the header span includes both parentheses.
The written body is retained, not desugared into a while loop. A loop condition
is not synthesized when omitted.

The private tree verifier recognizes these new kinds and validates their required,
optional and kind-specific links. Initializer statement classification reuses the
last successfully appended statement kind, avoiding a read-only borrow of a
mutably borrowed arena. It is read only after a nonzero statement result.

This does not implement propagation execution, constant evaluation, defer timing,
cleanup ordering, loop lowering or semantic validity. The for slice admits the
grammar's binding/assignment initialization and assignment post clause, not
arbitrary expression-only clauses accepted by Go's broader syntax layer for later
semantic rejection. Existing budgets remain unchanged: 32768 source bytes,
4096 expression/type/statement nodes, 1024 entries, 512 declarations, 32 block
levels and 64 active expression parses.

For loops hit the statement limit between 1023 and 1024 empty loops in one
function (4093 nodes versus exhaustion). Nested empty loops pass at 31 inside the
function body and exhaust at 32. These are targeted boundaries, not a general
hostile-input or semantic conformance claim.

## Existing fixture results

All 22 unchanged manifest cases were measured under low and optimized Windows
MSVC profiles. Complete native observation records agree across profiles.

| Classification | Previous | Current |
| --- | ---: | ---: |
| Supported valid syntax | 4 | 8 |
| Unsupported valid syntax | 8 | 4 |
| Matching rejection | 9 | 9 |
| File/module context gap | 1 | 1 |

Newly supported fixtures are `pass.precedence`, `pass.postfix-chain`,
`pass.lexical-defer` and `pass.classic-for`. All eight supported fixtures are
explicit regression/removal gates. Rejection means rejection agreement, not
identical diagnostic wording or spans. There was no budget exhaustion or
unexplained false acceptance in this fixture batch.

| Remaining valid fixture | First problem byte span | First blocker |
| --- | --- | --- |
| pass.literals | 134..144 | Floating literal `1_024.5e-2` |
| pass.compile-time-items | 55..68 | `static_assert` |
| pass.foreign-call | 51..57 | `unsafe extern` declaration |
| pass.package-state | 54..59 | `state` |

The literals fixture now gets past its constant declarations but still reports
two problems. First-blocker reporting is not an exhaustive diagnosis.
`fail.module-mismatch` remains a known input-contract gap: the candidate receives
bytes without the relative filename used by Go's module identity check.

## Validation scope

`fairyc_go_format`, `fairyc_go_test_parser_control_items`,
`fairyc_go_test_parser_evidence` and `fairyc_go_test_parser_surface` passed.
Both native profiles executed rather than skipping. All fifteen current candidate
sources parsed without problems and passed declaration comparison and integrity
checks. The changed source sizes are 31419 bytes for `bodies.fy`, 26518 for
`syntax.fy` and 8155 for `integrity.fy`, below the unchanged source cap.
Scoped strict mustflow and whitespace checks accompany delivery.

The first control-items run after adding for failed at compilation because a
mutable vector reference was passed to a read-only access function. The targeted
fix and rerun passed. A duplicate command name was corrected before execution;
shared root lock contention was retried without clearing another task's locks.

The full repository suite, remote CI and release checks are outside this bounded
private candidate change. No child documentation validator is configured.
The parent runner adds the bounded `fairyc_go_test_parser_control_items` intent
and exact commit scopes; no background process, deployment or push is added.
