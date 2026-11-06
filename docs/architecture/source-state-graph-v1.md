# Source state graph integration v1

Status: explicit source-to-HIR/state-MIR admission with Windows native C and
local-store evidence. [ADR 0503](../adr/0503-experimental-source-state-graphs.md)
owns the admitted source subset. This advances step 4 of
[native persistence integration](native-state-persistence-v1.md); the full
source/Complete/artifact/cache/ordinary-CLI acceptance stays open. Later helper
and root borrow steps (ADRs 0504/0505) connect the initial five-codec source
subset to the explicit native developer mode. ADR 0506 adds dynamic bytes/text
to that explicit path. Sections below retain their checkpoint scope; later
sections supersede the earlier pending statements only for their named slice.

The actual authored source in
`examples/contract_state_graph/transaction.fy` reads a key and flag, selects a
map value through nested branch/join, updates a cell, and reuses the key after
the join. Its five reads are actual native observations. Missing reads retain
canonical zero values separately from absence, so missing and present-zero
map entries do not collapse.

## Windows evidence

`TestExperimentalStateGraphSource` independently checks the selected map
values 0, 42 and 73 and path costs 10, 13 and 14 against source expectations.
Other source controls cover closed/open arms, typed copied aliases and found
aliases, scope escape, unsupported bodies/owners/effects, integer maxima and
overflow, and fixed-byte lengths immediately below/at/above 20 and 32. Source
construction accepts 1,023 and 1,024 operations and refuses 1,025; it accepts 61
and 64 blocks and refuses a 67-block request before building it. The state-MIR
verifier independently checks the resulting graph. Ordinary and original
single-cell admissions remain closed; input AST bytes remain unchanged.

`TestGraphSourceNativePersistence` compiles the authored source through the
new entry point and `PrepareCompiled`, builds real MSVC low/optimized graph
executables and performs 36 C attempts: commit/reject/trap, three different
seeded states and two attempts with close/reopen. Each attempt compares the
complete outcome and snapshot with independent Go MIR execution, and counts
one snapshot and at most one CAS. An unrelated map entry survives publication
and rollback. Source branch expectations 0/42/73 are checked separately from
the MIR comparison. `TestGraphSourceNativeCodecs` adds ten C executions across
the five source-admitted codecs in both profiles, using typed read values as
both map keys and values, including u32/u64 maxima, binary zero and 0xff.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/compiler -run '^(TestExperimentalState|TestLowerAnalyzed|TestCheckExperimental)'
go test -mod=readonly -count=1 -timeout 3m -v ./internal/contractstatenative -run '^TestGraphSourceNative'
go vet -mod=readonly ./internal/compiler ./internal/contractstatenative
git diff --check
```

Initial test compilation exposed incorrect harness API names, corrected to
the existing fallible store constructor and physical-key helper. Initial
source tests exposed the parser's expression-statement and final-if tail
representations; both are handled structurally now. The tests retain the
original category rules (canonical user categories are not a fixed enum).
No warning, runtime, codec, timeout, storage or graph-verifier policy is relaxed.

Native Linux, source helpers/borrows, dynamic byte/text schemas, ordinary CLI
integration, source/Complete/cache authority and final queue gates
remain unverified or unimplemented as stated above. No push/hosted dispatch,
release or deployment is part of this evidence.

## Explicit developer execution

`go run -mod=readonly ./tools/dev/fairyc-state <init|execute|snapshot> <directory>
<source.fy> --engine native-graph --profile low|optimized` now selects fresh
source-graph admission, a sealed Windows C image and the one-attempt CAS path.
The old positional Go-MIR invocation is retained independently. Details and
report v2 belong to [ADR 0500](../adr/0500-local-contract-state-execution-and-persistence.md#explicit-native-graph-developer-mode).
Only execute builds/runs C; init and snapshot never execute the source body.

`TestNativeGraphCommand` runs both profiles against an actual local directory:
init, repeated commits, source body/hash change, snapshot without execution,
reject/trap rollback, report-sink failure after publication, and reopened state.
The report-sink refusal advances the version exactly once and retains explicit
possibly-published diagnostics. Original Go-mode regressions still pass.
Invalid flags/cancelled contexts and invalid source refuse before store access;
graph source never falls back to original single-cell Go admission. There is no
cache lookup or reuse in this developer mode.

```powershell
go test -mod=readonly -count=1 -timeout 3m -v ./tools/dev/fairyc-state
go vet -mod=readonly ./tools/dev/fairyc-state ./internal/contractstatenative
go build -mod=readonly -o .cache/fairyc-state.exe ./tools/dev/fairyc-state
```

A separately built actual command also passes init, execute and snapshot with
the authored example, low profile, five native reads and reopened version 1.
The explicit mode completes developer selection for the currently admitted
source subset. Source helper/borrow and dynamic byte/text admission, trusted
current-source/Complete binding, package artifacts/cache, ordinary compiler CLI
selection, Linux execution and every final acceptance gate remain required.

## Source helper continuation

[ADR 0504](../adr/0504-source-state-graph-helpers.md) now connects up to 32
acyclic source helpers, including branch-local call sites, shared-root overlay,
return/reject/trap and checked ordinary call-only owner projections. The
projection is not the executed storage body or trusted Complete/cache evidence.
The graph and native route retain their independent checks. Source borrows,
dynamic byte/text schemas and the remaining queue above stay open.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/compiler -run '^TestExperimentalStateGraphSourceHelper'
go test -mod=readonly -count=1 -timeout 2m ./internal/contractstatemir
go test -mod=readonly -count=1 -timeout 3m -v ./internal/contractstatenative -run '^TestGraphSourceNativeHelpers$'
go vet -mod=readonly ./internal/compiler ./internal/contractstatemir ./internal/contractstatenative
```

The helper native root passes in 4.808s, with twelve actual executions and no
selected skips. The first harness build failed on an unused test import; it
was removed. No compiler-warning or policy relaxation was used. Existing
linear and graph callers retain their exact original bounds and result policies.

## Root source read borrows

[ADR 0505](../adr/0505-source-state-read-borrows.md) now connects explicit
borrow/view/copy locals, same-block aliases, frozen presence and copied-value
transport. Handles never enter graph edges; helper ownership and dynamic map
borrow keys remain closed. Bounds, exact-key invalidation, writing-helper
invalidation and typed canonicality re-enter the existing state verifier/runtime.
Dynamic byte/text schemas and Complete/artifact/cache/ordinary-CLI integration
remain required; the existing five source codecs retain their identities.

The new native roots pass 42 actual executions in 17.986s: 24 local-store
branch/commit/reject/trap/helper/close-reopen comparisons, ten five-codec map
borrow comparisons, and eight local-store runtime failures with zero CAS.
The developer command additionally executes both authored helper and borrow
sources under both profiles. It checks three reads, two writes, version 1 and
independent source costs 12/16, then reopens through snapshot. No Go result is
used to construct native output and no selected native case is skipped.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/compiler -run '^TestExperimentalStateGraphSourceBorrow'
go test -mod=readonly -count=1 -timeout 3m -v ./internal/contractstatenative -run '^TestGraphSourceNative(Borrows|MapBorrowCodecs|BorrowFailures)$'
go test -mod=readonly -count=1 -timeout 3m -v ./tools/dev/fairyc-state
go vet -mod=readonly ./internal/compiler ./internal/contractstatemir ./internal/contractstatenative ./tools/dev/fairyc-state
```

Native Linux and all final queue gates remain separate. No stable source/CLI/
SDK/runtime ABI, DB/provider, dependency, default limit or runner change.

The final combined `^TestGraphSourceNative` replay passes in 38.284s with all
100 actual C attempts (46 original source, 12 helper and 42 borrow cases).
The complete developer-command package passes in 15.465s. A separately built
executable also passes init/execute/snapshot for helper optimized and borrow low,
with costs 12/16, three reads and reopened version 1. Focused compiler controls,
full state/state-MIR/fixture regressions, scoped vet, formatting and diff checks
pass. Logs remain local under `.cache/state-graph-source-all-20261107.log`,
`.cache/state-graph-owner-commands-20261107.log` and
`.cache/state-graph-owner-smoke-20261107/receipt.json`. Full Go/Fairy comparison,
consecutive self-build, distribution, hosted CI and Linux remain separate gates.

## Dynamic bytes and text source codecs

[ADR 0506](../adr/0506-source-state-dynamic-codecs.md) adds `bytes`/`text` cells
and map keys/values to the explicit source graph compiler and existing native
developer selection. The ordinary semantic entry and original single-cell
source compiler retain the five-codec vocabulary. Layout analysis and the
helper call-only projection both receive the explicit opt-in; neither produces
Complete/artifact/cache authority. Original source AST bytes are unchanged.

The authored `dynamic/transaction.fy` example combines helper writes, dynamic
binary/UTF-8 map addressing, nested byte views, a complete text-scalar view,
copy-before-replacement and branch control. Independent successful expectations
are seven reads, six final writes and cost 25. The byte copy is `AB` followed
by 0x1a and the text copy is `나`, even after both original cells are replaced.

All 54 new Windows low/optimized native attempts pass through actual generated
C and one-attempt local CAS, independent Go outcome/snapshot comparison and
real close/reopen. Twelve authored-example attempts cover repeated commit,
reject and trap; eight cover absent/present-empty and NUL/0xff/CRLF/0x1a;
ten cover valid/fragmented UTF-8 views and distinct Unicode-equivalent keys;
24 cover each codec at 1,023/1,024/1,025-byte keys and
65,535/65,536/65,537-byte values. Text limits count bytes, not scalar values.
Invalid UTF-8 writes and over-limit attempts perform one snapshot, zero CAS
and preserve all durable bytes. Every case retains unrelated state.

The complete `^TestGraphSourceNative` replay passes in 66.804s with all 154
actual C attempts, including the previous 100 source/helper/borrow cases.
Source/semantic/developer focused validation passes in 0.324s/0.113s/4.281s.
The developer command checks init/execute/snapshot for helpers, borrows and
dynamic codecs under both profiles; only execute runs a transaction. Complete
state/state-MIR/fixture regression passes in 0.381s/0.123s/0.077s and the full
developer-command package passes in 11.618s. Scoped vet and formatting/diff
checks pass.

```powershell
go test -mod=readonly -count=1 -timeout 5m -failfast -v ./internal/contractstatenative -run '^TestGraphSourceNative'
go test -mod=readonly -count=1 -timeout 3m ./internal/compiler ./internal/semantic ./tools/dev/fairyc-state -run '^(TestExperimentalStateGraphSource|TestAnalyzeExperimentalState|TestAnalyzePackages(LowersSourceStateDeclaration|RejectsInvalidSourceStateDeclarations)|TestNativeGraphHelperBorrowCommand)'
go test -mod=readonly -count=1 -timeout 3m ./internal/contractstate ./internal/contractstatemir ./internal/contractstatefixture ./tools/dev/fairyc-state
go vet -mod=readonly ./internal/compiler ./internal/semantic ./internal/contractstatenative ./tools/dev/fairyc-state
git diff --check
```

Logs remain local under `.cache/state-graph-source-dynamic-final-20261107.log`,
`.cache/state-graph-dynamic-source-owner-20261107.log` and
`.cache/state-graph-dynamic-regression-20261107.log`. Native Linux, exact
current-source/Complete provenance, artifact/cache invalidation, ordinary
compiler CLI selection and every final acceptance gate remain required.
No stable language/CLI/SDK/runtime ABI, dependency, DB/provider, default limit,
repository-policy or runner changes occur here.

## Separate current-source analysis for cache admission

`AnalyzeExperimentalStateGraph[Context]` now rebuilds the current layout,
owners, helper projection and typed source graphs without publishing MIR.
`CheckExperimentalStateGraph[Context]` uses that analysis and the unchanged
atomic `LowerAnalyzedContext` boundary. Existing callers keep their signature.
Canceled requests publish neither ordinary nor state MIR, and nil explicit
contexts refuse. This is a prerequisite for current-source cache admission;
it does not itself create a cache hit or bind an artifact to current source.

The four authored source families pass analysis plus independent check/lower
equivalence, exact stable body hashes under package 57 and source AST
immutability. Source refusal and cancellation controls pass. The complete
`^TestExperimentalStateGraphSource` selection passes in 0.201s. The initial
test attempt used a file basename inconsistent with the authored module and
failed parsing; the harness now uses the required `transaction.fy` identity.
The native dynamic/helper/borrow replay passes all 48 actual C attempts under
both Windows profiles in 15.951s. Scoped compiler vet and diff/format checks
pass. Logs: `.cache/state-graph-analysis-source-20261107.log` and
`.cache/state-graph-analysis-native-20261107.log`.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/compiler -run '^TestExperimentalStateGraphSource'
go test -mod=readonly -count=1 -timeout 3m -v ./internal/contractstatenative -run '^TestGraphSourceNative(Dynamic|Helpers|Borrows)$'
go vet -mod=readonly ./internal/compiler
```

Current-source artifact/cache use, Fairy Complete authority, ordinary CLI,
native Linux and all final acceptance gates remain required.
