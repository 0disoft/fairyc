# ADR 0485: Explicit C enum array transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0478 enum identities and ADR 0484 bounded array transport

## Decision

Admit `1..=4096` element fields of payload-free, explicitly C-integer-based enums inside nonempty
acyclic `@repr(C)` records. All thirteen existing bases remain subject to target availability;
`c::ssize_t` is POSIX-only and `c::long` follows LP64/LLP64. An ordinary private enum is not a C
array element. Resolve representation metadata before checking aggregate fields so declaration
order does not affect eligibility.

Retain the nominal enum identity and array length through HIR, MIR and package/public-interface
transport. The C member is an actual array of the enum's exact integer typedef. Whole-field reads,
typed/inferred locals, internal array arguments/results, record reconstruction, nested reads/copies,
and direct whole-field replacement reuse the existing wrappers and element-copy loops. Different
enum types and their underlying integer arrays are not interchangeable.

Copying a C-produced array preserves every representable underlying integer, including values
outside the declared variants. It does not validate, clamp or assume the ordinals. Existing scalar
enum matching retains its trap rule. Array literals, element access/mutation, length operations,
views/iteration, field borrowing, nested field stores, bare foreign arrays, direct private/resource/
tagged payload arrays and new pointer forms remain closed in this transport slice.

Each compiler stage resolves the enum definition before admitting the array. The MIR verifier also
rejects attempts to introduce executable array element operations into a transport-only enum
array. No new runtime, opcode, dependency, cache format, SDK export or workflow is introduced.

## Evidence

Focused compiler tests cover all thirteen bases, later enum declarations, both target emission
profiles, unavailable Windows `ssize_t`, MIR round trips, malformed enum bases and source operation
boundaries. A forged MIR length operation is rejected independently of source checking. Public
interface tests distinguish enum name, base and array length.

The existing separate-object native probe now exercises twelve Windows enum bases with three
rotations each, using the C provider's integer extrema as well as ordinary values. Independent C
checks every element and caller-copy isolation through nested reads, internal calls, reconstruction
and direct/referenced replacement. Length-one records cover small aggregate crossings. All four
MSVC 14.51.36231 C11/C17 low/optimized executions pass with exit 42 and no output.

The POSIX probe additionally includes `ssize_t`, and its Linux-profile C emission passes on the
local Windows host. Native Linux execution remains pending; earlier hosted F2-6 results do not
attest this extension. Target maturity remains unchanged.
