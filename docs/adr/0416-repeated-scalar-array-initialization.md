# ADR 0416: Repeated scalar fixed-array initialization

- Status: implemented
- Date: 2026-10-01
- Owner: [Surface syntax](../language/0005-surface-syntax-v0.md)

Admit the already parsed `[initializer; count]` form for the existing inert
copy-scalar array element set: `bool`, `u8`, `i32`, `i64`, `u32` and `usize`.
The expected `T[N]` supplies the initializer's element type; otherwise the
initializer selects the type. The actual count determines the array length.
Annotation length or element mismatches retain E1006 and literal range errors
retain E1013. Resource owners, borrows, records and nested arrays remain outside
this admission, including count one: repetition must never duplicate ownership.

The count is a positive integer literal or a resolved, evaluated global integer
constant, including a qualified constant path. It is resolved at compile time,
never evaluated at runtime. Local variable counts are unsupported. Decimal,
hexadecimal, binary and underscore-separated literals retain existing parsing.
The count must be in `1..=4096`, the existing provisional fixed-array expansion
guard. Zero and greater counts are rejected before executable MIR construction.
Empty arrays and zero-size native storage remain unsupported; a zero repetition
does not supply an exception to that boundary and has no executable initializer.

Evaluate the initializer exactly once, then copy that scalar value into every
element. Effects, calls and traps occur once, not once per element. Each element
is independently stored; editing one does not alias the others. A trapping
initializer does not complete construction. This is deliberately distinct from
list initialization, which evaluates each written element left to right.

HIR retains one initializer with operator `repeat`. Lowering materializes that
initializer once and expands its operand into the existing verified MIR
`array-construct` list, bounded by 4096. Existing interpreter, artifact and C
transport array representations remain unchanged. No new MIR opcode, runtime
operation, public SDK API or grammar spelling is introduced. Language `/20`,
MIR v1, SDK stage0.73 and runtime ABI `/0` remain unchanged.

The [example](../../examples/repeated_array/README.md) covers the scalar set,
one-time side effects, computed global count, inference, internal by-value calls,
the maximum buffer length and independent element edits. Related semantic and
artifact/interpreter checks and Windows low/optimized execution validate normal,
initializer-trap and bounds-trap behavior. Native results and executable hashes
are recorded in the [evidence](../ops/evidence/repeated-array-20261001.json).
