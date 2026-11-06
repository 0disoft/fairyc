# Source state graph integration v1

Status: explicit source-to-HIR/state-MIR admission with Windows native C and
local-store evidence. [ADR 0503](../adr/0503-experimental-source-state-graphs.md)
owns the admitted source subset. This advances step 4 of
[native persistence integration](native-state-persistence-v1.md); the full
source/helper/borrow/Complete/artifact/cache/ordinary-CLI acceptance stays open.
The explicit developer mode below now connects the admitted source subset.

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
