# ADR 0539: Fairy Windows one-attempt persistent graph execution

- Status: Accepted for the existing explicit Windows graph/local-store family
- Date: 2026-11-07

`module_state_graph_windows_persistent` composes the sealed Built/runner from
ADR 0534 with the real local provider from ADR 0538. No arbitrary executable,
request, response, callback or provider receipt can substitute for those owners.
The graph's retained Complete supplies the expected layout identity.

One call validates options and cumulative ledgers before I/O, takes exactly one
snapshot, encodes its detached state and invokes the actual Built-backed runner
once. Only a successfully admitted commit reaches one complete overlay CAS.
Reject/trap and runtime/process/admission errors perform no publication. An
empty commit still performs the layout/version CAS without advancing version.
Nonempty exhausted versions refuse. Conflicts, busy storage and other failures
remain visible; neither the transaction nor CAS is automatically retried.

The storage and execution ledgers remain separate cumulative budgets. Attempt
counters include refused operations. The final overlay alone becomes the
canonical mutation batch; event writes and speculative intermediate values
cannot escape response admission. Caller source, process and scratch buffers
may be released while the returned response remains owned.

Typed failures retain the original storage cause or owned runner diagnostics.
Failures after successful response admission also retain that exact Admitted
owner. Capacity for those retention slots is reserved before I/O, avoiding a
new fallible allocation solely to preserve an existing failure/result. Nonzero
runtime status is an error with its admitted response; explicit reject/trap
remain outcomes with no publication acknowledgment.

Successful CAS acknowledgment is irrevocable. Receipt layout, version and write
count are checked against the attempted commit. A mismatch is returned as an
acknowledged but inconsistent outcome, preserving the actual receipt and
response; it must not be converted into a retryable publication error.

This is an explicit compiler-private Windows binding, not ordinary CLI selection
or a new public SDK/provider/chain contract. The existing runner's bounded
deadline is retained. It has no compatible cancellation-token API, so this
binding does not claim context cancellation. Broader command integration,
cancellation, Linux, full comparisons/self-build and final acceptance remain
separate required work.

Evidence: `docs/development/candidate-windows-persistent-state-graphs-20261107.md`.
