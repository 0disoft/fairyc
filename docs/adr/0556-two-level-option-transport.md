# ADR 0556: Two-level Option value transport

- Status: Accepted for bounded prototype
- Date: 2026-11-14
- Extends: [ADR 0547](0547-flat-option-generic-transport.md)

Admit root `Option<Option<S>>` for the existing seven extended scalar success
leaves, including direct generic substitution of S. Construction, inert copies,
parameters/results and exhaustive nested matching distinguish outer none, inner
none and a present scalar. The permission stops after two Option levels.

No new scalar operations, Option propagation operator, composite generic
arguments, storage/deferred captures, references, pointers or foreign ABI are
introduced. Other wrapper combinations remain separately admitted. Existing
type/artifact vocabulary represents the new form without changing old IDs.

C declarations must precede by-value uses of their child types. The nested
transport and its flat child use the existing tagged dependency visitor rather
than the ordinary aggregate graph. Stable numeric IDs retain their sorting.
Native observers inspect active values only and independently cover all three
states, copies, extrema and floating signed zero, from source and restored MIR.
Candidate source/artifact and common CLI/cache checks remain separate gates.
