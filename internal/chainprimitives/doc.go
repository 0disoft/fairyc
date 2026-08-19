// Package chainprimitives owns the host-independent value semantics needed by
// future contract targets. It is compiler-internal and does not make u64, u128,
// u256, Address, Hash256, Signature, Decimal18, or crypto intrinsics available
// in Fairy C source by itself.
//
// The package deliberately separates three concerns:
//
//   - checked arithmetic remains the ordinary default;
//   - wrapping and saturating behavior require explicit names;
//   - cryptographic execution costs belong to a versioned target schedule.
//
// A later language ADR must wire these values through HIR, MIR verification,
// interpretation, backend lowering, ABI layout, diagnostics, and fixtures as a
// complete vertical slice. This package is the shared arithmetic and identity
// oracle for that work.
package chainprimitives
