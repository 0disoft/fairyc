# Candidate source stamp bounds, 2026-11-07

The authored source-provenance tables now distinguish source cardinality from
package-owner cardinality. A workspace can contain more than 512 source files
without containing more than 512 packages.

## Changed admission

| Owner | Source-domain check |
| --- | --- |
| `generic_package_bindings::build` | Original source count uses `candidate_limits::source_files()`; resolved binding count uses `candidate_limits::project_imports()` |
| `generic_package_bindings::copy` | Detached module/source stamp count uses `source_files()` |
| `generic_package_context::compute` | Context source stamp count uses `source_files()` |
| `module_public_type_ids::current` | Public type identity source stamp count uses `source_files()` |

At this change's integration point, the shared bounds are 1024 source files and
8192 project imports. Each module imports `candidate_limits` directly; no new
capacity constant or storage array is introduced. Source stamps remain owned
`Vec` records. Existing digest, source-size, metadata-count, work and allocation
checks continue to run after admission.

Package owner IDs remain in `[0,512)`. Both
`generic_package_bindings::build` and `module_public_type_ids::owner` still reject
owner 512. The separate type-row cap 4096, detached-copy import guard 32768,
name pools, generic context work/encoding/hash ledgers, individual joins,
specialization seeds and cleanup limits are unchanged. This does not widen
the independent limits of any compilation unit.

## Narrow acceptance

`TestCandidateSourceStampBounds` compiles an authored probe, not the full CLI.
It exercises:

- 1024 distinct sources assigned to one package, actual binding collection and
  detached copying; 1025 sources rejected before consuming work;
- context hashing and 1024 retained source stamps;
- 1024 public type source stamps authenticated against full-source signature
  metadata and a real one-source scalar type graph; a changed last digest rejects
  with stale-source code 12;
- owner 511 accepted and 512 rejected independently in binding and public type
  identity APIs;
- 8192 resolved binding rows passing the input-count guard and reaching the
  unchanged exhausted-work rejection (10), versus 8193 rejected immediately (9).

The last case proves the resolved-binding admission boundary, not successful
resolution of a complete 8192-import graph. The existing work budget remains
independent. The probe's 90-second native process allowance is test-only.

Executed from the repository root:

```text
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateSourceStampBounds$/^low$'
go test -mod=readonly -count=1 -failfast -timeout 5m -v ./internal/cli -run '^TestCandidateSourceStampBounds$/^optimized$'
git diff --check -- examples/lexer_candidate/src/generic_package_bindings.fy examples/lexer_candidate/src/generic_package_context.fy examples/lexer_candidate/src/module_public_type_ids.fy internal/cli/candidate_source_stamp_bounds_test.go docs/development/candidate-source-stamp-bounds-20261107.md
```

Both native profiles passed: low 15.15 seconds (profile 10.27 seconds), optimized
31.07 seconds (profile 26.62 seconds). The initial probe attempted to access
opaque resource fields; the test was corrected to use public accessors and
assemblers before these passing runs. No product API was added for testing.
Whitespace checking passed. Logs are retained under
`.cache/sol-source-stamp-bounds-20261107/`: `low-tool-result.txt` captures the
observed tool result; `optimized.log` is the directly captured process output.

Final isolated whole-source check, lock refresh, fresh compiler seed and actual
Fairy self-build belong to the parent integration task. These narrow tests do
not establish whole-CLI emission or linked fixed-point acceptance.

No package-owner, public API/SDK ABI, dependency, DB, runner or CI policy change
is made. No commit, push, deployment or remote action belongs to this task.
