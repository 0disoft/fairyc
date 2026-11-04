# Seven review findings: Windows validation, 2026-11-04

The seven reviewed findings were implemented as seven local commits on `main`.
Linux execution is deferred at the maintainer's request and added to the
[batched validation queue](../development/ordered-work-20261031.md#batched-native-linux-validation-2026-11-01).
No push, hosted CI dispatch, release or deployment was performed.

| Finding | Implemented result and evidence |
| --- | --- |
| Nested resource cleanup expansion | Shared pointer-receiver helpers replace repeated owning-field path expansion. The depth-20 fixture has 1,048,576 leaf paths and emits 20 helper definitions, including partitioned output and source-specific drop traces. MSVC low/optimized execution checks cleanup mutation and order. |
| LSP workspace ownership | The nearest explicit ancestor workspace catalog owns its declared member. Sibling dependencies and unsaved overlays resolve together; unrelated nested projects keep their own scope. Tests cover the initialized-folder boundary and symlink rejection. |
| Blocking input admission | Explicit source, native stdin-file and cache directories use opened-identity admission. Linux file/directory opens are nonblocking, with directory-only admission where appropriate. Windows regular-file, directory, replacement and symlink tests pass; Linux FIFO tests are authored and cross-compiled, not executed. Final file components that are symlinks are rejected. |
| Borrow owner lookup | A cumulative owner-to-binding index filters candidates against the exact fork's environment and live uses. A fixture with 192 unrelated live borrows and 192 sink assignments visits zero owner candidates instead of scanning 36,864 unrelated bindings. Fork, last-use, parameter-origin, alias and loop behavior remains covered. |
| Semantic report growth | Up to 256 combined error/unsupported facts are admitted, followed by one explicit `E0011` when more facts are omitted. Direct type errors, test declarations and post-semantic generic transport checks use bounded admission. A real LSP notification for 3,000 unresolved calls contains 256 errors plus the marker; compiler tests prohibit verified MIR from exhausted analysis. |
| Directory materialization | Cache scans, source notices and optional test-directory discovery read at most 128 entries per batch, charge a shared 100,000-entry guard before retaining entries and use one overflow probe. Incomplete cache lookup is a miss; incomplete retention/statistics scans return an error before cleanup or eviction. Notices retain partial findings and explicitly report exhaustion. |
| Retained LSP analyses | A 128 MiB source-byte proxy covers unique current/last-clean analyses, the full workspace catalog and closed error sources retained by diagnostic positions. LRU eviction clears all document references; definition/completion reload only the requested project. Shared views, exact limits, unreachable members, failed loads, closing and three-project recovery are tested. |

## Executed Windows validation

All of the following passed. Checks run earlier on unchanged inputs were reused.

- Full `internal/cbackend` suite; `TestNestedDropFanoutEmitsLinearHelpers` for both
  target profiles, with and without drop traces; `TestPartitionedNestedDropHelpers`
  and compiler `TestPrivateVectorNestedResourceDrops`/`TestPrivateVectorNestedResourceDropTrace`.
  Native execution used Windows MSVC low and optimized; Linux profile emission is not execution.
- Full `internal/languageserver`, `internal/semantic` and `internal/diagnostic` suites.
  LSP was rerun after adding failed-workspace diagnostic-position accounting.
- Focused compiler tests `TestCheckPreservesSemanticReportExhaustion`,
  `TestRestrictGenericSignaturesPreservesReportExhaustion` and
  `TestRestrictGenericSignaturesBoundsTransportUnsupportedAtCollection`.
- Borrow/owner/loop/call semantic regressions and
  `BenchmarkBorrowOwnerAssignmentUnrelatedLoans` with `-benchtime=1x`.
- `go test -mod=readonly -count=1 ./internal/directoryscan ./internal/sourcenotice ./internal/workspaceloader ./internal/packagecache`.
- `go test -mod=readonly -count=1 ./tools/dev/fairyc-review`.
- Durable-file `TestOpen(Regular|Directory)` and CLI regular/special-path,
  source containment and stable native-input tests. Windows symlink cases executed.
- Linux/amd64 cross-compilation of durable-file, package-cache and CLI test binaries.
- `go vet -mod=readonly` for `internal/cbackend`, `internal/languageserver`,
  `internal/semantic`, `internal/diagnostic`, `internal/compiler`, `internal/durablefile`,
  `internal/cli`, `internal/packagecache`, `internal/directoryscan`,
  `internal/sourcenotice`, `internal/workspaceloader` and `tools/dev/fairyc-review`.
- `git diff --check`.

## Contract documents updated

- [C backend safety](../mir/c-backend-safety-v0.md).
- [Editor preview](../editors/lsp-preview.md).
- [CLI input contract](../cli/command-contract.md).
- [Diagnostic catalog](../diagnostics/catalog-v0.md).
- [Immutable disk cache ADR](../adr/0163-bounded-immutable-disk-cache-store.md).
- [Workspace loader](../package/workspace-loader-v1.md).
- [Source notice contract](interface-source-review-20261101.md).

## Limits and deferred evidence

The LSP limit measures retained source bytes, not total AST/HIR heap or peak working
memory. One incoming workspace can still be loaded/parsed under the existing 64 MiB
workspace source guard before retention admission. Directory-limit tests use small
injected budgets; no 100,000-entry filesystem throughput benchmark is claimed.
Partial oversized directory results are sorted admitted subsets, not a complete
global lexical prefix.

Native Linux FIFO/replacement behavior and GCC/Clang cleanup execution remain
unexecuted. Full repository `go test ./...`, a new release ZIP and hosted CI were
outside this bounded change. Public CLI flags and required function signatures
are preserved; `E0011` is an added prototype diagnostic and input admission now
rejects final symlinks. DB, dependencies, repository management rules, workflows
and runner configuration were not changed.
