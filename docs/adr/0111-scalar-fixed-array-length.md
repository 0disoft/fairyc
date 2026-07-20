# ADR 0111: Scalar fixed-array length

- Status: Accepted for the Stage 0 executable subset
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`

## Context

ADR 0110 admits bounded positive scalar fixed arrays, but leaves every property closed. Programs
therefore have to repeat `N` beside a value whose canonical type already retains it. Opening general
array fields, zero-length arrays, slices, or layout reflection would add unrelated representation and
borrowing decisions.

## Decision

1. Every admitted `[T; N]` value exposes the read-only property `.len` with result type `usize` and
   exact value `N`.
2. The receiver evaluates exactly once before the length result is produced. This remains true when
   the receiver is a list literal or an internal function call, even though `N` is known from its
   type.
3. Length observation performs no allocation, scan, memory load, runtime call, or trap. It does not
   depend on the private C wrapper's byte size or element representation.
4. Typed HIR and MIR use distinct `array-length` forms. The MIR verifier requires one live local
   receiver with an admitted fixed-array type, field `len`, and a `usize` destination.
5. The interpreter validates the runtime array shape and returns the retained canonical length. The
   C backend emits canonical `N` as `uintptr_t`; it does not use `sizeof` or inspect wrapper storage.
6. `.is_empty`, `.capacity`, general reflection, zero-length arrays, slicing, general
   array-to-slice conversion, mutation, and stable layout remain closed. ADR 0112 separately admits
   the contextual shared borrow from `[u8; N]` to `&[u8]`. Since the current subset requires
   `N >= 1`, adding `.is_empty` now would only freeze an unnecessary constant property before
   zero-length semantics exist.

## Evidence

- semantic tests distinguish `.len` from unsupported array fields;
- MIR tests preserve `array-length` and reject a mutated field marker;
- interpreter and constrained-C fixtures observe exact lengths for all five admitted element types;
- the Windows quickstart compiles and executes those observations through local x64 MSVC without
  WSL.
