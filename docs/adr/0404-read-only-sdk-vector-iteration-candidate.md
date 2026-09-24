# ADR 0404: Candidate for read-only SDK vector iteration

- Status: Candidate; compiler-private source and Flow proof, no stable source admission
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
activate them. Stable workspace semantic analysis still rejects the vector
source form before lowering, so it emits no C for that form. Exact type and
borrow checking, Flow and verified-MIR execution remain future gates for stable
admission. A separate compiler-private proof lane now checks the exact
authenticated, specialized `std::vec::Vec` resource identity, the written
shared item and eligible view shape, then runs ordinary type checking and Flow
over the prepared loop. Its first empty-vector and named-borrow cases reach
verified MIR; wrong element type, mutable item, in-loop owner replacement and
returning an item borrowed from a local vector are rejected. Stable workspace
commands still use the closed source gate.

The private Windows AMD64 MSVC probe now runs both the candidate and explicit
index form under low and optimized profiles. Empty vectors execute no body;
four initialized scalar elements produce the same ordered `continue`/`break`
result. The C probe observes no live vector allocation after function return.
This proves these control and cleanup cases, not the representative lexer
workload or the cumulative C-relative cost gate.

A second private native probe now inserts a shared `Vec<bytes::Buffer>` loop
into the existing nine-buffer ownership workload. The candidate and explicit
index forms both sum the nine five-byte views to 45, then complete the same
pop, move and resource-cleanup checks under low and optimized Windows AMD64
MSVC profiles. This covers resource elements and borrowed byte views without
admitting the syntax in stable workspace commands. It still does not compare
the actual lexer output or generated C size. The cumulative 8% C-relative
budget also remains unmeasured.

The real lexer source comparison loads the checked-in scanner package graph
with the authenticated private SDK. It tests the original explicit-index
source, each of the three loop rewrites separately, and all three together.
All five forms pass source analysis, Flow and verified MIR; the stable source
lane rejects every rewritten form. Native runs cover empty input and a
cooked-number plus unterminated-comment input. Exact console output matches
under low and optimized Windows AMD64 MSVC profiles. Storage counts match the
independent Go lexer observation, and the allocator probe finds no live SDK
allocation after the scanner calls and owned stream return. The C-relative
cost gate remains open.

A [one-run size comparison](../ops/evidence/vector-lexer-size-20260924.md)
found unchanged function, foreign-symbol and specialization counts and EXE
size for all three rewritten loops together. Generated C grew by 711 bytes;
optimized `unit.obj` grew by 24 bytes against the indexed Fairy source. This
does not establish check/build time, runtime or the independent-C budget.

An independently written checked C vector-read kernel and the indexed Fairy
SDK equivalent passed ten shared boundary executions and a
[five-pair optimized comparison](../ops/c-budget-vector-read-index-20260924.md).
The indexed Fairy kernel's paired runtime median was 1.38544 relative to C,
above the cumulative 1.08 limit, while its EXE ratio was 1.01824. The
candidate source form was not measured in that stable harness, so this result
cannot isolate the syntax cost. It keeps source admission open and points to
the existing vector access path as a performance investigation target.

The [private candidate C comparison](../ops/c-budget-vector-read-candidate-20260924.md)
now measures the proposed loop against the same independent checked C source.
Its five-pair runtime median is 1.37255 relative to C, still above 1.08;
EXE ratio is 1.01824. Ten boundary executions and the stable source rejection
preflight pass. The candidate and earlier indexed Fairy reports come from
separate runs, so their difference cannot yet be attributed to iteration
syntax. A directly paired index/candidate measurement remains open before
any source admission decision.

[Generated-access inspection](../ops/vector-read-access-inspection-20260924.md)
finds identical optimized disassembly for the indexed and candidate Fairy
units after excluding the dump-file path header. Both retain an inner-loop
call to the private vector slot lookup in `std::vec::at<T>`. That common access
path is the next optimization experiment; preserving safe bounds and the
owner-bound shared borrow is required. This evidence does not admit the source
form or explain the entire C-relative runtime gap.

[ADR 0405](0405-direct-address-shared-vector-access.md) now uses a checked
typed element address for shared `Vec.at<T>` in SDK `0.0.0-stage0.67` while
retaining the owner-bound borrow. A preliminary diagnostic measurement removes
the helper call and brings both indexed and candidate kernels close to, but
slightly above, the 1.08 paired runtime limit in that pre-commit diagnostic.
Stable source admission still requires the representative real-workload and
remaining cost evidence.

The [post-commit shared-access comparison](../ops/c-budget-vector-read-address-20260924.md)
observed both indexed and private-candidate four-element kernels below 1.08
on paired runtime medians and EXE size. Individual pairs cross the runtime
limit, and the representative lexer and unused-feature gates remain open.

A [prebuilt-image three-way comparison](../ops/vector-read-three-way-20260924.md)
rotates C, indexed Fairy and the private candidate through twelve measured
triads. Candidate/C is observed within 1.08 on paired and lane-median ratios,
and candidate/indexed is close to parity. Indexed/C is inconclusive because
its two median estimators straddle 1.08. This remains microkernel evidence;
the representative lexer, cleanup and unused-feature gates are open.

The [real-scanner execution comparison](../ops/vector-lexer-execution-20260924.md)
builds both optimized Fairy forms once and times the checked-in lexer's two
storage loops over 10,000 scans. Both EXEs are the same size, but the runtime
samples exceed the predeclared stability span; they support no speed verdict.
The token reporter runs outside the timed section, and the independent-C
full-scanner budget remains unmeasured.
A predeclared 25-times-longer window also exceeded the stability limit and
reversed the point-estimate ordering. Runtime cost remains inconclusive; more
repetitions under the same uncontrolled host are not an admission substitute.

The private rejection matrix also rejects moving a borrowed vector owner
(`E2105`), treating a scalar as the vector owner (`E1006`), and using a
computed vector expression as the iterable. The latter cannot acquire an
implicit temporary owner or hide evaluation in this source form.

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
