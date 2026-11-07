# Body AST projection regression, 2026-11-07

Observed base: `4eba784788f83abf36cb521e353b5476d373f5ce`, with the parent's dirty
`module_unit_body_ast.fy` optimization. This sidecar changes only the new Go test,
its testdata fixture, and this document. Production FY, lock and commits remain
parent-owned.

## Actual public AST builder

`TestCandidateBodyASTProjection` uses existing `candidateAtlasProbeSize`, copies
the real authored import closure, and executes public `ast::build` and
`ast::build_cleanup`. It does not extract the private mapping helper or fabricate
an `Ast` value. The Go test records the production and fixture hashes and rejects
a source change between native build and execution.

The fixture generates one parsed source below 64 KiB and the independent
128-declaration policy: 80 small functions, an empty-cleanup resource, and a filler
function placed between the first and last 40 small functions. It has 83
declarations and 1800 `discard(1+1);` filler statements. The initially proposed
2000 statements would exceed the unchanged 4096 statement-row cap once block
links and the small functions are included; reducing the filler preserves the
counterexample without raising a limit.

All 80 public AST builds share **one** work counter. An independent full-scan
verifier selects source rows by the existing span/reach rules, then checks:

- Expression, entry and type original IDs, ascending source order, source spans,
  token IDs and all remapped child/type/entry links.
- Statement order and every expression, target, first/second and entry link,
  including the distinction between statement links and raw block counts.
- Declaration source identity, body/result roots and ordinary parameter ranges.
- Late functions retain original expression IDs beyond half the complete arena;
  sparse source positions are not mistaken for dense local IDs.

For each ordinary stress function the measured cost must be exactly
`N + retained_rows`, where N is the combined expression/statement/entry/type arena
count. Mapping retention shares its already charged scan visit; only the selected
copy visit adds another charge. There is no additional charge for appending the
selected original ID. Over the whole stress input:

```text
functions=80 arena=12004 selected=1200 work=961520 legacy_lower=1920640
dense=ok cleanup=ok reach=ok
```

`961520 = 80*12004 + 1200 < 1048576`. The former two full arena scans alone would
require `2*80*12004 = 1920640 > 1048576`. This old-cost value is an analytic lower
bound from the previous loops, not execution of an old compiler binary.

## Additional guards

- Dense single-function input retains every arena row: cost remains exactly
  `2*N`, so the optimization does not worsen its old charged visits. Starting at
  `1048576-2*N` succeeds exactly at the cap; one unit less available work refuses
  with code 10 without exceeding the cap.
- A prepared copied-closure source projects ordinary and lifted functions with
  the real reach mask. The owner excludes unreachable nested rows despite
  overlapping source spans. Every AST and its charged visits are checked against
  the full-scan oracle; the new cost does not exceed `2*N + reach_work`.
- Empty cleanup remains one charged visit even with the large unrelated arena,
  including exact-cap success and exhausted-cap code 10. Nonempty cleanup keeps
  its original links and measured `N + retained_rows + reach_work` behavior.
- Invalid declaration zero retains code 9 with no work; an already over-cap
  ledger retains code 10 and its original value.

## Windows verification

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateBodyASTProjection$' -count=1 -timeout 6m -v
gofmt -l internal/cli/candidate_body_ast_projection_test.go
git diff --check
```

**PASS**, low and optimized: package 21.747s (test 21.57s), low 6.33s,
optimized 11.91s. Both profiles produce the identical report above; subtest
durations include native compilation and are not speedup claims. Formatting and
diff checks pass. The fixture's initial reserved-name and statement-cap failures
are retained separately and are not passes.

| Input or evidence | SHA-256 |
| --- | --- |
| Production `module_unit_body_ast.fy` | `18c03e53facb404566cc62b621675523233a4f1bd2ed94c5daf1b802bb02dd32` |
| Go test | `f42b5aec7e0794e243e47e4fff48402fa6f709b85e5d13f2d334c3150c6319c6` |
| FY testdata | `778dbef245b730c37aac0474584eb4cdf03281078ff265f6eb7ad8022a6c2d1f` |
| `windows-native.log` | `9d5f2615890048343ffc8fa3f95f4a63dfb738603ecb964643a5e36850ac35ac` |
| Low temporary PE | `08e233277026635f9f0d62391d48636894026f2ea328dcda4a574074f2aeb69d` |
| Optimized temporary PE | `720b70fdf2378171cc4b7eae1f73e6dd4310e8f433a6a647752d7d8f5b7c7235` |

Evidence lives in `.cache/body-ast-projection-20261107/`; `worker-freeze.json`
records all three owned file hashes and the observed production hash. Images are
temporary Go test outputs. No native/global timeout, runner, dependency or policy
was changed. Linux, allocation-failure injection, full source/locked checks and
whole authored self-build/fixed-point acceptance were not run by this sidecar;
the parent owns integration acceptance. No commit or amend was made.

## Integrated verification

The same production hash passes the full isolated source check and locked check
in 18.807s and 17.246s: 755 files, 9504 MIR functions, all 1225 source hashes
matching, and the verified lock copied back. Evidence is
`.cache/current-source-check-ast-projection-final-20261107.log`.

The optimized Go-seeded complete CLI regression passes in 238.220s (205 C parts,
14 independent CLI processes, four cold/warm trap pairs and six Go-matching test
reports). Evidence is `.cache/current-whole-cli-ast-projection-20261107.log`.
`go vet -mod=readonly ./internal/cli` and `git diff --check` also pass.
These are single-run observations under concurrent host load. They do not yet
establish a full authored self-build, fixed point, or complete current-source
Go/Fairy parity. Native Linux execution remains deferred.
