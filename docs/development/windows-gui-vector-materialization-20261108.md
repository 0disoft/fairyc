# Vector push materialization for the GUI driver

The GUI companion graph initially required 1,050,570 materialized AST nodes,
above the unchanged 1,048,576 cap. `std::vec::push<T>` now invokes its existing
`storage_put<T>` intrinsic directly. The removed private `write_slot<T>` did
nothing except that same move and an unconditional `return 1`; its caller
asserted that constant result. Allocation failure still returns the element,
and the null check, move, commit and returned length keep their original order.
No public signature, intrinsic, runtime C, budget or accounting changes.

`go test -mod=readonly ./internal/sdkstdlib -run
'^TestVectorSharedChecksPreserveConcreteElements$' -count=1` passed for scalar,
plain struct and owning resource elements through source/Flow/verified MIR.
Native companion and GUI execution are recorded separately in the GUI handoff.
