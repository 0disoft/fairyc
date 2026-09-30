# Shared reference reborrows

An existing `ref mut T` can be passed to an internal `ref T` parameter or
shared method receiver. The example reads and then edits a scalar, retains
a returned reader until its last use, carries it through Option, reads an
exclusive byte view, and calls a shared struct method through a mutable receiver.

The parent becomes mutable again after the shared result's last use.
Writing through the parent or original owner before that point is rejected.
The conversion is available at calls; direct bindings and returns keep exact
type checks. See [ADR 0414](../../docs/adr/0414-call-shared-reborrows.md).

Repository validation uses `fairyc_reborrow_check` for semantic regressions,
verified artifact round trips, malformed MIR and interpreter execution, and
`fairyc_reborrow_native` for Windows MSVC low/optimized execution. Native
evidence and executable hashes are recorded in `.cache/shared-reborrow-native/`
and summarized in the committed [evidence](../../docs/ops/evidence/shared-reborrow-20261001.json).
