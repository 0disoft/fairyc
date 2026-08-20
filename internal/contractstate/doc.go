// Package contractstate provides the host-independent semantic kernel for a future
// Fairy C contract profile. It owns canonical typed storage layouts, append-only
// layout compatibility, snapshot transactions, bounded write overlays, and atomic
// in-memory commit or rollback behavior.
//
// The package deliberately does not add Fairy C syntax, MIR operations, a chain
// target, persistence, consensus, or a public runtime API. MemoryStore is an
// executable oracle for the state behavior that later compiler and backend work
// must preserve.
package contractstate
