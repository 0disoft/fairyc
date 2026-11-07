# Bounded Windows compiler-host C compilation, 2026-11-07

Private compiler-host builds compile at most two literal/body units concurrently;
one available CPU/GOMAXPROCS selects one worker. The runtime's separate 8 MiB
source always compiles alone. Ordinary target builds retain sequential
compilation. Header/body/source, command deadline, flags, COFF inspection,
image-only ownership, deterministic link order and single-link policy remain.

The coordinator owns scheduling and errors. Each worker writes its own files,
waits for its own command, inspects one COFF object and releases its bytes.
Failure stops new dispatch, cancels later active jobs and waits for earlier
source indices to finish under their existing deadlines, preserving source-order
errors. Parent cancellation reaches all owned commands; cleanup follows Wait.
No global process-name termination is used. Two workers bound concurrency and
source admission, not compiler RSS; no physical-memory measurement is claimed.

## Executed checks

Evidence and exact hashes/commands are retained in
`.cache/windows-project-parallel-20261107/worker-freeze.json`, `native.log` and
`bounds.log`. Windows tests pass for worker count, concurrency peak, ordered
failure, cancellation, real owned parent/child termination and handle cleanup.
Native serial and parallel builds produce exactly equal PE bytes in both
profiles, both executing with status 42:

| Profile | Bytes | SHA-256 |
| --- | ---: | --- |
| low | 107520 | `e2da14933292ad2923bc6768cbf37159dd6f8a558dbf3935298eb42cf612dea9` |
| optimized | 107520 | `9180a3de06337de60a0d0f65504f93893d24e735f5b7e3c670a50fff5fa8f921` |

Existing source bounds, scoped vet, formatting and diff checks pass. `-race`
was not executed because the repository Go wrapper sets CGO_ENABLED=0. Other
native work shared this notebook; these results do not establish a throughput
improvement or a full CLI self-build pass. No public API/SDK ABI, DB, dependency,
runner, CI, FY source or source snapshot changed. Native Linux remains deferred.

## Bootstrap observation

The preceding authenticated low seed consumed 900 seconds on the current
491-source/3732-import compiler-only input without producing a packet. Its
Python timeout is retained at `.cache/current-compiler-seed-imports-20261107.log`;
the owned frontend terminated and no native build/pass is counted. A fresh
optimized Go-emitted full CLI seed is the next step. Product deadlines are
unchanged; this patch speeds the subsequent native C compilation stage.
