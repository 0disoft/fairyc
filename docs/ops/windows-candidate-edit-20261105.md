# Windows Fairy structural body edit, 2026-11-05

## Syntax-only replacement core

The private `project_edit_body` module computes an owned before/after plan without
filesystem access. It admits one private ordinary safe non-entry function without
attributes, owners or generic parameters, checks the exact raw source SHA-256 and
replaces only its original AST body span. The original input is borrowed and kept
unchanged; the result owns its original bytes, edited parser and both raw digests.

The replacement's raw size is checked before trimming, with a 64 KiB maximum.
Trimming matches the reference `bytes.TrimSpace` Unicode whitespace set, including
vertical tab and form feed; BOM and zero-width space do not trim. Re-parsing must
find the same function name/start and an exact replacement-body extent. Closing
the body early to insert another declaration fails. Existing scanner/parser and
shared work bounds apply, and no partial result escapes a failure.

`TestCandidateEditBody` runs 50 cases under each Windows MSVC C11 profile. The
independent Go `sourceedit.PlanBody` owner supplies body spans, raw hashes and
successful edited bytes. Cases cover CRLF and Korean comment preservation,
nested braces, comment trivia, all 25 whitespace scalars, 65,535/65,536/65,537-byte
inputs, stale identity, work exhaustion, forbidden subjects, lexical/parse failures
and sibling injection. The Go oracle prepares an actual current workspace lock;
a missing lock cannot masquerade as a rejection of an invalid edit.

An empty i32 function body explicitly distinguishes syntax acceptance from
semantic validation: the private core may construct it, but the Go workspace
validator rejects it. The core does not emit verified=true, create an edit report,
change files or implement the command by itself.

Executed and passed:

```powershell
go test -mod=readonly -count=1 -timeout 5m ./internal/cli -run '^TestCandidateEditBody$'
go test -mod=readonly -count=1 ./internal/atlasmcp ./internal/sourceedit
go vet -mod=readonly ./internal/cli
git diff --check
```

The native suite completed in 44.182 seconds with both profiles passing. Its
duration includes compilation, child processes and independent Go oracle setup;
it is not a throughput measurement. An initial test-harness run omitted the
oracle's lock and failed; that setup was corrected before the final passing run.

## Remaining integration

Current symbol/root selection, full edited-workspace semantic/MIR verification,
the bounded preview report and actual CLI dispatch remain the next slice. Apply
also needs fresh closure fingerprints, exclusive edit locking, same-directory
staging/sync, final byte rechecks, opened-root atomic rename and explicit
post-publication failure handling. None is established by syntax-only evidence.

Native Linux stays in the combined queue. Full repository tests, hosted CI,
clean-machine distribution and push/release/deployment were not performed.
No public language/SDK ABI, dependency, DB, repository hygiene, CI or runner
setting changed. The owning structural-edit contract remains unchanged.
