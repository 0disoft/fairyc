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

## Checked-body observation retention and private collection

`module_unit_borrow_sites.fy` detaches the converged semantic facts after the same
type, move, origin, cleanup and Result gates succeed. `module_unit_checked_bodies`
keeps only source spans and mutable/escape flags, plus primary-frame ranges. Provider
frames used solely for origin inference are not copied as duplicate local owners.
The large origin input/provider/liveness graphs still leave scope normally.

Capture validates the primary context prefix and declaration containment. It keeps
the existing 16,384-fact bound and uses an independent bounded capture history.
Bottom-up merge sorting preserves semantic multiplicity with O(B log B) work and
O(B) temporary space per frame; equal spans are not unconditionally deduplicated.
Ordering matches Go: source start/end, kind (`mutable` before `shared`), then
non-escaping before escaping. Failed capture never publishes a checked prefix.

`project_atlas_borrow_sites::collect` and `collect_all` consume the successful checked
owner. They do not inspect syntax, repeat type/borrow analysis, accept a raw fact
list, invoke Go, or turn a directly discarded/call-bounded loan into a retained fact.
The APIs keep one-based frame selection, per-frame count refusal, a total batch
bound, and the caller-owned 16,777,216 work history. Errors are 9 invalid selection,
3 work/count capacity, and 2 allocation failure.

The source/native corpus adds mutable slices, array elements, retained/returned
subviews, tagged returns, loops, mixed kinds, conditional origins, two generic
function instantiations and two concrete resource cleanups to the original
seventeen families. It compares complete original Go source-owner inventories,
frame multiplicity, spans, kind and escape flags under both Windows profiles,
with empty PATH and workspace byte preservation. Each frame's canonical order
and standalone/batch equality are also checked. Twenty controls per profile
cover invalid frames, zero count for empty/nonempty input, and one-short/exact/
one-more work and count boundaries. This is not yet proof of exact generic IDs
after whole-project frame remapping or of produced `.fymap`/CLI borrow records.

The initial harness attempts failed before native execution: one omitted import
closure, one ambiguous numeric/Result inference, and one conditionally consumed
catalog resource. The final harness retains a complete import closure, explicit
numeric/Result types and branch-local factory ownership. No compiler rule was
weakened to accept the harness.

Final Windows validation:

- `go test -mod=readonly -count=1 -timeout 15m -failfast -v ./internal/cli -run '^TestCandidateAtlasBorrowSites$'`:
  pass, 77.221s; twenty-seven source cases plus twenty controls per profile,
  ninety-four native processes, no selected skips. The final replay also asserts
  canonical order before aggregating same-declaration specialization observations.
- Authored `check` / `check --locked`: pass, 681 source files / 8,506 MIR functions.
- Scoped `go vet -mod=readonly ./internal/cli`, Go formatting and diff hygiene: pass.
- Logs: `.cache/atlas-borrow-collector-windows-20261106.log`,
  `.cache/atlas-borrow-checked-source-20261106.log`,
  `.cache/atlas-borrow-checked-locked-20261106.log`.

Public CLI/API, SDK/runtime ABI, DB, dependencies, runner settings and existing
product timeout/warning/stack policies are unchanged. The checked-body observation
owner is private and is not added to the serialized IR or cache wire format.

Complete-bound ordinary/generic/cleanup publication, cache/restoration metadata,
counts/completeness and actual commands remain required. All broader and final
acceptance work remains open; native Linux stays batched.
