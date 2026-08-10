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

ADR 0330 separately admits acyclic generic copy-only tag-to-tag dependency graphs. Nested
resource-bearing tags, recursive tags, unsupported leaf payloads, and tagged fields outside private
aggregates remain unsupported and require separate dependency and ownership decisions.

## Evidence

The 50th C-emission fixture freezes `Option<Result<i32, i32>>` and
`Result<Option<i32>, Result<i32, i32>>` construction, read, and switch with interpreter result 42.
A dedicated Windows test independently compiles and runs the same generated Fairy program with low
and optimized MSVC profiles and requires native result 42.
