# Inline immutable locals: Windows implementation receipt (2026-11-07)

Scope: [ADR 0509](../adr/0509-inline-immutable-local-bindings.md). This is local
bounded product support in the shared main checkout, not closure of the queue.

## Product support and source owners

Go HIR and Fairy source admission accept up to sixteen immutable i32 locals,
with a shared sixty-four-expression-node limit, followed by the existing return.
Real locals pass through existing lowering and C generation. Canonical v2 bodies
include ordered initializers, including unused ones, and local-reference positions;
existing local-free v1 identities retain their bytes. Detached Fairy IR reconstructs
and verifies the ordered identity instead of dropping local statements.

Source admission owners are `internal/hir/inline.go` and
`examples/lexer_candidate/src/module_inline_declarations.fy`. No extra shared
implementation owner was needed. The preexisting negative identity test in
`internal/cli/candidate_inline_identity_test.go` was updated, after notification
and parent acceptance, from immutable to mutable locals. Mutation remains closed.

Existing Go semantic checking and MIR lowering/verifying already represent real
bindings; neither the shared semantic function checker nor MIR/C emission needed
an extension. Calls, mutation, control flow and effects remain closed.

## Recovery and final verification

A normalization script opened the fourteen owned files for writing before reading
them, truncating them. It was stopped, the seven tracked baselines were recovered
from the current HEAD, and this agent's earlier patch commands were reapplied.
The new tests and both documents were also recovered. No other worker's files or
parent commits were reset. All fourteen files were then checked nonempty, UTF-8
without BOM, LF only, with a final newline, using read-before-write normalization.
The parent-mentioned `.cache/inline-source-recovery-20261107` snapshot directory
was absent under both this repository and the workspace root when inspected;
recovery used this agent's exact earlier patch records, not arbitrary temp files.

The final commands run from the repository root on Windows AMD64, with MSVC
14.51.36231. CLI probes copy their dependency closure into temporary projects;
their generated locks do not modify the repository example lockfile.

```powershell
go test -mod=readonly ./internal/compiler ./internal/workspacecompiler -run 'Test(InlineLocals|CrossPackageInlineLocals|CrossPackageInlineIdentityCodecAndEmission|CrossPackageInlineRejectsClosedForms|InlineCacheRejects)' -count=1
```

Passed after recovery: compiler **0.120s**, workspacecompiler **3.737s**.
The compiler selection covers a fixed independently constructed v2 SHA-256 vector,
return 42, canonical rename/spelling stability, exact 64/65-node and 16/17-local
boundaries, unused initializer values/order in the identity, first unused
initializer traps, and rejection of calls, mutation, forward/self references,
non-i32 locals and control flow. Existing local-free identity, codec and emission
tests also pass.

Cache coverage includes memory/disk invalidation, a fresh persistent session,
selected and unused initializer edits, private edits, canonical renames/spellings
and marker removal. Missing/changed/extra marker rejection and healing cover
ordinary/generic cache paths. C11/C17 low/optimized native probes use an independent
C observer to prove restored cross-package local bodies, once-only argument order
and exit 42.

The final opt-in command is:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineLocals(Identity|Pipeline|TrapReference)$' -count=1 -v -timeout 12m
```

Final CLI selection **passed after recovery (235.926s)**: source identity 12.88s,
native pipeline 185.38s and Go CLI trap reference 37.54s. Both compiler profiles
and both target consumer profiles passed the pipeline.

The pipeline reconstructs body/program/project artifacts, rejects missing/extra
source markers and changed body markers, and directly rejects five forged
inline bodies: mutable local kind, initializer mismatch, forward local use,
assignment statement and changed returned local reference. It produces three
independent target packets: a successful ordered-argument program, first unused
initializer division failure, and a later unused initializer overflow. Each
compiler image and each target consumer use low and optimized profiles. An
independent C observer checks private trap categories, exact provider source
`p2/helper.fy` and independently counted provider byte ranges. Source identity
probes independently run under both profiles and check Go/Fairy hash agreement
plus 16/17 and 64/65 boundaries. The Go CLI trap reference checks exact source
columns 55 and 72 through ordinary/generic callers, interpreter/native engines,
and a passing test after both traps.

## Blocked or deferred gates

An attempted full authored CLI replay reached `invalid bounded project C inputs`
under both host profiles. A focused follow-up captured the actual dimensions:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineLocalsCLIHostDimensions$' -count=1 -v -timeout 4m
```

The dimension probe passed (24.152s), reproducing rejection before C compilation.
This is measurement/admission evidence, not full authored CLI execution acceptance.

| Input | Actual | Existing limit | Result |
| --- | ---: | ---: | --- |
| Header bytes | 2,099,985 | 2,097,152 | Exceeds by 2,833 |
| Literal-unit bytes | 24 | 2,097,152 | Within bound |
| Runtime bytes | 153,115 | 8,388,608 | Within bound |
| Unit count | 205 | 4,096 | Within bound |
| Largest body bytes | 314,396 | 2,097,152 | Within bound |
| Total bytes | 52,071,171 | 100,663,296 | Within bound |

The private harness requested 262,144-byte partitions. Unit count is not the
reason for rejection, and changing partition size cannot fix this header bound.
Full CLI/persistent-cache scope remains open for a separately bounded compiler-host
path owned by the parent; the ordinary 2MiB driver contract stays intact. No shared
driver/backend change was made by this slice.
The focused pipeline/artifact restoration and Go cross-package persistent cache
checks are the available evidence. Native Linux, full example admission, full
suite, hosted CI, push, release and deployment remain deferred. Parent separately
reported unrelated `tests/module_unit_bind.fy` E4503 in all-example native admission;
these checks use narrow copied closures.

Parent owns `fairy.lock` refresh, staging, commits, integration and shared queue/
index updates. Atlas and module-state files are outside this slice. Public API,
DB, runtime ABI, dependencies, repository hygiene and CI/runner are unchanged.

## Exact owned paths

- `examples/lexer_candidate/src/module_inline_declarations.fy`
- `examples/lexer_candidate/src/module_inline_identity.fy`
- `examples/lexer_candidate/src/module_inline_ir.fy`
- `internal/hir/inline.go`
- `internal/compiler/cross_package_inline_test.go`
- `internal/compiler/inline_locals_test.go`
- `internal/workspacecompiler/inline_cache_test.go`
- `internal/workspacecompiler/inline_locals_native_test.go`
- `internal/cli/candidate_inline_identity_test.go`
- `internal/cli/candidate_inline_locals_cli_test.go`
- `internal/cli/candidate_inline_locals_identity_test.go`
- `internal/cli/candidate_inline_locals_pipeline_test.go`
- `internal/cli/candidate_inline_locals_trap_test.go`
- `docs/adr/0509-inline-immutable-local-bindings.md`
- `docs/development/candidate-inline-locals-20261107.md`

Final hygiene: all fifteen owned files are nonempty UTF-8/LF with final newlines;
scoped `git diff --check` passes. No staging, commit or lock refresh was performed.

Coordinator acceptance after worker handoff: a freshly built Go bootstrap passes
the complete example `check` and `check --locked` at 701 source files / 8,705 MIR
functions. The example lock and current inline architecture/index/queue are
updated in this implementation slice. The separately bounded compiler-host
header path and full authored CLI follow-up remain independent work; the narrow
pipeline results above do not substitute for that full command acceptance.
