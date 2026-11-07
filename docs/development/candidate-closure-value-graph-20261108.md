# Candidate closure value graph leaf, 2026-11-08

ADR 0551 graph stage in the shared detached checkout
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/closure-packages-20261108`,
baseline `94eac0fc97e4c7b513313192aa31d2aa466138fe`.
The Go oracle and real small Windows native fixtures are now **verified**.
The earlier parent-only-verification interpretation was incorrect and is superseded
by this executed receipt. No commit, main-tree copy, push, CI, dependency install,
bounds increase, or shared lock/queue/runtime/source-preparation edit was made.

## Representation and exact scope

Graph tag **23** is the canonical safe `fn i32(i32)` leaf. Its scalar, module,
declaration, first and second words must all be zero. It does not create child
scalar nodes. Ordinary and aggregate graph publication deliberately validate it;
duplicate records and nonzero payload words are invalid. Direct graph wrappers
around this leaf remain invalid independently of resolver admission.

This baseline uses graph tag **22** for ordered generic scalar tuples, not C
callbacks. That tuple family is preserved. The separate syntax namespace keeps
safe function tag **17** and C callback tag **15**; the latter never enters the
new safe leaf branch.

`closure_value_type_source.fy` validates structural child kinds and backwards
arena links, one direct named i32 result and exactly one direct named i32
parameter. Comments, whitespace and a trailing parameter comma retain their
normal structural meaning. It does not infer signatures from whole source text.
Source/catalog bytes must match before resolution. The leaf must be a direct
source root of a nongeneric ordinary function result, parameter or immutable
typed binding, or an ordinary non-repr(C) struct field. Extracted children of
forbidden wrappers/groups do not become independent roots. Generic source
functions, foreign declarations, resources, state, constants and mutable bindings
gain no closure-root permission.

`closure_value_type_nominal.fy` traverses the already authenticated **current
Parsed source**, without accessing private catalog fields or reparsing source.
It rejects current-source closure-bearing nominal wrappers, resource/enum/repr/
generic owners and wrapped fields, including an Option<Self> cycle next to a
closure field. Closure-free recursive nominal identity remains supported.

The catalog exposes identities rather than provider source/layout facts. Imported
nominal field proofs and raw nominal-identity graph records therefore retain
their existing ownership boundary; this stage does not claim a cross-provider
closure-bearing layout proof. Graph validation rejects direct tag-23 wrappers,
not arbitrary unrepresented nominal field graphs. Existing nominal identity APIs
do not constitute storage or execution admission. Full source-set interface/
layout consumers must independently reject closure-bearing wrappers and resource
storage when they adopt this leaf. Their executable gates were not opened here.

Generic argument resolution refuses the direct closure ID and current-source
closure-bearing nominal transport. Scalar aggregate arguments and Vec element
admission retain their existing exact leaf restrictions. Resolver and ordered
batch errors preserve the prior graph prefix. Existing capacities are unchanged;
no additional source scan/parse or full-compiler performance claim is involved.

## Executed oracle and native coverage

The new Go oracle used the original structural AST and real semantic analyzer:
20 original written type cases, **3 TypeClosureI32 admissions**, and **8 malformed
controls**. Admitted result and parameter identities are distinct from C callback
HIR types. The native fixture compared original root/result/ordered-parameter
byte intervals, including Korean comments and CRLF, and the Go E0007
missing-close origin. Noncanonical signatures, grouped children, direct wrappers
and C callbacks do not acquire the safe leaf identity.

`tests/closure_value_graph.fy` executed all four fixture functions in both native
profiles: ordinary/aggregate resolution and publication, canonical identity,
source-owned result/parameter/local/field roots, source mismatch, ordered batch
rollback after a successful prefix, local nominal/storage/generic restrictions,
cycles, forged payloads/wrappers, duplicates and exact/over 4096-node capacity.
Its synthetic capacity prefix is never published as a valid graph.

Execution separation was checked through the **public** cleanup MIR constructor,
using otherwise identical scalar and closure records with valid matching glue:
the scalar control succeeds; tag 23 fails with code **1001**. A typed closure
literal still fails unchanged normal `syntax::parse` with code **4**. No private
Program construction or replacement runtime is used. Constructor/invoke/body
planner/ABI/executable serialization gates were not added.

The older `closure_value_types.fy` assertion now expects deliberate graph tag 23,
while its existing harness name is preserved. Its original type-retention and
JSON/bounds/native tests also passed with the new module graph implementation.

## Validation results and preserved attempts

Windows AMD64, `GOMAXPROCS=1`, from this detached checkout:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueGraph(Oracle)?$' -count=1 -timeout=8m -v
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValue(TypeOracle|Types)$' -count=1 -timeout=8m -v
gofmt -l internal/cli/candidate_closure_value_graph_test.go
git diff --check -- examples/lexer_candidate/src/module_types.fy examples/lexer_candidate/tests/closure_value_types.fy
```

- New graph attempt **4 PASS**: Go oracle 0.03s; small native test **18.66s**;
  package **18.829s**. Both `run -- <oracle>` (low) and
  `run --release --locked -- <oracle>` (optimized) exited **0**, empty stdout and
  stderr. Log: `.cache/closure-value-graph-attempt4.log`.
- Existing type regression attempt **1 PASS**: oracle 0.01s, with 10 sources,
  141 structural types, 25 safe signatures and 8 malformed controls; small native
  test **15.11s**; package **15.263s**. Its harness executes both low and optimized
  native profiles. Log: `.cache/closure-value-graph-types-regression-attempt1.log`.
- Go formatting: no unformatted paths. Scoped tracked whitespace check: exit 0.
- Failed attempts **1-3** are retained as
  `.cache/closure-value-graph-attempt1.log`, `attempt2.log`, `attempt3.log`
  (each latter basename has the same `closure-value-graph-` prefix). Initial
  failures exposed the private-resource access, private-module syntax path,
  conditional scalar inference, and unsupported usize-to-u32 fixture cast.
  The repaired implementation uses public/source-owned inputs; the executable
  proof uses the public constructor plus a positive scalar control.

Both native harnesses copied only their actual authored import closure into
separate temporary projects. They ran source `check` followed by real low and
optimized compilation/execution. Only those temporary project locks were
created; the shared example lock was not changed. No full candidate compiler
host, self-build, Linux batch, cache/distribution replay, CI or remote action was
run. Those are separate consumers/gates; a standalone vet pass was unnecessary
for this small test-only Go change already compiled by `go test`.

## Final source hashes

SHA-256; this receipt is excluded. First five paths are relative to
`examples/lexer_candidate/`. These are the actual source bytes used by the
passing graph and regression runs.

| File | SHA-256 |
| --- | --- |
| `src/module_types.fy` | `3017463724E852E8E73E909E628930E4F9C25522F989B42B6619DFFB0E2AFA87` |
| `src/closure_value_type_source.fy` | `4543122303DEA1AEC08097F03A0D63F781482DA5DF9BCBDA611ADF4EADABB495` |
| `src/closure_value_type_nominal.fy` | `C02AFE1640718B116CEEB0F1804F96EFBC2BA1F010654EFB891EC1154F40134B` |
| `tests/closure_value_types.fy` | `A61B89A1F708FA7679BB3B36C8CB25E6939E26C140B8FDF502F9FCB31CE7300A` |
| `tests/closure_value_graph.fy` | `A208559F5C45CA6C5092873C3B65DE51045C81E28BE791723BDEDF0F3BB4A2AF` |
| `internal/cli/candidate_closure_value_graph_test.go` | `8204891EA04C1B5EB027136749EEE1BBD393767567D891BFCC769CEA96BEEFAB` |

These six files plus this receipt are the complete worker scope. Concurrent
Go packageartifact/moduleinterface/workspacecompiler edits are other owners'
work and are excluded. No ADR/normative source-of-truth document, public API,
DB, runner, dependency or repository-hygiene configuration changed.

## Remaining consumers

Fairy source-set signatures/interfaces and ownership/layout proofs, body closure
construction/invocation, cleanup/executable MIR, public type text, project
remapping and artifact/cache formats still need deliberate adoption. In
particular, `module_public_type_text.fy` still falls through to code 9 for tag 23,
and cleanup MIR still rejects it. Imported nominal closure containment belongs
to source-set interface/layout adoption. Go stable helper IDs and Result CLI
remain with their assigned owners. This receipt establishes the exact structural
graph leaf and its small compiled fixtures, not executable closure support.

## Coordinator integration

All seven authored implementation/test/receipt files were copied to main with
matching SHA-256 values. The byte-identical small Windows low/optimized graph
and existing type-regression results above were reused. Go package/cache
restoration was integrated separately in `b63ac743`; it does not open Fairy
executable closure admission.

The full main example source was then checked with the current Go compiler:

```powershell
# Working directory: examples/lexer_candidate
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

Both commands passed: 2 package instances, 760 source files and 9,604 MIR
functions. The first command refreshed only the root source identity and its
references in `fairy.lock`; SDK/dependency versions did not change. The second
verified that lock. Logs are `.cache/closure-graph-main-check.log` and
`.cache/closure-graph-main-locked.log` at the main repository root.

No full compiler self-build, distribution, Linux native run, push or hosted CI
was performed. Source-of-truth contracts, public SDK/FFI ABI, DB, dependencies,
runner and repository-hygiene policy are unchanged.
