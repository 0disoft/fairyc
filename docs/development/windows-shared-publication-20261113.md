# Shared asynchronous publication — 2026-11-13

`modules/win32_jobs/src/publication.fy` owns staging, bounded exclusive temporary
name retries, flush, protected input identity checks, explicit commit and cleanup.
The caller pumps events and explicitly authorizes publication. Before that point
Cancel/Close preserves the prior destination; after authorization it must wait
for publication/cleanup. A storage driver can delay completion. This does not
provide an atomic input filesystem snapshot or rollback after commit.

Hash's `saving.fy` now supplies only window/status policy. The starter uses the
same controller to save displayed text. A completed input read supplies a protected
file identity; a new text transformation clears it. The CLI's optional synchronous
source guard is only used without a window. Payloads may be empty, up to 1 MiB.

Validation passed: Hash check, pure publication test, locked release build,
three native writer safety cases and 57 packaged hash/compare/checksum CLI cases.
Generated starter check, all three pure tests, release build and 26 actual CLI
cases passed: nine read cases and twelve save cases plus five original cases.
Save cases cover Unicode/empty/replace, cancellation preserving old or absent
results, invalid destinations, temporary collision/exhaustion and actual source/
hard-link protection. No owned temporary survives; unowned colliding files retain
their bytes. Shared snapshot and scoped scaffold/CLI Go tests also passed.
The copied-compiler/empty-PATH outside-repository generator passed the complete
build/package/resource and CLI checks; local receipt:
`.cache/windows-jobs-20261113/publication.json`.

Hash EXE: 212,992 bytes (208 KiB); starter EXE: 171,520 bytes (167.5 KiB).
Source guides, embedded ZIP/checksums, the Hash lock and fixture source copies
were updated. No language/public SDK, DB, dependency, CI/runner or version change.
No window, physical input, clipboard or screenreader test was run. GUI save/close/
focus behavior is compiled only. Local commits only; no signing/push/publication.
