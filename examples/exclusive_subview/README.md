# Exclusive byte-view subviews

Slicing a `ref mut u8[]` retains exclusive access to the selected bytes.
The example writes through a nested subview, passes it to an editor and a
shared reader, returns a subview, creates an empty range at the end, and
checks that explicit bounds run once from left to right.

The parent, its aliases and the original owner remain frozen until the
child's last use. Two retained siblings are rejected even when their
numeric ranges do not overlap. See [ADR 0415](../../docs/adr/0415-exclusive-byte-subviews.md).

Repository validation uses `fairyc_subview_check` for semantic, MIR,
artifact and interpreter coverage, and `fairyc_subview_native` for
Windows MSVC low/optimized execution. The native suite also checks reversed
and out-of-range explicit and omitted bounds. Executable hashes and results
are recorded in `.cache/exclusive-subview-native/` and summarized in the
committed [evidence](../../docs/ops/evidence/exclusive-subview-20261001.json).

Windows C transport keeps byte-view index writes observable with a volatile
byte access to avoid the optimized store-loss observed with MSVC 14.51.
This is an internal code-generation workaround; it adds no atomicity,
thread-safety or public volatile semantics. Its performance cost has not
been measured.
