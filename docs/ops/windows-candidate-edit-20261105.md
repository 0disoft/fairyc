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

## Remaining integration

The actual body-file invocation/CLI dispatch remains next. Apply also needs fresh
closure fingerprints, exclusive edit locking, same-directory
staging/sync, final byte rechecks, opened-root atomic rename and explicit
post-publication failure handling. None is established by syntax-only evidence.

Native Linux stays in the combined queue. Full repository tests, hosted CI,
clean-machine distribution and push/release/deployment were not performed.
No public language/SDK ABI, dependency, DB, repository hygiene, CI or runner
setting changed. The owning structural-edit contract remains unchanged.
