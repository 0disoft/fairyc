# Fairy flat Option full CLI and cache, 2026-11-07

Owner: [ADR 0547](../adr/0547-flat-option-generic-transport.md). This test-only
follow-up connects the [restored-artifact source family](candidate-flat-option-transport-20261107.md)
to the real authored main, OS arguments, workspace commands and fresh-process
disk cache. The [Go cache receipt](go-flat-option-cache-20261107.md) remains a
separate implementation/format proof.

## Exact execution scope

`TestCandidateCompositeOptionCLIAndCache` retains the full `file_cli` dispatcher.
A two-package local workspace exposes a public scalar-generic pick function
whose private helper transports `Option<T>` parameters/results. The source
constructs Some/None, branches on None without reading its payload, copies and
assigns Option, returns it through the generic helper and extracts Some.
Scalar type arguments and all previous wrapper/FFI exclusions stay unchanged.

Each compiler profile runs 15 independent CLI processes and two built products:
cold/locked warm checks; development/release build and run; one isolated release
native test; locked refusal after a provider body edit; unlocked rebuild changing
42 to 43; locked reuse; invalid typed source refusal without changing valid
cache generation names/bytes; restored valid source; corruption/rebuild and
locked reuse of the selected generation. Native test JSON reports one pass,
zero failures and no stderr. No test result is cached.

The existing capture proxy forwards the exact Fairy C packet to the ordinary
C-only companion. A separate native observer consumes each selected current
packet; it does not supply source checking, replay an earlier packet or modify
the actual product build. Five observations per profile check all seven typed
scalar entries, including signed/unsigned boundaries and floating signed-zero
bits. The provider edit switches first to second arguments, and all seven values
must change accordingly after rebuild, warm reuse and corruption repair. Each
additional observer image uses the existing low C consumer profile; the actual
product builds still cover development and release.

Source-only recursive scalar seeds remain behind a literal-false branch and
never run. They retain uniquely shaped three-argument typed probe entries in
the current packet; the observer discovers them from exact unique C signatures
rather than hard-coded function IDs or new FFI exports. Actual scalar values
come from the independent C observer.

## Commands and authenticated hosts

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location) '.cache/flat-option-cli-cache-20261107/hosts'
$env:FAIRYC_CLOSURE_CLI_PROFILE=$null
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateCompositeOptionCLIAndCache$/^optimized$'
go test -mod=readonly -count=1 -failfast -timeout 15m -v ./internal/cli -run '^TestCandidateCompositeOptionCLIAndCache$/^low$'
go test -mod=readonly -count=1 -timeout 2m -v ./internal/cli -run '^TestCandidateCompositeOptionCLIFixture$'
```

Both selections pass: **optimized 50.900s**, **low 225.018s**, with no selected
skips. Each profile executes 15 CLI processes, two direct products and 35
independently observed active scalar payloads (30 processes / four products /
70 payload observations in total).
The final fast Go fixture passes in **0.163s**, independently admitting both
accepted provider bodies and rejecting the invalid one with source status 3.

The existing exact generated-C/image digest helper verifies copied host seeds
before reuse. Seeds are copied into a separate artifact directory from the
parallel closure run; neither companion proxy nor image writes touch that
worker's host directory. Both current-source C digests authenticate successfully:

| Profile | Generated-C/profile SHA-256 | Complete image SHA-256 |
| --- | --- | --- |
| low | `429e846711cc30a5f914b28da12681084b7f3c22e8b453f28e7fea9771f61da9` | `5b3210e6f9ba605ef6b519bc8165c9493ebbce01b6e56499b37e0a58772125c9` |
| optimized | `7743161be3093332a56258f6206d5404a00c8b100138d6aebc076e884bde648c` | `c775f0a5928f8b0e7e312541d9bc762ecfe59f5beb1032600b019b28dea4cf9a` |

Existing explicit compiler-host header/stack policies and native command limits
remain unchanged. Host/source identity is not a clean single-commit self-build
or a native-companion fixed-point claim.

An initial optimized run reaches the accepted commands and source refusal, then
fails a test-only locked-recovery expectation. Ordinary typed source errors can
follow publication of the changed source-set lock. Returning to valid source
therefore requires unlocked check before demanding locked reuse again. The
corrected test requires a real cache hit during that check and then verifies
locked corruption repair and execution. No product lock policy is changed.
The failed 45.668s attempt stays in
`.cache/flat-option-cli-cache-optimized-attempt1-20261107.log`.
Final profile logs are `.cache/flat-option-cli-cache-{optimized,low}-20261107.log`;
the final Go fixture log is `.cache/flat-option-cli-fixture-final-20261107.log`.
Scoped CLI vet, gofmt and diff checks pass. The source/lock check from the
unchanged production files is reused; no whole compiler build is repeated for
this test-only addition.

## Remaining scope

This closes only the tested flat Option Windows command/cache family. Native
Linux, broader wrapper/closure families, refreshed whole
Go/Fairy comparisons/self-build/fixed-point, distribution, actual clean machine
and hosted CI remain separate. The closure-local source-admission status gap
(Go 3 versus Fairy package failure 7) is not a flat Option typed-source result;
it remains an explicit outstanding diagnostic-parity task.

No production source/cache schema, grammar, public flag, SDK/runtime ABI, DB,
dependency, repository policy, product bound/timeout or runner change. No push,
dispatch, release, deployment or publication.
