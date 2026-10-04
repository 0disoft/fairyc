# ADR 0466: Checked reference compound assignment

- Status: implemented; Windows native and public CLI verified; Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0216, 0400, 0401, 0442, 0445, 0452, 0463

## Decision

Admit statement-only compound assignment through a named mutable integer reference:
`*name +=`, `-=`, `*=`, `/=`, `%=`, `<<=`, `>>=`, `&=`, `^=`, and `|=`. The referent type is
exactly `i32`, `i64`, `u8`, `u32` or `usize`, and the operator must be defined for it by the
existing `hir.CompoundAssignmentType` predicate. The grouping form is around the named reference,
`*(name)`; a group around the whole dereference `(*name)` is not admitted because the prefix
dereference must itself be the assignment target.

```fairy
fn void adjust(ref mut i64 total) {
    *(total) += 5;
    *total >>= 1;
}

fn i32 main() {
    var i64 x = 10;
    var i64 y = 20;
    var ref mut i64 selected = ref mut x;
    *selected += { selected = ref mut y; 3 };
    assert(*selected == 20);
    assert(x == 13 && y == 20);
    return 0;
}
```

The reference must be a named `ref mut` path. Computed and call borrow roots, temporary
references, shared (`ref`) references, `bool`, struct, resource and raw pointer referents, and
aggregate or pointer compound operators remain closed. Direct scalar field compound stores stay
owned by ADRs 0442 and 0463; a field is reached here only through an admitted named reference.

## Evaluation and lowering

A compound reference assignment selects the original referent address once, captures the old
scalar before the right-hand side, evaluates the right-hand side exactly once, applies the
existing checked binary operation, and performs the existing `deref-store` only on success. The
compiler snapshots the reference descriptor into a private temporary before the right-hand side,
so a right-hand side that rebinds the mutable reference still stores to the originally selected
address with the originally captured value. The frozen old value is loaded from that address
instead of being re-read through the possibly rebound name.

Right-hand-side effects are retained even when a later checked operation or the final store
traps; this admission does not roll back right-hand-side effects. An early `return`, a Result `?`
propagation, or a trap in the captured value, the right-hand side, or the checked binary skips the
final compound update. The destination keeps whatever the right-hand side left there, which is the
prior scalar only when the right-hand side did not write the referent.

`internal/mir/deref_store.go`'s `lowerDerefStore` handles the compound operator: it lowers the
reference into a temporary, emits the existing unary `*` load into a second temporary, lowers the
right-hand side, combines both operands with the existing checked binary through
`lowerBinaryOperands`, then emits the unchanged `deref-store`. `internal/semantic/typecheck.go`
admits a `derefAssignmentTarget` in the compound place check while `hir.CompoundAssignmentType`
continues to decide operator and type admission.

## Borrows

The existing `deref-store` flow retains the exclusive loan across the whole statement. A
right-hand-side read or write through the same selected reference is allowed, but competing owner
reads, owner writes, owner moves or replacement and live sibling loans are rejected (`E2101`); a
shared reference receiver is rejected as `E1008`. This reuses the dereference-store rule instead
of adding a compound-specific restriction.

## Compatibility

No MIR operation, wire field, grammar production, SDK export, runtime symbol or target flag is
added. Language `/20`, SDK stage0.86, runtime `/0`, MIR/wire schemas, public ABI and target maturity
stay unchanged, and this is an additive admission. Exact compiler revision/executable identity
separates cached generations.

## Verification and limits

The related Go suite passes 149 tests with no failures or skips, and the Windows native harness
passes 16 Go test entries with no failures or skips: 14 cases run in both dev and release for 28
actual outcomes, plus a separate cache test. The cases cover all five referent types across all
ten operators, the grouped `*(item)`, a right-hand-side write to the referent (the frozen old
value plus two wins over the right-hand side's 100), a descriptor rebind whose right-hand side
selects another referent while the originally selected address receives the frozen value plus the
right-hand side and the other referent is left unchanged, an early `return` helper leaving the
original unchanged, a Result `?` `.err` leaving it unchanged, a struct field reference above `i32`
width, and a negative signed right shift. Further cases cover a scalar generic function
`add<T>(ref mut T item, T delta) { *item += delta; }`, `/=` and `%=`
by zero, `i64` MIN divided by -1, `i32` MIN `%-1`, an invalid shift, a left-shift overflow, a
right-hand-side `assert` trap, and addition overflow for all five types.

Compiler round-trip tests confirm retained-AST MIR byte identity, MIR codec/interpreter execution,
and both the Windows MSVC and Linux C emission profiles. A MIR mutation that drops the arithmetic
check and a shared-store mutation are rejected; the compound path verifies as one load plus one
store. `go vet`, formatting and diff checks pass. The standalone public CLI passes the matrix
`check`/`build`/`run` in dev and release for 2 outcomes and rejects five source forms: a shared
receiver (`E1008`), an owner write (`E2101`), a width mismatch (`E1006`), `bool` (unsupported), and
a temporary reference (`E1008`). It records raw UTF-8 argv, cwd, stdout, stderr and exit status
together with the compiler and EXE SHA-256 receipt at `.cache/deref-compound-cli/receipt.json`.
Native and public outcomes total 30 actual Windows executions. Linux actual execution remains
deferred to the owner's batch. Evidence is recorded as
`docs/ops/evidence/deref-compound-20261005.json` with `.cli.json` and `.native.json` companions.
