# Private package syntax and filename identity

This increment changes only the private candidate and its probes. The owning Go
parser, Stage 0 grammar, stable compiler/SDK/runtime versions and the checked-in
22-case fixture manifest are unchanged. The byte-only entry point remains intact.

## Preserved syntax

- Declaration 9: mandatory `unsafe extern C fn`, exact declaration/name spans,
  written result type, ordered parameters and terminating semicolon; no body.
- Expression 31: unsafe keyword and complete block span, a statement-block child,
  postfix/infix participation and the existing shared depth/node budgets.
- Declaration 10: nameless state item and ordered tagged fields. Field kind 0 is
  struct/resource, 1 is cell, 2 is map; only maps have a required key type in
  addition to the value type. All share the existing 1024-field limit and rollback.
- `file_syntax::parse`: caller-supplied relative path, lexical slash/backslash and
  dot/parent cleanup, ASCII identifier and `.fy` checks, original source tokens
  including trivia retained. There is no filesystem access. Private errors 6 and
  7 distinguish invalid identity from module mismatch; syntax failures remain
  owned incomplete results. Limits are 32768 path bytes and 1024 live segments.

The candidate is a bounded surface representation, not the Go lossless CST.
Kinds and source/name ranges are compared at declaration level; focused tests
also compare state field and key/value type spans. No storage semantics, ABI
calling behavior, unsafe execution permission or constant evaluation is added.

## Verification

`fairyc_go_test_parser_package_items` passed in low and optimized Windows MSVC
profiles (56.806 seconds in the final complete package run). It covers foreign
declarations and malformed-prefix recovery; unsafe nesting, postfix combinations,
semicolon rules and depth exhaustion; cell/map shape, rollback and 1024/1025 field
boundaries; filename normalization, mismatch, invalid paths, segment/byte limits,
syntax-error precedence and unchanged byte-only behavior.

`fairyc_go_test_parser_evidence` passed (15.012 seconds in the final run). The
unchanged manifest now reports 12 supported valid cases and 10 matching rejections,
with zero unsupported valid cases and zero context-gap exceptions. Both profiles
produce identical observation records. The module-mismatch fixture reports private
status 7, not Go diagnostic-ID/span parity. Integrity validation also passed.

`fairyc_go_test_parser_surface` passed (43.309 seconds): diagnostics, postfix
shape, struct regressions and all 16 current candidate source files. Both profiles
fully parse each source and match Go declaration kind/order/full/name spans.
The filename adapter is 3839 bytes with 11 declarations; the largest candidate
source remains `bodies.fy` at 31625 bytes, below the unchanged 32768-byte input cap.

Implementation-time native compilation rejected repeated fixed-array initialization,
a test-helper result branch, path-dependent resource moves and an `i32` to `usize`
reporting cast. These were candidate/probe integration failures, not language
extensions: bounded `Vec` storage, a consuming result helper, the existing buffer
copy pattern and explicit status constants resolved them before successful reruns.

## Boundaries

This is bounded fixture evidence, not complete language conformance, full expression
equivalence, semantic validity, cross-arena acyclicity, unique ownership or fuzz/
scale evidence. Raw pointer and foreign function types remain outside the candidate
subset. No stable API, database, deployment or generated artifact changes are
included. The new parent-owned verification lane is a closed-stdin, offline,
180-second one-shot with cache-only writes. Full compiler suites and benchmarks
were not selected because no compiler, runtime or performance contract changed;
no child-specific documentation validator is configured. No push was performed.
