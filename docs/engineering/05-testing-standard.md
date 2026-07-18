# Testing Standard

- Status: Draft
- Owner: UNASSIGNED
- Semantic sources: `spec/grammar/stage0.ebnf`, `spec/fixtures/parser/manifest-v0.json`, `spec/fixtures/semantic/manifest-v0.json`, `spec/fixtures/resource/manifest-v0.json`, `spec/fixtures/region/manifest-v1.json`, `spec/fixtures/result/manifest-v0.json`, `spec/fixtures/mir/manifest-v0.json`, `spec/fixtures/mir-verifier/manifest-v0.json`, `spec/fixtures/package/manifest-v0.json`, `docs/language/`, `docs/mir/`, `docs/package/`, `docs/abi/`, `docs/runtime/`, `docs/compatibility/`, `docs/diagnostics/catalog-v0.md`, `docs/atlas/`
- Merge-blocking validation: `VALIDATION.md`

The Go frontend executes every case in `spec/fixtures/parser/manifest-v0.json` through
`internal/parserfixture`. Source, lexer, syntax, AST, budget, containment, and manifest behavior
also have focused unit tests. Lexer and parser fuzz targets run their deterministic seed corpus
during ordinary Go tests and assert lossless reconstruction, stable output, valid spans, bounded
exhaustion, and error-free AST lowering. Experimental CLI tests cover argument routing, reserved
stable commands, deterministic content-safe JSON, schema synchronization, human diagnostics,
bounded file reads, source-root escape rejection, output failure, clean syntax checks, semantic
success, semantic diagnostics, and unsupported-subset rejection.

Experimental semantic CLI coverage also verifies explicit multi-file module sets, input-order
determinism, cross-file visibility diagnostics, duplicate rejection, bounded file counts, and a
repository-owned qualified-import example.

`internal/semanticfixture` executes all 11 cases in `spec/fixtures/semantic/manifest-v0.json`
against name resolution, typed HIR, and local safe-borrow Flow. It verifies exact failure phase,
complete diagnostic set, related spans, selected resolution coordinates, selected canonical type
facts, borrow origins, last uses, and escape facts. `internal/resourcefixture` executes all 7 cases
in `spec/fixtures/resource/manifest-v0.json`, comparing complete-root states, cleanup obligations,
and the complete deterministic `E11xx` set after parser, resolver, and type setup.
`internal/regionfixture` executes all 6 cases in `spec/fixtures/region/manifest-v1.json`, compares
hidden allocation origins and complete `E31xx` diagnostics, and admits the lexical Arena case only
after proving its root, dependency, lookup, instance class, provider role, source root, and exported
`std::alloc::Arena` against the referenced package graph. `internal/resultfixture` executes all 6
cases in `spec/fixtures/result/manifest-v0.json`, comparing handling modes, canonical payloads,
single-evaluation facts, and complete `E41xx` diagnostics. MIR, MIR-verifier,
resolved-package-graph, interpreter, backend, ABI, runtime, and Atlas runners do not exist yet;
their fixtures remain future evidence requirements.

## Oracle order

1. Product specification and accepted ADRs define the intended boundary.
2. Versioned language and MIR fixtures become the executable contract.
3. The MIR verifier rejects structurally invalid compiler output.
4. The MIR interpreter provides the first executable semantics oracle.
5. Generated C is compared against the interpreter under the C compilers and profiles declared by the target fixture.
6. Later backends are compared against the same interpreter and C evidence rather than defining new behavior.

## Fixture classes

| Class | Proves |
| --- | --- |
| Parse pass and fail | `spec/fixtures/parser/` currently proves grammar acceptance, precedence trees, literal cooking, lossless reconstruction, recovery provenance, byte spans, and deterministic diagnostics through the Go parser runner |
| Semantic pass and fail | `spec/fixtures/semantic/` currently proves all 11 v0 cases: resolution coordinates, canonical types, local last-use loans, borrow return origins, phase ownership, related spans, and deterministic cascade suppression |
| Module pass and fail | File-to-module identity, explicit aliases, forbidden import forms, cycles, and source visibility |
| Type and initialization fail | Inert/resource separation and initialized-local rules |
| Resource flow pass and fail | `spec/fixtures/resource/` currently proves all 7 v0 cases: whole-root moves, scope obligations, branch and loop states, and deterministic `E11xx` spans; reassignment, consuming close, and concrete drop traces remain future groups |
| Borrow-origin pass and fail | Local last-use loans, conflicts, single-origin return, and storage bans |
| Region pass and fail | `spec/fixtures/region/manifest-v1.json` currently proves all 6 v1 cases: parameter and package-resolved local origins, permitted parameter-origin returns, `E3101` local escape, capability restrictions, and no `NeedsDrop` placement |
| Result pass and fail | `spec/fixtures/result/` currently proves all 6 v0 cases: return, discard, must-use behavior, `?` classification, exactly-once evaluation, and error compatibility; constructors and payload traces remain deferred |
| MIR normalization | `spec/fixtures/mir/` proves source-linked evaluation order, call successors, ownership transfer, cleanup order, Result branching, trap paths, and eliminated source forms once a runner exists; graph identity remains excluded |
| Resolved package graph | `spec/fixtures/package/` proves instance equality, feature splits, direct visibility, module isolation, ambiguity, transitive hiding, and explicit standard-library injection without choosing public package-manager syntax |
| Built-in tagged resources | `Ok`, `Err`, `Some`, discard, match, propagation, early return, and nested active-payload drop traces |
| MIR verifier mutation | `spec/fixtures/mir-verifier/` supplies a verified named seed and one isolated mutation for every current `Vxxxx` invariant without serializing arbitrary MIR |
| MIR interpreter run and trap | Scalar results, branches, moves, drop traces, region reset, and trap categories |
| Drop elaboration | Partial construction, reassignment, early exits, loops, and cleanup-suffix merging |
| C differential | Interpreter parity for output, status, trap category, and drop trace across declared compilers and profiles |
| C safety | Overflow, division, shifts, evaluation order, aliasing, alignment, inactive payload, and sanitizer-assisted checks |
| ABI | Target layout, calls, callbacks, raw bindings, and boundary shims for candidate target profiles |
| Runtime and stdlib | Zero-size normalization, allocation failure, reserve rollback, UTF-8 invariants, partial I/O, Linux close consumption, and retained-pointer wrappers |
| Compatibility | Exact pre-1 matching, cache misses, runtime mismatch, diagnostic identity, stale maps, target mismatch, and relation-specific completeness |
| Atlas schema | Deterministic identity, record compatibility, unknown edges, provenance, and whole-package rewrite |
| Atlas context | Budgets, stable ordering, omissions, stale source, text caps, and prompt-injection isolation |

## Diagnostic assertions

Compile-fail fixtures assert the stable diagnostic code, primary failure category, and required related spans. They do not freeze full prose unless wording itself is the contract. A single malformed input may produce recovery diagnostics, but the fixture names which diagnostic is primary.

Parser fail fixtures are stricter: they list every `E00xx` result, use half-open byte spans, and record whether recovery inserted, skipped, synchronized, or exited. Parser pass fixtures assert syntax only and may intentionally fail later name or type analysis. Operator fixtures compare normalized tree shape rather than only evaluation results.

Semantic fail fixtures list every independent diagnostic after cascade suppression, identify the owning resolver, type, or Flow phase, and use role-labelled related spans. Semantic pass fixtures may assert declaration coordinates, canonical types, and borrow origins, but never whole AST or HIR dumps or build-local IDs.

Resource fixtures sample complete-root state immediately before or after a source span. They assert that `Live` owns one cleanup obligation, `Dead` owns zero, and a source-visible `MaybeLive` join is rejected. MIR drop flags and executable drop order remain separate fixture contracts.

Region v1 resolves its lexical escape source through the explicit standard-library package instance and opaque fixture `Arena`; it does not pretend a user-declared type is the standard-library arena. Result fixtures accept typed parameters so they can test handling without choosing `Ok` or `Err` constructor syntax. Each manifest lists its deferred coverage explicitly.

MIR fixtures compare source-linked observations after verification. They do not compare textual dumps, block or local numbers, temporary names, or generated C. MIR-verifier fixtures separately require each named baseline to verify, apply one typed mutation, and return the complete one-code result; arbitrary malformed graph input remains deferred.

Package fixtures start from an already-resolved graph. They do not prove public manifest parsing, version solving, registry behavior, source acquisition, lockfile encoding, or a concrete package-instance hash.

## Differential matrix

The first planned matrix targets `x86_64-linux-gnu`, uses the MIR interpreter plus generated C, and compares Clang with GCC at low and optimized profiles. Exact compiler and glibc baselines remain UNDECIDED. Fairy-to-C and C-to-Fairy fixture directions are both mandatory. A new target or backend cannot replace the interpreter and reference-C oracles merely because its outputs agree with itself.

ABI fixtures cover scalar and pointer layouts, nested and padded `@repr(C)` structs, fixed arrays, explicit C enums, register exhaustion, mixed integer and floating arguments, narrow extension rules, by-value aggregate calls and returns, hidden return storage, null raw pointers, and the permitted synchronous callback. Every excluded C ABI feature has a negative fixture.

## Fuzzing and sanitizers

Lexer and parser fuzz entry points now exist, and their fixed seed corpus runs under the ordinary
test command on Windows and Linux CI. A timed or continuous fuzz campaign is not configured yet.
Type, Flow CFG, MIR verifier, and schema fuzzing remain later validation surfaces. Generated-C
sanitizer runs are backend evidence, not proof of language semantics. Fuzz duration, retained corpus
policy, and coverage thresholds remain uncommitted until reproducible campaign evidence exists.

## Review blockers

- A semantic rule has no positive or negative observable fixture class.
- A grammar, literal, precedence, or recovery change updates prose without changing `spec/grammar/stage0.ebnf` and the parser manifest in the same review.
- A resolution, typing, or Flow rule changes without a schema-compatible semantic pass or fail fixture and synchronized diagnostic meaning.
- A whole-root ownership rule changes without synchronized resource-state assertions, `E11xx` spans, and drop-elaboration review.
- A Region or Result rule changes without its dedicated manifest, deferred-coverage review, `E31xx` or `E41xx` spans, and MIR-lowering review.
- A MIR rule changes without synchronized source-linked observations, deferred-coverage review, verifier impact, and interpreter or enabled-backend impact.
- A MIR verifier rule changes without synchronized seed entities, one-mutation isolation, complete code coverage, and harness-failure review.
- A package identity or visibility rule changes without synchronized resolved-graph cases, module lookup review, compatibility review, and Atlas package-edge review.
- A backend test compares only generated text instead of executable behavior where execution is available.
- A diagnostic code changes meaning without the catalog and compile-fail fixtures changing together.
- A trap, move, or cleanup result differs by optimization profile.
- A foreign nonlocal-control test crosses a Fairy C frame instead of using a boundary shim.
- An Atlas test treats missing or opaque edges as a complete graph.
- A target is called conformant or supported while its declared fixture subset contains a skip.
- A runtime wrapper normalizes Linux-specific behavior and then documents it as portable POSIX behavior.
- Exact test counts, durations, or coverage numbers are promised without an executable harness and source of truth.
