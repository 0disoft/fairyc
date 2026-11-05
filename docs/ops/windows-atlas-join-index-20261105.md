# Windows Atlas checked-table and specialization indexes, 2026-11-05

## Scope

The authenticated SDK Vec source-production work exposed the existing shared
query-work limit during map joining. Its development probe used 15,717,260 units
before join and reached 16,777,105 before the next charge failed. The 16,777,216
unit limit remains unchanged. That Vec producer is a separate implementation
slice; the acceptance below exercises the existing reader/join/impact owners.

Partition admission now retains the symbol-to-declaration position already
proved while validating exact symbol identities. It merges the independently
validated, key-ordered declaration and ordinary-function tables once to prove
required-function coverage and retain their association. A query cannot supply
these indexes, and source/current identity checks remain mandatory.

Joining groups specialization records by original declaration once. Reverse
insertion preserves ascending specialization-ID order within each symbol;
package/symbol order and function-before-cleanup order are unchanged. Empty
specialization sets require no group allocation. Every new allocation and loop
retains failure and cumulative-work handling.

For D declarations, F ordinary functions, S symbols and G specializations,
required ordinary-function coverage changes from O(D log F) to O(D + F).
Ordinary-summary selection changes from O(S log F) to O(S). Specialization
selection changes from O(S * G) to O(S + G log D). Added storage is O(D + S + G),
bounded by the existing validated row limits. Binary target/implementation
lookup and all resolved-call checks remain in place.

## Windows verification

```powershell
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlas(GenericJoin|GenericValidation|Validation|Current|Impact)$'
```

The final command passes in 222.399s, with MSVC low/optimized execution for all
five roots and no selected skips. It compares exact ordered ordinary/generic
and cleanup edges with the Go oracle; current-source drift, missing/incorrect
implementation endpoints, malformed maps, duplicates, call/package limits and
impact traversal retain their existing assertions.

The initial merge helper used a resource-valued conditional return and failed
the ownership checker with E1104. An explicit error return before moving the
completed index fixes that development failure; the final command above uses
the corrected source.

Native Linux remains in the maintainer's batched queue. No public language/CLI,
SDK/runtime ABI, map wire schema, DB, dependency, default timeout, work limit,
repository hygiene or CI runner changes. No push, dispatch or publication.
