# Values and Resources

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0003-stage-0-safety-boundary.md`

## Type properties

Every fully resolved type has compiler-known properties:

| Property | Meaning |
| --- | --- |
| `Copyable` | Reading a value may duplicate it without transferring ownership. |
| `NeedsDrop` | A live value owns a cleanup obligation. |
| `ContainsSafeBorrow` | The representation contains a safe `&T` or `&mut T`. |
| `BorrowOrigin` | The hidden origin that bounds a safe borrow, if one exists. |

These are semantic properties, not user-implementable traits in Stage 0.

## Value categories

| Category | Copyable | Needs drop | Stage 0 aggregate rule |
| --- | --- | --- | --- |
| Scalar and raw pointer | Yes | No | May appear in inert values. |
| Inert `struct` | Structural | No | Cannot contain resources or safe borrows. |
| `resource` | No | Yes | Safe borrow fields and partial field moves are forbidden. |
| `Option<T>` | Follows `T` | Follows `T` | Active payload owns the obligation. |
| `Result<T, E>` | Follows both payloads | Follows active payload | Exactly one variant is active. |
| `Region` capability | No | Compiler reset | Cannot be stored, copied, moved out, or returned. |

## Initialization

Every source-level local declaration has an initializer. Stage 0 has no declaration that creates readable but uninitialized storage.

Compiler IR may create uninitialized temporaries while evaluating an aggregate. Those temporaries are not source-level values, cannot be observed, and are governed by drop elaboration. A failed aggregate expression drops only the fields that completed initialization.

`MaybeUninit<T>`, unions, arbitrary placement initialization, and user-observable partially initialized values are deferred.

## Binding and move syntax

`let` prevents reassignment but may still be consumed. `var` permits reassignment and is required when creating a mutable borrow. Ownership transfer has no source-level `move` operator: assignment to a by-value destination, a by-value argument, or a return consumes a resource, and diagnostics name that transfer site.

## Affine resource ownership

A resource local has one of three flow states at a program point:

- `Live`: usable and carrying one cleanup obligation;
- `Dead`: moved, consumed, or not yet installed, with no cleanup obligation;
- `MaybeLive`: different incoming paths disagree; the value is unusable until control flow is restructured.

Reading a copyable value copies it. Passing, returning, assigning, or otherwise reading a resource by value moves the complete resource root. Moving `r.field` when `r` is a resource is rejected even when the field itself is a resource.

At a control-flow join, equal states remain equal. `Live` joined with `Dead` becomes `MaybeLive`. A loop back edge must reproduce the ownership state expected at the loop header; Stage 0 does not synthesize a conditional ownership protocol for mismatched iterations.

## Construction and destruction

Construction is an ordinary function that returns a fully initialized value or a `Result`. `drop` is compiler-invoked, infallible, and runs exactly once for each live resource on every normal scope exit. Resource fields drop in reverse initialization order.

A trap, process abort, or forbidden foreign nonlocal control transfer is not a normal scope exit and does not run cleanup.

Safe code cannot forget or disarm a resource. A resource implementation may expose a narrow unsafe internal operation that clears its cleanup obligation after ownership has been transferred to an external system; that operation must be represented explicitly in MIR and carry source provenance.

## Fallible close

A fallible close operation must make ownership unambiguous.

- If the operation consumes responsibility on both success and failure, it may return `Result<void, E>`.
- If the caller must retain responsibility after failure, the failure payload must return the resource, for example `Result<void, CloseFailure<R, E>>` where `CloseFailure` contains both `R` and `E`.

An API that returns an error while leaving it unspecified whether the original resource is still live is invalid.

The initial Linux `File.close(self)` consumes ownership on every outcome and is not retried. This is a target-specific rule, not a portable POSIX promise. A general fallible-close protocol for composite resources is deferred because partial field moves are unavailable; composite resources rely on infallible reverse-order drop or keep fallibly closable leaves separate.

## Reassignment

For `old = make_resource()?`, the right-hand side is evaluated into a temporary before `old` changes. If evaluation fails, `old` remains live. If it succeeds, the compiler drops the old value, clears its obligation, then moves the new value into the local and marks it live. Reassignment of a `MaybeLive` resource is rejected.

## Required diagnostics

The compiler must distinguish use after move, resource partial move, branch-dependent ownership, loop-state mismatch, resource in an inert struct, fallible drop, and ambiguous close ownership. Diagnostic namespaces and provisional codes are defined in `docs/diagnostics/catalog-v0.md`.
