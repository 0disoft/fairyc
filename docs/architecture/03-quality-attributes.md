# Quality Attributes

- Status: Draft
- Measurement source: `docs/engineering/03-performance-budget.md`

## Semantic integrity

- Safe-code meaning is independent of optimization level and backend.
- Left-to-right evaluation, traps, moves, drops, and error propagation are explicit before interpretation or code generation.
- Every MIR handoff passes the verifier invariants in `docs/mir/mir-v0.md`.
- Unsafe assumptions are searchable and never inferred from debug-only assertions.
- Diagnostics are deterministic for the same source, target, language revision, and compiler version.

## Build responsiveness

- `check` avoids code generation.
- Development builds favor latency over maximum optimization.
- Release, LTO, PGO, sanitizer, full-debug, and cross-target work remain explicit profiles.
- No-op and incremental work reuse valid frontend results without retaining unbounded intermediates.
- Latency, memory, artifact, and cache claims remain measurement fields until a reproducible harness establishes a baseline and adoption threshold.

## Storage discipline

- The project directory contains intentional outputs and a small state file.
- Shared cache content is addressed by immutable inputs, deduplicated across projects, size bounded, and automatically collected.
- Generated C, assembly, full-debug objects, and obsolete compiler-version entries have low retention priority.

## Runtime performance

- The safe subset should be competitive with equivalent optimized C when checks are provably removable.
- Strong alias, non-null, alignment, ownership, and escape facts are optimization inputs only when proven by the language rules.
- Benchmarks compare equivalent algorithms, allocators, safety semantics, optimization features, and hardware conditions.

## Portability

- Target descriptions own layout and ABI facts.
- The Go host, C host compiler, and backend version cannot silently redefine Fairy C semantics.
- Target status is reported separately as `emittable`, `conformant`, or `supported`.
- A target becomes conformant only when its declared semantic, ABI, runtime, standard-library, and toolchain fixtures pass without skips; support additionally requires maintained CI, distribution, diagnostics, and provisioning documentation.
- The first planned conformance target is `x86_64-linux-gnu`; no target is currently supported.

## Maintainability

- IRs use stable typed IDs and deterministic ordering.
- Frontend, Flow CFG analysis, MIR, interpreter, backend, runtime, standard library, `SemanticSnapshot`, and Atlas boundaries are independently testable.
- Self-hosting cannot remove the bootstrap or reference path.

## Code intelligence and safety

- Atlas exports compiler facts separately from human-authored text.
- Every fact carries source and schema provenance; uncertain edges retain an explicit certainty class.
- Truncated context reports omissions instead of implying completeness.
- Source metadata cannot grant filesystem, process, network, credential, or approval authority.
