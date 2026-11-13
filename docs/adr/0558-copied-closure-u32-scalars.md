# ADR 0558: Copied closure u32 signatures and captures

- Status: Accepted for bounded prototype
- Date: 2026-11-14

Direct nonescaping copied closures admit named i32/u32 result, capture,
parameter and explicitly typed immutable local types. Existing bounds remain:
four captures, four parameters, sixteen immutable locals and sixty-four pure
arithmetic expression nodes. Ordinary typing rejects mixed arithmetic or an
incorrect actual capture type; there are no implicit scalar conversions.

Capture values are read once when the closure binding is initialized. Later
changes to the original variable do not affect the snapshot. Actual arguments
execute once in source order. Lifting uses private ordinary functions with
typed hidden capture arguments; it adds no callable descriptor, heap allocation,
foreign ABI, artifact tag, new scalar operation or language version.

First-class closure values retain the exact existing `fn i32(i32)` signature,
i32 captures and i32 locals. Their descriptor/invocation ABI is not widened by
the direct-call permission. Closure-valued, resource, borrowed and pointer
captures, escaping direct closures, mutation inside the literal and other
control-flow/body families remain separately closed.
