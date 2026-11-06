# Focused Linux native replay collector, 2026-11-07

`tools/collect-linux-native-validation.py` plans existing Go test roots and,
only with `--execute` on Linux amd64, runs the selected groups sequentially.
It uses only Python's standard library. A plan invokes no subprocesses, Go,
compiler, package manager or CI. Every invocation requires a new named output
directory; existing evidence is never overwritten.

## Selection and replay

From the repository root, on Windows or Linux:

```text
python -B tools/collect-linux-native-validation.py --output .cache/linux-native-plan-20261107
```

The small default selects `candidatebuild` and `trap-runtime`. Repeated
`--group` options replace the defaults, deduplicate repeated group names and
run only the exact anchored roots below. There is no implicit `./...` run.

| Group | Package | Existing roots |
| --- | --- | --- |
| `candidatebuild` (default) | `internal/candidatebuild` | `TestLinuxPacketBuildAndPublication`, `TestLinuxPacketProcessArguments` |
| `trap-runtime` (default) | `internal/cruntime` | `TestLinuxTrapReceiptProgressNative` |
| `runtime` | `internal/cli` | `TestLinuxGNUCompanionRuntime`, `TestLinuxGNUCompanionRuntimeAssets`, `TestLinuxGNUCompanionNative` |
| `cli` (expensive, opt-in) | `internal/cli` | `TestLinuxFileCLI` |
| `fetch-native` | `internal/cli` | `TestLinuxRemoteObservedReadNative`, `TestLinuxRemoteOfflineNative`, `TestLinuxRemoteOnlineNative` |
| `fetch-host` | `internal/candidatehost` | `TestLinuxPackageDirectoryAdapterNative`, `TestLinuxPackageFileAdapterNative`, `TestLinuxClockAdapterNative`, `TestLinuxHTTPSWorkerRejectsUntrustedNative`, `TestLinuxHTTPSParentProcessNative`, `TestLinuxHTTPSParentRejectsUntrustedNative`, `TestLinuxHTTPSParentTLSDeadlineNative` |

The planner checks each root has exactly one declaration in the current package
test sources and records its owner path. Missing or renamed roots fail planning.
The selected roots need no `FAIRYC_*` environment opt-in. Inherited `FAIRYC_*`
helper modes and unrelated integration opt-ins are removed from child processes.

`TestLinuxFileCLI` builds the full Linux entry and reuses its low/optimized
images for workspace/cache/build/run/test/new/fetch and Atlas impact/MCP/
symbols/calls/pack/edit. `checkLinuxAtlasMCP` invokes
`verifyCandidateMCPCommandWithHelp`, which includes
`verifyCandidateMCPBrokenOutput`: a pipe with its reader closed must produce
exit 4 and the write-failure diagnostic. Atlas and closed-pipe replay therefore
live in the explicit `cli` group; the default does not claim to cover them.
This root currently selects GCC, not an invented GCC/Clang full-CLI matrix.

`TestLinuxFetchCommand` and the separate `TestLinuxAtlas*Command` roots are
Windows command probes; they are deliberately excluded from Linux native
selection. `checkLinuxFetch` inside `TestLinuxFileCLI` supplies actual Linux
fetch replay, including cached fetch and loopback untrusted-TLS rejection.
The optional fetch groups add the existing runtime/host tests without rebuilding
the full CLI. No public download acceptance or consecutive self-build proof is
claimed. The DNS-deadline test is not selected.

Example **future Linux execution**, after preparing the local toolchain and
dependency cache outside this collector:

```text
python3 -B tools/collect-linux-native-validation.py --execute --group candidatebuild --group trap-runtime --group runtime --group fetch-host --group fetch-native --group cli --timeout-seconds 3600 --output .cache/linux-native-replay-20261107
```

Use a fresh directory for each attempt. The per-group Go deadline defaults to
900 seconds, accepts 1..3600 seconds and has an outer timeout 30 seconds longer.
No Linux execution is authorized or performed by this document's creation.

## Output and failure handling

`plan.json` records the exact argv, cwd, test roots/owner paths, timeout and
allowlisted environment overrides. Execution additionally writes `receipt.json`
after each observation and separate raw `.stdout` / `.stderr` files for every
tool probe and selected group. Go stdout remains the original `go test -json`
stream, including test-output text and skip reasons. There is no source ZIP.

The child environment forces `GOTOOLCHAIN=local`, `GOPROXY=off`, `GOSUMDB=off`,
`GOWORK=off`, `GOENV=off`, empty `GOFLAGS`, Linux/amd64 and CGO enabled. A missing
Go toolchain or cached dependency fails locally rather than bootstrapping from
the network. No installation, source rewriting, push, dispatch, release or CI
change is performed. Existing tests build in their own temporary workspaces;
ordinary local Go cache writes still occur. Tests retain their existing local
socket/loopback behavior; this is not an OS network sandbox.

Version probes record Git SHA/dirty state, Go version/target settings, GCC,
Clang and libcurl's pkg-config version, plus Python/host identity. No raw
environment dump or credential values are serialized. Probe failures remain
failures in the receipt and prevent `complete`, even if selected tests pass.
Required tools, compiler development headers/libraries and dependencies must
already exist; the collector does not repair missing prerequisites.

Each command records start time, duration, exact argv, original subprocess exit
code and effective collector exit code. A timeout uses effective code 124;
launch failure uses 127; interruption uses 130. Original signal return codes
remain intact. The first ordinary nonzero command/probe code is returned, while
later groups still run. Interruption stops selection and marks remaining groups
`not-run`. `SIGTERM` is routed through the same interruption path as Ctrl-C and
the previous handler is restored. Each child starts a fresh session; completion,
timeout and interruption kill the owned process group and reap the direct child.
Processes that deliberately detach into another session are outside that group.

Root `skip`, missing terminal events, failed roots and a passing root with skipped
descendants (`partial`) are distinct observations. Invalid JSON/field shapes
and duplicate root terminal events make the evidence incomplete without losing
the final receipt. If commands exit zero but roots skip or are missing, the
collector exits 3. `complete` only covers the explicitly selected roots with no
skips or parser failures and successful metadata probes; it is not completion
of the implementation queue or a hosted validation receipt.

Existing tooling was inspected first: `prepare-candidate-linux-cli.py` selects
an isolated workspace, `package-candidate-linux-runtime.py` emits runtime assets,
and the native/fixed-point verifiers operate on separately prepared compiler
artifacts. None collects this bounded set of current Go roots.
`.github/scripts/linux_phase2_evidence.py` consumes an existing Go JSON stream
but requires its own fixed ten-root/four-profile Phase 2 set and receipt schema.
Reusing that completion predicate here would falsely require unrelated roots;
it remains unchanged. This helper retains raw JSON and only classifies the
explicitly selected roots, without introducing a new product contract.

## Windows validation

```text
python -B -m unittest discover -s tools -p test_collect_linux_native_validation.py -v
```

17 portable tests pass on Windows / Python 3.14.6 (final run: 1.161s). They cover current
root discovery, bounded/default selection, no-subprocess planning, Windows
execution refusal, offline child settings, fake argv/log/timing/exit recording,
skip/partial/missing/failure handling, malformed event shapes and retained final
receipts, interruption, fake SIGTERM cleanup/handler restoration, and fake timeout
cleanup. All subprocess execution is fake in these tests. **This is not a Linux
native pass.** Actual Linux execution and process-group behavior remain pending
the authorized Linux batch. No broad Go suite, native matrix or CI was rerun.

The actual default-plan command above also passes, using the fresh local output
`.cache/linux-native-plan-sidecar-20261107/plan.json`; it creates only that plan
and invokes no tools. Per-file `git diff --no-index --check -- NUL <path>` passes
for all three added files (the normal no-index difference exit 1 is accepted;
there are no whitespace diagnostics).

The sidecar adds only the collector, its portable unit tests and this note.
The coordinator reviews/integrates them and links this note in the shared work
queue. Product API/DB/schema, source index/lock, existing CI runners and product
contracts are unchanged. No remote operation is performed.
