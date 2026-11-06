# Source state graph integration v1

Status: explicit source-to-HIR/state-MIR admission with Windows native C and
local-store evidence. [ADR 0503](../adr/0503-experimental-source-state-graphs.md)
owns the admitted source subset. This advances step 4 of
[native persistence integration](native-state-persistence-v1.md); the full
source/helper/borrow/developer/Complete/artifact/cache/CLI acceptance stays open.

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

Native Linux, source helpers/borrows, dynamic byte/text schemas, developer and
stable CLI integration, source/Complete/cache authority and final queue gates
remain unverified or unimplemented as stated above. No push/hosted dispatch,
release or deployment is part of this evidence.
