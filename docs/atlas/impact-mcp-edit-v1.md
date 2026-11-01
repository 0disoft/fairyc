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

## Read-only MCP over stdio

```sh
fairyc experimental atlas-mcp --allow-impact --max-packages 32 --max-calls 8192 --max-depth 8 --max-symbols 256 --max-output-bytes 262144 --max-requests 100
```

Launch from the selected workspace with a current lock. The invocation grants the capability and
every limit. The client cannot widen the root, change budgets or enable another tool. Only
`atlas_impact` is advertised, with exact `packageUid` and visible/full `symbol` strings. Results
contain structured evidence and fixed explanatory text, not authored source/doc bodies or host paths.

The MCP 2025-11-25 stdio profile implements newline-delimited JSON-RPC, initialize/version
negotiation, initialized notification, ping, tools/list and tools/call. An unsupported client
version receives the supported server version; the client decides whether to continue. Tool calls
require the ready phase. There is no shell, network listener, sampling, prompts, client-root
request, credentials, tasks or write tool. Metadata cannot grant authority.

Limits: 1-1,024 messages (notifications count), 64 KiB/frame, nesting 16, 512 encoded bytes/ID,
and 2 KiB-4 MiB/response payload. Duplicate keys, batches and invalid framing/IDs reject. Stdout
contains only protocol frames. EOF ends the session. Every query revalidates locked current
source; unavailable/stale input becomes `isError` without raw diagnostics. Session policy failures
return 2, output failure 4; tool failure can leave the session usable.

References: [stdio](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports),
[lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle),
[tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).
This is the named profile, not every optional MCP feature or a client-product certification.

## Single-function structural body edits

```sh
fairyc experimental atlas-edit replace-body --package-uid <root-uid> --symbol <symbol-id> --body-file edits/body.txt --expected-source-sha256 <raw-file-hash>
# Add --apply to authorize the current validated replacement.
```

The body file holds one complete braced body, at most 64 KiB, in a portable regular workspace
file outside production roots. Exact current IDs select root-owned, private, safe, non-entry,
nongeneric functions without owners/attributes. SDK/member/vendor writes, renames, public/foreign/
generic/attributed declarations and multi-file edits remain excluded.

Require the exact raw source SHA-256. Replace the original AST body span and preserve all outside
bytes. Parsing must find the same function/body start and exactly the replacement extent; early
closing to inject siblings fails. Recompile the complete production workspace through an immutable
overlay and require verified generic-aware semantic/MIR output. This validates accepted source,
not behavioral equivalence or safe-change proof.

Preview returns bounded `fairyc.structural-body-edit/v1`: portable path, module/function, original
body byte range, before/after raw source hashes and verified/applied states. It writes no source or
lock. Reports over 8 KiB fail before application. Apply prepares the current body file independently,
not a serialized old plan; keep that input fixed between review and application.

Apply acquires `.fairyc-edit.lock`, repeats local manifest/production-source/graph-lock fingerprints
and candidate validation, writes/syncs a same-directory temporary, rechecks the closure and original
file, then performs one opened-root rename. No delete-and-retry fallback is used. Ordinary staging
errors clean temporaries/locks. Crash leftovers require inspection; hostile concurrent writers and
physical-media durability are not guaranteed. Output or lock-cleanup errors can follow publication;
inspect hashes before retrying. Keep `fairy.lock` unchanged and run ordinary `check` before locked
build/run/test. Language/runtime/SDK/map schemas do not change. MCP never exposes apply.
Neither write authority nor certainty follows from an impact result.
All tools use explicit package/symbol selectors and treat attributed text or client strings as
untrusted data, never instructions or authority to run a command. A tool result cannot grant
write permission or widen the root selected by the process invocation.
