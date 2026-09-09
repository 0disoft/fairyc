# Resource-bearing user enum admission assessment

Status: Deferred candidate; no source admission or implementation claim.

## Current decision

Do not remove the Stage 0 rejection as a small backend follow-up. The owning
contracts `docs/language/0005-surface-syntax-v0.md` and
`docs/language/0004-results-traps-and-control-flow.md` restrict user-enum fields to
inert copyable values. Resource payload moves and active-variant cleanup currently
belong to built-in `Option` and `Result`.

The lexer candidate already retains owned cooked bytes in a resource token inside
`Vec<Token>`. Its real-source and long-input execution does not demonstrate a need
for a resource-bearing user enum. ADR 0218 requires demonstrated unsafe behavior
or repeated code and a materially worse library alternative before acceptance.
The present request identifies a desired capability, but does not yet supply that
workload comparison. Keep the existing rejection and compatibility identity.

## Bounded reopening plan

1. Check in a concrete workload with at least two distinct owned payload variants.
   Compare its current resource/Option representation and invalid-state handling
   with the proposed enum. Do not use a synthetic roundtrip alone as product need.
2. Accept a feature ADR defining enum-wide move-only classification, exact
   constructor transfer, consuming exhaustive switch, discarded active payload,
   partial-move rejection, and cleanup order. Define how no-drop payloads affect
   the whole enum; an inactive variant must not grant permission to bypass an
   obligation. Initially exclude recursive graphs, borrowing payload projections,
   generic enum definitions and public/foreign ABI.
3. Implement ownership/type facts and flow diagnostics first, with explicit
   unsupported executable lowering until MIR understands the new ownership kind.
   Do not globally mark the enum inert to reuse existing value-copy lowering.
4. Implement and verify construction, move, active-payload extraction and drop in
   MIR, interpreter and native emission as one vertical slice. Observe exactly one
   cleanup for success, early return and discard, and reject use after transfer.
5. Before source activation, review language revisioning, previous-revision
   rejection, interface/cache identity, private artifact verification and every
   consumer that currently assumes enums are inert. Run the owning negative tests
   and default/optimized native execution on each claimed target.

Each step is a separate reviewable unit. This assessment does not reserve syntax,
change SDK APIs, advance a language/runtime version, or promote self-hosting.
