// Package chainprimitives owns a host-independent fixed-width unsigned
// arithmetic oracle for possible future contract targets. It is
// compiler-internal and does not make u64, u128, or u256 available in Fairy C
// source by itself.
//
// The package deliberately separates three concerns:
//
//   - checked arithmetic remains the ordinary default;
//   - wrapping and saturating behavior require explicit names;
//   - parsing and byte conversion use exact target-independent widths.
//
// A later language ADR must wire these values through HIR, MIR verification,
// interpretation, backend lowering, ABI layout, diagnostics, and fixtures as a
// complete vertical slice. This package is only the arithmetic oracle for that
// work; target identities, nominal values, cryptography, fixed-point values,
// and execution costs remain unselected.
package chainprimitives
