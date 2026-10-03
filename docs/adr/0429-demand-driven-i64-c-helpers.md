# ADR 0429: Emit only required i64 C helpers

- Status: Implemented
- Date: 2026-10-03

## Defect and decision

Drimi's Linux run of source revision
`d866070a3001e63e6ff4c48e9b6c9a220e4bc3c4` found that any i64 binary/unary
operation emitted all five static inline i64 helpers. Clang 19's unchanged
strict C11 `-Wall -Wextra -Werror` rejected unused right-shift and overflow
helpers when compiling the actual Fairy task manager.

`internal/cbackend/emit.go` now records helper demand from verified MIR:
bit-preserving unary/binary operations need bitcast; arithmetic right shift needs
right-shift and bitcast; checked addition/subtraction/multiplication need their
matching overflow predicates. Checked left shift also needs right-shift support
for the signed lower-bound check. Comparison, unary plus, borrow and ordinary
positive division/remainder need no i64 helper. Demand is taken from the whole
reachable emitted program, not guessed from source spelling.

`internal/cbackend/i64.go` emits each demanded helper and its dependencies only.
Helper function bodies, arithmetic semantics, trap categories, warning flags,
public syntax, SDK/language/runtime ABI identities and CLI schemas are unchanged.
The i32 unary-helper predicate also groups the operator alternatives correctly,
so i64 negation cannot accidentally emit an unused i32 bitcast helper.

## Evidence boundary

The returned patch was reviewed and applied to the current checkout without
replacing unrelated files. The return archive
`fairyc-drimi-linux-d866070a-result-20261003.zip` has SHA-256
`90fe77ee16e93f022f4d2bf02249c70cbe94d6edd1c5999e0aa99d7e84fe3dab`;
all 1,553 files listed in its evidence manifest were hash-verified locally.

Drimi observed GCC 14.2 and Clang 19.1.7 on Debian 13/glibc 2.41. After this
two-file patch, its fault-save receipt contains 40 passing cases, four native
images and no failed flag. An independent 32-source i64 oracle/native matrix
passed 128 executions. These results describe that external patched snapshot,
not the later native runtime-object cache or every test at the current HEAD.
Its original nine-test suite was 7 pass / 2 fail / 0 skip; the rerun retained a
separate stale-lock expectation failure. No full Linux suite pass is claimed.

Repository regressions cover demanded/absent helper sets on both C targets and
strict GCC/Clang native compilation/execution in both profiles on Linux. The
existing Windows i64 arithmetic/format/trap matrix remains applicable. Local
Windows execution cannot replace Linux native evidence; no warning suppression,
new runner, dependency, benchmark or power-loss experiment is introduced.
