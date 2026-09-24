# ADR 0406: Null niche for shared scalar borrow options

- Status: Implemented private C transport
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

The constrained C backend represents `Option<ref bool>`, `Option<ref u8>`,
`Option<ref i32>`, `Option<ref u32>`, and `Option<ref usize>` with one pointer
field. `None` is a null pointer; `Some` stores the existing non-null safe borrow. The MIR and
interpreter retain the ordinary two-variant Option semantics. This changes no
source syntax, borrow origin, payload extraction rule, or cleanup obligation.

The whitelist excludes mutable borrows, byte and string views, raw pointers,
aggregate borrows, resources, and nested tagged values. A raw pointer can itself
be null; a view may use null data for an empty value. Neither representation
provides this spare value. Safe scalar borrows are non-null by contract; unsafe
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

Whole-program build time, generated C bytes, object size, executable size, and
the cumulative 8% C-relative budget are not established by this local layout
proof. Those require representative comparisons before claiming a product-wide
cost result.
