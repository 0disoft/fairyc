# Source state graph workspace artifacts and cache v1

Owner: [ADR 0507](../adr/0507-source-state-graph-workspace-cache.md).
Scope: explicit Go workspace state-graph analysis and existing verified package
artifact/cache reuse. Fairy Complete, ordinary CLI and final gates stay open.

## Current-source and artifact ownership

`internal/workspacecompiler/state_graph_session.go` constructs the existing
bounded memory or exact-build disk `Session` with the state-source analyzer.
`Workspace.SemanticInput` and `SemanticPackages` retain the loaded workspace's
original source, coordinate and digest metadata. Each check performs source
graph analysis before interfaces/body hashes and cache keys are computed.
Invalid graph schemas, operands, helper effects and unsupported multi-file or
imported closures never authorize an old executable. Ordinary constructors
remain closed to the explicit graph source family.

Existing interface v5, state-body v1-v4 and stable compilation formats are
unchanged. Compiler-build keys additionally contain the explicit source
admission profile. Complete package artifact verification/reassembly still
precedes a hit. Memory hits lower current state MIR after ordinary MIR reuse;
disk hits reconstruct the existing stable ordinary/state bundle. No snapshot,
transaction result, native process outcome or native image enters this cache.

## Restored namespaces and native admission

Ordinary artifact reassembly assigns fresh function symbols. The explicit
profile clones the MIR, matches every exact package/module/function owner,
rewrites function and call symbols into current HIR, and binds the state root
according to memory/disk ownership. Void returns restore their inactive zero
destination and an absent helper list returns to nil. These representational
repairs preserve all active code, source spans, values, calls and state facts.
The ordinary MIR verifier runs again. Native `PrepareCompiled` still requires
the complete exact current HIR/MIR/state relationship before building C.

Rebind failures invoke ordinary cold fallback. Disk failures discard only
observed rejected generations before repair so immutable generations do not
conflict. A subsequent valid hit must restore the exact original program
identity. Cached/source objects are not mutated by namespace rebinding.

## Windows results

Four source families (base, helpers, borrows, dynamic bytes/text) pass memory
and disk cold/warm with exact C, body and framing-program identity. Warm checks
replace the coordinator's lowering entry with a failing sentinel and still
succeed. Four separate executions of the Go test binary reopen persistent
artifacts and produce the same independently retained parent identities.
Comment edits miss without changing state-body identity; helper byte edits
miss and change it. Current effect failures publish no MIR/state/artifact;
cancellation leaves cache observations unchanged; a second source file refuses.
The final portable selection, including repair, passes in 1.378s, with no
selected skips.

Both Windows profiles build all sixteen warm reconstructed images across
four source families and two cache policies. Each image performs two real C
commits with local-store close/reopen, yielding 32 attempts. Complete outcome,
ordered reads/presence/codec/bytes, receipt, cost and reopened state match
independent Go persistent MIR. Every attempt records one snapshot and one CAS.
The native selection passes in 15.017s, with no selected skips. The Go oracle
does not construct the native response. The actual Windows C process runs it.

Rejected-owner rebind/fallback/repair controls pass both cache policies in
0.188s. Existing ordinary session/persistent session/package-key regressions
pass in 0.791s. Scoped compiler/workspace/native vet passes.

```powershell
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/workspacecompiler -run '^TestStateGraphSession(PersistenceProcess|MemoryAndPersistent|SourceInvalidation|AdmissionAndCancellation|RejectedRebindFallsBack)$'
go test -mod=readonly -count=1 -timeout 3m -failfast -v ./internal/workspacecompiler -run '^TestStateGraphSessionNative$'
go test -mod=readonly -count=1 -timeout 3m -v ./internal/workspacecompiler -run '^TestStateGraphSessionRejectedRebindFallsBack$'
go test -mod=readonly -count=1 -timeout 3m ./internal/workspacecompiler -run '^(TestSession|TestPersistentSession|TestPackageKeys)'
go vet -mod=readonly ./internal/workspacecompiler ./internal/compiler ./internal/contractstatenative
git diff --check
```

Logs are local `.cache/state-graph-cache-portable-final-20261107.log`,
`state-graph-cache-native-20261107.log`, `state-graph-cache-repair-20261107.log`
and `state-graph-cache-ordinary-regression-20261107.log` in the same directory.
Earlier attempts exposed fresh MIR symbol mismatch, inactive return-local
rebasing and empty/nil helper lists; all are fixed in the source-only rebind.
The first native harness used a nonexistent layout convenience constructor;
it now uses the existing checked builder. The next compared zero-byte read
slice allocation identity; it now compares exact bytes together with the
independent presence bit. No runtime, storage, warning, timeout or native
admission check was weakened to make these attempts pass.

Native Linux, Fairy Complete/source/artifact/cache integration, ordinary CLI,
full Go/Fairy comparison, consecutive self-build, clean-machine distribution
and same-commit hosted CI remain required. No stable language/CLI/SDK/runtime
ABI, DB/provider, dependency, cache wire, default bound, repository policy or
CI/runner changes accompany this explicit workspace constructor.
