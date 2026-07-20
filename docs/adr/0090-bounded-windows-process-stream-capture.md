# ADR 0090: Bounded Windows process stream capture

- Status: Accepted
- Date: 2026-07-20
- Scope: local Windows MSVC execution driver

## Context

The Windows driver previously attached stdout and stderr to one bounded writer because every host,
target, layout, ABI, and unwind product was required to emit zero bytes. That is sufficient for an
empty-output assertion but destroys stream identity. Adding console output on top of that shape
would force the CLI either to misroute bytes or to pretend cross-stream ordering could be recovered
after execution.

## Decision

1. `WindowsHostOutcome` carries distinct `Stdout` and `Stderr` byte strings plus the exact signed
   `i32` exit bits.
2. Both process pipes write through synchronized stream writers into one shared 64 KiB budget. The
   driver fails the execution if their combined captured bytes exceed that ceiling.
3. Stream identity is preserved. Ordering between stdout and stderr is deliberately unspecified
   because the operating-system pipes are drained independently and do not expose one total write
   order.
4. Existing host, target, layout, ABI, and unwind paths still require both streams to be empty. This
   change supplies an honest transport boundary only; it does not expose source-visible console I/O
   or change a CLI receipt.
5. Unit tests cover exact shared-budget admission and truncation after one stream consumes most of
   the budget. The actual Windows smoke continues to require zero bytes from both streams.
6. No WSL or Linux compatibility layer participates.

## Consequences

- A future console API can decide explicitly how the CLI forwards stdout and stderr instead of
  inheriting an irreversible combined buffer.
- Cross-stream ordering requires a separate event protocol or pseudo-terminal decision and cannot
  be inferred from these captured byte sequences.
- The memory ceiling remains shared rather than silently doubling to one limit per stream.

## References

- `internal/cdriver/windows_host.go`
- `internal/cdriver/windows_host_test.go`
- `docs/architecture/02-runtime-flow.md`
- `docs/cli/output-and-exit-codes.md`
