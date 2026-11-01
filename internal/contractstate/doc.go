// Package contractstate provides the host-independent semantic kernel for a future
// Fairy C contract profile. It owns canonical typed storage layouts, append-only
// layout compatibility, snapshot transactions, bounded write overlays, and atomic
// commit or rollback behavior, and an explicit persistence compare-and-publish port.
//
// The package deliberately does not add Fairy C syntax, MIR operations, a chain
// target, consensus, or a public runtime API. MemoryStore is an
// executable oracle for the state behavior that later compiler and backend work
// must preserve. LocalPersistence is a bounded Windows/Linux filesystem prototype
// for restart-visible state, not a certified power-loss durable production engine.
package contractstate
