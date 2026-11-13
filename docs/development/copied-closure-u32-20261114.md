# Direct copied-closure u32 family — 2026-11-14

ADR 0558 extends direct copied closures with u32 captures, arguments, results
and immutable locals, using the existing private ordinary-function lifting.
First-class `fn i32(i32)` values retain their existing all-i32 family and ABI.
Resource/borrow/pointer captures and escaping direct bindings remain closed.

Executed Go checks:

- Existing copied-closure and first-class closure compiler/semantic controls
  pass (38.832 s compiler, 0.693 s semantic).
- Focused u32 source/native/MIR restore and first-class ABI refusal checks pass
  (2.417 s). Authored syntax remains unchanged. The source exercises snapshots
  after original mutation, UINT32_MAX identity, four captures/four arguments,
  and typed immutable u32 locals.

`TestCandidateCopiedClosureU32Pipeline` passes (280.13 s), with two producer
and two consumer profiles. All four actual products return 42 with empty
output, through Fairy source checking, project codecs, current-source restore
and C partitioning. Existing i32 ordered argument evaluation, borrowed-view
iteration and scalar Vec iteration remain active alongside reachable u32 tests.

The replay additionally required synchronizing the already-changed owning Vec
SDK with its stale candidate source pin/data; see the separate SDK sync receipt.
No descriptor/artifact tag or language version was changed. No UI/input test.
Wider first-class signatures and owned/borrowed captures remain future families.

The separate native candidate admission/value-policy guard passes in both
profiles (21.41 s), including rejection of u32 locals in first-class literals.
