# ADR 0372: Keep contract-state retries caller-visible

- Status: Accepted
- Date: 2026-09-01
- Product source: `docs/product/04-contract-profile-candidates.md`
- Kernel source: `docs/contract-state/transaction-kernel-v0.md`
- Fixture source: `spec/fixtures/contract-state-retry/manifest-v0.json`
- Implementation: `internal/contractstate/`, `internal/contractstateretryfixture/`

## Context

A stale transaction returns `ErrConflict`. Retrying inside `Execute` would evaluate the callback more
than once without the caller observing the extra work. That is unsafe groundwork for future events,
external calls, metering, tracing, and target-specific execution costs. A callback can also perform
ordinary host effects that the state overlay cannot roll back.

The kernel therefore needs an exact retry contract before persistence or an SDK exposes it.

## Decision

1. `Execute` performs exactly one callback evaluation and one commit attempt. It never retries
   `ErrConflict`, rejection, panic, verifier failure, limit failure, layout mismatch, or any other
   error.
2. A caller that chooses to retry must invoke `Execute` again. Every invocation receives a fresh
   transaction and captures the then-current committed layout and version. No transaction object,
   snapshot, overlay, receipt, or managed capability is reused.
3. Attempt identity is caller-owned and visible. The first call is attempt one; any maximum attempt
   count, exhaustion result, scheduling, cancellation, and backoff belong to the caller. The kernel
   adds no exhaustion error and performs no wall-clock sleep.
4. The complete callback runs again on each explicit attempt. Callers must not place non-replay-safe
   effects outside the transaction overlay in a retried callback. Future event emission, external
   calls, gas, and metering must account for each visible attempt before any higher-level retry helper
   can be admitted.
5. Add `fairyc.contract-state-retry-fixtures/v0` as a versioned caller-policy corpus. It proves
   success after one visible conflict and caller-owned attempt-budget exhaustion. Injected conflicts
   publish complete competing two-key overlays; failed attempts publish none of their own overlay.
6. The Go oracle and fixture runner remain experimental implementation evidence. This ADR does not
   create a stable Fairy source API, runtime ABI, persistence protocol, or SDK retry helper.

## Fresh-snapshot rule

An explicit retry is not a continuation of the failed attempt. It reads the state committed by the
winning transaction and recomputes the body from that state. The success fixture proves this by
observing value `10` after the injected winner and committing `11` on the second call. Reusing the
first attempt's computed value would produce `1` and fail the fixture.

## Exhaustion rule

The kernel reports only the final result of each individual `Execute` call. In the exhaustion fixture,
the caller stops after two `ErrConflict` results and reports `exhausted` as its own policy result. The
store contains only the two complete competing commits and no bytes from either losing overlay.

## Consequences

The policy is deliberately boring and safe. Every repeated evaluation is explicit to logs, meters,
cancellation, and future effect accounting. Applications that need retries can implement a bounded
loop now without the kernel guessing idempotence. A reusable helper can be admitted later only after
its accepted callback/effect boundary is narrower than arbitrary Go code and has versioned execution
evidence.
