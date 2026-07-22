# ADR 0159: Expose process-local MIR reuse through a bounded workspace check session

- Status: Accepted for experimental CLI
- Date: 2026-07-22
- Command source: `internal/cli/workspace_session.go`
- Compiler source: `internal/workspacecompiler/session.go`
- Protocol schema: `spec/cli/experimental-workspace-check-session-v0.schema.json`

## Context

ADR 0156 introduced a reusable in-process compiler Session, and ADRs 0157–0158 proved its cache
paths through test and measurement harnesses. The user-facing CLI still created a new process for
each check, so none of that cache state survived long enough to help an editor, build tool, or
developer issuing a second request.

Repeating the same check a fixed number of times inside one invocation would demonstrate a hit but
would not let a caller edit the workspace between requests. A filesystem watcher or daemon would
add platform-specific notification behavior, background lifecycle, cancellation, stale-event, and
socket-security policy before Stage 0 has evidence that it needs those costs.

## Decision

1. Add the unstable command `fairyc experimental check-workspace-session`. It owns one
   `workspacecompiler.Session` and accepts sequential check requests from stdin.
2. The command is a foreground bounded process, not a daemon, watcher, socket server, network
   service, or persistent cache. It stops at EOF or after the explicit `--max-requests` bound.
3. Every request and response is exactly one UTF-8 NDJSON object using
   `fairyc.experimental.workspace-check-session/v0`. Requests require `kind: "check"` and contiguous
   `sequence` values beginning at one. Responses use `kind: "check-result"` and repeat the sequence.
4. Request objects reject duplicate keys, unknown keys, missing fields, wrong types, trailing JSON,
   invalid UTF-8, empty lines, and lines above 1,024 bytes. A protocol error terminates with
   prototype invocation code `2` before performing that request.
5. The caller must supply `--target`, `--max-requests`, `--cache-max-entries`,
   `--cache-max-bytes`, and `--cache-max-artifact-bytes`. There are no hidden product cache
   defaults. The safety ceilings are 256 requests, 65,536 entries, 1,073,741,824 retained bytes,
   and 1,073,741,824 artifact bytes; the artifact limit may not exceed the retained-byte limit.
6. Each request reloads the bounded manifest-owned workspace from disk and applies the existing
   ordinary or `--locked` lockfile policy. Filesystem mutation and loading do not share source bytes
   across requests. Only verified process-private MIR artifacts persist in the Session.
7. Source diagnostics, unsupported source, and package failures produce a response and leave the
   session available for the next request. Human diagnostics remain on stderr. Internal invariant,
   input-read, and output-write failures terminate the process because response framing or Session
   trust can no longer be guaranteed.
8. The response records status, per-request prototype exit code, lock state, workspace and MIR
   counts, diagnostic counts, cache decision counts, retained entries and bytes, and cumulative
   verified hits. It contains no source text, source path, absolute path, artifact bytes, or local
   toolchain path.
9. When EOF or the request limit ends a nonempty session, the process returns the last completed
   request's exit code. An empty session is an invocation failure. Earlier transient diagnostics do
   not force a nonzero process result after a later successful request.
10. This command establishes an honest product boundary for process reuse. It does not establish a
    speedup, default cache policy, file watcher, LSP, concurrent request model, cancellation
    protocol, persistent cache, or stable CLI/API.

## Consequences

- an editor or tool can keep stdin open, modify the workspace, and request another check without
  restarting the Go compiler;
- requests execute strictly in sequence, so response order and cache mutation have no concurrency
  ambiguity;
- lockfile writes remain the existing ordinary-mode behavior and `--locked` remains exact and
  no-write;
- cache memory is bounded twice by entry/total-byte limits and once by artifact-byte admission;
- the configured standalone Windows compiler smoke proves two requests cross the real process
  stdin/stdout boundary and that the second request is an all-hit reuse;
- reference-machine timing, process peak memory, noise policy, cancellation, editor integration,
  and stable command naming remain separate gates.

## Rejected alternatives

- A fixed `--repeat` loop was rejected because no real edit can occur between buffered checks.
- A package-global cache inside ordinary one-shot commands was rejected because process exit would
  erase it and disguise non-reuse as integration.
- A background daemon, local socket, or filesystem watcher was rejected because Stage 0 has no
  lifecycle, authentication, cancellation, event-coalescing, or recovery contract for one.
- Hidden cache defaults were rejected because the current measurements do not justify a product
  memory policy.
- Concurrent request handling was rejected because it would add ordering, cancellation, shared
  filesystem snapshot, and cache synchronization questions without measured need.
