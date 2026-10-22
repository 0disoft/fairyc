# Phase 0 contract and fixture audit

- Date: 2026-10-23
- Inspected source baseline: `893cfbc98284d0c5d6634fbd5121dca88b44a85a`
- Scope: Phase 0 written-contract consistency and mandatory fixture-class presence
- Method: inspection of authority docs, owning contracts, manifests, runners, and supporting source;
  documentation corrections recorded below
- Supplied build evidence: root Go CI passes all package fixtures; latest local E1 and E2 MSVC checks pass

The initial inspection reused those results. Remediation checks below add focused execution;
passing fixtures alone do not establish that contradictory prose is correct.

## Inspected owners

`docs/README.md` establishes the authority order. I inspected `docs/product/02-spec.md`,
`docs/product/01-roadmap.md`, `spec/grammar/stage0.ebnf`, the relevant language, runtime, package, ABI, and
compatibility contracts, the fixture routing pages, and the mandatory manifests and runner expectations. Historical
and superseded documents were excluded as contradiction evidence.

## Mandatory fixture classes

| Class | Checked-in current expectation |
| --- | --- |
| Parser pass/fail | `spec/fixtures/parser/manifest-v0.json`: 22 cases |
| Name/type/Flow acceptance | `spec/fixtures/semantic/manifest-v0.json`: 63 cases |
| Whole-resource Flow | `spec/fixtures/resource/manifest-v0.json`: 12 cases |
| Region Flow | `spec/fixtures/region/manifest-v1.json`: 6 cases |
| Result Flow | `spec/fixtures/result/manifest-v0.json`: 24 cases |
| Verified MIR | `spec/fixtures/mir/manifest-v0.json`: 13 cases |
| MIR verifier mutations | `spec/fixtures/mir-verifier/manifest-v0.json`: 11 seeds, 23 mutations |
| Resolved package graph | `spec/fixtures/package/manifest-v1.json`: 8 included v0 cases plus 1 cycle case |
| Run/trap/bounded execution | `spec/fixtures/interpreter/manifest-v0.json`: 94 cases |
| C emission/drop trace | `spec/fixtures/c-emission/manifest-v0.json`: 56 cases plus generated trace unit |
| Target layout | Linux 16 cases; Windows MSVC 9 cases |
| C ABI | Linux 17 cases; Windows MSVC 15 cases |

Package identity, public-interface v5, and package-source manifests are also present, with 7, 13, and 5 cases.
Native run, sanitizer, runtime, and standard-library observations are executable tests under
`internal/cbackendnative/`, `internal/cruntime/`, and `internal/cstdlib*`, rather than one JSON manifest. A static
path check found no missing schema, case root, or declared source file in the language/backend manifests. No named
top-level Phase 0 fixture family is absent.

## Verified findings

### P0-1: Runtime array exclusions were stale (corrected)

The original `docs/runtime/runtime-stdlib-v0.md:18-19` said empty/repeated literals, array fields, mutation, properties other
than `.len`, and other array borrows remained outside the compiler-owned slice. These clauses contradicted current
language ownership and implemented source:

- ADR 0416 admits repeated scalar initialization; `internal/semantic/repeated_array.go:10` implements it.
- ADR 0125 admits mutable fixed-array byte views; ADR 0126 admits indexed mutation;
  ADR 0430 adds typed i64 array views.
- ADRs 0468/0471 admit bounded inline scalar and plain-struct array fields;
  `internal/cbackend/aggregate_types.go:179` and `internal/interpreter/field_store.go:32` implement the field path.
- `docs/language/0005-surface-syntax-v0.md:393-400` already records the current repeated/list construction,
  replacement, views, and field subset.

The `empty` clause remains correct. The grammar parses `[]`, while `internal/semantic/check_arrays.go:13-23`
rejects it as `empty fixed-array literal`; ADR 0416 keeps empty and zero-count arrays unsupported. This is the
runtime boundary retained by the corrected paragraph.

### P0-2: Compatibility contract repeated the stale boundary (corrected)

The original `docs/compatibility/versioning-v0.md:220-221` described repeated literals, fields, and mutation as
unstabilized or excluded in its current 1..=4096 array paragraph. The implemented sources above supersede those
clauses. Empty arrays, nested arrays, resource elements, broad coercions, and foreign signatures remain closed.
This was a second canonical prose contradiction.

### P0-3: Interpreter fixture documentation count drifted (corrected)

The original `spec/fixtures/interpreter/README.md:5` said 92 cases. The manifest contains 94, the runner requires exactly 94 at
`internal/interpreterfixture/runner_test.go:36`, and `docs/product/01-roadmap.md:152` also records 94. The executable
expectation agrees with the manifest; the fixture README needed correction.

## Remediation (2026-10-23)

- P0-1: narrowed the runtime exclusions to the still-closed array shapes and linked current surface syntax.
- P0-2: replaced the stale compatibility exclusions with the later repeated-initialization, mutation,
  element, field, and typed-view decisions. The provisional size ceiling and foreign ABI boundary remain.
- P0-3: corrected the interpreter README to 94 cases, matching the manifest and runner.
- Updated the emission README to distinguish the 56-case/224-comparison/112-sanitizer matrix
  from the historical 55-case/220-comparison/110-sanitizer receipt.

Local remediation validation passed:

- `go test -mod=readonly -count=1 ./internal/interpreterfixture ./internal/cbackendfixture`.
- Manifest counts: 94 interpreter, 56 emission and 63 semantic cases.
- 610 schema, case-root and declared source paths in mandatory manifests exist.
- All 69 relative link targets in the eight remediation documents exist; `git diff --check` is clean.

The full local suite was not repeated: hosted Frontend owns the cross-platform execution, and the
earlier focused MSVC and CI-helper results remain valid for the unchanged implementation.

## Hosted validation

[Frontend run 39817319718](https://github.com/0disoft/fairyc/actions/runs/39817319718) at
`5f62c9c3e8d4ceeed8830c96b82215c96784ea29` passes all nine jobs and the aggregate run.
The [Linux Go job](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119959193676)
passes the full fixture suite and the v4/32-case focused receipt without missing, skipped or
erroneous cases. The [native job](https://github.com/0disoft/fairyc/actions/runs/39817319718/job/119965856725)
records 56 emission cases, 224 outcome and 224 exact drop-trace comparisons, and 112 sanitizer
executions. Windows core, native smoke, lexer and all three parser proof groups also pass.

## Phase 0 judgment

**The audited contract/fixture scope passes.** The mandatory fixture classes are present, all three
identified documentation defects are corrected, and current hosted execution passes, including the
56-case emission matrix and v4/32-case focused Linux receipt. Full Phase 0 exit is not asserted:
this bounded inspection does not establish that every written contract is free of contradictions.
It also does not promote Phase 2 target maturity.
