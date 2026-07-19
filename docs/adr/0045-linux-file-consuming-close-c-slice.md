# Linux File Consuming-Close C Slice

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-19
- Contract source: `docs/runtime/runtime-stdlib-v0.md`

## Decision

The provisional Linux File helper validates a bounded NUL-terminated path and closed read/write,
create, truncate, and mode options. `read` and `write` expose actual partial counts and retry only
`EINTR`; `write_all` reports bytes completed on failure and rejects zero-progress success. `sync`
uses `fsync`.

`close` consumes the descriptor record before invoking the kernel exactly once. Success and failure
therefore both leave the File closed and no automatic retry is possible. Drop performs the same
single best-effort close without allocation, output, or error reporting.

The native probe writes, syncs, closes, reopens, reads in three-byte partial chunks, verifies EOF,
then forces `EBADF` and proves failed close still consumes the File. `fairyc.native-c-report/v11`
adds four `linuxFileExecutions`. Hosted evidence remains absent. ADR 0063 later adds a separate
generated-Fairy source crossing for the successful write/sync/close/reopen/partial-read/EOF/close
lifecycle without replacing this C-only failure probe.

## Consequences

- This is an `x86_64-linux-gnu` contract, not portable POSIX File semantics.
- Durability-sensitive callers must sync before consuming close.
- Safe source effects, ownership-aware open-option types, and error resources remain to be connected.
