# ADR 0478: Explicit C enum fields

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)

## Decision

Use the existing attribute grammar to spell an explicit C integer enum:

```fairy
@repr(c::uint)
enum Status { Ready; Busy; Done; }
@repr(C)
struct Reply { Status status; u32 count; }
```

This attribute is valid only on a nonempty, payload-free enum. Its one argument must name a C
integer scalar; `c::bool`, `c::float`, `c::double` and Fairy integer types are excluded. Repeated
representation attributes are rejected. Discriminants are zero-based declaration ordinals, bounded
by the base's portable range for the supported targets. Explicit values, aliases, negative values
and enum arrays are deferred. Plain enums retain their existing private tagged representation.

The declared base travels through HIR, verified MIR, public interface identity and stable package
context encoding/rebinding. Changing it invalidates the corresponding interface/context identity.
Absent base fields retain existing encodings. No language revision, SDK version or target maturity
flag changes. MIR independently rejects invalid bases, overflowing variant counts and payloads.

The C transport uses the selected integer scalar, not the host's default C enum size. Record fields,
copies and foreign record arguments/results retain the nominal Fairy enum type; standalone foreign
enum arguments/results and enum raw pointers remain closed. Matching a foreign value outside the
declared variants traps as unreachable; reading/copying it does not assume a valid ordinal.

## Delivery

Commit `9b8ba65f` preserves and validates representation metadata. The following implementation
connects the existing target C-enum layout to integer typedefs, construction and matching. Direct
enum-field replacement is supported on writable owners with an exactly typed value. Nested enum
leaf replacement and enum-field borrowing remain closed; nested records may carry/read enum fields.

The shared native record probe includes independently declared C integer members at 1, 2, 4 and
8 bytes, small and large record argument/results, scalar neighbors, caller-copy isolation and direct
field replacement. The `c::long` field also checks the LP64/LLP64 distinction. Existing Linux and
Windows lanes consume this source without new jobs or report counters.

Local MSVC 14.51.36231 passes all four C11/C17 low/optimized combinations. Compiler tests check all
13 C integer bases in both target emission profiles, with Windows rejecting unavailable
`c::ssize_t`. MIR codec/interpreter, public interface invalidation and stable context rebind tests
pass. Hosted Linux execution remains pending; older array/record receipts do not attest enums.
The full Windows smoke and full repository Go suite were not repeated for this bounded addition.
