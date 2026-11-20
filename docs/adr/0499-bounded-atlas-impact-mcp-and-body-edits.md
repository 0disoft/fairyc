# ADR 0499: Bounded Atlas impact, read-only MCP and checked body edits

- Status: Accepted bounded Go prototype; each slice requires its own executable evidence.
- Date: 2026-11-02
- Extends ADR 0004 and existing exact-current map/query ownership.
- Contract: [impact, MCP and structural body edits](../atlas/impact-mcp-edit-v1.md).

Admit reverse reachability only over compiler-resolved direct calls joined from the complete
exact-current map partition set. Preserve current source/compiler/interface identities and all
completeness facts. Return bounded/partial evidence and explicit depth/node cutoffs, never safe
change, full points-to, outside-workspace callers or behavior-equivalence claims. Existing interface
review supplies its separate bounded semantic/effects diff; it is not inferred from call reachability.

Admit a separate opt-in stdio MCP server with only a read-only impact tool. Root, capabilities
and construction/traversal/output limits are fixed by the process invocation, not model arguments.
Every tool call revalidates the locked current workspace. No shell, network, source-body delivery,
write tool, prompt execution, sampling or ambient credential surface is exposed.

Admit a separate explicit single-private-function body transform. Bind the exact current symbol,
raw source SHA-256 and AST body span. A preview must replace exactly that span, then reparse and
verify the complete production workspace and ensure no sibling/signature was injected. Applying
requires an explicit CLI switch and repeat input/identity checks before one contained atomic file
replacement. Keep `fairy.lock` unchanged so a later locked compile rejects until ordinary check.
A separate bounded Go batch preview verifies multiple private body replacements in one
workspace overlay. Explicit batch apply stages originals and replacements, then publishes
by file and attempts reverse-order restoration on failure. Failed restoration retains
its recovery journal and backups. Multi-file visibility and crash recovery follow the
limits in the owning contract. Go edits to private generic functions require concrete
specializations in the baseline and successful verification of all their original bindings
in the edited workspace. Rename inference, public/foreign/attributed functions,
SDK/vendor writes, formatting rewrites and MCP writes remain excluded.

No target/language/SDK promotion, dependency or hosted runner change accompanies these local tools.
Windows provides local evidence; native Linux execution remains in the combined deferred queue.
