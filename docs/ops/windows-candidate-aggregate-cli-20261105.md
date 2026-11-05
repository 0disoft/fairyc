# Fairy aggregate workspace commands, 2026-11-05

Scope: actual Fairy-authored Windows candidate `check/build/run/test`, including
explicit project/unit persistence. This does not promote compiler-private
aggregate syntax into Go's stable workspace admission or change language/SDK
versions. The scalar-argument aggregate implementation and its source/ownership
contracts remain owned by ADR 0387 and the aggregate implementation plan.

## Implementation

`project_compiler_source_mode` now supplies the shared current-source selector
for Atlas, editing and command compilation. Only generic struct/resource source
declarations select the aggregate route; the authenticated SDK Vec source keeps
its existing ordinary route. Neither cache framing nor a selected symbol can
choose compilation authority. Discovery work is bounded and retained.

`project_compiler_command` selects the existing complete checked aggregate or
ordinary factory. Check, emission and test commands call it only after the
existing loader, lock, coordinate and input gates. Failed cache initialization
uses the same source-selected cold path. Existing explicit ordinary factories
and decoders retain their closed admission.

Project cache lookup, restoration, mixed unit compilation and publication select
the matching aggregate/ordinary families. Test overlays use only package-unit
reuse and still rebuild their linked Complete owner; each test executes again
after reuse. Source diagnostics and disposable cache misses remain distinct.
The Windows candidate dispatcher now explicitly selects the existing cached
test parser, admitting the same private opt-in test-cache flags as its Linux
counterpart. This supersedes the candidate's non-cache-only Windows test
selection. Without those flags it follows the original uncached path. Go's
stable options, public help, cache defaults and result schemas do not change.

The interpreter's record admission, constructor storage and field projection now
include concrete kinds 20/21 only inside aggregate-verified IR. Existing immutable
heap cells, copy-on-write places, layouts and cleanup CFGs still own execution.
Tuple metadata kind 22 does not become an executable value.

## Acceptance

The full authored CLI test is `TestCandidateGenericAggregateCLI`, opt-in through
`FAIRYC_RUN_PARITY_INTEGRATION=1`. Its Go oracle uses current loaded sources,
package coordinates and `compiler.CheckGenericPackages`; it derives check counts,
test selection and interpreter JSON through the existing Go owners. Go's stable
workspace CLI deliberately still refuses these aggregate templates, so a stable
`check` failure is not used as a negative candidate oracle.

The three-package fixture constructs and moves resource instances, scalar-pair
records and generic factories/helper calls. It accesses a resource's private
payload only inside its owning provider module. A helper edit changes the
observable program result, rather than merely changing a discarded cleanup
result. The host retains authored main/dispatch; its Go sidecar consumes only
fresh generated C, with no frontend fallback. Runtime PATH is empty.

The final full-command roots pass in low (434.19s) and optimized (167.723s),
27 independent candidate CLI processes per profile. The ordinary SDK control
also passes both profiles in 143.793s (low 44.53s, optimized 86.10s), with
Go-exact counts/output and unchanged locked bytes. The matrix includes
uncached/cached commands, both program profiles, published image execution,
exact Go check/list/interpreter reports, test reruns on unit hits, stale source,
corrupt cache recovery, failed Complete preserving cache bytes and missing-cache
cold fallback. `TestCandidateAggregateCommandOrdinarySDK` separately checks that
an authenticated SDK Vec workspace still follows its original command path.

## Development failures

The first full-host run fails in 116.636s because the test incorrectly asks Go's
stable CLI to compile private aggregate syntax. A subsequent oracle replay fails
in 20.156s on the fixture's cross-module private resource-field access. The
fixture now exposes a provider-owned consumer and executes the private Go
compiler/interpreter oracle. The next replay fails in 30.547s because an ordinary
cache snapshot helper intentionally rejects aggregate envelope bytes; the
aggregate fixture now observes its actual flat cache directory instead. These
are harness failures, not successful command acceptance. The next 95.649s replay
passes check, cold/warm cache, both build/run profiles and listing, then reproduces
the missing aggregate interpreter admission with code 2. Record/field execution
is now connected; the strengthened fixture additionally checks mutation after a
value copy, record arrays, nested owning tags and an empty generic resource.
The next 170.165s run passes exact uncached Go interpreter JSON, then finds that
the Windows dispatcher still selects the non-cache-only test parser. That
private selection is now explicitly connected as described above.
An optimized run started before that selection fix reproduces the same parser
refusal in 348.156s; it is not counted as passing the final source. The final
aggregate host builder uses the existing 2 MiB partition target (24 parts),
with unchanged consumer/timeout limits; other callers keep their 256 KiB target.
The combined low/SDK command returns failure in 497.347s after its complete
aggregate root passes: the SDK control compares a locked candidate response
against an unlocked Go response. Both report the exact same 3 instances,
10 source files and 178 functions; only `lock verified` versus `lock updated`
differs. Its oracle now explicitly runs Go with `--locked`; the selected SDK
root is rerun independently. The unchanged aggregate root's two profile results
are retained rather than reexecuted after this SDK-only test correction.

Final commands:

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_PROFILE='low'
go test -mod=readonly -count=1 -timeout 20m -failfast -v ./internal/cli -run '^TestCandidate(GenericAggregateCLI|AggregateCommandOrdinarySDK)$'
$env:FAIRYC_CLOSURE_CLI_PROFILE='optimized'
go test -mod=readonly -count=1 -timeout 20m -failfast -v ./internal/cli -run '^TestCandidateGenericAggregateCLI$'
go test -mod=readonly -count=1 -timeout 12m -failfast -v ./internal/cli -run '^TestCandidateAggregateCommandOrdinarySDK$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_generic_aggregate_cli_test.go internal/cli/candidate_copied_closure_cli_host_test.go
git diff --check
```

The first command's complete aggregate low root passes; its later SDK-only
oracle failure is accounted above and closed by the third command. Both final
aggregate roots have 27 CLI invocations plus two separately executed published
development/release images per host profile. Scoped vet, formatting and diff
checks pass. Retained hosts reuse only matching generated-C/profile/image
digests; no old pre-dispatch binary attests the final source.

Native Linux, full current-source/self-build, SDK Atlas production,
distribution/clean-machine and same-commit hosted CI remain pending. No stable
API/SDK ABI, schema, DB, dependency, timeout/cache-default, runner or hygiene
change; no push, release or deployment.
