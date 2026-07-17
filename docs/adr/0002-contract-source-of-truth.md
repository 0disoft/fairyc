# Contract Sources of Truth

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-16

## Context

Fairy C combines a language specification, compiler implementation, multiple backends, runtime and standard-library code, build metadata, and semantic maps. Without explicit ownership, generated artifacts or backend accidents can silently become the language definition.

## Decision

Contract ownership is layered as follows:

| Contract | Source of truth | Derived evidence |
| --- | --- | --- |
| Product scope and non-goals | `docs/product/02-spec.md` | README and roadmap summaries |
| Accepted design decisions | `docs/adr/*.md` | Architecture explanations |
| Product language semantics | `docs/product/02-spec.md` and accepted ADRs | Detailed `docs/language/*.md` contracts |
| Versioned grammar and parser acceptance | `spec/grammar/stage0.ebnf` and `spec/fixtures/parser/manifest-v0.json` | Pratt tables, CST and AST structures, examples, and parser diagnostics |
| Typed behavior | `docs/language/*.md` and `spec/fixtures/semantic/manifest-v0.json` | Typed HIR, `SemanticSnapshot`, and Atlas type and ownership facts |
| Resource Flow behavior | `docs/language/0001-values-and-resources.md` and `spec/fixtures/resource/manifest-v0.json` | Flow CFG state, cleanup elaboration, MIR moves, and drop traces |
| Executable control flow | `docs/mir/mir-v0.md`, then versioned MIR fixtures | Interpreter, C, later native, object, and assembly output |
| Target layout and C ABI | `docs/abi/*.md`, then versioned target descriptions and ABI fixtures | Backend attributes, generated headers, and toolchain receipts |
| Runtime and standard library | `docs/runtime/*.md`, then versioned runtime and library contracts | Target runtime bundles, headers, and executable fixtures |
| Compatibility and artifact identity | `docs/compatibility/*.md` | Build receipts, cache keys, package metadata, and compatibility diagnostics |
| Diagnostic meanings | `docs/diagnostics/catalog-v0.md` | Compiler messages, explanations, and diagnostic JSON |
| Atlas map and context contracts | `docs/atlas/*.md` | Emitted maps and bounded context packs |
| CLI behavior | `docs/cli/command-contract.md` plus implementation tests | Help text and examples |
| Package ownership | `docs/monorepo/package-ownership.md` | Build graph and cache entries |
| Validation names | `VALIDATION.md` | CI configuration and command runners |
| Scaffold ownership | `.ssealed/manifest.json` | Generated document checksums |

Build caches, generated C, LLVM IR, object files, assembly, semantic maps, benchmark reports, and LLM output are never authoritative by themselves.

## Change rules

- A semantic change updates the owning specification, fixtures, HIR or MIR contract, affected backends, diagnostics, and tests together.
- A syntax, precedence, literal, or recovery change updates the normative EBNF, parser fixtures, diagnostic catalog, explanatory language docs, and language revision decision together.
- A resolution, typing, or Flow rule change updates the owning language contract, semantic schema-compatible fixtures, diagnostic catalog, and typed-HIR or Flow implementation together.
- A whole-root resource-state change updates the resource manifest, cleanup-obligation contract, `E11xx` meaning, Flow analysis, and drop elaboration together.
- A target or support-status change updates the ABI profile, toolchain requirements, runtime and standard-library surface, CLI target report, and fixture matrix together.
- A compatibility change names the affected vector member instead of using the compiler release as a catch-all version.
- A backend optimization cannot change observable language behavior.
- An Atlas fact must be reproducible from the same compiler inputs and carry its schema and source hash.
- Human documentation explains rationale and boundaries; it does not duplicate compiler-known facts as unchecked prose.
- External AI opinions, benchmarks, and language comparisons remain design inputs until adopted by an ADR or specification change.

## Consequences

The repository carries more explicit contract files, but disagreements have an owner and generated output can be discarded safely. Early prototype syntax remains provisional until a versioned language specification is introduced.
