# Initial Compiler Architecture Boundaries

- Status: Accepted for prototype
- Owner: UNASSIGNED
- Date: 2026-07-16

## Context

Fairy C needs a compiler implementation that can change quickly while the language is unstable, preserve target semantics independently of its host language, and avoid turning backend integration into the source of truth. The project also values fast builds and bounded artifacts.

## Decision

1. Implement the Stage 0 compiler in Go.
2. Store syntax and semantic graphs in typed integer-ID arenas rather than pervasive Go interface and pointer graphs.
3. Start with a trivia-preserving token stream, compact arena-backed syntax structures, typed HIR, analysis-only Flow CFG, and explicit MIR. Defer an immutable green tree and incremental reparse until formatter or IDE work needs them.
4. Lower resource moves, initialization state, drops, `Result` propagation, bounds checks, overflow checks, and traps into MIR before backend emission.
5. Use generated C as the first reference transport under an explicit target, C dialect, toolchain, libc or CRT, and ABI profile. It is not universally portable C.
6. Build a small MIR interpreter as the semantic oracle before adding a second native backend.
7. Evaluate QBE or another simple backend only after MIR is stable and measured.
8. Add LLVM as a production optimization backend only when representative workloads justify its build and distribution cost.
9. Delay self-hosting until the language, standard library, diagnostics, package build, and bootstrap subset meet the roadmap gate.
10. Retain the Go implementation as a bootstrap compiler and reference semantics implementation after self-hosting.

## Rationale

Go reduces compiler implementation overhead for parsing, diagnostics, caching, testing, and cross-platform binary distribution. The generated program does not inherit the Go runtime. Typed IDs and target descriptions prevent Go object layout and host integer choices from leaking into Fairy C.

HIR keeps user-facing language concepts available for diagnostics and analysis. Flow CFG separates fixed-point ownership analysis from executable lowering. MIR gives the interpreter and every backend the same explicit control-flow and cleanup contract. A constrained C transport makes the first executable language slice cheaper and permits two independent C compilers to check it, while delaying native backends avoids early binding and version-management cost.

## Rejected alternatives

- Rust: strong IR representation but unnecessary lifetime and build-loop cost for the mutable Stage 0 design.
- C++: direct LLVM integration but excessive implementation-language ownership and build complexity.
- Python or TypeScript: useful for disposable syntax experiments but likely to force an early rewrite for the durable compiler.
- LLVM-first: too much backend and toolchain surface before language semantics stabilize.
- Early self-hosting: freezes immature choices and creates a bootstrap dependency before the language can support its compiler.

## Consequences

- Go is a bootstrap dependency but not a Fairy C runtime dependency.
- C-backend correctness requires explicit helpers and differential tests against C undefined behavior.
- The first planned conformance target is `x86_64-linux-gnu`; the Windows development host is not target-layout evidence.
- Stage 0 exposes only explicit C ABI v0 and makes no stable native Fairy ABI claim.
- Multiple backends increase verification cost; they must share MIR fixtures.
- Stage 0 prioritizes semantics and diagnostics over peak generated-code performance.

## Revisit conditions

Revisit this decision if Go prevents required compiler performance or distribution targets, the C backend cannot preserve specified semantics, a fast backend lacks required ABI support, or self-hosting prerequisites are demonstrably complete.
