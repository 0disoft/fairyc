# Fairy Complete-bound graph response admission, 2026-11-07

ADR 0533 adds a sealed Fairy result owner over three disjoint Sol high worker
implementations: bounded framing, Verified CFG/call-return planning and
snapshot/overlay observation admission. The coordinator owns shared types,
sealed composition, adversarial integration, final review and commits.

The integration emits the actual Fairy transport C and immutable request,
builds/runs them with MSVC and gives the received bytes to the Fairy decoder.
The existing Go decoder is only an independent test oracle. Successful Fairy
results are compared with Go and, for fresh snapshots, the independent MIR/
reference-state oracle. The original receive buffer is overwritten and freed
before result reads, final writes and trap categories are inspected.

Wire tests cover 650 authored cases per profile: header/frame truncation,
identity/digest/policy/count/length corruption, maximum-width integers, 64MiB
framing and shared exact/one-short work. Path tests use eight actual
current-source Verified graphs, eleven CFG paths and 109 events per profile,
including both branch arms, helper return continuations, trap numbering,
counter drift and exact/short/overdrawn work. These detached prerequisites
cannot construct the sealed Admitted owner.

Initial preparation failures are retained separately. The first wire attempt
used an unsupported narrowing cast; it is replaced with bytewise u32 decoding.
The coordinator's first wire run then found that the small parser did not
cross the shared test helper's partition threshold. A 4KiB test-only partition
budget fixes the harness; both profiles pass in **7.054s**
(`.cache/graph-response-wire-final.log`). No product limit is changed.
The first three integration attempts fail before execution on mutable resource
field projection/whole-field assignment in the new Evidence implementation.
They are not passes (`.cache/graph-response-attempt1.log` through `attempt3.log`).
The existing language rule is retained. Evidence now has the existing Buffer
ABI prefix and one uniquely owned allocation with offset-only values and
geometrically grown packed tables. The snapshot remains an owning immutable
Vec. A view is never used across reserve; cleanup drops the single allocation
and the compiler drops the snapshot owner. Attempts 4–6 expose test-only import
ordering, an argument-evaluation loan conflict, and a reserved test symbol.
Those are corrected without changing compiler admission. Attempt 7 passes
the real empty-commit integration in **20.822s**.

The broad integration command executes **67 graph sources/profile** and
**134 actual Fairy-generated C images** in **167.284s**. Normal results match
Go response admission and independent fresh-state MIR semantics. Both profiles
pass the selected identity/header/event/literal-key/observation/final-write
corruption controls, all eleven checked failure-prefix statuses (including a
lost next charged event), restored owner 511 and exact/one-short/overdrawn
ledgers. The command nevertheless exits **1**: the allocation injector stays
armed after a successful decode and later breaks test-result key formatting.
The failed command is retained as `.cache/graph-response-integration.log`, not
relabeled as an aggregate pass. Its unaffected normal/negative observations
remain valid on the unchanged product inputs.

The test hook now disarms immediately after decode, before test output.
The borrow/copy case then passes both profiles, including **48 allocation
positions/profile**, **five injected failures/profile**, copied read/write
lifetime, exact work, restored owner and all its corruption controls. Every
native fixture invocation checks **zero live SDK heap allocations** after the
Fairy entry returns. The intermediate corrected command still exits **1**
solely because the new large-read test hits the generic C driver's 64KiB output
default; its owner-refusal tests also pass in **13.70s**. That record remains
`.cache/graph-response-final-corrected.log`.

Only the large-response test opts into the driver's existing 64MiB bound.
The fresh **37.800s PASS** (`.cache/graph-response-large-final.log`) executes
both profiles with a **2MiB present snapshot read**, version **73**, a **4KiB
write-value policy**, cost **2** and one operation. The copied result survives
destruction of receive bytes. This verifies that runtime reads do not inherit
the source-literal or write-value ceilings. No production default is changed.

An additional independent source writes/reads 20 map entries and deletes ten,
forcing both index tables across 8/16/32 capacities, physical-key order shifts,
overlay replacement and rollback-safe allocation failures. Its final command
passes in **40.895s** (`.cache/graph-response-growth-final.log`), with **50
Fairy processes/profile**, **48 allocation positions/profile** and **ten
injected failures/profile**. Every call returns with zero live SDK heap blocks.
Together these sources cover **69 distinct
graphs in both Windows profiles**, with the broad run's unchanged valid
observations reused explicitly rather than repeating all native compilations.

Complete isolated `check` then `check --locked` pass in **15.736s / 12.326s**,
with **741 source files / 9,270 MIR functions**, empty stderr. All **1,211**
authored `.fy` files match the checked snapshot; only its successful lock is
copied back. `.cache/graph-response-check-latest.json` records snapshot
`.cache/graph-response-check-11138396` and the unchanged bootstrap SHA256
`4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`.
Scoped `go vet -mod=readonly ./internal/cli`, gofmt, diff and UTF-8/LF checks pass.

Linux remains batched. Full repository Go tests, allocator failure in every
unrelated compiler path, native orchestration/CAS, ordinary CLI selection, clean-machine/
consecutive-self-build/distribution and hosted same-commit CI remain separate.
Public CLI/SDK/runtime ABI, DB/provider, dependencies, default wire limits,
CI/runners and repository policy are unchanged. No push, dispatch, release or
deployment.

Selected commands (successful prerequisite results are reused on unchanged inputs):

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphResponseWire$'
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateStateGraphResponsePath$'
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateStateGraphResponse($|SealedOwner$)'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphResponse($|SealedOwner$)/(low|optimized)/(present-borrow-copy|snapshot-above-literal-cap)$'
go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateStateGraphResponse$/(low|optimized)/snapshot-above-literal-cap$'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphResponse$/(low|optimized)/overlay-table-growth$'
go vet -mod=readonly ./internal/cli
```

The broad command and first corrected command above are the explicitly retained
failures described in this receipt. They are not aggregate success receipts.
The last two commands and the corrected borrow/copy/owner observations close
their actual failed cases; none requires a product-code change after the
complete-source check. A complete 64MiB semantic decoder replay is not inferred
from the framing parser's exact-size controls or the separate 2MiB read test.
