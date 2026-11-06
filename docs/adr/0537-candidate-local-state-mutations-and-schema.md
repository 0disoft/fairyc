# ADR 0537: Fairy local-state mutations and schema reconstruction

- Status: Accepted for the existing Fairy graph layout family
- Date: 2026-11-07

`module_state_local_codec` now admits structural mutation input into a sealed
Canonical owner before any storage I/O. All spans, deleted/null payload rules,
count/per-value/total byte limits and duplicate keys are checked. Unsorted
caller rows are accepted: a bounded heapsort orders copied indices in
O(M log M), without mutating caller data. Canonical copies retain nil versus
empty bytes. Applying the overlay merges sorted entries in O(N+M), preserving
all unmentioned physical rows. Final admission includes JSON/base64/checksum
expansion under the existing full-state policy.

Layout comparison precedes version comparison. Empty overlays preserve the
version, including UINT64_MAX; every nonempty overlay advances it exactly once,
including deletion of an absent key and replacement by equal bytes. Nonempty
UINT64_MAX refuses. The private errors preserve malformed/policy/allocation
1001/1002/1003, layout/version mismatch 1004/1005, exhaustion 1006 and
incompatible layout extension 1007.

Append-only schema publication validates the complete next Plan, compares the
same UID and exact ordered prior field signatures/prefixes, and preserves all
entries. An identical layout is a no-op; a genuine append advances once. It
does not infer compatibility merely from matching names or an external hash.

`module_state_local_schema::read` reconstructs the current canonical layout
header into independently owned names/fields. This permits detecting a valid
foreign/upgraded stored layout instead of misclassifying every prefix change
as corrupt data. It retains the existing graph family's seven codecs and
1,024-field ceiling. Numeric/string/array boundaries and complete layout
digests/prefixes are rechecked under the caller's cumulative ledgers.

A reconstructed header is structural only. Its caller must immediately use
ADR 0535's complete record decoder with that Plan and the same received bytes
and work history. Header success grants no checksum/content, execution or
publication authority. Actual file locking, atomic publication and one-attempt
native graph persistence remain separate integration work.

Evidence: `docs/development/candidate-local-state-mutations-20261107.md`.
