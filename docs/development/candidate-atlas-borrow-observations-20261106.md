# Complete-owned borrow observations, 2026-11-06

Continues the [checked-body collector](candidate-atlas-borrow-sites-20261106.md).
This retains semantic metadata through compilation and restoration; it does not
publish borrow relations or promote Atlas completeness. Native Linux stays batched.

## Ownership and current-source restoration

`module_borrow_observations` retains flat source-site rows and per-frame ranges.
An absent table differs from observed empty frames. Local builders publish a whole
immutable table after every frame is seen. Aliases must agree in count, order,
spans, mutability and escape flags, including empty/nonempty contradictions.

`module_identified_ir` retains the original backend owner as a whole `Core` and
attaches observations derived from successful checked bodies. Project linking
follows the exact linker frame map and checks every contributing alias. The same
table is available from the checked program retained by `compiler::Complete`.
No owning resource field is partially moved or mutably borrowed.

Unit and whole-project artifacts do not serialize trusted borrow observations.
Their existing current-source body/Flow, recipe, identity, state and SDK admission
reconstructs observations before Complete publication. Restoration copies a fresh
table into the new owner; it never moves from or reuses facts in a prior owner.
The artifact wire format and executable IR remain unchanged.

`module_artifact_borrow_observations` measures bounded observation work separately
and folds it into the existing 1,073,741,824 artifact work history, on success and
failure. A prior artifact counter above 16,777,216 must not be interpreted as an
already exhausted observation counter. Fold failure explicitly consumes the
operation's Result before rejecting. Product work limits remain unchanged.

Project tables join separately bounded source units. The private collector's
65,536 batch limit must not cap the complete project's cumulative observations.
The project work ledger and each checked source unit's existing limits still apply.

## Windows acceptance

`TestCandidateAtlasBorrowObservations` compares cold and restored Complete facts
with independent original Go inventories for twenty-seven source families plus
one cross-package alias family, at both Windows profiles. It compares ordinary,
generic and concrete cleanup owners, exact IDs, spans, mutability, escape flags,
empty frames, canonical order and multiplicity. Each unit is also encoded,
restored, compared and re-encoded with byte-identical executable IR. The alias
fixture requires pre-link frame counts to exceed linked frame counts. Native
runtime PATH is empty and source workspace bytes must remain unchanged.

Controls cover one-short/exact artifact work, a preexisting 16,777,217 history,
absent/incomplete tables, repeated identical aliases and conflicting aliases.
`TestCandidateAtlasBorrowObservationCapacity` copies 65,535, 65,536 and 65,537
observations across frames of at most 16,384 sites and compares every retained
field. Before removing the accidental cumulative cap, the admitted native low
probe rejects 65,537 with code 3; the original source checker accepts the harness.

The original combined Complete/unit-cache command passed in 462.481s before
the capacity correction: Complete observations 107.97s; existing generic aggregate
unit-cache low/optimized 207.23s/135.22s. Its unchanged cache-path result is retained
as preceding regression evidence, not a fresh final-source cache replay.

The final observations/capacity command passes in 113.115s with both Windows
profiles and no selected skips: 56 source processes plus two capacity processes.
The capacity root passes in 9.66s, including all three counts and exact site fields.
Authored check and locked check pass 684 files / 8,541 MIR functions. Scoped CLI
vet, complete atlasmap/contextpack/semanticsnapshot suites, Go formatting,
UTF-8/LF/final-newline checks and `git diff --check` pass.

Commands and logs:

```powershell
go test -mod=readonly -count=1 -timeout 8m -failfast -v ./internal/cli -run '^TestCandidateAtlasBorrowObservation(s|Capacity)$'
go vet -mod=readonly ./internal/cli
go test -mod=readonly -count=1 ./internal/atlasmap ./internal/contextpack ./internal/semanticsnapshot
# From examples/lexer_candidate:
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

The `.cache/atlas-borrow-observations-final-20261106.log` receipt retains the final
native run. `atlas-borrow-observation-capacity-repro-final-20261106.log` retains
the reproduced rejection; `atlas-borrow-complete-{source,locked}-final-20261106.log`
retain the final authored checks. The preceding combined command is retained in
`atlas-borrow-observations-and-cache-final-20261106.log`.

## Failed attempts and remaining work

Earlier source attempts rejected missing semicolons, mutable owning-field borrows
and an unconsumed Result on work-fold failure. Whole-owner composition, local
vectors and explicit Result disposal correct these without language changes.
The first alias fixture did not merge frames; the current admitted resource
factory asserts actual cleanup aliases. The new capacity harness initially used
the wrong Vec allocation-error propagation and an oversized C-partition threshold;
both are harness-only failures, retained separately from the native reproduction.

Final borrow map output, relation counts/completeness and actual command acceptance
remain required, together with all broader implementation and final acceptance
gates in the original queue. No public CLI/API, SDK/runtime ABI, DB, dependency,
repository hygiene or runner policy changes. No push, dispatch or release.

The subsequent [Complete-bound production receipt](candidate-atlas-borrow-production-20261106.md)
records map/count/completeness and selected actual command acceptance separately.
It supersedes that bounded production gap, not the broader implementation or
complete final-acceptance queue retained above.
