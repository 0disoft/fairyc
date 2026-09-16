# Parser public-readiness snapshot

- Date: 2026-09-16
- Reviewed compiler baseline: `39221d05a8375aa25fef2ecc60eddef3974a6377`
- Status: local documentation review, not release approval or a current CI attestation

## Decision boundary

Publishing the Go Stage 0 prototype and promoting the Fairy-written parser are
different decisions. The Go compiler is the current user-facing implementation;
the native candidate remains experimental. Recent parser comparisons and loop
regressions do not establish an installable release, complete language support,
stable language/ABI compatibility, or self-hosting readiness.

This snapshot supports a clearly labelled experimental Go-prototype preview
after the publication checks below are closed. It does not declare those checks
complete, authorize publication, or change existing compatibility contracts.

## Three distinct surfaces

| Surface | What it provides | What must not be claimed |
| --- | --- | --- |
| Go Stage 0 frontend | Lossless lexing, recoverable syntax and AST lowering used by workspace loading, followed by Go semantic analysis | Parsing a construct does not prove backend execution or target support; stable command contracts do not make the entire language stable |
| Fairy-written lexer/parser candidate | Native recognition and bounded syntax records compared with the Go reference; detailed forms and independent budgets in the candidate inventory | No standalone name/import resolution, type checking, complete Stage 0 grammar, executable MIR, or replacement of the Go bootstrap |
| Private scalar test adapter | Converts a narrow native tree into Go AST for semantic comparison: one ordinary function in `module main`, `i32`/`bool` parameters and result, scalar expressions, bindings, assignment, conditionals and while loops | Not a public AST protocol or an independent semantic implementation; references, calls, Result handling, for and control exits are outside its admitted subset even where the candidate parser recognizes their syntax |

Sources: [workspace loader](../../internal/workspaceloader/loader.go),
[stable CLI contract](../cli/stable-cli-contract-v0.md),
[candidate inventory](../../examples/lexer_candidate/README.md),
[private adapter](../../internal/cli/fairy_scalar_adapter_test.go), and
[scalar corpus](../../internal/cli/fairy_scalar_semantic_test.go).
The adapter's separate limits are 4096 source bytes, 262144 wire bytes, 36000
fields, 256 nodes and depth below 64. These are not general Go compiler limits
or a promise that all candidate parser output is semantically supported.

The normative language remains the [Stage 0 grammar](../../spec/grammar/stage0.ebnf)
and its owning language documents; this table is an orientation aid, not a new
grammar or support contract.

## User path versus experiment path

Users should start from the [human quickstart](../../README.md#human-quickstart)
with a built Go `fairyc` executable: create an explicit-identity starter, check it,
and run its documented tests. Native execution in that quickstart requires
Windows/AMD64 and local x64 MSVC. Check the target-specific contract rather than
generalizing this path to every platform. These are human usage instructions,
not permission for an agent to run unconfigured product commands.

The `experimental` namespace, candidate programs under `examples/lexer_candidate`,
numeric syntax records and `_test.go` adapter are experiments, not an alternate
installation path. Promotion remains governed by the separate
[self-hosting gate](../architecture/self-hosting-readiness-v0.md), including
same-commit CI and stage1/stage2 evidence. This review does not change that gate.

## Retained local evidence

The [loop diagnosis](../../examples/lexer_candidate/loop-cost-diagnosis.md) records
the exact baselines, measurements and limitations:

- `b7241dfc` and `5be39d06`: narrowly guarded scalar borrow/Result optimizations,
  with comparisons against the original analysis paths.
- `19f9f0f4`: restored 30-loop scalar regression and retained rejection of the
  extra loop; the body-tree suite passed in 50.484s under its 120-second limit.
  Both native profiles retained 115 semantic comparisons, 58 unsupported shapes,
  50 malformed transports and seven producer-budget rejections.
- `39221d05`: 36 Go-only stateful-loop samples at depths 1/4/8, with expected
  diagnostics, borrow/handling facts and repeatable complete semantic results.
  Deeper stateful loops were not measured; no general scalability claim follows.

These are prior local results, not tests rerun by this documentation change.
They do not substitute for a release candidate's complete required CI, clean
installation, target checks or end-user smoke test.

## Required before a public preview

| Required item | Current evidence or gap | Closure condition |
| --- | --- | --- |
| Define the publication | No release candidate, channel or artifact selected by this review | Name source-only versus binary preview, exact commit/version and supported host/toolchain; label it experimental |
| Set ownership and distribution terms | Release owner is UNASSIGNED; no tracked LICENSE/COPYING/SECURITY-named files were found in this checkout | Owner selects and documents distribution terms and a usable support/security-reporting route; this review selects no license |
| Verify the chosen candidate | Recent local regression evidence only; current remote CI and packaged artifacts were not checked | Record required CI outcomes on the exact candidate, then verify clean installation/build and quickstart on each advertised target |
| Publish accurate boundaries | This snapshot separates Go, native syntax candidate and scalar adapter | Keep preview instructions explicit about experimental interfaces, unsupported target/features and known analysis limits; do not describe the native candidate as the compiler |
| Record stop/recovery ownership | Release document still has UNDECIDED blocker and escalation fields | Assign release/rollback owner and stop on failed candidate checks or unverified artifact identity |

The [release document](release.md) remains the owner of release decisions, and
[versioning](../compatibility/versioning-v0.md) remains the compatibility authority.
Missing publication decisions must not be silently filled in by a parser test pass.
No code, runtime behavior, API, DB, language revision, ABI, SDK or version changes
are made by this documentation-only snapshot.

## Local executable starter verification (2026-09-16)

On baseline `f1a305a72b74827c759a7ca20934f199c3d0e775`, the configured
`fairyc_starter_executable_smoke` built `cmd/fairyc` through the existing offline
Go wrapper and called that executable as separate processes, not in-process CLI
test functions. The [bounded smoke script](../../tools/starter-smoke.mjs) creates
a unique empty scratch parent under ignored `.cache/`, accepts no arbitrary paths,
closes stdin, checks output/status and preserves its local receipt.

| Step | Local result | Elapsed ms |
| --- | --- | ---: |
| Build compiler executable | success | 5401 |
| `new hello-fairy --uid example.hello-fairy` | starter created | 252 |
| `check` | lock created | 172 |
| `check --locked` | lock verified | 187 |
| `test --locked` | one passed, zero failed | 170 |
| `test --native --locked` | one passed, zero failed | 1739 |
| `run --locked` | exit zero, no stdout | 1714 |

Every step returned zero and empty stderr. Generated manifest, source, test,
README and the lock after its initial creation retained their hashes. Compiler
SHA-256: `9c244cbdcf6c54684e951a98856a09608de1433899b296827469fafd9faf2470`.
Receipt: `.cache/starter-smoke-fA71mR/receipt.json` (ignored, local-only).
These timings are single-run observations, not performance thresholds.

This closes the basic executable quickstart check on this Windows/AMD64 host with
its installed MSVC toolchain. The scratch project is fresh, but the host and Go
dependency cache are not clean installations. No installer/downloaded artifact,
release-mode matrix, interactive stdin, other platform or current hosted CI was
verified. Publication terms, ownership and candidate-specific release checks
above remain open. The added smoke runner does not change compiler behavior,
language/API/DB/ABI/SDK versions or grant installation/publication authority.

### Release-mode starter verification (2026-09-16)

On baseline `0a746f70c89c342c4cc86b6a7d5ea614a2e5608e`, the same executable
workflow also passed `test --native --release --locked` and
`run --release --locked`. Semantic, default-native and release-native test output
matched exactly: one passed, zero failed. Default and release runs both returned
zero with empty stdout; all commands had empty stderr. Source/manifest/test/README
and lock hashes were checked after each locked step and remained unchanged.

The fresh scratch receipt is `.cache/starter-smoke-fny90B/receipt.json`;
compiler SHA-256 is
`46ab755143b846151d6da3d086af521d048ef10a605592c4100807a246ee47bf`.
The default/release native tests took 981/931ms and runs 894/883ms in this single
local observation, not a speed comparison. The existing 150-second workflow
deadline and 180-second wrapper limit were unchanged.

This closes release-mode coverage for the generated starter on the current
Windows/AMD64 MSVC host. It does not verify clean-PC installation, other targets,
hosted CI, a packaged release or arbitrary user programs. No compiler production
code or version changed; only the smoke runner and this evidence were extended.

### Local-only ZIP rehearsal (2026-09-16)

On baseline `d9936781cf0a39a70944662b9d5ee901722f42e7`,
`fairyc_local_preview_zip` built an executable, packaged only `fairyc.exe`,
`README.txt` and `SHA256SUMS.txt`, then extracted it into a separate folder.
All extracted file hashes and the exact entry inventory matched the staged
files. The same full default/release starter workflow passed using the extracted
executable. No test used the original unarchived binary after extraction.

The fixed `--package-local` mode writes only unique ignored scratch outputs;
the archive is explicitly LOCAL-ONLY and NOT FOR PUBLIC DISTRIBUTION. The
[included guide](windows-preview-README.txt) states prerequisites, commands and
unresolved distribution terms. Checksums detect changes, not publisher identity;
no signing or authentication claim is made.

- Archive: `.cache/starter-smoke-XtQlDi/fairyc-windows-amd64-LOCAL-ONLY.zip`
- Archive SHA-256: `4cc59b0fa6faee2d37456c262c575d4c012418f1f95efc2c1d0c7528595d1389`
- Compiler SHA-256: `911fc1f19ab3fe85c82e7915b161c96969111e78afff7a1777ea88be744edd67`
- Local receipt: `.cache/starter-smoke-XtQlDi/receipt.json`

This is packaging mechanics evidence on the existing development host, not a
clean-machine test, reproducible archive claim or release. ZIP bytes may vary
with timestamps. Source publication, license selection, third-party notices,
signing policy, release ownership and external distribution remain unperformed.
No runtime, API, DB, language/SDK version or native parser promotion changed.
