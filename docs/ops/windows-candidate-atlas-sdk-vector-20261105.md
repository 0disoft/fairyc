# Windows Fairy Atlas SDK vector source production, 2026-11-05

## Scope

The private producer now admits exact authenticated SDK Vec source separately
from the user aggregate mode. Original generic declarations, binder spelling,
public/generic interface identities, nominal kind-13 arguments, physical layout
and cleanup remain distinct from concrete function implementations. Map/cache
headers and nominal names cannot grant source authority.

The checked compiler inlines Vec cleanup's private `take_slot<T>` call into a
storage intrinsic. Its five ordinary-view element types therefore need ten
source implementations despite having no five separate helper frames. The
producer requires fresh Complete-bound source/current IR, an authenticated Vec
declaration and helper, matching scalar plan arguments, and exactly one checked
storage-take call observation inside each retained cleanup body. It retains the
original source declaration, span and canonical implementation ID. A source
plan node alone cannot authorize a frame-less implementation.

An ordinary `pop<i32>` call also materializes a callable `take_slot<i32>` frame.
The same cleanup inline observation joins that existing implementation. Function
summaries and both direct/inline call endpoints use one original identity.

The producer retains existing explicit unavailable coverage for relations it
does not collect. It does not copy Go's richer unsafe/borrow/effect facts into
an exact-coverage claim. The preceding
[checked join indexes](windows-atlas-join-index-20261105.md) keep current-map
admission plus joining within the unchanged shared work allowance.

## Acceptance boundary

The Windows native probes execute the Fairy producer and current/join APIs;
Go independently prepares the expected v3 maps. Fixtures cover an unused Vec
import (ten implementations), empty/push/at use (fourteen), and empty/push/pop
use (fourteen). They compare original declarations and IDs, interfaces, all
type arguments, all six physical fields, cleanup presence, exact ordinary and
specialized calls and global completeness. SDK declarations and the generic
interface remain independent of which root operations are used.

Control cases cover changed authenticated source, an ordinary collector's
refusal, exact work exhaustion, missing coordinates and source/export/current/
join budget exhaustion. Source work, export work and shared current/join work
retain separate bounded ledgers; the query ledger is never reset per map.
Workspace bytes remain unchanged and PATH is empty during native probe use.

## Windows verification

```powershell
go test -mod=readonly -count=1 -failfast -timeout 25m -v ./internal/cli -run '^TestCandidateAtlasSDKVector(Authority|Production)$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateAtlas(GenericProduction|AggregateProduction|Types)$'
go vet -mod=readonly ./internal/cli
```

The SDK command passes in 706.465s: authority 41.11s and production 665.05s.
Production profiles are low 548.50s and optimized 95.58s; each includes unused,
at-use and pop-use source cases. The unused low subtest includes five separate
negative budget/current controls and takes 385.10s overall. Each native process
retains the existing three-minute diagnostic cap. This is not a product build
timeout or runtime-speed acceptance.

The source-production regressions pass in 591.060s: aggregate 349.58s, generic
160.25s and types 81.08s. Both Windows profiles execute with no selected skips.
Scoped vet and Go formatting pass. Before the final full command, the focused
optimized unused case passes in 160.675s. Earlier development attempts exposed
the missing five inline implementations and shared-query work exhaustion; the
temporary numeric product diagnostics are removed.

The two preceding join-index modules initially had working-copy CRLF endings.
Final cleanup restores their repository-owned LF bytes and refreshes the raw
source-bound workspace lock. The full native matrices above describe the
pre-cleanup representation; final LF join replay and workspace checks are
recorded separately below. Source logic and Git-normalized module blobs are
unchanged by that cleanup.

Final LF `TestCandidateAtlasGenericJoin` replay passes both profiles in 50.611s
(low 24.66s, optimized 21.71s), including exact Go edge order and all existing
negative endpoint/limit controls. Workspace `check` passes with two instances,
641 sources and 8,080 MIR functions and updates the source-bound lock; its
matching `check --locked` is the final source/lock gate.

The command Vec guard, actual impact/MCP/edit execution, SDK/user-aggregate
combinations and the final complete authored-CLI replay remain subsequent
gates. Native Linux stays in the maintainer's batched queue. No source target
program execution or distribution proof is inferred from map-probe acceptance.

No public language/CLI, SDK/runtime ABI, map schema, DB, dependency, default
timeout, work limit, repository hygiene or CI runner changes. No push, dispatch
or publication.

## Impact and read-only MCP command follow-up

After the producer/current/join acceptance at `1cb4f678`, the actual source
query removes its SDK Vec guard. Impact and MCP use the same fresh locked
Complete-bound source preparation; no map, cache header or selected SDK name
grants source authority. The obsolete vector-only impact diagnostic is removed.

The impact fixture queries unused, `at<i32>` and `pop<i32>` workspaces, selecting
both visible and full declaration IDs. The independent Go workspace call index
supplies exact subjects, reachability and current contexts. Native image identity
is independently bound to the actual executable bytes. Exact call/package limits,
one-short cutoffs, changed-source rejection and relocked recovery pass, with
unchanged workspace bytes and an empty child PATH.

The read-only MCP fixture selects `std::vec::take_slot` in a live stdio session
with 127 resolved calls. It preserves the existing initialize/permission/tool
protocol checks, exact Go impact expectations, unknown-package isolation, stale
source refusal and relocked recovery in that same process. Stdout contains only
bounded protocol frames; raw source/host paths are absent. SDK declarations
remain noneditable, and MCP exposes no write tool.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 25m -v ./internal/cli -run '^TestCandidateAtlasSDK(Vector)?Command$'
go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run '^TestCandidateMCPSDK(Vector)?Command$'
```

Both commands pass without selected skips on Windows amd64 with MSVC:

| Root | low | optimized | Total |
| --- | ---: | ---: | ---: |
| Ordinary SDK impact | 341.87s | 91.61s | 452.62s |
| SDK Vec impact | 340.94s | 78.92s | 435.12s |
| Ordinary SDK MCP | 175.62s | 104.15s | 298.18s |
| SDK Vec MCP | 170.04s | 81.49s | 274.46s |

Complete Go command durations are 887.890s and 572.800s. Logs are retained in
`.cache/atlas-sdk-vector-impact-20261105.log` and
`.cache/atlas-sdk-vector-mcp-20261105.log`; timings include native preparation
and diagnostic controls and are not benchmarks. The source-bound workspace
`check` and `check --locked` pass with 641 sources and 8,080 MIR functions.
Scoped `go vet -mod=readonly ./internal/cli`, Go formatting and
`git diff --check` pass for this follow-up.

The full authored-main impact/MCP verifiers are registered, but that full CLI
replay is not repeated for this command-leaf slice. Vec user-body editing,
SDK/user-aggregate combinations, native Linux, current-source full comparisons,
consecutive self-builds, distribution and hosted CI keep their separate gates.
This supersedes only the earlier Vec impact/MCP refusal statements above.

No public language/CLI shape, SDK/runtime ABI, map schema, DB, dependency,
product timeout, work allowance, target, repository hygiene or CI runner change.
No push, hosted dispatch or publication.
