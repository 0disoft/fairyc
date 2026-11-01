# Windows cross-package inline and cache v5 receipt, 2026-11-01

- Core implementation: `b76ab97d59df7301b96e78e2bbb538b93c167f80`.
- Disk healing correction: `5fd8a761`.
- Native probes and this receipt: commit containing this file.
- Host: Windows amd64, MSVC 14.51.36231.
- Owners: [ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md),
  [cache key v5](../package/whole-package-cache-v5.md) and
  [Phase 3 reopening](../development/phase-status.md#phase-3-inline-reopening-2026-11-01).

## Executed evidence

| Scope | Result |
| --- | --- |
| Native cross-package execution | Four passes, C11/C17 x low/optimized. A dependency exports `@inline value(a,b) = (a*3+b)/2`; the root calls it after a process-cache warm hit with MIR lowering skipped. Separately compiled C observer functions require arguments to execute exactly once in source order and the result to be 42. All images exit 42 with empty streams. Generated caller and observer are distinct objects; PE/AMD64 unwind inspection passes. |
| Partition visibility | Two native low/optimized passes. Multiple caller partitions consume the single private shared-header inline definition; ordinary source partitions contain no duplicate definition. Both images exit 42. This probe isolates partition mechanics in one module; the workspace probe above supplies the cross-package evidence. |
| Identity and codec | Function identity survives parameter rename, whitespace/comment/grouping and equivalent integer spelling. Body changes change identity; marker removal removes it. MIR codec round trip and both POSIX/MSVC profile emission preserve the marker and private body visibility. |
| Cache invalidation | Ordinary memory and fresh-session disk cold/warm, inline-body edit, private-body edit, formatting edit and marker removal pass. Inline edits/removal miss owner and dependent. Private/formatting edits retain the dependent key. Partial-hit requests still perform unified lowering; this is not selective per-package compilation. Restored MIR executes with the interpreter and emits both C profiles. |
| Mismatch recovery | Missing, changed and extra inline markers are rejected. Twelve injected stale-cache cases cover ordinary/generic x memory/disk x three mutations. Each forces verified lowering, repairs the store and reaches all-hit on the very next request. Disk cases reopen the session before consuming the stale records. These test injections are not claims that an ordinary lowering path produces stale markers. |
| Exact quarantine | Three adapter cases cover ordinary, generic and storage-transfer disk namespaces. Rejection removes only the originally observed immutable generation, survives caller mutation of decoded payloads and preserves a different generation published between lookup and rejection. |
| Scoped regression | HIR, semantic, MIR, public-interface, package-artifact, package-cache, package-cache-fixture and C-backend checks pass. Focused compiler/workspace inline checks, scoped Go vet, formatting and diff checks pass. Unchanged core results from the implementation slice were reused. |

The native tests establish functional execution and body visibility, not that MSVC actually
substituted the body at each call site or improved throughput. `static inline` remains a hint.
The C observer tests successful checked arithmetic; this receipt adds no native overflow-trap
matrix. Normal arithmetic/trap lowering remains the existing MIR contract.

The additional disk probe initially reproduced one extra miss after repair: stale and correct
immutable generations coexisted under one key, so the following lookup rejected the conflict.
The correction quarantines the observed generations on inline reassembly mismatch before
publishing repairs, using byte-exact generation receipts. It retains the existing best-effort,
concurrency-safe deletion policy and all store bounds.

## Reproduction

```powershell
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler ./internal/cbackend -run '^(TestCrossPackageInline|TestInlineCacheRejects|TestPartitionedInlineBody)' -count=1 -v
go test -mod=readonly ./internal/hir ./internal/semantic ./internal/mir ./internal/moduleinterface ./internal/packageartifact ./internal/packagecache ./internal/packagecachefixture ./internal/cbackend -count=1
go vet -mod=readonly ./internal/hir ./internal/inlinebody ./internal/semantic ./internal/mir ./internal/cbackend ./internal/compiler ./internal/packagecache ./internal/packageartifact ./internal/workspacecompiler
git diff --check
```

The commands consolidate the final focused and package checks run in separate development
steps. Repeated runs of the same matrix are not additional distinct coverage.

## Limits and deferred work

The admitted source family is public, safe, nongeneric i32 functions with zero through four
immutable i32 parameters and one bounded pure returned arithmetic expression. Helper calls,
effects, generic inline functions, aggregates, resource/borrow transport and broader bodies
remain closed. The generic-cache tests combine an ordinary inline export with an already
supported generic function; they do not admit inline generic declarations.

The marker is a consistency binding, not authentication of hostile executable cache contents.
Public-interface v5 remains body-independent; package-cache keys advance to v5, with frozen
legacy-v4 byte evidence and disposable old-version misses. Fixture JSON remains format v8.

Native Linux is deferred by maintainer request; the guarded GCC/Clang low/optimized probe
compiles and skips on Windows. No WSL, Linux sanitizer run, full repository `go test ./...`,
hosted CI, push, release or publication was attempted. The new pure-inline feature is not
implemented or promoted in the Fairy-authored candidate compiler by this Go Stage 0 slice.

Source-of-truth changes are ADR 0493, cache-v5/v4 status, Phase 3 status, ADR/package indexes,
package-cache fixture baseline/schema and the ordered queue. Existing attribute grammar is
reused. SDK/CLI APIs, runtime/foreign ABI, dependencies, DB, repository hygiene and CI/runner
configuration remain unchanged.
