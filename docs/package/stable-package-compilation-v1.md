# Stable package compilation artifact v1

- Status: Experimental compiler-private contract
- Decision: `docs/adr/0352-stable-package-compilation-artifact.md`
- Implementation: `internal/packageartifact/stable_compilation.go`
- Schema: `fairyc.stable-package-compilation/v1`
- Version: `fairyc-stable-package-compilation/1`

## Contents

One artifact contains exactly:

- one stable package MIR v3 value with its package instance, coordinate, projected type context,
  stable function bodies, and exact imports;
- one contract-state package-body artifact v1 with the same coordinate, exact layout and body hash,
  stable function descriptors, canonical operations and values, and explicit exits.

The outer artifact does not rewrite either nested representation. Each nested validator remains the
authority for its own identity and semantics.

## Required invariants

- both nested artifacts validate completely before encoding or after decoding;
- the MIR coordinate and state-body package coordinate are byte-identical;
- the caller supplies a positive maximum encoded byte count;
- JSON has the exact schema/version, field set, field order, and canonical byte representation;
- build-local package/function numbers, source spans, and transaction capability tokens do not enter
  the state payload;
- local state package renumbering under the same coordinate preserves complete artifact bytes.

## Closed gates

Stable cache records still contain MIR v3 alone. This artifact has no cache adapter publication,
expected-body lookup, malformed-record quarantine, combined MIR/state reassembly, fresh-symbol
alignment, workspace all-hit execution, source syntax, backend, runtime, persistence, or target
admission. Those require later versioned slices rather than implicit use of this codec.
