# Native graph framing prerequisites, 2026-11-07

Two disjoint Sol high workers implement ADR 0530. The parent reviews and
commits their detached framing plan and explicit C-frame bundle together.

The plan command `go test -mod=readonly ./internal/cli -run
'^TestCandidateStateGraphTransportPlan' -count=1 -v -timeout=12m` passes in
**20.365s** (wall **35.975s**): **22** independent Go Program cases, **22** native
cases per Windows low/optimized profile and **six** ownership refusal cases.
Independent arbitrary-precision bottom-up Go expectations cover shared/repeated
helpers, both branch arms, depth 32, saturation, sparse IDs, literal spans,
zero/one/1024 fields, map exclusion and all codec mappings. Exact/short/overdrawn
work and immutable ownership are exercised. Two initial harness failures are
retained and distinguished in `.cache/graph-transport-plan-review.log`.

`go test ./internal/cli -run '^TestCandidateStateGraphFrameAssets' -count=1 -v
-timeout 10m` passes in **10.564s**. Windows low/optimized compare all **8,260**
frame bytes against Go parser output and the public GraphTransport.Source
suffix. The **8,284**-byte distribution pack has SHA256
`320ac1faf4137b3915ebc7c4c48505b55ed56b149a0d0ba2d86df595cc761335`.
Explicit pin drift, corrupt/truncated/oversized input, work, lifetime and private
owner controls pass. Initial failure and corrected run remain in
`.cache/graph-frame-assets-windows*.log`; packaging and UTF-8/LF checks are
recorded separately there. Only `--update-pin` authorizes regeneration.

These tests do not attest request encoding, complete framed C or storage CAS;
their owning integration receipts supply that evidence. Complete source/lock
checks follow final transport integration. Linux, allocator-failure injection,
full Go suite and hosted CI are unexecuted. Public API/SDK/runtime ABI, DB,
dependencies, default limits, repository/runners and existing C are unchanged.
No push, dispatch, release or deployment.
