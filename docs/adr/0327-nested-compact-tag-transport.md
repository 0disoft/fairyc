# ADR 0327: Nested compact tag transport

- Status: Accepted
- Date: 2026-08-10

## Decision

The constrained C backend admits copy-only `Option` and `Result` transports whose direct payload is
the already-fixed compact `Option<i32>` or `Result<i32, i32>` transport. The compact inner typedef is
emitted before generic tagged records, and target layout is computed from that same closed record.

This is private compiler transport for source programs already accepted by the frontend. It changes
neither the public Fairy ABI nor the language, SDK, or runtime identity.

## Boundary

General tag-to-tag dependency graphs, nested resource-bearing tags, recursive tags, and nested tagged
fields inside aggregates remain unsupported. Expanding those cases requires a separate dependency and
ownership decision rather than recursive admission through this rule.

## Evidence

The 50th C-emission fixture freezes nested construction, read, and switch with interpreter result 42.
A dedicated Windows test independently compiles and runs the generated Fairy program with low and
optimized MSVC profiles and requires native result 42.
