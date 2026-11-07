# Same-module inline calls: full CLI/cache acceptance (2026-11-07)

Status: **PASS**. Go fixtures passed (21.058s). Full authored Windows CLI/cache
acceptance passed both host profiles (1599.238s) after coordinator authorization
following `.cache/inline-calls-artifact-final.log` (162.401s). This receipt records
actual full CLI execution; prior source/cache/artifact receipts remain separate.
Source scope is frozen per profile below and excludes the later unrelated
grouped integration into current main.

Scope: [ADR 0548](../adr/0548-inline-same-module-call-graphs.md). This sidecar
owns only `internal/cli/candidate_inline_calls_cli_cache_test.go` and this receipt.
No production, existing test, queue, ADR, lockfile, staging or commit changes.

## Authored path and independence

`TestCandidateInlineCallsCLICache` uses the existing `copiedClosureCLIHost`
private compiler-host builder. It copies the complete authored `file_cli` main;
each command launches a new executable process. The Go companion consumes the
unchanged freshly emitted Fairy C packet. It supplies no Go frontend or AST.
The existing capture proxy records that packet before forwarding it unchanged.
The existing private header/stack bounds, ordinary product policy limits and
strict compiler warnings remain intact. Both low and optimized host profiles
run sequentially with `GOMAXPROCS=1`, one host preparation per profile.

The helper authenticates any reusable host with both generated C-source and
executable SHA-256 digests in `host.json`. Source edits invalidate reuse. Native
workspace sources, cache generations and captured packets are retained under
the attempt directory, including failed attempts. Linux remains batched.

`TestCandidateInlineCallsCLIFixtures` runs independently first. It checks Go
execution, interpreter tests, source refusal, exact trap provenance and fixed
canonical markers without building the large authored Fairy compiler host.
The native suite additionally gets independent Go trap reports for interpreter,
development native and release native execution before building either host.

## Fixtures and assertions

The provider's public nongeneric `@inline value` calls later-declared public
`@inline leaf` both in `first=leaf(48,4)` and `return leaf(first,2)`. Positional
subtraction makes reversed arguments observable. The baseline is 42; changing
only `leaf` from `a-b` to `a-b+1` returns 44 while the outer text and marker stay
fixed. An alternative marked target returns 40. An unused inline export and an
unrelated private ordinary function are edited separately. All inline
declarations remain nongeneric. Only dependent callers specialize `evaluate`
with i32 and bool markers. The ordinary/generic tests additionally check ordered,
once-only actual argument evaluation and legacy v2 immutable locals.

Each host and caller family covers:

- Cold check, locked byte-stable hits, development/release build and direct
  executable execution, development/release run, test listing, interpreter
  tests, native development tests and native release tests.
- Fresh processes after leaf-only, unused inline export, private ordinary and
  marked-target edits. Stale locks refuse before execution; unlocked compilation
  refreshes generations; subsequent locked processes preserve names/bytes/lock.
  The private edit requires mixed unit reuse rather than a whole-project hit.
- Exactly one newly selected whole-project generation is corrupted after the target
  edit; repair must recover its exact original bytes and preserve other records.
  Identify its first framed member as `FYCKEYS0`; unit records share the same
  `fyproject0-` filenames and `FYCACHE0` envelope but carry `FYPUNIT0` instead.
  Valid surviving units may heal the project envelope via `cache: units hit`;
  the corrupt project itself must never produce `cache: hit`.
- Annotation removal, retargeting to an ordinary function and public argument
  labels refuse without changing good project cache bytes. Restoring valid
  source must hit the preserved cache. Both Go and Fairy source refusals return
  exit 3; Fairy emits exact `cache: cold uncached` and the outer declaration's
  unsupported-source diagnostic. Restored good source refreshes the lock through
  an unlocked project hit before subsequent locked stability checks.
- Captured packets contain the two reachable inline prototypes/definitions in
  the shared header, none in body units. Unused inline exports stay unreachable.

Unused initializer traps exercise `outer` and `reverse`, whose initializer order
differs: division runs before overflow, or overflow runs before division.
Ordinary and generic callers compare exact Go/Fairy diagnostics at provider
line 2 column 47 and line 3 column 43. Cold and warm/release processes must
preserve the same first trap and warm cache bytes. All three test engines must
match independent Go JSON reports, including the passing sentinel after traps.
Their packets contain three reachable inline functions in the shared header.

No parser depth change is made. The independent identity owner covers 63 nested
calls / 64 expression nodes. The 64th nested call is refused earlier by the
existing Fairy parser depth status 4 and is not claimed as inline admission.
Public argument labels remain forbidden.

## Reproducible commands

From the repository root, first:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineCallsCLIFixtures$' -count=1 -v -timeout 5m
```

After coordinator authorization and fixture success:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_CLOSURE_CLI_ARTIFACTS=Join-Path (Get-Location).Path '.cache/inline-calls-cli-20261107/hosts'
$env:FAIRYC_INLINE_CALLS_CLI_ATTEMPT=Join-Path (Get-Location).Path '.cache/inline-calls-cli-20261107/attempt3'
go test -mod=readonly ./internal/cli -run '^TestCandidateInlineCallsCLICache$' -count=1 -v -failfast -timeout 30m
```

The test explicitly clears the shared host profile selector to retain both
profiles. Record each new log and attempt path instead of overwriting failures;
host files alone may be reused after exact source/image authentication.

## Fingerprints and execution record

Baseline provider SHA-256:
`9f1a435f4ea611cc5e454ef9313d3ad557e5978bba1d39f9c2523eff93857224`.

| Body | Fixed marker suffix (after `fyinlinefn1-sha256-`) |
| --- | --- |
| value, v3 | `9b8978c9825c3bcc925ddeb7fe1c1b7f023aad5cc2756916b82a7e1061c887af` |
| leaf, v1 | `e1c7867e72719b32aaf40f42cbe5afec989b512e5f88e14cd312525177e2ebe2` |
| legacy, v2 | `6aac2a75fa26963e634daa3970fe133abdd3cd7e440ae96e9a12cefd94231aaf` |

Every generated main/test/provider source, edited provider and captured native
packet is fingerprinted in test logs. Source and marker fingerprints are kept
separate: a leaf-only edit intentionally preserves the outer marker.

| Source | SHA-256 |
| --- | --- |
| Ordinary main | `df0cd8e3b494c209e8ff61256f43963efbbe498c077a4ce4b7092e6ca87dda4a` |
| Ordinary tests | `268806d61e2bd0a07483fcaa401c3ac5f2b700bd8c33ad78f1e64e55a75ba927` |
| Generic main | `cbeceab0d6f1236531409b7ff341d769a8dace69c3d68411a1456415a3df606e` |
| Generic tests | `b0431b60278ae5dda896a3ac296ac282d844d3e968177be36930ac5cc9053f0d` |
| Leaf-only provider edit | `10066173f96f60e1b3fdce09a342df9d70cb9c6142e1c0c6e66da86372ce3757` |
| Unused inline export edit | `b896d005c94cec1f697109069bdcbf9d1b7b7bebc987932b7cf79f535148f059` |
| Private ordinary edit | `101adcc756420b1e0b4f98614931982f3adba8a577ffb8d7273d8176bb28f965` |
| Inline target edit | `28b642d061cfd21c87ec96fcb5c1a9101e972fe930318d9ad3e98bd50d9dad49` |

The leaf-only marker changes to
`fyinlinefn1-sha256-a1b1c2330ff85d874d52111e7f14a602ca89d5752415684be05ee4da60102d83`.
Changing the target changes the outer marker to
`fyinlinefn1-sha256-d921e16adfc667940b8ed167f42dd2228d01ba5920cc7701c9fedc8496d8c029`.

Initial fixture attempt failed only because the handwritten trap column expected
44 instead of the actual expression start at column 47. Corrected the fixture's
literal column expectations to 47/43, without changing compiler behavior.
Fixture retry log: `.cache/inline-calls-cli-20261107/fixtures-attempt2.log`.

Fixture retry: **PASS**, test 20.90s, ordinary 10.83s, generic 10.06s, package
21.058s. Actual values 42/44/40, v1/v2/v3 fixed markers, all three source
refusals, unused-initializer order and exact provider trap locations passed.

Native attempt 1 failed a harness assumption after low ordinary check/build/run,
all test engines and all four edit/warm-hit cases passed. The test selected new
files by filename alone, accidentally counting per-unit records as projects.
Stopped that run after the assertion to avoid repeating its failure across the
remaining matrix. Its log, host receipt and workspaces remain under
`.cache/inline-calls-cli-20261107/{native-attempt1.log,attempt1,hosts}`.
No production failure was observed. Corrected selection using the existing
framed member magic and allowed valid unit reuse specifically during repair.

Native attempt 2 authenticated/reused the low source/image digests, then was
stopped early after a separate retained low-host refusal probe established the
correct exit/status. The initial test had copied locals' package-input refusal
(7/skipped), but these inline-call refusals reach ordinary source diagnostics
(3/cold uncached). All three direct probes returned the identical outer source
span `[bytes 15..22]` at line 2 column 1. The prototype was behaving correctly;
the test expectations were corrected before spending time on the full replay.
Attempt 2 log/workspace and the final probe workspace remain under the cache
tree; the three probe outputs are also retained in the execution transcript.

Native attempt 3 **passed**, with exit 0 and package duration **1599.238s**
(test 1599.03s), inside the 30-minute harness deadline. Parent Go PID 5636 started
at 23:27:38 KST; output session 88478 completed normally before 23:55:13. No live
owned native test remains. Low passed 1205.81s; optimized passed 364.84s.
Exact-digest host reuse, fail-fast on a new assertion failure and separate
retained workspaces remained enabled.
Log: `.cache/inline-calls-cli-20261107/native-attempt3.log`.

| Host | Ordinary CLI/cache | Generic CLI/cache | Traps | Fresh CLI processes | Direct built executions |
| --- | ---: | ---: | ---: | ---: | ---: |
| low | 467.93s | 506.69s | 197.28s | 108 | 4 |
| optimized | 35.94s | 34.92s | 41.56s | 108 | 4 |

Each profile's ordinary and generic caller groups passed 45 CLI processes each;
ordinary and generic trap families passed nine each. Combined: **216 fresh
authored CLI processes and eight direct standalone executions**, plus independent
Go references. The two profiles passed twelve cold/warm trap pairs and twelve
exact Go-matching test reports. Selected
whole-project corruption healed through **`cache: units hit`**, restoring the
exact snapshot of all good project/unit generation names and bytes. Annotation
removal, ordinary-target retarget and public labels each returned source exit 3
without changing that snapshot; restored source hit again and subsequent locked
check/release-run pairs preserved the lock and cache. No production failure or
policy limit change was needed. The only failures were the documented harness
assumptions in prior attempts; neither passed profile was rerun after success.

Both host receipts independently checked against the executable bytes:

| Profile | Source SHA-256 | Image SHA-256 | Image bytes |
| --- | --- | --- | ---: |
| low | `4e48948505f992475d8634aae1393987a260da3187969c341efb96ec3d5ced0c` | `553679d4cd79d0489eb5a47018ce80efb9c8aed7035f5476304a468b51068bd5` | 16,225,280 |
| optimized | `01a57df261288e9d7de7a81681f24b3f224d7cf73667f66b5ac524a70776baf9` | `9279c67798299fc0e98bc1d2c908aba0a2d37a835459637c56797e7d38c6fa96` | 6,102,528 |

## Per-profile source scope after concurrent integration

The coordinator integrated unrelated grouped static-operation Fairy sources
while this suite ran. Attribute each frozen host separately; neither this
receipt nor the low pass certifies the complete current grouped integration.
Do not infer source inclusion from the later native compilation timestamp.
No rebuild is justified solely by that unrelated change; complete current-source
comparisons remain later. Main eventually reached integration commit `7a49b76d`.

- Low: original copy directory created at **23:18:21 KST**. Attempt 3 regenerated
  the host C and authenticated the same source/image digests above. Its pass
  belongs to this pre-group snapshot, with the inline implementation later
  committed as `c5b6c9b4`; it is not current grouped-source coverage.
- Optimized: actual helper source copy created at **23:48:44 KST**; native image
  published at **23:52:52 KST**. Inspection of that copy found the old
  `static_operation_declarations.fy`, `static_operation_identity.fy` and
  `generic_source_catalog.fy`, whose individual Git blob IDs match `c5b6c9b4`.
  Both new grouped modules are absent. The five inline owners match current main
  byte-for-byte. This also excludes the later grouped integration; no claim is
  made that every host source equals an entire named Git tree.

Both source directories and matching per-file `scope.json` manifests are retained
under `.cache/inline-calls-cli-20261107/host-sources/{low,optimized}/`. The eight
present owner files below match between these two retained copies. The absent
files are `static_operation_groups.fy` and `static_operation_group_identity.fy`.

| Frozen source | SHA-256 |
| --- | --- |
| static_operation_declarations.fy | `5c4f54d4a0f87b94c94253d394a4e3a9869e527ce1a0ee6cdb6bd6904cd8dde8` |
| static_operation_identity.fy | `2d1e87e0f2415dd7d9f58151633f95064200d92a82164f8e5ca402a78d6bc93e` |
| generic_source_catalog.fy | `b7b5d90d940b97cd05e6dc0ec126197cdcce83111b93776e84bbf5fccbc70ca0` |
| module_inline_declarations.fy | `13a5c5a55a2dfdfeefb068534e2a02bf8395b3957fb00358d09ecab8e5feff7f` |
| module_inline_identity.fy | `576750911854d8787731ee002908f0fe1f9ba471e42d9f7aa61125e636fd3213` |
| module_inline_ir.fy | `9136f83d0ea3454667f978f51bf1b73bfeb3480ec10aa5c197832a598e3c076c` |
| module_inline_graph.fy | `aaacca8510af0f6ed6bb93ec19c2389d1cff3f863701123d571d754119d02d1a` |
| module_unit_body_ir_verify.fy | `492a41dc9fb2a2dde27fedded9ce46764626634ef63528f41ec7189a3b50f6a3` |

Full repository suite, Linux, hosted CI,
distribution refresh, release, push and deployment are outside this sidecar.
No public API/ABI, database, dependency, runner or repository hygiene changes.
No source-of-truth contract document is changed.

Scoped final static validation on integrated main `7a49b76d`:
`GOMAXPROCS=1 go vet -mod=readonly ./internal/cli` **passed**, exit 0.
This vets the CLI package and tests without rerunning native acceptance or
claiming that the frozen hosts include current grouped sources.

Final hygiene: only the two assigned new files are untracked. Go file is
gofmt-clean; both files are UTF-8/LF with final newlines and below 40KB. No other
test, source, queue, ADR, lockfile, staging, commit or remote action was performed
by this sidecar. The coordinator owns any eventual commit.
