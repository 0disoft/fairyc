# Go flat Result transport, 2026-11-07

Implements the Go compiler/backend slice of
[ADR 0550](../adr/0550-flat-result-success-generic-transport.md) in the isolated
detached checkout `.cache/flat-result-20261107`, baseline
`bf719aa872375ac22706c20cb232db42efb28700`.

## Bounded implementation

The only production changes are:

- `internal/compiler/generic_admission.go`: admit a shallow root
  `Result<S,u32>` after scalar substitution, where S is a direct named `i8`,
  `i16`, `isize`, `u16`, `u64`, `f32` or `f64`. This also admits `Result<T,E>`
  when the concrete bindings give S and exactly u32. Other error families,
  nested wrappers and composite generic arguments keep their existing rules.
- `internal/cbackend/tagged_types.go`: recognize that exact Result root and
  resolve its success leaf's scalar layout locally. Error-payload and shared
  aggregate predicates are unchanged. The existing tag/success/error fields,
  active-member handling and ordinary call/result/reassignment machinery are
  reused. Previously retained flat candidates do not authorize nested roots.

No shared scalar/aggregate admission predicate, MIR opcode, codec version,
public-interface encoding, source identity recipe, CLI option, foreign ABI,
allocation/boxing, dependency, runner or global bound is changed. No Fairy,
queue, lock, ADR or main-root file is edited by the Go owner; the coordinator's
concurrent Fairy files remain outside this owner's write scope. No commit or
remote action is made by this owner.

## Executed focused checks

All checks ran in this checkout with `GOMAXPROCS=1`:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 ./internal/compiler ./internal/cbackend -run '^(TestComposite(Result|Option)|TestRestrictGenericSignatures|TestGenericSignatureAdmission|TestGenericSignaturesAdmit|TestEmitScalarOptionAndResult$|TestEmitCopyResultWithStructError$|TestEmitCopyResultWithBorrowedText$|TestEmitResultWithEnumPayloads$|TestEmitResourceBearingResultLayout$|TestTaggedCandidates)' -v
go vet -mod=readonly ./internal/compiler ./internal/cbackend
```

Final test result: **28 top-level tests passed** (18 compiler, 10 backend),
compiler **7.938s**, cbackend **0.060s**, no failed or skipped tests. Targeted vet
exited **0** without output. Owned production diff whitespace checks passed.

The new compiler coverage checks all seven scalar success leaves through:

- Private-module, explicit-file and package-aware admission, including ordinary
  named signatures, scalar generic substitution, and `Result<T,E>` specialized
  with E=u32. Written source ASTs remain unchanged.
- `.ok`/`.err`, ordinary and generic parameters/results, inert copies, explicit
  handling, reassignment, exhaustive extraction, and propagation.
- Each copied/evaluated Result's existing must-handle rule. Handling the source
  does not discharge the destination's pending evaluation. Handling a stored
  copy does not permit an unhandled later evaluation of the original. E4101
  keeps the exact original producer/evaluation span.
- Ordinary error-type mismatch E4103 at the original `?`, with actual-error u32
  and expected-error i32 related spans.
- Nine generic shape refusals per scalar: wrong error family, error=S, both
  payloads=S, Option/Result nesting, reference, pointer and array wrappers.
  The original generic signature spelling and both refusal spans are retained;
  failed preparation does not mutate source ASTs. Composite generic type
  arguments remain refused for each of the seven scalars.
- Ordinary nested/wrong-error wrappers retain semantic admission where it
  existed and fail both target emitters. Result struct-field storage remains
  refused by the backend; defer capture retains its source-level refusal/span.

The new backend test independently builds expected tag/success/u32 layout fields
and compares sizes, alignments and offsets. It tests allowed-root-first and
allowed-root-last orders, nested/wrong-error refusal, unchanged shared payload
and aggregate layout predicates, and continued foreign value/pointer/reference
refusal. Existing flat Option, old scalar/struct-error/borrowed-text/enum/resource
Result, compact deep-tagged-graph and semantic-report-budget tests also pass.

## Actual Windows native receipts

MSVC **14.51.36231** compiled and executed C11/C17 consumers at both low and
optimized profiles. All **12** final native processes returned **42** with empty
stdout and stderr, with no skips:

| Input | MIR functions | MSVC C bytes | C11 low image | C11 optimized image | C17 low image | C17 optimized image |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Result source | 76 | 133079 | 149504 | 117760 | 149504 | 117760 |
| Result verified-MIR encode/decode restoration | 76 | 133079 | 149504 | 117760 | 149504 | 117760 |
| Existing flat Option regression | 47 | 74046 | 128512 | 112128 | 128512 | 112128 |

The Result program uses independently authored C record seeds and observers for
signed minima (`i8`, `i16`, `isize`), unsigned maxima (`u16`, `u64`), ordinary
nonintegral f32/f64 values, and both signs of floating zero: eleven observations
cases across all seven scalar types. No new scalar source literal or arithmetic
is assumed, and Result itself never crosses a foreign ABI.

Each native Result process asserts **33** active success-payload bit comparisons,
**22** operand marks and **22** seed calls. Each case observes the original,
ordinary/generic copied/reassigned value and successful propagated value. It
also checks propagated error 77 without reading an inactive success payload.
The marks/seeds detect duplicated operand execution and an incorrectly executed
success branch after propagation failure. The copy/ordinary/generic paths and
both error alternatives must all satisfy existing must-handle checking.

The restored path uses the existing verified MIR codec with an explicit 16MiB
caller byte limit. It reruns MIR verification during decoding before both target
emissions and native execution. Both source and restored executions compare
against the same authored C constants/counters; neither run's output supplies
an oracle for the other. This is MIR codec restoration, not a claim of stable
per-package artifact, disk-cache or full authored CLI acceptance.

## Initial failed attempt

The first source test pass admitted and emitted all seven allowed payloads, and
the backend boundary test passed. That command nevertheless exited 1 because two
new fixture assumptions were wrong:

- Result storage transfers the obligation to the destination, and its diagnostic
  producer is the initializer evaluation. The first test incorrectly expected
  the new binding's name and an independently pending original without a new
  evaluation. The corrected cases assert the existing transfer/evaluation
  contract and exact producer offsets.
- Matching an inner Result into `_` still creates a must-handle obligation. The
  initial nested-wrapper fixture therefore failed Flow before reaching its
  intended emitter refusal. It now names and explicitly discards the inner
  Result, so the test reaches the backend boundary it owns.

No production Flow rule or limit was changed to resolve these failures. A later
Result-only command passed (compiler 3.998s, backend 0.052s); the final combined
command above includes source/restored native evidence and old Option/Result
regressions.

## Remaining gates

No full Go suite, Fairy-authored acceptance, whole CLI/cache matrix, stable
package-artifact restoration, self-build/fixed-point, hosted CI or Linux native
execution was run by this owner. Both target emitters ran, but Linux execution
remains the separate batched gate. Shared-graph rollback and Fairy metadata
acceptance belong to the coordinator's concurrent implementation. No Go-owner
failure remains; all owned validation sessions have completed.
