# ADR 0328: Production workload capability admission gates

- Status: Candidate
- Date: 2026-08-10
- Product source: `docs/product/02-spec.md`
- Roadmap source: `docs/product/01-roadmap.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Depends on: ADR 0179, ADR 0218, ADR 0223

## Context

Fairy C may eventually serve long-running services, command-line tools, data processors, and other
production workloads. Those uses share requirements that the Stage 0 language and runtime do not
yet claim: supported Linux distribution, cancellation and deadlines, bounded concurrency and I/O,
data interchange, and operational evidence. Recording those requirements now can prevent isolated
features from fixing incompatible scheduler, ownership, generated-code, or deployment contracts.

A broad list of server features is not evidence that Fairy C should become a framework platform.
HTTP, databases, schema formats, telemetry protocols, and deployment systems evolve independently,
while Fairy C deliberately keeps its language, runtime, ABI, and build model small. This ADR
therefore records admission gates, not accepted syntax, public APIs, or an implementation schedule.

## Candidate gates

### 1. Linux production-target promotion

Promotion beyond the current provisional and conformance profiles requires all of the following:

- one exact target, libc, linker, loader, runtime, standard-library, and distribution identity;
- reproducible release builds with declared dynamic dependencies and a tested deployment closure;
- same-commit interpreter, generated-C, ABI, sanitizer, optimized-native, and artifact evidence;
- `.fy` source recovery through optimized symbols or an explicitly bounded source-map contract;
- signal and process-lifecycle behavior with bounded graceful-stop and forced-stop rules;
- representative long-running, concurrent, memory-budget, descriptor-budget, and failure tests;
- separate promotion decisions for GNU, musl, x86_64, and arm64 rather than implied portability.

This gate does not demote Windows evidence. Each supported target retains independent maturity and
distribution receipts.

### 2. Cancellation, deadlines, and structured concurrency

Any concurrency or asynchronous proposal must first define these ownership invariants:

- spawned work has one visible scope and cannot silently become an orphan;
- scope exit completes, cancels and joins, or explicitly transfers every outstanding task;
- cancellation and timeout paths execute the same verified resource cleanup obligations as other
  exits and cannot skip Region reset or active-payload drop;
- queues, streams, body-like inputs, decompression, buffering, and blocking pools have explicit
  capacity, byte, work, and deadline bounds;
- `drop` remains synchronous, nonblocking, allocation-free, and unable to fail; waiting, flushing,
  commit, rollback, and graceful close remain explicit transitions;
- cross-thread transfer and sharing require separately proven capabilities rather than inheriting
  safety from ordinary ownership or borrowing;
- a lexical or request-like Region cannot escape through detached work.

No `async`, `await`, task, scope, channel, cancellation-token, `Transfer`, or `Share` spelling is
reserved. A runtime or library experiment may precede a language proposal. Hidden executors and
ambient unbounded queues remain inadmissible.

### 3. Bounded data interchange and generated declarations

Serialization, validation, configuration, row mapping, and schema export may return to the
admission queue only after representative checked-in source demonstrates material repeated code.
The first proposal must:

- name exact input-size, nesting, token, string, collection, and output-size limits;
- define duplicate-field, unknown-field, UTF-8, numeric-range, and partial-output behavior;
- produce inspectable ordinary Fairy declarations or equally explicit compiler facts;
- preserve declaration identity, source locations, deterministic diagnostics, cache invalidation,
  and generated-artifact compatibility;
- avoid runtime reflection, hidden allocation, arbitrary AST rewriting, unrestricted user macros,
  and annotation-driven framework behavior;
- keep each generated capability separately selectable instead of admitting one universal derive
  or reflection system.

JSON, database rows, configuration, API schemas, and client generation are possible evidence
sources, not accepted formats or products. No `derive`, reflection, macro, or code-generation
syntax is reserved.

### 4. Vendor-neutral operational context

Long-running workloads may later need structured events, traces, metrics, profiles, and request or
task context. Reopening this candidate requires:

- an explicit owner and lifetime for context rather than an unbounded mutable process global;
- bounded propagation across admitted task, callback, and foreign-call boundaries;
- deterministic source-location and symbol identity where the compiler supplies them;
- redaction, cardinality, payload-size, buffering, backpressure, and failure behavior;
- proof that observation cannot change safe-program results or bypass ownership cleanup;
- library-neutral compiler/runtime facts before choosing any external telemetry protocol.

Logging during `drop`, hidden allocation for observation, and treating telemetry delivery failure
as permission to lose application cleanup remain disallowed.

## Explicit non-decisions

This candidate does not accept or schedule:

- an HTTP client, HTTP server, web framework, router, middleware, TLS stack, or WebSocket API;
- PostgreSQL, SQLite, Redis, ORM, migration, query-macro, or connection-pool APIs;
- `async`/`await`, coroutines, virtual threads, a scheduler, recoverable task traps, or unwinding;
- traits, interfaces, associated types, generic services, implicit conformance, or dynamic dispatch;
- `derive`, runtime reflection, arbitrary macros, OpenAPI, Protobuf, or a specific telemetry vendor;
- a public registry, version solver, package publication, container tool, or deployment platform.

Those areas require their own evidence and accepted ADRs. Package acquisition remains owned by the
existing manifest and lockfile readiness contract; generic constraints remain behind ADR 0223.

## Reopening and implementation evidence

Each capability advances independently from `candidate` to `accepted` only when its proposal names
the source contract, ownership and effect rules, bounds, failure states, target impact, compatibility
cut, diagnostics, and positive and negative fixtures. `Implemented` additionally requires the
owning semantic or runtime tests and real generated-Fairy native evidence on every target claimed.

## Consequences

The project retains durable production-workload questions without turning them into Stage 0
promises. Future work can build small vertical slices against shared invariants, while Fairy C
continues to reject features whose scheduler, allocation, cleanup, ABI, or build cost is hidden.
This ADR changes no grammar, SDK surface, runtime ABI, target maturity, version identity, or current
implementation status.
