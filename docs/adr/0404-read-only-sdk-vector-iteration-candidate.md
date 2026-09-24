# ADR 0404: Candidate for read-only SDK vector iteration

- Status: Candidate; inert AST preparation only, no source admission
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)
- Related boundaries: [byte-view iteration](0403-read-only-byte-view-iteration.md), [public vector](0390-public-windows-sdk-vector.md), [owner-bound storage borrows](0389-owner-bound-storage-borrows.md)

## Workload

The Fairy-written lexer candidate has three read-only `Vec<T>` loops that use
their index only to visit every initialized element in order:

- `scanner::report_work` visits `Vec<Token>` and holds `ref Token` for one body;
- `scanner::storage` visits `Vec<bytes::Buffer>` and `Vec<diagnostics::Entry>`
  to accumulate byte lengths.

These are three local uses in one application, not proof of demand across
applications. The diagnostic reporter keeps indices for stable ordering and is
not part of this candidate. The file-syntax parser also uses indices to treat
the last segment specially; changing it would hide required state.

Current code already expresses these loops with `v::len<T>` and `v::at<T>`.
The proposed benefit is removing repeated indexing without introducing a
callback, a collection copy or a runtime iterator. The following is an
illustration, **not accepted Fairy syntax**:

```fy
for (let ref Token token in ref tokens) {
    number(token.kind as usize);
}
```

The existing parser's `for (let type name in expression)` shape can retain this
source; semantic analysis currently admits only an immutable `ref u8[]` byte
view with a copied `u8` item. Parsing a shape is not language admission.

The first experimental implementation step prepares a separate AST block for
eligible shared-reference item loops. It finds the actual `std::vec` import
alias, captures a named borrowed view or explicit borrow of a path/field place
once, and sketches calls to that alias's `len<T>` and `at<T>`. Generic cloning
keeps the written loop and prepared block detached from the original source.
Only the authenticated SDK package graph now exposes those prepared calls to
bounded generic planning; an import spelling or unowned provider alone cannot
activate them. Semantic analysis still rejects the vector source form before
lowering, so there is no new accepted syntax or generated C. Exact type and
borrow checking, Flow and verified-MIR execution remain future gates.

## Proposed first boundary

The iterable would have to be an explicit shared borrow of an exact SDK-owned
`Vec<T>` place, or a named immutable `ref Vec<T>` local or parameter. The item
would have exact type `ref T` and be valid only within one loop-body entry.
`T` would follow the already admitted SDK vector element set, including named
resource elements by shared reference. No element move, mutable element view,
user-defined iterator, other generic collection, implicit borrow, or hidden
allocation would be admitted. An effectful iterable expression or a vector
from an unauthenticated package would be rejected.

The vector length would be observed once before entry. At each index the
implementation must apply the same initialized-prefix and bounds checks as
`v::at<T>`, return a shared owner-bound element reference, and keep the vector
borrow live through loop exit. An empty vector executes no body. A safe move,
resize, push, pop, clear or exclusive borrow of that vector in the body must
remain an error. The hidden index is not source-visible. `continue` advances
it once; `break` skips the advance. `return`, `?`, `defer`, drops and Region
resets must take the same paths and order as the explicit indexed loop. A
borrowed item must not escape a local owner.

This first boundary must not broaden SDK generic arguments or expose the
private storage layout. A backend implementation that adds a per-element
runtime helper, copies `T`, duplicates a specialization or weakens checked
access is not equivalent to the current source loop.

## Evidence required before acceptance

1. Compare the three real loops with explicit-index Fairy source and the
   proposed form. Preserve exact scanner output, storage counts, failure
   behavior and ownership. Keep the index-dependent loops unchanged.
2. Prove source and Flow rejection for wrong item/owner types, forged vectors,
   mutation and move during the loop, item escape, effectful iterable and
   unsupported targets. Check empty, ordered, `continue`, `break`, early
   `return` and `?`, with resource and Region cleanup in verified MIR and
   native execution.
3. Compare normalized executable MIR, generated C, specialization and helper
   counts, object and EXE sizes, check/build time and process runtime against
   the explicit-index variant. Compare an equivalent independent C workload
   under the optimized profile against the cumulative 8% runtime and EXE
   budget. A new source form cannot claim a zero-cost result from syntax or
   interpreter parity alone. Separately measure unused-feature overhead.
4. Decide the exact source eligibility, diagnostic, cross-package identity,
   formatter and release pin only after the proof. A failed gate keeps this
   feature a candidate; it does not authorize relaxing checks.

No grammar, product spec, language revision, SDK, runtime ABI, MIR, cache or
target contract changes in this preparation. Other collection types, mutable
iteration and user-defined iterators remain outside this candidate.
