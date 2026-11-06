# Authored state-graph combined operation limits, 2026-11-07

`candidate_state_graph_source_limits_test.go` checks the real source producer,
source-bound artifact roundtrip and portable body identity against the independent
Go producer. This supplements the authored-values regression; no production
source or limit changes were needed.

Each fixture has a root and two helpers in reverse canonical declaration order.
No individual function approaches the 1,024-operation bound. The complete program
contains exactly 1,023, 1,024 or 1,025 operations, including helper calls and reads.
The root appears both before and after the helper declarations. At exactly 1,024,
32 aliases in the root and 32 in the last canonical helper remain accepted:
aliases and exit terminators do not consume graph operations. An unknown alias
at that same bound remains semantic error 1001 rather than policy error 1002.

The test inspects actual Go graph counts for positive fixtures. The 1,025 cases
assert the actual Go operation-limit error with no semantic diagnostics; no Go
canonical body is claimed for these refusals. Native positive fixtures compare
exact canonical body bytes, original/restored/decoded identity and exact/one-short
work allowances through the shared authored probe. Native overflow must be 1002.

Executed by the GPT-6.1 Sol high boundary worker:

```powershell
go test -mod=readonly ./internal/cli -run '^TestCandidateStateGraphSourceLimits$' -count=1 -v
```

PASS in **26.969s**: 10 independent Go cases and **20 Windows native processes**
(10 low, 10 optimized), no failures or selected skips. The parent reviewed the
new test and reused the already passing current-source check/lock because no
production source changed. Scoped vet, formatting and diff checks are finalized
with this test-only commit. Linux execution remains batched. Ordinary Complete,
cache/commands/native persistence and final acceptance are separate gates.

No public API/SDK ABI, DB, dependency, runner or product policy changes. No push,
CI dispatch, release or deployment.
