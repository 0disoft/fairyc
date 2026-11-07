# ADR 0541: Cancellation through Fairy Windows graph build and persistence

- Status: Accepted for the existing explicit Windows state-graph family
- Date: 2026-11-07

The existing sealed builder, runner and persistence binding add `build_cancel`
or `run_cancel`, taking a borrowed ADR 0540 Token immediately before the final
mutable work argument. Existing entry signatures delegate with none and preserve
their behavior, bounds and deadlines. This adds no main CLI or SDK/native ABI.

Pure options/work admission retains precedence. Valid pre-canceled builds
refuse before charging/copying source or performing host I/O. Runner and
persistent pre-cancellation leave their ledgers and call counters unchanged;
the persistent path does not take a snapshot. Source copies, stage transitions
and process capture observe explicit tokens without transaction/CAS retries.
Actual compile/link and graph execution use the existing job-owned cancellable
capture. Builder cancellation uses stage 11, runner cancellation stage 9, with
status 1009. Token observation failures use 1010 and retain the native OS cause.

Persistence observes cancellation before snapshot, after snapshot/detachment,
and on successful commit responses before mutation preparation and before CAS.
Post-run cancellation retains the actual owned admitted response. Already
admitted runtime error/reject/trap outcomes keep their existing precedence and
no-CAS semantics. The runner may observe cancellation before response admission.
Runner failures retain their typed nested stage/status/OS information.
Persistent checkpoints use cause stage 9/code 1009; token observation failure is
1010 with separate `cancellation_kind` and `cancellation_os_code` fields.

The last token checkpoint gates entry to the provider's atomic compare/publish.
Once entered, that provider operation is not interrupted by the token. A signal
arriving during CAS may therefore coexist with successful publication. No token
poll follows CAS; acknowledgment and the exact receipt remain irrevocable,
including a defensive receipt-consistency disagreement. This boundary matches
the existing non-contextual persistence port rather than pretending to cancel
an acknowledged rename.

The source, graph and concrete provider owners remain sealed. No callback,
arbitrary result, weaker response admission, fallback interpreter, automatic
retry or provider/chain selection is introduced. Token-to-console and actual
authored command selection, Linux and the broader final queue remain separate.

Evidence: `docs/development/candidate-state-graph-cancellation-20261107.md`.
