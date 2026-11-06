# Fairy Complete to genuine operation C, 2026-11-07

ADR 0529 completes the explicit Fairy operation-C emission unit. Three Sol high
workers had disjoint writes: runtime assets, structural plan and operations.
The parent implements bounded C writing, body/edge/terminator emission,
Complete-bound artifact publication, integration tests and these documents.
ADRs 0527/0528 are committed separately as verified prerequisites.

The independent current-Go source/HIR matrices supply the expected full C,
layout/body/program identities and sorted trap JSON. Both observer modes retain
exact bytes for empty exits, seven codecs, u32/u64 limits, binary/present-empty
bytes, Unicode, dynamic map keys/values, SSA edges, transitive helpers and
borrow/view/copy. Native tests compile/execute the returned Fairy buffer; they
never substitute Go-generated operation C for that buffer. Audited calls use
the existing graph test framing and independent Go MIR/reference persistence
expectations, rather than compiling a duplicate byte-identical Go image.

Restored Complete with a rebound owner emits identical C. Artifact output
survives destruction/mutation of caller input and destruction of Complete and
Bundle. Exact/one-short/overdrawn emission work, wrong target, bad runtime and
source refusals preserve their own failures and publish no artifact. Private
construction/mutation are checked separately.

```powershell
go test -mod=readonly -run '^TestCandidateStateGraphNative' -count=1 -timeout 8m -v ./internal/cli
go vet -mod=readonly ./internal/cli
```

The final combined command passes in **131.962s**, with no selected skips or
failures. Emission integration passes in **78.86s**: **172** Fairy processes per
profile / **344 total**, and **48** actual Fairy-produced C images execute
across both observer modes and profiles. Its **24** audited executions compare
with the independent MIR interpreter and reference-store bytes. Present binary
borrow/view/copy and dynamic Unicode map keys/values additionally succeed.
Both artifact-forgery controls pass; formatting, scoped vet and staged diff
checks pass. Final log: `.cache/graph-native-final.log`.

Logs retain four preparation failures: an unused Go import and
nonexistent test-only default-limits helper, an inferred numeric type, a
currently unsupported narrowing cast and overlapping argument loans. All are
corrected within the new code/harness; no language rule, warning or limit is
relaxed. The first complete integration passes in **95.542s**, including **336**
emission/refusal processes and paired C executions. The final harness eliminates
duplicate C compilation and strengthens independent semantic checks.

Full isolated current-source `check` then `check --locked` pass: **730 source
files / 9,112 MIR functions** in **13.542s / 12.346s**, empty stderr. All **1,200**
authored `.fy` files match the snapshot and current source. Only the successful
snapshot lock is copied back. The unchanged Go product bootstrap is reused at
SHA256 `4e392248a97b8351c8563c904910fcf88c648a9b4d1323be8cda45dc711eb084`;
snapshot/commands/timings: `.cache/graph-native-check-latest.json`.
The plan worker's CRLF working copy is normalized to the repository's LF
contract before the final lock refresh. Its Git content is unchanged; the
current-byte slot/owner replay passes again in **22.387s** in
`.cache/graph-native-plan-lf-final.log`. The working bytes and checked lock now
also match the repository's committed LF form.

This proves C emission and test-driven execution for the admitted family.
Fairy-authored native request/response transport, one-attempt persistent CAS,
ordinary workspace/CLI selection and the complete remaining queue are pending.
Linux native execution stays in the requested batch. Full Go suite, allocator
failure injection, self-build/distribution acceptance and hosted CI are
unexecuted. Public CLI/SDK/runtime ABI, DB, dependencies, repository/runners and
existing defaults are unchanged. No push, dispatch, release or deployment.
