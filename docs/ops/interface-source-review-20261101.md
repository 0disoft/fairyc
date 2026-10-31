# Public-interface/effects review and large-source notice - 2026-11-01

Status: Opt-in developer tool, verified on Windows. Linux execution remains deferred.

## Commands

From the repository root:

```powershell
go build -mod=readonly -o .cache/fairyc-review.exe ./tools/dev/fairyc-review
.cache/fairyc-review.exe diff --before C:/work/before --after C:/work/after
.cache/fairyc-review.exe diff --before C:/work/before --after C:/work/after --format json --fail-on-change
.cache/fairyc-review.exe notice --root . --threshold-bytes 40000
```

Use two existing workspace directories; the tool does not check out Git refs or create copies.
Text is the default, and `--format json` selects private versioned developer receipts. The
executable returns 0 for a complete review, 1 only for a complete diff with known changes when
`--fail-on-change` is requested, and 2 for invalid arguments or incomplete comparison/scan.
`go run` is convenient interactively, but wraps nonzero child exits; use the built executable
when distinguishing 1 from 2 in automation. A large-file finding alone always remains advisory.

## Interface and effects contract

`internal/interfacereview` (`03195a2e`) uses the existing read-only workspace loader and
`workspacecompiler.CheckContext`. Both source closures must verify before ordinary typed facts
can be compared. No source regex substitutes for the semantic compiler. Source and v5 projection
hashes, root UID/name/version/language/features and gaps remain in the report. Successful output
does not embed absolute workspace paths, timestamps or disposable package numbers.

The comparison covers the **root package**, with dependency types represented by the existing
stable coordinates. It does not recursively review every dependency's exported API. Matching
requires equal package UID, language and feature context; a version change alone does not rename
the package or create an interface difference.

Matched items use exact module/kind/name keys. Added and removed declarations remain separate;
the tool never infers a rename. Known facts cover ordered parameter types and result, unsafe and
must-use flags, C ABI/link names, owner types, repr(C)/C-enum representation, ordered public record
fields and enum variants, resource abilities, and declared function/foreign/drop effects.
Canonical effect additions/removals are separate from other changed fields. Module existence
is compared too, including empty public-item modules. Persistent state-layout hashes are opaque
exact identity changes, without inferring layout migration or compatibility.

Function bodies, private resource representation and documentation are outside the ordinary v5
projection. They can change the source hash while the projected interface stays equal. This is
deliberate scope, not proof of unchanged behavior, ABI, optimizer output or semantic versioning.
The tool reuses [v5 interface identity](../package/public-interface-hash-v5.md) and the existing
[effects contract](../language/0008-effects-and-capabilities-v0.md); neither contract or hash/cache
encoding changes. Current argument labels remain private-call-only under ADR 0396 and do not add
public parameter-name compatibility rules here.

| Evidence state | Meaning |
| --- | --- |
| `exact` | One observed item/module/layout change in the verified projection |
| `bounded` | Complete comparison within the named ordinary root-package projection |
| `partial` | Verified ordinary modules can be compared, with explicit omitted modules |
| `unknown` | No safe comparison is available, or package/language/feature context differs |

Modules containing generic declarations are omitted as a whole before extraction, including
generated specializations. The union of omitted modules on either side is excluded from the
comparison, preventing an unavailable old declaration from becoming a false addition/removal.
When all modules are unavailable, the result is unknown. These rules are conservative even for
private generics and do not replace the compiler's separate generic-interface identity.
With partial coverage, the projection hash identifies only the retained subset; it is not a
complete package identity or a cache-admission key.

Invalid/unsupported source, loader errors, cancellation, extraction errors or altered projection
hashes cannot yield an unchanged result. JSON `changed` is null when no change can be concluded
under incomplete coverage; a partial report may retain `changed: true` for exact observed changes
but still exits 2. `behaviorCompatibility` is always unknown. The private JSON schema is
`fairyc.interface-review/v1`; descriptor values are compiler-owned facts, not a new SDK format.

## Large-source contract

`internal/sourcenotice` (`586ecba4`) scans directory/file metadata without opening source contents,
loading manifests, parsing, compiling or following encountered symlinks. The default threshold
is decimal **40,000 bytes**, with a strict greater-than comparison: 40,000 is quiet and 40,001
produces a notice. UTF-8 counts bytes, not characters. `--threshold-bytes` accepts a positive
override. Sources and compiler limits are never truncated, rewritten or raised by the tool.

The JSON `fairyc.large-source-notice/v1` records threshold, scan budget, extension/exclusion
policy, visited/source counts, sorted relative-path findings, skipped symlinks and errors.
Supported extensions are `.c`, `.cc`, `.cpp`, `.css`, `.fy`, `.go`, `.h`, `.hpp`, `.js`, `.jsx`,
`.ps1`, `.py`, `.rs`, `.scss`, `.sh`, `.svelte`, `.ts`, `.tsx` and `.vue`; extension matching ignores
case. Named excluded subdirectories are `.cache`, `.git`, `.venv`, `build`, `coverage`, `dist`,
`node_modules`, `target`, `vendor` and `venv`. Their children are not traversed.

The scan stops at 100,000 visited entries. Cancellation, unreadable metadata or other scan errors
retain partial findings with non-success status. Invalid roots/policies are unknown. No finding
claims every possible source extension, an atomic filesystem snapshot or a required refactor.
Generated source remains visible when it matches the extension policy; the tool does not guess
whether it is authored or whether manual splitting is appropriate. No watch process, editor hook,
default build warning or workflow invocation is installed.

## Windows verification

Compiler-backed tests cover effects, parameter/result/unsafe changes, record order/representation,
enum variant order and foreign link names. Additional cases cover private/body/order-only changes,
deterministic JSON, exact add/remove rather than rename, invalid after-input, package mismatch,
bad projection hash, explicit partial generic coverage, resource effect deltas, opaque state-layout
identity and canceled/missing inputs. Notice tests cover byte boundaries, Unicode byte counts,
invalid syntax without parsing, policy exclusions, unchanged source/mtime, deterministic order,
scan-budget exhaustion, invalid inputs and actual symlink exclusion. No symlink test was skipped.

A newly built Windows executable passes six invocations: changed diff with opt-in exit 1,
unchanged exit 0, invalid-source exit 2, text diff exit 0, help exit 0, and an advisory large-file
notice exit 0. Workspaces retain only their original manifest/source directories, and the large
source retains all 40,001 bytes. Output-write failure and argument rejection are also checked.

Manual repository runs produce `.cache/interface-review-20261101.json` for a verified unchanged
`std_alloc_quickstart` root and `.cache/large-source-notice-20261101.json` for the source-size scan.
That scan observed 3,877 matching source files and 12 candidates above 40,000 bytes, including
generated parser C. These are point-in-time local findings, not a directive to refactor other work.

Passed:

```powershell
go test -mod=readonly ./internal/interfacereview ./internal/sourcenotice ./tools/dev/fairyc-review -count=1 -v
go vet -mod=readonly ./internal/interfacereview ./internal/sourcenotice ./tools/dev/fairyc-review
git diff --check
```

Linux/WSL, full `go test ./...`, native C smokes and hosted CI were not run for this pure-Go
developer-tool change. No push, workflow dispatch, publication or deployment ran. Public
language/SDK API, DB, dependencies, compiler flags/limits, grammar, interface/cache format,
repository hygiene rules, CI workflows and runners remain unchanged. The developer command,
its private output formats, tools guide, source index and ordered-work evidence are added/updated.
