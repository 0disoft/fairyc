# ADR 0546: Copied closure immutable i32 locals

- Status: Accepted for bounded prototype
- Date: 2026-11-07
- Extends: [ADR 0496](0496-explicit-copied-local-closures.md)
- Related: [ADR 0509](0509-inline-immutable-local-bindings.md)

## Source contract

An existing eligible copied local closure may contain zero through sixteen
immutable `let` bindings followed by a pure i32 tail or a final explicit return.
Every local has an explicit i32 type or infers i32 from its initializer. Captures
and parameters keep their existing zero-through-four i32 limits. Local names
must be unique across captures, parameters and preceding locals, and cannot use
the existing `_fy_closure_` compiler-private prefix.

Initializers and the return share the existing sixty-four-expression-node
budget. Their vocabulary remains integer literals, capture/parameter/preceding
local names, parentheses, unary +/-, and binary +,-,*,/,%. Groups and local-name
references each count as nodes; initializers are not substituted into later
references. A local becomes visible after its initializer, so self and forward
references fail. Mutable or non-i32 locals, calls, assignments, discards,
intermediate expression statements, control flow, nested block expressions and
early returns remain closed. No grammar production or API is added; the existing
capture, parameter and expression-node limits are unchanged.

The AST retains literal spellings rather than token kinds. New initializer
admission distinguishes lexical integer literals without accumulating their
values: decimal, existing hexadecimal/binary spellings and separators retain
their parser-defined grammar. Ordinary contextual i32 checking still owns value
range errors, including integers larger than uint64. Inferred bool, float and
string locals are rejected even if unused. Zero-local return admission and its
existing typed diagnostics remain unchanged.

## Execution, ownership and identity

Captures are copied once when the outer closure binding is reached. Local
initializers execute in declaration order once per call, after ordered call
argument evaluation and before the return. Unused initializers still execute
and trap. Creating an unused closure does not execute its body initializers.
Private helper lifting must retain these statements and their source spans,
without modifying the caller's AST or erasing/reordering initializers.

The existing ordinary typed-HIR, ownership, checked arithmetic, MIR and native
paths govern the helper body. Existing source/generic definition identities
retain every written local initializer, including unused ones and their order;
an initializer edit must change exported generic template closure and concrete
specialization identities. No new callable/environment representation, runtime
dispatch, heap allocation, MIR opcode or ABI is introduced.

Closure values remain unavailable for copying, passing, returning, assignment
or storage. Capture mutation/inference, resources, borrows, raw pointers,
asynchronous retention and cross-thread delivery remain outside this slice.
This advances the wider closure queue without completing first-class closures.

## Implementation and validation ownership

Go admission is owned by `internal/closureprep/prepare.go`; ordinary helper
lifting already preserves the original body. Focused parser-backed boundary
tests live in `internal/closureprep/locals_test.go`, and compiler/interpreter,
Windows native and generic-identity regressions live in
`internal/compiler/copied_closure_locals_test.go`.

The parent owns Fairy admission/lifting, combined source/identity and native
acceptance, central queue/index updates, lock refresh and commits. Native Linux,
full CLI/self-build and broader acceptance are separate; this ADR makes no
claim that those checks ran. No dependency, DB, runner, packaging or publication
change is part of this slice.

## Go Windows evidence

With `GOMAXPROCS=1`, the focused command passes:

```text
go test -mod=readonly -count=1 -timeout 6m -v ./internal/closureprep ./internal/compiler -run '^(TestImmutableLocals|TestCopiedClosure)'
```

The compiler package passes in 8.745s, including the original copied-closure
execution, ordered-argument, escape, generic-detachment and view/vector for-in
regressions. The new value/snapshot/per-call program independently executes under
Windows MSVC 14.51.36231 in C11/C17, low/optimized: four native processes each
return 42 with empty streams. Interpreter controls verify no initializer execution
at creation, a later call trapping after an earlier successful call, and the
first division/overflow failure from unused initializers.

Parser-backed admission tests cover 16/17 locals, 64/65 shared nodes including
groups, original refusal spans, disallowed names/types/forms and unchanged
zero-local bodies. Eighty-digit integers pass preparation and then fail ordinary
contextual i32 typing. Equal-length unused initializer value/order edits change
both generic public interfaces and specialization IDs without mutating source ASTs.

Initial harness runs failed because a `module main` source used the filename
`locals.fy`, and a spelling fixture used unsupported uppercase `0B`. The final
tests use the actual `main.fy` module path and existing lowercase `0b` grammar;
neither failure required a product grammar or limit change. These failed runs
are not counted as passing evidence. Native trap provenance, Fairy integration,
full CLI and Linux execution are not established by these Go-focused checks.
