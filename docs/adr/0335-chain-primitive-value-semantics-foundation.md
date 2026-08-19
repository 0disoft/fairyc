# ADR 0335: Chain-primitive value-semantics foundation

- Status: Accepted implementation foundation
- Date: 2026-08-19
- Product source: `docs/product/02-spec.md`
- Admission sources: `docs/adr/0218-modern-language-concept-admission-boundary.md`, `docs/adr/0254-defer-nominal-scalars-pending-user-domain-evidence.md`, `docs/adr/0328-production-workload-capability-admission-gates.md`
- Implementation: `internal/chainprimitives/`

## Context

Future contract targets need arithmetic and cryptographic values that ordinary Stage 0 applications do
not yet require. The practical set is unsigned 64-, 128-, and 256-bit arithmetic, opaque addresses and
hashes, algorithm-tagged signatures, eighteen-decimal token amounts, and deterministic target-owned
costs for cryptographic operations.

Adding only source spellings would be unsafe. The current language deliberately treats integer
semantics, MIR operations, traps, C construction, target layout, ABI identity, interface hashing, and
fixtures as one vertical contract. A premature `u256` keyword would either leave those layers
inconsistent or force one chain's address width, signature encoding, and gas schedule into the general
language.

The implementation also needs three visibly different overflow policies. Ordinary arithmetic remains
checked. Wrapping and saturation are useful for cryptographic and accounting algorithms, but making
either implicit would silently weaken the existing arithmetic contract.

## Decision

1. Add `internal/chainprimitives` as a compiler-internal, host-independent semantic oracle. This
   package is implemented and tested, but it does not add Fairy C source-visible types or intrinsics.
2. Unsigned arithmetic has three separate APIs:
   - `Checked` operations return `ErrOverflow` or `ErrUnderflow`;
   - `Wrapping` operations reduce modulo the exact bit width;
   - `Saturating` operations clamp to zero or the exact maximum.
   Ordinary future Fairy C operators map only to the checked APIs. Wrapping and saturating behavior
   require explicit named calls.
3. `Uint128` uses two little-endian `uint64` limbs and `Uint256` uses four. Their external byte
   representation is exactly big-endian 16 or 32 bytes. Decimal and hexadecimal parsing reject signs,
   malformed input, and overflow. No host `int`, floating point, C extension integer, or architecture
   width participates in their meaning.
4. `Uint256` provides checked, wrapping, and saturating addition, subtraction, and multiplication;
   exact division and remainder; checked shifts that distinguish invalid amounts from result overflow;
   bitwise operations; fixed-width byte conversion; and canonical decimal formatting. Multiplication computes the complete eight-limb product before
   classifying overflow. Division uses bounded 256-step binary long division. Shift amounts outside
   the exact width return `ErrInvalidShift`; representability failures return `ErrOverflow`.
5. `Address` is opaque and initially admits exact 20- and 32-byte values. Keeping the width in the
   value avoids choosing EVM-style 20 bytes or 32-byte account identifiers as a language-wide ABI.
   A contract target must later select one admitted width in its target specification.
6. `Hash256` is exactly 32 opaque bytes and carries no algorithm identity. `Signature` carries an
   explicit algorithm identity and exact byte count. The first admitted encodings are compact and
   recoverable secp256k1, Ed25519, and BLS12-381 signatures. Construction validates length but does not
   claim cryptographic validity.
7. `Decimal18` is an unsigned fixed-point amount backed by `Uint256` atoms. Parsing accepts at most
   eighteen fractional digits, formatting is canonical, and addition, subtraction, integer
   multiplication, and integer division are explicit. General decimal-by-decimal multiplication is
   withheld until a 512-bit intermediate and rounding contract are accepted.
8. Cryptographic operations use closed `IntrinsicID` values. Zero-cost rules are rejected. Each target
   supplies a versioned `CostSchedule` whose rules compute `Base + PerChunk * ceil(inputBytes / ChunkBytes)` under checked
   `uint64` arithmetic and an explicit input ceiling. Construction rejects unknown IDs, duplicates,
   zero revisions, empty targets, zero chunk widths, and zero input ceilings.
9. A schedule has a canonical SHA-256 identity over the target, revision, sorted rules, and every
   numeric field. Target descriptions, cache keys, deployment receipts, and future contract artifacts
   can bind to that digest so a cost change cannot drift silently.
10. The only executable reference intrinsic in this change is SHA-256 because Go's standard library
    supplies it without a new dependency. Every other intrinsic fails closed. A target adapter must
    provide implementation and conformance evidence before enabling another algorithm.

## Source-language admission gate

No spelling is reserved by this ADR. Exposing any of `u64`, `u128`, `u256`, `Address`, `Hash256`,
`Signature`, or `Decimal18` requires a later accepted ADR and one complete implementation slice that
names all of the following evidence:

- exact literal, cast, comparison, arithmetic, and diagnostic rules;
- HIR identity and public-interface encoding;
- verified MIR operations, traps, and artifact-codec versioning;
- interpreter agreement with this package;
- target layout and ABI classification;
- constrained-C, WASM, RISC-V, or another selected backend lowering;
- positive, negative, overflow, boundary, differential, and native fixtures;
- target cost-schedule identity in cache and deployment artifacts.

The first source-visible proposal should expose `u64` as the smallest complete vertical slice. `u128`
and `u256` follow only after the selected backend proves exact arithmetic and layout. Opaque chain
values should be nominal types rather than aliases for byte arrays.

## Rejected alternatives

### Make every value a user-defined struct

The current language has no generic operator or nominal-scalar contract. Duplicating wide arithmetic
in source would expose limb representation, inflate generated code, and bypass checked operator
semantics.

### Use `math/big` in production semantics

`math/big` is useful as a test oracle but has allocation, aliasing, and representation behavior that
should not define generated contract execution. The implementation therefore uses fixed limbs and
bounded loops. Tests compare it against `math/big` over deterministic randomized inputs.

### Fix `Address` at 20 bytes

That would encode one ecosystem's choice into the general language before a contract target exists.
The internal value admits 20 and 32 bytes; each target must choose exactly one source-level width.

### Publish default gas numbers

Unmeasured numbers would look authoritative and immediately become compatibility baggage. This ADR
implements the formula, validation, and digest but deliberately supplies no production schedule.

## Evidence

`internal/chainprimitives` includes deterministic tests for:

- all three `u64` overflow policies;
- `Uint128` boundary behavior;
- 256 deterministic randomized `Uint256` pairs compared with `math/big` for checked and wrapping
  addition, subtraction, multiplication, quotient, and remainder;
- every checked shift amount and exact decimal and byte round trips;
- address widths, defensive copies, hash encoding, and signature lengths;
- decimal parsing, canonical formatting, and integer arithmetic;
- schedule-order-independent digests, chunk charging, input ceilings, duplicate rejection, and
  fail-closed reference intrinsic execution.

## Consequences

The project gains executable wide-value and intrinsic-cost semantics without pretending blockchain
syntax or a deployable contract backend exists. Later frontend and backend work can share one tested
oracle instead of independently reimplementing arithmetic at every layer.

The immediate cost is an internal package not yet reachable from Fairy C source. That is deliberate:
source availability remains closed until the complete vertical evidence exists.
