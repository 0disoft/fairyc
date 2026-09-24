# ADR 0406: Null niche for shared scalar, aggregate and resource borrow options

- Status: Implemented private C transport (shared scalars, structures, fixed arrays and resources)
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

The constrained C backend represents `Option<ref bool>`, `Option<ref u8>`,
`Option<ref i32>`, `Option<ref u32>`, and `Option<ref usize>` with one pointer
field. `None` is a null pointer; `Some` stores the existing non-null safe borrow. The MIR and
interpreter retain the ordinary two-variant Option semantics. This changes no
source syntax, borrow origin, payload extraction rule, or cleanup obligation.

The same private representation now also applies to `Option<ref S>` when `S`
is a resolved structure. Its shared borrow is a non-null pointer, and the
tagged payload was already supported. Structure contents and layouts are
unchanged; only the surrounding Option record loses its redundant tag.

`Option<ref R>` now uses the same representation for a resolved resource `R`.
The borrowed pointer remains tied to the resource owner; the resource itself
is neither moved nor dropped by constructing or switching the Option.
An exact fixed scalar array borrow `ref T[N]` also uses one pointer; this does
not apply to a length-bearing `ref u8[]` view. This transport change does not
admit direct indexing of a fixed-array borrow; the native proof explicitly
dereferences its payload before using the existing value-array index operation.

The whitelist excludes mutable borrows, byte and string views, raw pointers,
and nested tagged values. A raw pointer can itself
be null; a view may use null data for an empty value. Neither representation
provides this spare value. Included safe borrows are non-null by contract; unsafe
code that fabricates an invalid borrow does not create an additional safe value.

## Transport and compatibility

The emitted C type remains a private `fy_tagged_*` struct, now with only `p_0`.
The target layout engine and C `_Static_assert` agree on one pointer-sized,
pointer-aligned field. Construction and switching inspect that field rather
than a tag. Other Option and Result forms retain their existing tagged C
records. `@repr(C)` aggregates and foreign signatures do not acquire an
implicit nullable-pointer ABI. No runtime symbol, helper, type table, or
specialization is added.

The compiler commit identifies the changed private C transport. The language
revision, SDK version, public interface schema, and runtime ABI stay unchanged
because their source and external contracts do not change.

## Verification and limits

The backend test checks `Some` and `None` through the interpreter, asserts the
one-word generated C layout and null tests, then compiles and runs the result
under low and optimized Windows AMD64 MSVC profiles. The former tagged shape
contained a `u32` tag plus pointer, occupying two words on this target.
The `ref bool` case uses the same non-null borrow representation; its private C
transport was added after the initial four-integer whitelist without changing
the language or SDK pin.

The structure, resource and fixed-array cases check `Some` and `None` through the
interpreter, assert the one-word C record and null test, and execute both
Windows MSVC profiles.

Whole-program build time, generated C bytes, object size, executable size, and
the cumulative 8% C-relative budget are not established by this local layout
proof. Those require representative comparisons before claiming a product-wide
cost result.
