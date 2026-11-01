# ADR 0494: Broader direct scalar generic transport

- Status: Accepted for Go Stage 0 prototype
- Date: 2026-11-01
- Extends ADR 0385's executable signature subset.

Generic functions may now transport direct i8, i16, isize, u16, u64, f32 and f64
parameters/results, in addition to bool, u8, i32, i64, u32 and usize. The existing
explicit argument planner, monomorphization, bounded specialization graph and
ordinary semantic/Flow/MIR checks remain authoritative. No implicit numeric
conversion, new arithmetic, inference, aggregate admission or runtime boxing is added.
Composite signatures containing these seven transport scalars retain their previous
closed boundary. Existing source spans and name-resolution diagnostics are preserved.

The user-requested wider generics slice makes an identity/forwarding algorithm useful
for all thirteen existing scalar types without seven manually duplicated overloads.
The new native fixture transports seven boundary values through two generic functions
and independently compiled repr(C) C observer records, including unsigned maximums,
signed negative boundaries and exact floating values. It does not reinterpret ordinary
f32/f64 as direct extern-C scalar ABI types.

Private, explicit-file and package-aware compilation pass; generic memory/fresh-session
disk cold/warm and both emission profiles are checked. Windows MSVC 14.51.36231
C11/C17 x low/optimized runs return 42 with empty streams. Native Linux remains in the
maintainer's batched queue. SDK/candidate parity, language-version promotion, full-suite
and hosted CI evidence are not claimed; API flags, ABI, dependencies and runners do not change.

Reproduce with `go test -mod=readonly ./internal/compiler -run '^(TestGenericSignature|TestBroaderGenerics)' -count=1 -v`
and `go test -mod=readonly ./internal/workspacecompiler -run '^TestBroaderGenericsScalarCache$' -count=1 -v`.
