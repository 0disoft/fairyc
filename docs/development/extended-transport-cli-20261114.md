# Extended transport CLI/cache replay — 2026-11-14

The complete authored optimized CLI passes all three new concrete families:
flat Result with signed i32 errors, two-level Option, and straight-line inline
mutation. The Go compiler bootstraps the host; target checking, IR/artifact/cache
policy and C packet generation execute in Fairy. Native transport consumes the
fresh packet. No GUI or input automation is involved.

`TestCandidateExtendedTransportCLIAndCache` passes in 462.18 seconds, with one
210-part host build shared by the three families. Each fixture executes nine
separate CLI processes and two standalone products: 27 CLI processes and six
products in total, plus independent seven-scalar C packet observations.

The replay covers cold store, locked warm reuse, low/release builds, native
tests, a provider-only edit, stale-lock refusal, rebuilding to exit 43, locked
reuse of the updated program, and invalid-source refusal without changing valid
persistent generation bytes. Inline mutation edits change the dependent result
without changing ordinary seven-scalar payload helpers. Signed error probes
execute -77 checks; nested Options copy and exhaustively match active values.

Current log: `.cache/extended-cli-20261114.log`; exact source/image receipt:
`.cache/extended-cli-20261114/optimized/host.json`.
Low whole-CLI replay was not repeated for this batch. Both producer/consumer
profiles already passed the focused source/artifact/native gates for each family.

This closes the batched CLI/cache gate for these three implementations. Broader
composite generic arguments, inline branches/loops and inline-to-inline calls
across modules remain separate language designs; this receipt does not claim
general unrestricted generics or inline bodies.
