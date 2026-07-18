# Target-layout fixtures v0

`manifest-v0.json` is the versioned executable contract for the first host-independent layout
engine. `internal/layoutfixture` decodes it with Go `encoding/json`, rejects unknown fields and
trailing documents, binds the named target through `internal/target`, computes every case through
`internal/layout`, and compares exact size, natural alignment, top-level field offsets, padding
ranges, or one closed failure code.

The 14 cases cover language and C scalars, data and function pointers, explicit C-enum bases,
fixed arrays, internal and tail padding, nested aggregate placement, empty aggregates, duplicate
fields, zero-length arrays, unknown scalars, and `uint64` size overflow. Case IDs are unique and
strictly sorted so manifest order cannot become an accidental precedence rule.

This runner is host independent. It does not ask Go, Clang, GCC, libc, or the operating system for
layout facts. It also does not prove source syntax, semantic acceptance, SysV argument/return
classification, generated C, object metadata, or bidirectional C ABI behavior; those remain named
in the manifest's deferred ledger.
