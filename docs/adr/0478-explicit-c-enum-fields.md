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

The first slice preserves and validates representation metadata while C emission explicitly rejects
the new representation. The next slice connects target layout, scalar typedefs, construction and
matching, then extends the existing shared native record probe. A layout fixture alone is not
source-to-C execution evidence. Linux execution remains pending until the expanded probe runs there.
