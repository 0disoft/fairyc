# ADR 0427: Shared persistent cache retention

- Status: Implemented
- Date: 2026-10-03

## Decision

The explicit persistent workspace cache applies its existing entry and byte
bounds to the combined published generations in ordinary `v1`, `generic-v0/v1`
and `storage-transfer-v0/v1`. Record addresses, codecs and lookups remain
isolated. All three adapters open the same filesystem containment root for
collection and statistics while retaining their own namespace for record I/O.

Opening a store and publishing a record collect the oldest generations across
these three directories, with relative path as the timestamp tie breaker.
Reported retained entries/bytes describe the same combined set. Hits, misses,
rejections and evictions remain operation counters for the reporting store.
Changing source families does not multiply the selected retention allowance.
Opening with a smaller allowance can evict compatible sibling records; the
next lookup safely falls back to current-source compilation. Missing/corrupt
records never acquire authority over source checking or verified reassembly.

`MaxRecordBytes` remains a per-reader/writer bound: a smaller reader must not
classify a larger sibling record as corrupt. That record still counts toward
the shared retention allowance and can be evicted by the aggregate policy.

## Limits

This bounds published encoded record bytes, not total disk allocation or peak
build space. Unexpired temporary files are retained; expired compiler temporary
files in the supported directories use the existing caller-selected age.
Unknown versions/namespaces, unrelated files and symlink entries are preserved
and excluded from this allowance. Empty directories are not recursively removed.

No new global lock or daemon is added. Independent concurrent writers can
temporarily exceed the allowance; collection converges after publication/open
or an explicit sweep once writes stop. Concurrent callers with different
allowances apply their own policy, so shared roots should use one agreed budget.

Stable `build/run/test` still do not use persistent compilation caches. This
change does not reuse native executables or claim faster incremental builds.
Native reuse needs exact dependency invalidation for generated C, runtime,
compiler, linker, headers and libraries before it can bypass compilation.
SDK, language, MIR and cache record versions are unchanged. Existing cache paths
need no migration; only retention policy and experimental aggregate counters
change. No CI runner or dependency is added.

## Validation

Focused tests cover aggregate entry/byte limits across three stores, reopening
with a smaller allowance, namespace-local lookups, adapter integration, expired
sibling temporary files, preservation of live/unknown/unrelated files, and
namespace validation. Existing disk publication, corruption, subprocess,
filesystem-containment, artifact and source-session regressions remain required.
