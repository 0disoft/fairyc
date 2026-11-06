# Fairy Atlas borrow sites, 2026-11-06

Part of the complete implementation queue. This receipt distinguishes the original
Go semantic inventory from Fairy collection, production and actual command evidence.
Native Linux remains batched; no push, dispatch or release is authorized here.

## Original Go inventory

`internal/cli/atlas_borrow_sites_oracle_test.go` fixes seventeen independent source
cases against the original Go semantic facts after v3 map encode/decode. Expectations
name written source selectors, kind and escape status, never candidate output.

Retained local shared/mutable borrows, copies, static bytes/C strings, returned static
and parameter values, forwarded parameters, branches, fields, explicitly borrowed
array slices and retained call results are covered. Direct call-bounded borrows,
discarded borrows and sites after an unconditional trap produce no retained facts.
A returned parameter with no prior loan uses the returned expression span; an
existing loan keeps its original borrow site and becomes escaping.

The initial oracle incorrectly expected complete return-statement spans and used
two unsupported forms. Inspection of `internal/semantic/borrow_flow.go` confirmed
the returned-expression contract. Dereferenced explicit reborrows and unborrowed
array slicing now have two separate source-diagnostic rejection controls; their
admitted replacements retain the original planned coverage. No Go production
semantics were changed to fit the oracle.

Windows validation:

- `go test -mod=readonly -count=1 -timeout 3m -v ./internal/cli -run '^TestAtlasBorrowSites(Oracle|UnsupportedSource)$'`:
  all nineteen cases pass, 1.264s.
- `go test -mod=readonly -count=1 ./internal/atlasmap ./internal/contextpack ./internal/semanticsnapshot`:
  pass, 2.231s / 0.240s / 0.202s.
- `go vet -mod=readonly ./internal/cli`: pass.
- Log: `.cache/atlas-borrow-sites-go-oracle-20261106.log`.

Fairy collection, Complete-bound production, counts/completeness and actual commands
remain required. The full original implementation queue, final comparison/self-build,
distribution/clean-machine/hosted gates and batched native Linux work remain open.
