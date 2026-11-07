# Ordinary AST reuse regression, 2026-11-07

Prepared at HEAD `7a2ac59033db1e6578559980cc90bb2dcafa5a42`. The parent owns
the production changes in `module_unit_ordinary_names.fy` and
`module_unit_values.fy`, source checking, lock and integration. This sidecar owns
only `internal/cli/candidate_ordinary_ast_reuse_test.go`, its
`testdata/ordinary_ast_reuse/main.fy`, and this document.

## Actual pipeline and measured work

`TestCandidateOrdinaryASTReuse` uses the existing `candidateAtlasProbeSize`
helper, actual authored module dependencies and 256 KiB C partitions. The helper
builds/runs low and optimized Windows amd64 profiles. The regression records the
two production source hashes, body AST source hash, fixture hash and stress PE
hash, and refuses production drift during the test.

The stress fixture executes public `ordinary::build`, `closure::build` and
`values::build`. It compares 80 ordinary functions against fresh public
`ast::build` projections, including declaration/body roots, expression links,
statement links, entry/type links, original IDs, tokens and source spans. It
checks the standalone ordinary plan and the ordinary plan owned by Ready. Values
must retain the exact source IDs/links, frame/module/declaration coordinates,
160 typed locals and 80 checked returns. The retained AST remains readable after
values finishes.

Two generated inputs select the same 80 functions. An unused generic filler is
placed between the first and last 40 functions: zero statements in the small
input and 1800 `discard(1+1);` statements in the large input. The filler is not
instantiated or selected as an ordinary function. Both inputs remain under
64 KiB and have 82 declarations. The filler changes original arena positions but
does not change the selected function bodies or their value types.

For each independently rebuilt AST, charged work must equal
`whole_source_arena_rows + selected_rows`. Summed projection cost must be
`80*arena + selected`, remain below 1048576, and exceed 800000 for the large
input. The increase in ordinary name work must equal `80*arena_growth`, while
**actual Values work must be identical for both inputs**. This fails if Values
rebuilds ordinary ASTs and charges those repeated scans again. The independent
rebuild cost measures the projection work removed from Values; it is not an
execution of an old compiler or a wall-clock speedup claim.

## Existing regressions reused

The `existing-boundaries` subtest adapts the real
`examples/lexer_candidate/tests/module_unit_values.fy` fixture and invokes only:

- `call_results_reach_generic_cleanup_and_ordinary_locals`: imported provider,
  generic, cleanup and ordinary value frames, with original source coordinates.
- `stale_sources_are_rejected_before_value_publication`: stale source code 12.
- `value_admission_keeps_the_active_local_bound`: 128 locals admitted, 129 refused.

Copied closure admission remains owned by the existing
`TestCandidateCopiedClosureCalls` regression. It was not rerun in this sidecar;
the new fixture does not duplicate its closure matrix or claim that merely
importing closure modules exercises copied closure syntax. The parent separately
reported low/optimized passes for `TestCandidateOrdinarySourceIdentities` and
`TestCandidateGenericAggregateValues` (102.558s); those results are reused as
parent-reported evidence, not attributed to this worker.

## Ownership, limits and memory

`ordinary::Function` owns its Ast; `ordinary::body_ast(ref Function)` returns an
immutable borrow. Origin getters delegate to that Ast. Values owns its projected
tables and does not retain references to Ready. The existing concrete generic
body and cleanup AST paths remain separate.

The lifetime of all retained ordinary AST payloads extends to Ready's cleanup.
The checked-bodies owner can keep Ready alive through later stages. Temporary
reach/mapping buffers and source byte storage are not transferred into Ast.
The test deliberately holds a standalone ordinary plan as an oracle alongside
Ready, so it is unsuitable as a production peak-memory measurement.

No RSS, allocation capacity, allocator overhead or peak lifetime was measured.
No memory-reduction claim is made. Existing 512-function, 16384-name/reference,
128-active-local and 1048576-work policies remain unchanged. Source freshness
validation and original-coordinate checks remain necessary when borrowing the
saved AST. The in-memory Function representation is not an artifact wire format;
no artifact version or cache admission relaxation is part of this change.

## Verification status

The parent authorized native execution after freezing the production patch.
The first attempt failed in fixture code: check 38 incorrectly equated two
lexical locals with exactly two AST entry rows. All preceding source/AST/value
checks in that small input had passed. The correction records the actual entry
count and last expression origin before Values, then compares them afterward.
The separate existing-fixture replay also initially omitted its extra
`module_unit_plan` import dependency; the harness now copies that real module
through the existing helper. Neither correction changes production behavior.
The initial failed run is retained at
`.cache/ordinary-ast-reuse-20261107/windows-native.log` (70.994s).

The corrected run **PASS**, package 117.767s (test 117.57s). Stress low 11.36s,
optimized 38.00s; existing-boundaries low 10.01s, optimized 41.91s. These timings
include native build/runtime under concurrent same-host work and do not establish
a performance speedup. Both stress profiles produced the same report:

```text
small functions=80 arena=1044 selected=1040 projection=84560 names=86083 values=4149 expressions=160 locals=160 returns=80
large functions=80 arena=11844 selected=1040 projection=948560 names=950083 values=4149 expressions=160 locals=160 returns=80
reuse=ok origins=ok links=ok values=ok
```

The 10800 unrelated arena rows add exactly 864000 visits to name analysis and
the independent projection oracle, and zero visits to Values. All counters stay
under their existing caps. These are charged-work observations, not RSS or
timing measurements.

The complete passing log is
`.cache/ordinary-ast-reuse-20261107/windows-rerun.log`. The three production
hash guards passed before and after all four native executions. `gofmt -l` and
whitespace checks passed. Final owned-file hashes are recorded separately in
`.cache/ordinary-ast-reuse-20261107/owned-files.json` to avoid a document self-hash.

SHA-256 inputs recorded by the corrected run:

| Input | SHA-256 |
| --- | --- |
| `src/module_unit_ordinary_names.fy` | `aa10a5458c9aa4c86083e0b3bc59694376eb35568cf0589c0cd46e2967127a58` |
| `src/module_unit_values.fy` | `9d7510c3755ab8eebde67ce96e254902cbb3ba0a9e7c09295d6bbd9efe7de1fe` |
| `src/module_unit_body_ast.fy` | `18c03e53facb404566cc62b621675523233a4f1bd2ed94c5daf1b802bb02dd32` |
| Stress fixture `main.fy` | `17fa70737f673282294f47372351e24829dac5dcf6d2f12096aa517f95f46a33` |
| Existing `tests/module_unit_values.fy` | `bb9cdc7bea540fe8f90359c4fa9aad2cf09cab8ffbf2d2595620294c92604e72` |
| New Go harness | `949f4cb89f4147f7118f0a819d1ba844c2dd7585c65f8ce09ea1b66d027980cf` |
| Stress low PE | `1abdbc9a50d5c7e0fbbdb859337e0d89f2b6ceee11efff1be5711189d2c6c820` |
| Stress optimized PE | `f685a6abb6a21044a9603012a7f7fda546a2eaafefa44619abc67e453a51de0b` |

The `src/` and `tests/` paths above are relative to
`examples/lexer_candidate`. The profiles have different PE bytes and identical
runtime reports. Images live in Go test temporary directories and are cleaned by
the existing helper; the hashes and output remain in the log.

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateOrdinaryASTReuse$' -count=1 -timeout 8m -v
gofmt -l internal/cli/candidate_ordinary_ast_reuse_test.go
git diff --check
```

Full CLI builds, selfbuild/fixed-point, full source checking and lock changes
remain parent-owned. Linux native execution is deferred.
