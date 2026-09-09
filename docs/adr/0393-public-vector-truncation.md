# ADR 0393: Public vector truncation

- Status: Accepted
- Date: 2026-09-09
- Depends on: ADRs 0388–0392; issue #44

## Decision

SDK `0.0.0-stage0.59` adds
`std::vec::truncate<T>(ref mut Vec<T> value, usize new_length) -> usize`
in the existing Windows vector surface. The notation here describes the API;
the SDK source uses the language's return-type-first declaration syntax.

The call removes elements from the last initialized index down to `new_length`
and returns the number removed. Requests at or above the current length return
zero without growing, allocating, touching element storage or changing capacity.
Empty vectors and repeated truncation at the current length are no-ops.
`clear<T>` delegates to truncation at zero and retains its existing removed-count
contract. Surviving elements retain their values and order; later pushes reuse
the retained capacity.

Each removed slot is popped from the initialized prefix, transferred into a
temporary owner and destroyed exactly once through existing cleanup. No runtime
symbol, descriptor layout, C transport, language revision or runtime ABI changes.
The same shared-element borrow exclusion applies even to a numerically no-op
truncation, because the signature requires exclusive access to the vector.

## Compatibility and evidence

This is an additive source API. Refresh SDK lockfiles for the new version and
source digest. It does not admit mutable element views, nested vectors, user
generic aggregates, new allocators, non-Windows execution or Atlas vector queries.

`TestWindowsVectorTruncate` exercises ordinary CLI low/optimized execution for
empty, equal, larger, smaller, repeated and zero lengths, capacity retention,
surviving values and a subsequent push. It requires `E2101` for a live shared
element reference crossing truncation. The quickstart uses zero-length truncation.

`TestPrivateVectorNestedResourceTokenOwnership` truncates an owned-Buffer token
vector, observes reverse destruction and surviving payloads, then clears and
reuses it. Its independent probe still requires two allocation-failure recoveries,
the exact cleanup sequence, balanced allocations/frees and zero live allocations
after all owners leave scope. Truncation itself reuses pop/take and does not
allocate; the failure injections exercise push recovery, not a fictitious
allocation-failing truncate operation.

Full lexer correctness, self-hosting and hosted target conformance remain separate.
