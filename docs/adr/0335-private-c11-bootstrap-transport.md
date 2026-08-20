# ADR 0335: Demote generated C11 to a private bootstrap transport

- Status: Accepted
- Date: 2026-08-18
- Product source: `docs/product/02-spec.md`
- Architecture source: ADR 0001
- Transport source: ADR 0137
- Supersedes: the stable product-surface decision in ADR 0180

## Context

Fairy C exists so users can write and ship Fairy C programs. Generated C was introduced as the
cheapest first native transport, a differential-test target for verified MIR, and a way to defer an
early dependency on a larger native backend. That bootstrap decision remains useful.

The stable `fairyc emit c` command changed the incentives around that implementation detail. It made
target-selected generated C look like a supported product artifact and required the project to treat
its command grammar, failure categories, whole-unit output, deterministic private symbols, target
profiles, and transport dialect as user-facing compatibility surfaces.

That surface does not help ordinary users build Fairy C applications. It also encourages feature
work to be framed as expansion of an ISO C source product instead of completion of Fairy C semantics,
standard-library APIs, diagnostics, tooling, and native application builds. Every additional public C
transport promise compounds work around C evaluation order, integer promotion, undefined behavior,
reserved identifiers, consumer modes, target layout, runtime headers, and ABI evidence.

C interoperability is a separate product requirement. `unsafe extern C fn`, `@repr(C)`,
`@link_name`, target-owned C scalar mappings, and verified caller/callee crossings allow Fairy C
programs to use existing native libraries. Removing a generated-C product command must not weaken
those boundaries.

## Decision

1. ISO C11 remains the sole bootstrap transport dialect for the current constrained C backend. This
   keeps one conservative consumer baseline and does not introduce C17 or C23 emitters.
2. Generated C is compiler-private transport. Its declarations, helper functions, private symbols,
   file structure, formatting, and whole-unit bytes carry no cross-release compatibility promise.
3. The stable `fairyc emit c --target <target-id> [--locked]` command is removed from the product
   surface.
4. A top-level `fairyc emit ...` invocation fails with code 2 and an explicit migration message. It
   does not silently preserve the old stable contract or guess that the caller accepted instability.
5. Compiler developers retain equivalent workspace inspection through
   `fairyc experimental emit-workspace-c --target <target-id> [--locked]`.
6. The experimental command uses the existing manifest, package graph, lockfile, verified MIR,
   target selection, and C11 emitter. It invokes no compiler or linker and writes the private
   application unit to stdout for debugging and differential investigation.
7. The experimental output is not a standalone distribution artifact. It may reference separate
   runtime and standard-library support units and may change whenever the compiler implementation
   changes.
8. Stable `fairyc build`, `fairyc run`, and native `fairyc test` may continue to use C11 internally.
   Their product contracts concern Fairy source, diagnostics, produced native artifacts, execution,
   and observable program behavior rather than generated C text.
9. Explicit C ABI interoperability remains a product feature. This decision does not remove or
   weaken foreign declarations, `@repr(C)`, link names, target layout checks, ABI classifiers, or
   independently compiled caller/callee evidence.
10. New constrained-C work is admitted when it is necessary to preserve verified MIR semantics,
    unblock a Fairy C build/run/test workload, support the standard library, or maintain bounded
    differential evidence. Expanding generated C as an independently useful source product is not an
    admission reason.
11. The MIR interpreter remains the semantic oracle. A simple direct native backend may be evaluated
    after MIR stability and representative workload measurements justify it. This decision neither
    selects such a backend nor begins a second backend implementation.

## Compatibility

This is an intentional breaking change to the Stage 0 CLI. Scripts using stable `fairyc emit c` must
move to the explicitly unstable `fairyc experimental emit-workspace-c` command or, preferably, use
`fairyc build` for product artifacts.

The language revision, MIR, runtime ABI, standard-library API, target identities, C ABI contracts,
manifest, lockfile, package identities, native executable behavior, and current C11 implementation
remain unchanged. C11 transport bytes were already disposable across compiler releases and gain no
new compatibility guarantee through the experimental command.

ADR 0137 continues to own the internal C11 dialect baseline. Its historical references to a product
C-emission dialect no longer create a user-facing output promise. ADR 0180 remains as the record of
the superseded stable command.

## Consequences

The project keeps the inexpensive bootstrap and differential path without optimizing the product
roadmap around a C transpiler. Users see a smaller stable CLI centered on creating, checking,
building, running, testing, explaining diagnostics, and inspecting target maturity.

Compiler developers retain a direct way to inspect target-selected generated C, including lockfile
verification and deterministic same-version reproduction. Calling that command requires an explicit
experimental namespace, so build scripts cannot mistake private transport for a durable interface.

Future language features still require executable backend coverage before they can ship through a
native product path. They do not require a new public C-source compatibility promise, a new C dialect,
or a standalone transport ADR unless C itself introduces a correctness or interoperability decision
that cannot be expressed by the owning MIR, target, runtime, or ABI contract.

## Verification

- CLI help contains `experimental emit-workspace-c` and no stable `emit c` command.
- The removed stable command returns code 2 with the exact migration direction.
- Experimental parsing requires one explicit supported target and rejects duplicate or unknown
  arguments.
- Linux GNU and Windows MSVC workspace emission preserve target-selected C11 differences and repeated
  same-version byte identity.
- `--locked` verifies without changing `fairy.lock`; ordinary mode retains existing lock maintenance.
- Source, package, unsupported-subset, missing-entry, zero-byte output failure, and partial-write
  failure categories remain covered.
- No experimental emission path creates a build directory or invokes a native toolchain.
