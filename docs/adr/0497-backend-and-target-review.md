# ADR 0497: Retain C; measure before a second backend or target

- Status: Accepted bounded review decision
- Date: 2026-11-01
- Extends ADR 0001; [comparison and adoption conditions](../development/backend-target-review-20261101.md).

Keep the constrained C backend for both existing x86_64 profiles. The current verified
MIR, checked operations, resource cleanup, aggregate C ABI, callback, PE/COFF, unwind
and debugging receipts would all need explicit owners under a new emitter. A tool's
supported architecture list cannot replace those Fairy/runtime/ABI obligations.

LLVM text IR with external tools is the first challenger to evaluate when tools and a
representative workload are available. This is experiment priority, not backend adoption,
a new CLI selector or a promise of better latency. Avoid embedding libLLVM/cgo in the Go
bootstrap before a measured benefit. Reuse the C runtime and system linker initially.
QBE remains deferred: its advertised platform list does not establish native Windows
MSVC/COFF, SEH and debug coverage, and no local QBE tool or fixture evidence is available.
This does not assert that forks or later upstream versions cannot target Windows.

Do not add target IDs now. Review aarch64-linux-gnu and aarch64-windows-msvc first if a
real deployment requires ARM64; selection remains UNDECIDED. Linux may reuse much of the
POSIX runtime but needs AAPCS64/aggregate/ELF proof; Windows requires ARM64-specific COFF,
unwind and debugging proof. wasm32-wasi needs a separate runtime/host model. C can target
a new architecture through its compiler; a native IR backend is not a prerequisite.

Commit the decision first, then a bounded manual study harness and Windows C baseline.
Reports must separate frontend/MIR time, emission, uncached native build, executable size,
launch-inclusive execution and unavailable candidates. Unsupported adapters have no timings.
Do not infer program throughput from process launch duration or compare a C cold build with
an IR warm cache. Full compiler/parser and SDK workloads plus semantic/ABI/runtime parity
and repeated paired measurements are required before selecting a replacement.

No product backend/target, support flag, ABI, cache key, SDK/CLI API, dependency, default
build policy, repository hygiene or CI runner changes are made. Linux execution remains
in the maintainer's deferred batch; no push, dispatch or publication is authorized here.
