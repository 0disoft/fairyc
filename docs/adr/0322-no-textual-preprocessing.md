# ADR 0322: No textual preprocessing in Fairy source

- Status: Accepted
- Date: 2026-08-09
- Language: `fairyc-language/10`
- SDK: `0.0.0-stage0.48`
- Runtime ABI: `fairyc-runtime/0`
- Depends on: ADR 0136, ADR 0143, ADR 0172, ADR 0320

## Context

C and C++ translation units can change before parsing through textual inclusion, macro expansion,
ambient command-line definitions, and conditional token removal. That model makes the effective
source and dependency graph depend on compiler invocation state that is not visible in the file.
Fairy C uses C as one constrained native transport, but does not need to inherit that source model
or promise compatibility with historical C headers and compiler flags.

## Decision

1. The Fairy compiler sends the exact validated `.fy` source bytes to the lexer. There is no
   pre-lexing include, macro-expansion, token-pasting, or conditional token-removal phase. `#` is
   not a directive introducer and remains an `E0001` invalid source byte outside literals/comments.
2. Every source file declares one module and imports semantic module identities. Dependency and
   source closure come from the explicit project/workspace graph, never from text copied into a
   translation unit or from implicit transitive visibility.
3. Manifest v1 dependency `features` are sorted identity inputs. They do not create ambient defines
   or an in-file `cfg` language. Any future target/feature implementation selection must be a
   separately versioned graph or manifest decision that selects complete package/module inputs and
   preserves explicit interface identities; it must not hide or rewrite source tokens.
4. Generated C, the thin C runtime, and target probes may use headers and the C preprocessor as
   compiler-owned implementation details. Those inputs belong to the backend/toolchain closure and
   do not become Fairy source semantics.
5. C interoperability remains opt-in and visibly unsafe through `unsafe extern C fn`, `@link_name`,
   and `@repr(C)`. A future header importer, if admitted, must be a bounded one-shot generator that
   produces reviewable versioned Fairy declarations; the compiler will not textually include a C
   header while compiling `.fy` source.
6. Fairy language, SDK, runtime ABI, target schema, MIR, cache, and lockfile identities remain
   unchanged because this ADR freezes existing accepted and rejected behavior.

## Evidence

- The normative grammar starts from `module_decl import_decl* item* EOF` and documents direct
  source-byte lexing without a preprocessing phase.
- Lexer regression tests require `#include`, `#define`, and `#if` to produce one `E0001` at `#` in
  both the current and immediately previous language revisions while preserving exact input bytes.
- The closed project-manifest parser rejects a `defines` field, and its documented feature contract
  grants graph identity only.
- Existing package-graph, public-interface, source-digest, and cache-key contracts retain explicit
  dependencies and compatibility inputs instead of compiler-invocation-dependent header text.

## Consequences

Fairy source has one visible parse input and one inspectable dependency graph. Platform-specific
implementations may require separate modules or packages, which is more verbose than local `#if`
blocks but keeps parsing, interfaces, caches, and Atlas facts deterministic. C transport can still
use platform headers internally, so this decision does not claim that the Stage 0 implementation
has eliminated all C toolchain dependencies. It prevents those dependencies from becoming the
Fairy language's public compilation model.
