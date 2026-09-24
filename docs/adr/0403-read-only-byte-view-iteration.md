# ADR 0403: Read-only byte-view iteration

- Status: Implementation candidate; optimized microkernel budget met, workload gate open
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)
- Reopens only the byte-view part of [ADR 0224](0224-defer-unproven-surface-shorthands.md)

## Workload and decision

The Fairy-written lexer candidate now has three independent, non-fixture loops
that read every byte of an immutable `ref u8[]` in order and increment an index
solely to reach the next byte: [diagnostics text output](../../examples/lexer_candidate/src/diagnostics.fy),
[pages text output](../../examples/lexer_candidate/src/pages.fy), and
[scanner numeric output](../../examples/lexer_candidate/src/scanner.fy). All
three belong to one application; this is evidence for repeated local code, not
for demand across unrelated applications. A library cannot express their common
loop body without a callback or collection-wide operation, neither of which is
part of the current language. The old one-loop audit in ADR 0224 is historical.

Admit one source form, subject to the release evidence below:

```fy
for (let u8 byte in view) {
    number(byte as usize);
}
```

`view` must resolve to a named immutable local or parameter of exact type
`ref u8[]`. The byte binding is a fresh immutable `u8` in the loop-body scope.
There is no index binding, ownership transfer, callback, user iterator,
generic collection, `Vec<T>`, mutable slice, string, array, range value, or
expression-valued loop in this decision. An eligible path has no call or other
effect to evaluate repeatedly. A non-path expression, mutable view binding,
or wrong type is rejected before MIR.

## Execution and ownership

The view's length is observed once before the first iteration. The compiler
retains the view's borrow origin until loop exit. Iteration visits indices
`0..len` in order; each body entry copies one checked `u8` element into the
binding. An empty view executes no body. A safe mutation or move of the
borrowed owner while iterating remains an error. The hidden index and bound
cannot be named by source code.

`continue` advances the hidden index exactly once before the next condition;
`break` skips that advance. `return` and `?` leave the function through their
existing cleanup paths. `defer`, resource drops and Region resets on every
exit follow the same order as the equivalent classic scalar `for`. The final
index increment is representable even when `len == SIZE_MAX`: the largest
executed index is `SIZE_MAX - 1`. Checked element access remains in safe mode
unless a later optimization proves it redundant without changing trap behavior.

The lowering reuses existing scalar-loop control flow, cleanup and
verified MIR. No runtime iterator, allocation, vtable, metadata, public ABI or
specialized function is admitted. Source spans and the byte binding must remain
available to diagnostics and formatting; implementation cannot silently rewrite
the user's source file.

## Implementation and release gate

The grammar and lossless parser retain this form, and AST lowering preserves
its name, type, view and body. It prepares a hidden length snapshot and scalar
`for` path without rewriting the source. Semantic analysis admits only the exact
`u8` item and named immutable `ref u8[]` local or parameter. Public Flow rejects
owner mutation during iteration and permits it after the loop. The ordinary
compiler publishes verified MIR, and interpreter tests cover empty, ordered,
`continue` and `break` cases. Windows MSVC low and optimized native tests cover
empty iteration, `continue`/`break`, and nested loops with an early `return`.
The nested return also executes in the verified-MIR interpreter. The backend
omits unreachable MIR blocks when emitting C, avoiding unreferenced labels
after an early return.

An executable proof now checks `defer` before resource cleanup, exactly once
per body, on byte-loop `continue`, `break`, and early `return`. It compares the
verified-MIR interpreter trace with low and optimized Windows native execution.
The same interpreter and native observer now check `Result` `?` error
propagation from inside the loop, including exactly-once `defer` and resource
cleanup before the caller handles the error. Region cleanup exits remain open.
Negative fixtures cover wrong or mutable iterable types,
effectful expressions, binding escape and owner mutation while iterating.
The [paired C comparison](../ops/evidence/byte-view-c-budget-20260924.md)
measured the new form, the equivalent explicit-index Fairy form and independent
C in both MSVC profiles. Optimized runtime and EXE size are within the 8% limit
for this microkernel. The `/Od` runtime is about 21% above C, similar to the
existing explicit-index form; it is a diagnostic profile, not the optimized
C-relative budget gate established by the [baseline](../ops/c-performance-budget-baseline.md).
The [real lexer-candidate comparison](../ops/evidence/byte-view-real-workload-20260924.md)
confirms matching output and identical EXE size for the three adopted loops
against their former explicit-index spelling in both Windows profiles. Its
process roundtrip samples are too unstable for a runtime verdict and are not
C-relative. The full-application C budget and broader release evidence remain open.
Generated-C inspection points to retained safe checks under `/Od`; this is
not yet a measured attribution to an individual check.
The Fairy-written parser candidate now retains this form as statement kind 18
and validates its node references. Its expression arena bound grows from 4096
to 4160 nodes to parse its own updated source without changing the separate
type and statement bounds. The three workload loops now use byte-view
iteration; their native behavior and C-relative cost comparison remain release gates.
Compare the same algorithm in explicit-index Fairy, the new form, and
independent C at low and optimized profiles. Check generated functions,
runtime and EXE size, build/check time, and unused-syntax overhead. Apply the
cumulative 8% runtime and EXE limit to equivalent C under the optimized `/O2`
profile used by the existing budget policy; use `/Od` for native correctness and
cost diagnosis. Build 8% is an observation marker. A syntax-only or
interpreter-only change does not complete this decision.

The source cutover advances `fairyc-language/18` to `/19` and SDK
`0.0.0-stage0.65` to `0.0.0-stage0.66`, synchronizing current manifests,
scaffolds, tests and examples. Runtime ABI, MIR version, public-interface
schemas and target maturity do not change. ADR 0224 still defers checked narrowing
and expression-bodied functions; named loops and nominal scalars have separate
decisions.
