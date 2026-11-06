# Fairy graph runtime distribution inputs, 2026-11-07

The parallel Sol high runtime worker implements ADR 0527 in four disjoint new
files: the Python packager, generated pin, immutable Fairy Bundle and dedicated
Go/native tests. The parent reviews and commits these prerequisites separately.

`go test ./internal/cli -run '^TestCandidateStateGraphNativeAssets' -count=1 -v`
passes in **10.376s** on Windows. Both low/optimized compare complete declaration
and runtime bytes to the current Go cstdlib outputs, and compile/link/execute
graph copy/write/read/digest/commit/closed-state calls. The bundle is **37,346
bytes**. Pin drift, corrupt/truncated/oversized inputs, exact/one-short/overdrawn
work, output lifetime and private construction/mutation are exercised.

An initial probe-input bound failure is retained in
`.cache/graph-native-assets-test.log`; the corrected harness passes in
`.cache/graph-native-assets-test-retry.log`. A preparation failure is not a
product pass. The unchanged selectors, C sources and limits are not regenerated
or weakened. This is a runtime-input prerequisite, not compiled transaction or
persistent-CAS acceptance. Complete source/lock verification is recorded with
the final native-emission integration receipt.

Linux execution, allocator-failure injection, full Go suite and hosted CI remain
unexecuted. Public API/SDK ABI, DB, dependencies, repository/runners and product
timeouts are unchanged. No push, dispatch, release or deployment.
