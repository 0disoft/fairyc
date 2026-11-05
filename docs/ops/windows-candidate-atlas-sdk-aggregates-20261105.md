# Windows Fairy Atlas SDK/user aggregate composition, 2026-11-05

## Scope and reproduced failures

The compiler-private aggregate route now preserves authenticated SDK Vec's
kind-13 storage type alongside user kind-20 struct and kind-21 resource instances.
The SDK constructor is resolved through its exact catalog owner, not its spelling.
SDK fields and cleanup still come from the existing authenticated SDK path; the
user aggregate preparation must not invent another SDK layout or scalar tuple.

Before the fix, mixed Vec/struct production fails in concrete type preparation
with E4502. After that gate is corrected, source-bound constructor demand still
fails in checked-unit closure with code 12, at the SDK `Vec<T>{...}` expression
(module 9, bytes 1647..1754). The latter reproduced command fails in 78.901s.
Allowing the source-prepared, authenticated kind-13 constructor removes the second
refusal; client structs named `Vec` retain their ordinary aggregate identity.

The initial corrected optimized probe passes four source families in 87.520s.
The final matrix adds a real `pop` request and shared bounded-failure controls;
that final command, not this preliminary run, owns the broader acceptance below.

## Acceptance boundaries

`TestAtlasSDKUserAggregateOracle` derives current Go v3 expectations for core
SDK/struct, Vec/struct, Vec/resource, a client struct also named Vec, and Vec pop
with a client struct. Each requires simultaneous SDK and client aggregate facts,
canonical codec roundtrip and a current-source call join. Each also requires
stable-source refusal, denied client storage intrinsics and denied nominal
arguments to a client aggregate. User aggregate parameters remain scalar.

`TestCandidateAtlasSDKUserAggregateProduction` runs the actual Fairy Complete,
producer, current-map reader and workspace join in a Windows native process, with
PATH empty. Every produced package is independently decoded and compared with the
Go partition for declarations, symbol/interface identities, specializations,
physical fields, cleanup summaries and resolved call endpoints. The explicit
candidate completeness limitations are checked rather than erased to match Go.
The pop case checks a callable `take_slot` instance together with its cleanup
observation. Source-instance collection, export, current admission and join exhaustion,
plus a missing coordinate, must reject without changing the fixture workspace.

This is source/map production evidence, not execution of the produced client
program, mixed-source persistent-cache acceptance or actual impact/MCP/edit
command acceptance. Those commands and full authored-main acceptance stay in the
composition plan. More source forms and the complete implementation queue remain
open.

## Commands and results

The combined Go-oracle/producer command passes in 751.183s. The Go-only oracle
passes in 1.91s and covers five positive source families, five stable-source
refusals and ten client-intrinsic/nominal-argument refusals. Native producer
acceptance passes low in 657.68s and optimized in 79.41s, with five complete map
comparisons and five deliberate failure modes per profile. These are acceptance
durations, not isolated performance measurements: the independent regression
command runs concurrently on the same host.

The existing non-SDK aggregate producer passes all eight source cases in both
profiles in 244.44s (low 176.13s, optimized 56.01s). The existing SDK Vec-only
producer passes unused/at/pop and its shared failure controls in both profiles
in 451.40s (low 370.16s, optimized 69.53s). The complete regression command
passes in 695.952s, with no selected skips or failures. Compiler source inputs
stay unchanged across the final production and regression commands. The
preliminary four-fixture optimized probe is not substituted for either final
profile of the expanded matrix.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 25m -v ./internal/cli -run '^(TestAtlasSDKUserAggregateOracle|TestCandidateAtlasSDKUserAggregateProduction)$'
go test -mod=readonly -count=1 -failfast -timeout 20m -v ./internal/cli -run '^(TestCandidateAtlasSDKVectorProduction|TestCandidateAtlasAggregateProduction)$'
go vet -mod=readonly ./internal/cli
```

From `examples/lexer_candidate`:

```powershell
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
```

Both workspace checks pass with two instances, 641 sources and 8,081 MIR functions;
the updated source-bound lock is verified. Scoped vet, Go formatting and diff
checks pass. Logs use `.cache/atlas-sdk-aggregate-{produce,regression,check,
check-locked,vet}-20261105.log`.

Native Linux, full repository tests, full authored CLI, final Go/Fairy comparison,
consecutive self-build, distribution/clean-machine and hosted same-commit CI are
not executed for this slice. Linux stays batched. No source grammar, public CLI,
SDK/runtime ABI, map wire, DB, dependency, default timeout, target maturity,
repository hygiene or CI runner change. No push, dispatch, release or deployment.
