# Candidate two-level Option transport — 2026-11-14

ADR 0556 is now connected to both candidate admission owners: executable type
walking and original-source generic signature diagnostics. The permission is
limited to a root `Option<Option<S>>` with one of the seven transport scalars.
Triple nesting, other wrappers and composite type arguments remain separate.

The shared private test bridge now emits the actual ordered scalar parameters
of each probe, instead of assuming a single parameter of the result type.
No public foreign ABI was added.

Executed checks:

- `TestNestedOptionTransport*`: Go source admission, unchanged AST, GNU/MSVC
  emission and source/restored native programs in both C standards/profiles.
  Original and copied values independently retain all three tags. Integer
  extrema and floating signed zero/nonzero payloads are observed in C.
- `TestCandidateNestedOptionPipeline`: actual Fairy source checking, body,
  program and project codecs, restore against source, partitioned C and native
  products from both producer and consumer profiles. All four products return
  42 with empty output. Each product makes 78 independent C payload comparisons;
  authored assertions additionally distinguish all three original/copied tags.
- `TestCandidateGenericTransportAdmission/generic_scalar_arguments`: existing
  materialization, rejection and rollback controls pass in both native profiles.
  Old newly-admitted rejection fixtures now use triple Option and i64 errors.

Final logs: `.cache/nested-option-final-20261114.log` (324.10 s candidate,
8.005 s Go) and `.cache/nested-option-boundary-20261114.log` (21.14 s).
No desktop/input interaction. Whole CLI/cache acceptance remains batched with
the next generic/inline family; this receipt does not claim that broader gate.
