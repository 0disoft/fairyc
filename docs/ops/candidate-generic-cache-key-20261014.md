# Candidate source-generic cache key v0

The Fairy generic key embeds the complete validated ordinary v4 encoding, then
the current generic interface hash, one exact-coordinate generic interface per
ordinary dependency, and sorted unique canonical `g0_` IDs emitted in this
package. Missing/reordered/extra dependency coverage and malformed or duplicate
specializations reject. The ordinary key exposes detached dependency facts and
its encoding separately, avoiding an unnecessary intermediate ordinary SHA.

The maintained Go `ComputeGeneric` supplies eight independent key observations.
A reconstructed Go envelope must hash to that key before it is written as the
fixture. Fairy then matches every envelope byte and key text in Go-native and
candidate C development/release. Cases change current/dependency generic
interfaces, specialization membership, ordinary state-body and compiler facts,
and cover no specializations and no dependencies. Three authored tests also
exercise eleven invalid fact/set cases, detached ownership and exact/exhausted
16MiB phase work. The four ordinary v4 authored tests pass again in all three
profiles after the encoding split. Private API images retain 16MiB stacks.
Root and locked checks pass 453 files / 5862 MIR functions.

The Go oracle is regenerated with `tools/run-go.ts run
tools/verify-candidate-generic-cache-key.go --output <fixture>` and passed through
the maintained private test verifier. A first direct fixed-array resource field
was rejected by the existing language boundary; the final owner retains that
array inside a plain record, with no language extension.

Receipt: [evidence](evidence/candidate-generic-cache-key-20261014.json).
Next: derive these required facts from the complete checked package owners and
environment, then verify/rebind reusable package artifacts before actual cache
lookup/publication. Both key APIs still accept caller facts; neither authorizes
a cache hit. Current full fixed point, product defaults/performance, Linux/full
suite and same-head CI remain open. Public CLI/SDK ABI and product stack/timeouts
are unchanged; no push.
