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

## Current-source preview and complete edited-workspace verification

`project_edit_preview::prepare` resolves the current graph with an existing lock,
requires the exact root UID and rederives the visible/full symbol from independently
verified current source. Only after this selection does it construct the body
change. `project_compiler_input::with_source` copies the complete input and its
ownership/origin metadata, preserves module/import headers and remaps selected
import declaration IDs with one checked index. Source and binding copying is
linear in retained source/arena size and binding count; the old input is unchanged.

The full generic-aware compiler checks the edited overlay through semantic, Flow,
MIR, project linking and interface publication. A report is constructed only after
that factory succeeds. It matches `fairyc.structural-body-edit/v1` exactly and
must fit 8 KiB. No workspace write, lock refresh or child compiler is involved.

`TestCandidateEditPreview` passes 21 actual preview-process calls per profile with
an empty PATH. Six successful body/selector combinations compare every report
field with Go `sourceedit.PlanBody`; ten invalid-body combinations distinguish
syntax/injection failures from complete semantic rejection. Additional controls
reject a member UID, unknown symbol, stale raw hash and stale graph lock, then
accept restored source. The entire fixture tree remains unchanged by previews,
including rejected ones. This also exercises a fresh cross-package call through
the preserved import binding in the edited overlay.

```powershell
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateEditPreview$'
```

Both Windows C11 profiles passed in 242.935 seconds. Initial integration runs
found an immutable local cursor and a test runner that did not supply stdin;
both were corrected before the passing run. This is private preview-host evidence,
not full CLI or atomic application evidence. Broader Atlas source families retain
their separate admission work.

## Invocation syntax

`project_edit_options` owns the four required values and explicit `--apply` bit.
It accepts the exact `replace-body` subcommand, checks bounded argv reads and
rejects missing/duplicate flags, missing values, unknown options and extra
positionals before workspace access. Selector/path/hash semantics remain the
host's responsibility, preserving selection before body-file access.

`TestCandidateEditOptions` passes 16 native invocations per profile in 5.552
seconds. Ordinary and reversed flag order are checked against the real Go
preview; all thirteen invalid invocations are compared with the real Go flag
rejections. The apply flag is parsed without executing an apply, and the entire
Go-oracle workspace, including the body file and lock, remains unchanged.

```powershell
go test -mod=readonly -count=1 -timeout 3m ./internal/cli -run '^TestCandidateEditOptions$'
```

## Actual body-file preview command

`project_edit_preview` now separates exact current root/symbol selection from
finishing the edited overlay. Selection owns its source, coordinates and copied
names, so no loader lifetime escapes. The existing already-owned-body entry is
retained. `project_edit_cli` reads a portable body path through the retained
Windows root/child-directory adapters only after selection. Paths within any
local production source root reject; a sibling such as `src2` remains allowed.

The Go command now enforces that same existing outside-production-roots contract
through `sourceedit.ReadBodyFile`; previously its bounded reader alone admitted
non-Fairy body files under a production root. No flag or report schema changed.

The actual command leaf passes Windows low/optimized acceptance in 183.671
seconds with an empty PATH. The successful report matches Go exactly. Controls
cover invalid semantic/syntax/injected bodies, canonical path and case checks,
root/member production roots, sibling directories, 65,535/65,536/65,537-byte body
files, an actual directory symlink, broken stdout and help outside a workspace.
All preview/error paths preserve the fixture tree. The explicit apply flag fails
until its publication owner is connected, rather than reporting preview success.

```powershell
go test -mod=readonly -count=1 -timeout 12m -v ./internal/cli -run '^TestCandidateEditCommand$'
go test -mod=readonly -count=1 ./internal/sourceedit ./internal/cli -run 'Test(AtlasEdit|Snapshot)'
```

Both Go suites pass, including root/member body exclusion and a valid `src2`
control. Full candidate dispatch is wired in source; its full CLI acceptance will
be refreshed together with apply, instead of repeating that expensive image now.

## Exact workspace fingerprints

`project_edit_snapshot` hashes the exact sorted path/NUL/raw-file-hash/NUL
representation used by the Go edit owner. It includes every local catalog
manifest and production source plus raw `fairy.lock` bytes. The existing bounded
merge-sort key index supplies deterministic O(N log N) ordering. The graph lock
is merged separately, preserving room for 512 manifests and 512 source records
without dropping the additional lock input.

`from_resolved` describes the supplied immutable loader inputs; it does not claim
they are still current. It hashes the already owned source and manifest bytes,
avoiding N additional file opens and repeated directory scans. `capture` always
loads and verifies the current locked workspace before constructing that value;
application rechecks must use that fresh entry. Only the lock bytes need an
additional retained-root read. No cross-attempt file or directory cache is added.

`TestCandidateEditSnapshot` passes nine native invocations per profile in 66.486
seconds. Independent Go construction checks every fingerprint, including raw root
and member manifest whitespace, raw lock whitespace, ignored body files, and a
member whose sorted path precedes the graph lock. The fresh locked entry refuses
changed source without repairing the lock and succeeds after restoration. Every
snapshot attempt preserves the entire observed fixture tree.

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateEditSnapshot$'
```

This is fingerprint-port evidence. Exclusive locking, application revalidation,
staging and publication are not supplied by this primitive.

## Fresh preview revalidation

`project_edit_preview` retains only the 64-byte digest of the exact owned loader
snapshot, releasing the fingerprint index after selection. `recheck` loads the
current locked graph and validates the complete overlay again using its already
owned replacement body. Fingerprints, paths and full before/after source bytes
must match. No source or graph-lock write occurs in this entry.

`TestCandidateEditRecheck` passes eight deterministic process-barrier scenarios
per Windows MSVC C11 profile (16 native invocations, 199.02 seconds). The test
mutates files only after Fairy has prepared its preview. Source and dependency
drift, raw root/member manifest whitespace, raw lock whitespace and a changed
dependency with a newly refreshed lock all reject. Unchanged inputs and a change
to the external body file succeed using the previously owned replacement. The
independent Go fingerprint and complete tree preservation are checked each time;
PATH is empty and no compiler/CLI subprocess implements the Fairy recheck.

```powershell
go test -mod=readonly -count=1 -timeout 8m -v ./internal/cli -run '^TestCandidateEditRecheck$'
go vet -mod=readonly ./internal/cli ./internal/sourceedit
git diff --check
```

## Remaining integration

Fresh closure fingerprints and overlay revalidation are available. Apply still
needs exclusive edit locking, same-directory
staging/sync, final byte rechecks, opened-root atomic rename and explicit
post-publication failure handling. None is established by syntax-only evidence.

Native Linux stays in the combined queue. Full repository tests, hosted CI,
clean-machine distribution and push/release/deployment were not performed.
No public language/SDK ABI, dependency, DB, repository hygiene, CI or runner
setting changed. The owning structural-edit contract remains unchanged.
