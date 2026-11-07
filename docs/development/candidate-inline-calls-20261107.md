# Fairy inline same-module calls: source admission, 2026-11-07

Owner: [ADR 0548](../adr/0548-inline-same-module-call-graphs.md). The Go
implementation and cache receipts are separate. This source checkpoint does
not establish full authored CLI/cache or final self-build acceptance.

## Actual source implementation

`module_inline_declarations` checks positional bare-name calls against actual
same-module declarations and rejects parameter/local shadowing. Every target
must have an admitted public safe nongeneric inline i32 signature and its body
must pass the same local/node limits. Calls in initializers, returns and nested
actuals produce the v3 canonical tuple; leaf v1 and no-call-local v2 identities
retain their bytes. Public labels stay closed under ADR 0396.

Each root owns a worklist of at most 64 distinct declarations and one 64-bit
adjacency mask per row on the existing x64 targets. Body checking is iterative
over that worklist. Leaf elimination rejects cycles in at most 64 passes / 4096
row checks. Repeated targets share one row; bodies are never substituted into
their callers. Graph depth no longer adds complete body-checker frames to the
native stack. Ordinary declarations without annotations do not allocate a graph.
Existing expression recursion, parser/work limits, stack policies and native
timeouts are unchanged.

## Execution and failed attempts

The actual source probe is `TestCandidateInlineCallsIdentity`. Go independently
checks each input and supplies every accepted declaration's full digest; a
literal v3 JSON tuple provides a separate fixed SHA-256 vector. The native Fairy
probe parses the same source and compares all eight hash words per declaration.
Both ordinary Windows compiler profiles execute the full matrix.

Final 23-case selection passes in **18.805s** (test 18.65s): forward/backward
declarations, locals, nested/repeated calls, a shared diamond, zero actuals,
flat 64/65 nodes, 63 nested calls plus one parameter node, 64/65 distinct
declarations and invalid target/signature/shadowing/cycle/public-label families.
One case deliberately observes the existing parser's status 4 at its 64th
nested call (`bodies.fy`, depth >= 64); it is a parser refusal, not credited as
inline admission or CLI diagnostic parity. The separate flat 65-node case does
reach and fail inline admission. Command:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateInlineCallsIdentity$'
```

Log: `.cache/inline-calls-identity-attempt8-20261107.log`. Optional
`FAIRYC_INLINE_CALLS_IDENTITY_SOURCE` retains the exact generated probe; normal
tests create no output outside their temporary directories.

The existing `TestCandidateInlineIdentity` and
`TestCandidateInlineLocalsIdentity` also pass both native profiles, package
**26.320s** (13.33s / 12.85s). This independently replays the no-call v1/v2
admission and canonical vectors after the new graph code. Log:
`.cache/inline-calls-legacy-identities-20261107.log`. Scoped
`go vet -mod=readonly ./internal/cli` passes with no diagnostics.

Earlier failures remain in the numbered attempt logs: an incorrect expectation
that public argument labels were supported; one oversized generated harness
main before assertions were split into bounded functions; actual stack overflow
from recursive whole-body graph checking, fixed by the iterative algorithm;
unsupported i32-to-usize casts in its first bitmask spelling, replaced with
typed usize operands; and two attempts incorrectly expecting the existing
parser to return a tree for the over-depth negative. The passing matrix
preserves all assertions and does not raise a product bound or stack limit.

## Full-source checkpoint

The final coordinator snapshot passes original `check` / `check --locked` in
**19.534s / 19.766s**: 756 checked source files, 9,563 MIR functions. All 1,227
authored Fairy files match the checked copy, and its lock is copied only after
that equality succeeds. The bootstrap SHA-256 is
`00be89383b0049a349b0bf8559ea2c81c86e8af85feddf349069001ce53982fa`.
Receipt: `.cache/inline-calls-source-20261107-all-check.json`; execution log:
`.cache/inline-calls-source-attempt3-20261107.log`.

The first in-progress source snapshot failed on unsupported mutable owning-field
borrows and an incomplete concurrent IR edit. No pass is claimed for it. A
second earlier snapshot passed before the iterative graph correction and is
not the final source evidence above.

No public API/ABI, DB, dependency, cache wire/key version, runner or target change.
Native Linux remains batched. Artifact/native-C results, complete CLI/cache,
full suite, current-source self-build/fixed-point, hosted CI and distribution
refresh are independently recorded gates. No push or publication.
