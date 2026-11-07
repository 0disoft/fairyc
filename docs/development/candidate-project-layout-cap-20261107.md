# Project compiled-layout cap, 2026-11-07

Base: `befb10d4e542152395b1e790f906dcf54dd3d751`, isolated worktree
`.cache/parallel-project-layout-cap-20261107`. This slice changes five FY files,
one focused native regression and its fixtures, plus the compilation-unit policy
appendix and this note. No root checkout source, lock, route or metadata-map
document is changed.

## Bounded policy and inventory

`candidate_limits::project_layouts()` is 2048 and replaces exactly four compiled
layout gates: `module_project_metadata` before a new merged layout,
`module_unit_body_ir_verify`, `module_metadata_artifact_model::bounded`, and
`module_metadata_artifact_decode`'s 104-byte-row `count_from`.
The merged-layout gate admits duplicates at capacity and refuses the next new
layout. All other limits and validation rules retain their existing values.

An additional compiled-IR metadata guard remains in
`module_project_public_view.fy:83`: `ir::layout_count(program)>1024`. That file is
outside this slice's authorized write set. Public-view extraction can therefore
still refuse a larger verified program. State layout/transport/local-schema
1024-field guards are separate storage contracts and were not changed. The
inventory found no other production FY compiled-IR 1024-layout guards.

## Wire/version decision

ADR 0008 and `docs/compatibility/versioning-v0.md` treat internal MIR/cache state
as exact-version disposable state. Cache keys include the full compiler build
identity; `module_unit_body_ir_model::cache_revision()` separately identifies the
private checked model. The model itself explicitly distinguishes that identity
from any independently versioned future envelope. The existing source-artifact
count extension likewise retained its marker (see
`candidate-source-artifact-bounds-20261107.md`).

Under that private exact-compiler policy, `FYMETA00` and the checked-model revision
remain unchanged here: only the maximum admitted count grows. The 8-byte count,
104-byte layout row, field order, representation and meaning do not change, and
all formerly admitted fixture bytes remain identical. An older decoder refuses
counts above 1024. There is no explicit general count-extension or cross-version
interchange promise in the repository; this decision must not be cited as one.
No cache key, source identity, provenance or cache-hit publication rule changes.

## Real-input diagnostic, owned by the parent

The root checkout's `.cache/current-project-metadata-replay-20261107/run.json`
records `diagnosticOnly=true`, `acceptanceEvidence=false`, unchanged inputs and
157 checked IR packets (496065867 bytes). Their 19873 input layouts, including
duplicates, coalesce to **1116 unique layouts**, 5708 fields, 5772 callables and
17685 parameters, with metadata work **266376**. The replay took **24.0749669 s**
and then failed in project cleanup with 44002, exit 2. The parent supplied the
2048-cap diagnostic engine; this sidecar inspected the receipt and did not rerun
the replay. This proves the observed metadata need exceeds 1024 and fits 2048;
it is not whole-program, clean-build, bootstrap or self-hosting acceptance.

## Focused Windows regression

`internal/cli/candidate_project_layout_limit_test.go` copies the authored import
closure and uses fresh fixture copies only. Three test-only constructors expose
otherwise private component/type-plan construction. The production gates,
verifiers and codecs are not replaced. The detached fixture has unique canonical
nominal coordinates and one i32 field per layout. Each input IR first passes the
real cleanup and body-IR assemblers; a canonical identity-prefix plan then feeds
the real project metadata builder. It is not a source-authenticated executable
fixture; the actual source-derived 1116-layout observation is the parent receipt.

The literal cases are **1024, 1116, 2047, 2048 and 2049**. An independent Go
big-endian encoder supplies complete `FYMETA00` bytes, including the otherwise
well-formed 2049-row refusal input. Accepted cases require every layout/field and
mapping, exact byte encoding, decode/re-encode equality, extract/restore through
full IR verification, truncated header/table/tail rejection, trailing-byte
rejection and codec work exhaustion. The merge receives <=512-layout units plus
a duplicate recipe after the last unique row; borrowed inputs remain intact.
At 2049, model assembly/check, encode, decode, full IR assembly and merged
metadata all must refuse with 1002 instead of returning a shorter success.

```powershell
$env:GOCACHE='C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/go-build'
$env:GOENV='off'
$env:GOTOOLCHAIN='local'
go test -mod=readonly ./internal/cli -run '^TestCandidateProjectLayoutLimit$' -count=1 -timeout 8m -v
git diff --check
```

Windows AMD64 MSVC: **PASS**, 22 C partitions, 16 MiB stack reserve. The focused
Go package took **27.800 s** (test 27.68 s); low and optimized subtests took
**6.56 s** and **12.49 s**, respectively. Both profiles produced the exact five
expected result lines. These are same-host observations, not performance claims.
The initial attempt failed in frontend lowering because the fixture directly
constructed private resources and inferred a mixed-width local; no native check
ran in that attempt. Test-only constructors and explicit usize locals resolved
those fixture errors before the successful low/optimized run.

The untracked worktree-local log is `.cache-project-layout-limit-test.log`.
Production hashes are logged **before** appending the test-only constructors:

| Artifact | SHA-256 |
| --- | --- |
| `candidate_limits.fy` | `48fac6de8bbfec28255b4a4fb6829bb87136c3cc72ba2bb61ba17e4f70cfb997` |
| `module_project_metadata.fy` | `64974d39353563003b3bcaa68a798b733763bf9843b301232efb1e630095ca40` |
| `module_unit_body_ir_verify.fy` | `7be5ac0b6402a35f1167de555870f0765e82dbb1beab284790b41e61b44d79b4` |
| `module_metadata_artifact_model.fy` | `12da97111f24e5bfae8676f6af67185e796a5c6ee235f9ffbe5c18723695a2fb` |
| `module_metadata_artifact_decode.fy` | `f168596d7d115130c3b0bb74554d0d63c07a85e55de1be7ab25942b66d3835e6` |
| FY main fixture | `520b5d963e213213a3cdf27e7be21f6d934dfc12948f59b6dd65159dbd093d96` |
| Independent 2048-layout wire | `ac3e78b1ef8dd13b5ebe8f486702ffa53fdd31150080a768e5c3170305927fff` |
| Low native PE | `bf9443a2be3e30bc8f36b374d1bd184ab91adc3c352bda9d3d3a30c4971fcf45` |
| Optimized native PE | `5a53da8cefffa1d865b9882307d5ea0af2172bc163ee9324a4670c8743950e67` |

Linux execution is batched by the parent. Full suites, full source/lock checks,
full self-build, CI and push are intentionally outside this bounded slice.
No public API, DB, dependency, repository hygiene or runner files change.
