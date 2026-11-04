# Six follow-up review fixes, 2026-11-04

The six findings from the follow-up review are implemented in separate local
commits. Linux execution remains in the maintainer-requested
[batched queue](../development/ordered-work-20261031.md#batched-native-linux-validation-2026-11-01).
No push, hosted CI dispatch, release or deployment was performed.

| Finding | Implementation and evidence |
| --- | --- |
| Quadratic dominance allocation | The dense MIR fixed-point table has a checked 64 MiB ceiling. Size arithmetic is checked before multiplication/allocation, including machine-integer extremes. Exhaustion rejects the first trapping operation with `V1501`; branch-only functions still avoid dominance. The reference-oracle graph tests remain passing. Linear graph metadata is outside the table limit. |
| Unbounded installed-tree enumeration | Shared admission accounts for at most 8,193 entries including the package root, with 128-entry read batches and one overflow probe. Enumeration handles close before descent. Root and opened-directory identities are checked; partial scans cannot certify package reuse. Tests cover shared sibling limits, exact/adjacent boundaries, cancellation and cleanup. |
| Linux descendant lifetime | Standalone targets use their own process group and the existing group cancellation helper. Timeout and parent trap-pipe failure cleanup terminate the owned group; timeout errors preserve `context.DeadlineExceeded`. New fork-descendant and binary-stdin/trap-receipt regressions cross-compile but have not executed on Linux. Programs that deliberately escape the group are outside this cleanup guarantee. |
| Stable JSON allocation | Compilation, MIR and body envelopes perform size admission before constructing full output. Canonical JSON bytes and semantic/encoding error precedence remain covered. This bounds encoding work, not allocations performed by the existing semantic validators. |
| Repeated exact-name scans | One verification-local name index replaces per-file sibling scans. File kind/size and opened identity remain checked per file. A final bounded pass verifies directory identities, exact names and entry kinds. Case renames, additions, parent replacement and fresh-invocation invalidation are tested. These observations are not a filesystem transaction. |
| Duplicate small-build emission | Small `EmitBuild` results reuse the measured function and cleanup bodies, restoring private linkage when assembling the single unit. Ordinary `Emit` keeps streaming. Byte equality covers literals, inline functions, callbacks, nested cleanup, both targets and drop instrumentation. |

## Measurements

Measurements are local single-sample controls, not portable latency promises.

`BenchmarkInstalledTreeNameAdmission -benchtime=1x -count=1` includes index
construction and the final directory pass for 512 empty sibling files:

| Path | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| Repeated scans | 4,062,795,400 | 28,404,928 | 355,935 |
| Indexed, including two directory passes | 743,644,300 | 1,408,328 | 23,177 |

At 4,095 stable sibling names, the old successful lookups compare 8,386,560
directory entries in aggregate; the two index passes enumerate 8,190 entries.
File opens, identity checks and byte comparisons remain per-file costs.

`BenchmarkSmallBuildBodyEmission -benchtime=1x` compares the former duplicate
body emission with reuse on the same analyzed nested-resource program:

| Path | ms/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| Duplicate body emission | 1.509 | 209,432 | 1,403 |
| Reused measured bodies | 1.467 | 163,077 | 917 |

This removes one body traversal per small build; it does not halve whole-build
time. Regular streaming emission measured 140,443 B/op and 709 allocs/op, within
the isolated unchanged implementation's 139-141 KB and 703-710 allocation range.

## Validation and limitations

The following executed checks passed on Windows:

- Full `internal/packageartifact` and `internal/hir` suites, including exact and
  one-byte-short limits, invalid UTF-8/HTML escapes, Base64, custom IDs and
  operation-kind errors, retained/deep type spelling and unsupported future
  schema rejection. The escaped-payload allocation regression uses a 16 MiB
  `<` string and a 32 MiB limit: the roughly 96 MiB JSON output is rejected
  with less than 2 MiB of additional allocation for each of the three encoders.
- Full `internal/mir`, `internal/packagesource`, `internal/directoryscan` and
  `internal/cbackend` suites. The C backend suite included MSVC low/optimized
  partition, callback, inline and nested-drop execution; the final full run took
  212.014 seconds. Both-target emission equality is not both-target execution.
- C driver `TestLinuxExecutionRejectsInvalidOptions` and
  `TestExecuteLinuxTargetRejectsUnsupportedHostOrArtifact`.
- CLI `TestRemotePackageCLIOfflineLockAndFailures` and
  `TestRemotePackageCLINative`, including actual Windows low/optimized runs.
- Workspace `TestPersistentSessionReusesStableMIRAcrossProcesses`,
  `TestPersistentSessionReusesStableMIRAcrossIndependentSessions` and
  `TestPersistentSessionCompilerIdentityBindsCompilationFormats`.
- `go vet -mod=readonly` for MIR, package sources, C driver and C backend.
- `go vet -mod=readonly` for package artifacts and HIR.
- Linux/amd64 cross-compilation of the C driver tests, including both new Linux
  execution regressions. No Linux test process ran.
- The two scoped comparison benchmarks above, formatting and `git diff --check`.

Linux execution is deferred because this Windows laptop has no usable WSL; cross-
compilation is not native execution. Full repository tests, clean-machine
distribution and hosted CI are outside this bounded follow-up.

Owning contracts updated:

- [MIR verification](../mir/mir-v0.md): dominance allocation ceiling and failure.
- [Remote sources](../package/remote-sources-v1.md): bounded enumeration and name-index lifetime.
- [Linux build](../architecture/candidate-linux-build-v1.md): standalone target cleanup.
- [Stable compilation](../package/stable-package-compilation-v1.md): JSON size admission.
- [C backend safety](../mir/c-backend-safety-v0.md): measured-body reuse and streaming output.

Product API, SDK/foreign ABI, DB, dependency versions, repository hygiene rules,
workflows and runner counts are unchanged. The changes are local compiler and
experimental package-source hardening, with their owning contracts updated.
