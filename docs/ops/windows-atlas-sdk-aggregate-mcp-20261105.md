# Windows mixed SDK/user aggregate read-only MCP, 2026-11-05

The actual Fairy `experimental atlas-mcp` session now has Windows low/optimized
acceptance for the mixed source defined by the impact command fixture: SDK
Vec<i32>, a same-name client Vec<T>, and a client Flag<bool> with nonempty
cleanup. Independent Go v3 preparation retains 127 resolved calls. The real
stdio command freshly derives its own checked source graph with PATH empty.

The shared SDK MCP verifier checks initialize/notification/tools-list protocol,
the single read-only tool and no destructive hint, then queries the SDK at
subject. Its structured impact result, original subject, current contexts,
traversal and omission facts match Go while retaining the actual image digest.
Changing root bytes produces an error tool result without changing files or
terminating the session. After independent lock/source refresh, the same session
returns the new Go-matching context. Every query preserves the root, dependency
and lock bytes it received. Responses contain neither local root nor source
contents, stdout contains protocol frames only, and EOF closes the process with
empty trailing output/stderr and success. The command owns no write capability.

```powershell
go test -mod=readonly -count=1 -failfast -timeout 12m -v ./internal/cli -run '^TestCandidateMCPSDKUserAggregateCommand$'
go vet -mod=readonly ./internal/cli
```

The native command passes in 221.177s: low 143.99s and optimized 64.00s. Neither
profile is skipped. Scoped vet, Go formatting and diff checks pass. The log is
`.cache/atlas-sdk-aggregate-mcp-20261105.log`; shared scoped vet is recorded in
`.cache/atlas-sdk-aggregate-commands-vet-20261105.log`. Other command probes can
run concurrently, so these acceptance durations are not isolated benchmarks.

The mixed MCP verifier is registered in full authored-main acceptance; that
complete replay is still pending. Mixed body-edit acceptance remains separate.
Product source is unchanged from `2b1a2d2a`; its source/lock and producer evidence
is retained. More Atlas source forms, final Go/Fairy comparisons, consecutive
self-build and Windows distribution/clean-machine acceptance remain open.
Native Linux stays batched; hosted same-commit CI awaits authorized push/dispatch.
No grammar, public CLI or protocol shape, SDK/runtime ABI, map wire, DB,
dependency, timeout, target, repository hygiene or CI runner change. No push,
dispatch, release or deployment.
