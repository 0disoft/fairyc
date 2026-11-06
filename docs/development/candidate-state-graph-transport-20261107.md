# Fairy Complete-bound graph transport preparation, 2026-11-07

ADR 0532 integrates three disjoint Sol high prerequisites (framing plan, pinned
frame and request encoder). The parent implements Complete-consuming transport
preparation, compiled-owner Input wrapping and full C/hash/request integration.
ADRs 0530/0531 are separately verified local prerequisite commits.

Both Windows profiles compare complete framed C and transport identity to live
Go GraphTransport.Source/ProgramHash, and complete request bytes/SHA256 to live
GraphTransport.Input. The admitted matrix includes all codecs, empty/binary/
Unicode values, literal/dynamic map operands, helper chains, branch/join and
borrow/view/copy. Maps-only layouts exercise the no-known-cell C variant.
Exact/one-short entry/framing work, overdrawn meters, invalid target/frame and
restored Complete with owner 511 are included. Input mutation cannot alter
Prepared's retained current-source owner or copied outputs.

Actual generated C executes empty commit/reject/trap, map operations, helpers,
present binary borrow/view/copy, a true branch and a nonempty version-one bool
snapshot. Eight ordinary native cases compare admitted response facts and
final reference-state bytes with the independent Go MIR interpreter. The
seeded branch additionally has authored fixed exit/cost/read/version facts.
Every built C image separately refuses a request with a foreign program hash
and emits no response bytes. Go response admission is explicitly a test driver
here: no Fairy result owner or storage publication is inferred from it.

The initial complete integration passes in **48.030s**, with **71** Fairy
processes and **eight** actual C images/profile before the additional seeded
case. Preparation failures are retained in
`.cache/graph-transport-attempt1.log` through `attempt3.log`: an import/accessor
name collision, direct fixed-array resource fields in the new request code and
a path-dependent resource move. These are fixed in the new files without
relaxing language rules, target/wire limits or diagnostics.

The broader selected run is retained in `.cache/graph-transport-final.log`.
Its frame asset, request and plan groups pass on the final unchanged product
inputs (request **70.08s**, plan **14.04s**). The overall command fails only
because the new seeded-branch assertion mistakenly expected cost 2. Actual C
and the existing accounting require cost **3**: read, branch terminator and
commit terminator. The expected value is corrected; no product code changes.
Those valid prerequisite results are reused, rather than re-running the large
request boundary matrix to fix an unrelated assertion.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateStateGraphTransport($|SealedOwner$)'
go vet -mod=readonly ./internal/cli
```

The corrected integration group passes in **48.65s**. **72** Fairy preparation/
request processes per profile / **144 total** compare exact Go bytes and
identities. **Nine** actual C images/profile / **18 total** each execute both
a valid Fairy request and a rejected foreign-identity request. Sixteen valid
executions match independent Go MIR/state outcomes; two seeded executions
match the separately authored version/read/branch/cost expectations. The three
sealed-owner controls pass in **15.71s**. The corrected command exits **0** in
**64.540s**, with no selected skips or failures. Log:
`.cache/graph-transport-final-corrected.log`. Formatting, scoped vet and staged
diff checks pass.

Full isolated `check` then `check --locked` pass in **13.740s / 12.313s**:
**736 source files / 9,169 MIR functions**, empty stderr. All **1,206** authored
`.fy` files match the snapshot and current source; only its successful lock is
copied back. `.cache/graph-transport-check-latest.json` records the snapshot and
unchanged owning Go bootstrap SHA256
`4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`.

Fairy-authored response/path/write admission, native executable orchestration,
one-attempt persistent CAS, ordinary workspace/CLI, broader families and all
remaining queue/final acceptance work stay open. Linux execution is batched.
Full Go suite, allocator-failure injection, clean-machine/self-build acceptance
and hosted CI are unexecuted. Public CLI/SDK/runtime ABI, DB, dependencies,
existing wire/default limits, repository/runners and C kernel are unchanged.
No push, dispatch, release or deployment.
