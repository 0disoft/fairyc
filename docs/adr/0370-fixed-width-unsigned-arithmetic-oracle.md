# ADR 0370: Fixed-width unsigned arithmetic oracle

- Status: Accepted implementation foundation
- Date: 2026-09-01
- Product source: `docs/product/02-spec.md`
- Admission sources: `docs/adr/0218-modern-language-concept-admission-boundary.md`, `docs/adr/0254-defer-nominal-scalars-pending-user-domain-evidence.md`, `docs/adr/0328-production-workload-capability-admission-gates.md`
- Implementation: `internal/chainprimitives/`

## Context

Future deterministic workloads may need unsigned 64-, 128-, and 256-bit arithmetic. Adding only
source spellings would be unsafe because language types, MIR operations, traps, interpretation,
backend lowering, layout, ABI identity, interface hashing, and fixtures form one vertical contract.

The project can still reduce later implementation risk by owning one allocation-free,
host-independent arithmetic oracle. That oracle must not silently choose a chain address width,
signature encoding, cryptographic roster, fixed-point scale, gas schedule, or backend.

## Decision

1. Add `internal/chainprimitives` as a compiler-private arithmetic oracle. It does not add Fairy C
   source-visible types, literals, casts, operators, intrinsics, or ABI promises.
2. Unsigned arithmetic exposes three distinct policies:
   - checked operations return `ErrOverflow` or `ErrUnderflow`;
   - wrapping operations reduce modulo the exact bit width;
   - saturating operations clamp to zero or the exact maximum.
   Any future ordinary Fairy C operator maps only to checked semantics. Wrapping and saturation need
   explicit source-level admission.
3. `Uint128` uses two little-endian `uint64` limbs and `Uint256` uses four. External byte encodings
   are exactly big-endian 16 or 32 bytes. Decimal and hexadecimal parsers reject signs, malformed
   input, and overflow. No host `int`, floating point, C extension integer, or architecture width
   participates in the result.
4. `Uint256` provides checked, wrapping, and saturating addition, subtraction, and multiplication;
   exact quotient and remainder; checked shifts; bitwise operations; fixed-width byte conversion;
   and canonical decimal formatting. Multiplication computes the complete eight-limb product before
   classifying overflow. Division uses a bounded 256-step binary long division.
5. Address, hash, signature, cryptographic intrinsic, fixed-point, gas, backend, and deployable-target
   semantics remain `UNDECIDED` and are not represented by this package.

## Source-language admission gate

No spelling is reserved by this ADR. Exposing `u64`, `u128`, or `u256` requires a later accepted ADR
and one complete slice covering:

- literal, cast, comparison, arithmetic, overflow, and diagnostic rules;
- HIR identity and public-interface encoding;
- verified MIR operations, traps, and artifact-codec versioning;
- interpreter agreement with this oracle;
- target layout and ABI classification;
- lowering through one selected backend;
- positive, negative, boundary, differential, and native fixtures.

The first proposal should admit only the smallest type justified by a real workload and backend.
This ADR does not assume that `u64` is necessarily that first public type.

## Rejected alternatives

### Admit chain-shaped nominal values now

Address width, signature format, cryptographic algorithms, fixed-point scale, and gas accounting are
target decisions. Freezing a speculative roster would create compatibility baggage without an
executable target or user-domain evidence.

### Use `math/big` in production semantics

`math/big` is useful as a test oracle but has allocation, aliasing, and representation behavior that
must not define generated execution. The implementation uses fixed limbs and bounded loops; tests
compare it with `math/big` over deterministic randomized inputs.

## Evidence

`internal/chainprimitives` includes deterministic tests for:

- all three `u64` overflow policies;
- `Uint128` boundary behavior;
- 256 deterministic randomized `Uint256` pairs compared with `math/big` for checked and wrapping
  addition, subtraction, multiplication, quotient, and remainder;
- every checked shift amount plus exact decimal and byte round trips.

## Consequences

Later compiler layers can share one tested wide-arithmetic reference instead of reimplementing the
semantics independently. The package remains unreachable from Fairy C source, so it is not evidence
for a public type, backend, ABI, runtime cost, or deployable contract target.
