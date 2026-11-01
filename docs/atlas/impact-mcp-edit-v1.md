# Bounded Atlas impact, MCP and structural edit v1

Owner: ADR 0499. These are separately tested Go Stage 0 experimental commands, not a new map
schema, full semantic diff, arbitrary refactoring engine or Fairy-authored candidate promotion.

## Impact

`fairyc experimental atlas-impact --package-uid <uid> --symbol <visible-or-full-id>
--max-packages <n> --max-calls <n> --max-depth <0..64> --max-symbols <1..4096>
--max-output-bytes <1..4194304> [--locked]` always verifies an existing `fairy.lock`.
Run ordinary `fairyc check` first. The query never updates that lock or fetches dependencies.

Rebuild logical v3 maps from verified current source, derive independent load expectations, and
join all root-reachable package partitions before traversal. Missing/stale/incompatible maps or
an incomplete join fail; no old cached map is silently substituted. Ambiguous feature-instance
UIDs reject. Package/call construction ceilings are explicit and at most 65,536.

The JSON `fairyc.atlas-impact/v1` lists the subject at depth 0, deduplicated reverse-call reachable
symbols, deterministic depths, and exact-current package contexts. It includes source/compiler/
interface identities and original completeness facts, not source text or host paths. Output is
buffered and size-checked before writing. No partial document is emitted on budget failure.

`state` is bounded or partial, never exact safe-change evidence. `traversalComplete` only describes
the admitted resolved-call graph. Unknown/external coverage, partial maps and explicit traversal
cutoffs remain visible. Non-call/type/effects dependencies, outside-workspace callers, dynamic
targets and callbacks are not inferred. Adding/removing a declaration does not imply a rename.
The existing interface/effects review remains the separate comparison tool for its documented
ordinary-interface subset; none of its uncertainty becomes an unchanged declaration here.

## MCP and structural edits

The separate following implementation slices own stdio lifecycle/capability checks and atomic
single-body edit preview/apply. Neither capability becomes enabled merely by an impact result.
All tools use explicit package/symbol selectors and treat attributed text or client strings as
untrusted data, never instructions or authority to run a command. A tool result cannot grant
write permission or widen the root selected by the process invocation.
