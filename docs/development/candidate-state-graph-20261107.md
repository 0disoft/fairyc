# Fairy state graph implementation, 2026-11-07

Part of the full `implementation-queue-20261105.md` objective. This record starts
with the independent seven-codec layout prerequisite; it does not replace the
objective with a layout-only milestone. Native Linux remains batched. No push,
hosted dispatch, release or deployment is authorized by this record.

## Explicit layout and artifact profile

ADR 0508 adds explicit graph declarations/validation, `FYSTATE1` encode/decode
and current-source layout comparison. Ordinary five-codec declarations,
validation and `FYSTATE0` remain unchanged. No current caller automatically
selects this graph profile and no layout-only result creates Complete authority.

`TestCandidateStateGraphLayouts` obtains independent expected source admission
from the Go frontend/experimental semantic owner and canonical identities from
the Go layout builder. Its separate uint64 wire encoder never uses the Fairy
codec to create expected bytes. Actual Fairy-generated C runs under Windows
low/optimized with an empty PATH.

Coverage includes all seven codecs, binary/text map keys and values, empty
fields, UID lengths 254/255/256, field-name lengths 126/127/128, unsupported and
duplicate fields; source/artifact roundtrips; exact and one-short independent
and aggregate budgets; unmodified ordinary wire bytes/admission; current UID,
name, codec, kind, key and span drift; valid rehashed alternative layouts;
trailing comments; truncated/trailing input; marker, digest, prefix, integer
width, codec and name corruption; and refusal to read the other profile's marker.

The first boundary attempt exposed a real error-classification defect:
failed byte/hash work could leave the counter below its cap and be reported as
allocation error 11 instead of work error 10. Checked append and its callers now
propagate the original status. Primitive prefix, layout hash, copy and validation
each have exact, one-short and initially-overdrawn controls. The next attempt
exposed a harness error: current-source matching uses the aggregate program's
1 GiB ledger, not the artifact codec's 16 MiB ledger. The test now reads the
actual aggregate limit; no production limit or refusal was weakened.

## Windows acceptance

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestCandidateStateGraphLayouts$'
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^TestCandidateOriginalStateLayouts$'
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/cli -run '^(TestCandidateCompleteCacheKeysV5|TestCandidateCacheKeyV5Parity)$'
```

The new graph matrix passes both profiles in 15.487s. The four original
authored layout tests, including frozen five-codec hashes, package ownership,
diagnostic ordering and linked-IR rejection, pass unchanged assertions in
114.684s. The replay copies their actual source into an isolated dependency
closure; it does not replace their logic. The first isolated attempt omitted
the diagnostic module dependency and failed before execution; that dependency
is now included.

Ordinary and generic cache-key parity pass in 33.592s. The separately gated
Complete key replay is explicitly skipped because its integration opt-in is
not enabled; this is not a Complete graph-cache acceptance claim. Reference
`check` and `check --locked` pass 701 files / 8,702 MIR functions before the
parallel unrelated source expansions. The staged-source snapshot also passes
both checks at 701 / 8,702 and supplies this commit's refreshed lock, excluding
other workers' source. Scoped `go vet -mod=readonly ./internal/cli`, diff,
UTF-8/LF/final-newline checks pass.

An attempted native replay through the complete example's test selector fails
at unrelated `tests/module_unit_bind.fy` with E4503 before selected tests run.
It is recorded as failed, not a native pass. The isolated original-layout
replay above and independent graph/ordinary codec matrix provide this slice's
actual execution evidence. The original artifact filesystem suite was not
executed through that blocked all-example entry.

Logs are local `.cache/candidate-state-graph-layout-20261107.log`,
`candidate-state-graph-legacy-20261107.log` and
`candidate-state-graph-cache-regression-20261107.log`. These outputs are not
committed or treated as source authority.

## Remaining sequence

1. Add verified Fairy transaction graph IR and current-source admission,
   helper calls, root borrow/view/copy and typed branch/join behavior.
2. Bind exact state-body/source identity to the current Complete owner.
3. Connect graph artifacts to verified memory/disk cache reuse and rebind.
4. Select the admitted profile through ordinary compiler commands.
5. Complete independent Go/Fairy comparisons, consecutive self-build, batched
   Linux, distribution/clean-machine and same-commit hosted acceptance.

Other source families/deferred language/package/backend/provider designs in the
full queue remain required. This slice changes no stable public API, SDK/runtime
ABI, database/provider, dependency, default bound, runner or repository policy.
