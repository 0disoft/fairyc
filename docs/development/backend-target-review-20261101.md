# Backend and additional-target review, 2026-11-01

Decision owner: [ADR 0497](../adr/0497-backend-and-target-review.md).
Source inspected: `ff333d8d`; Windows amd64, MSVC 14.51.36231.

## Comparison

| Candidate | Current evidence | Integration and continuing cost | Direction |
| --- | --- | --- | --- |
| Existing verified MIR to C11 | Actual Windows/Linux receipts, checked arithmetic/traps, resource cleanup and C ABI. Windows COFF/CRT/unwind/PDB and recent generics/interface/closure probes exist. | Maintain profiled C emission/runtime and external toolchain. C parsing plus optimization remains a build cost; partitioning already exists for Linux. | Retain the product backend and create a reproducible baseline. |
| MIR to LLVM IR | Official LLVM Windows support and integer-overflow intrinsics are relevant building blocks. No Fairy MIR emitter or installed LLVM tool was found in the checked host locations. | Implement explicit arithmetic/trap semantics, layout and target data; C ABI lowering, ownership, calls, cleanup, debug/unwind, tool version compatibility and distribution remain Fairy obligations. | First future challenger via textual IR and external tools; no adoption or performance claim. |
| MIR to QBE IL | Official overview advertises amd64 Linux/macOS, arm64 and riscv64 and a compact SSA IL. No native Fairy/QBE evidence or installed QBE tool was found. | A second lowering plus verified Windows ABI/object/unwind/debug path would be required for this host. Vendor size/performance goals are not Fairy measurements. | Defer; do not adopt a private Windows fork merely to complete a chart. |

Clang compiling emitted C is still the C backend; it does not measure MIR-to-LLVM IR.
Assembly/object emission alone is not executable, runtime, ABI or debugging conformance.
A retained C backend also supplies a semantic/ABI comparison path for a later IR experiment.

## Target shortlist, not admission

| Candidate | Potential reuse | Required new evidence | Decision |
| --- | --- | --- | --- |
| aarch64-linux-gnu | LP64 and much of the POSIX runtime may carry over. | Target-owned layout, AAPCS64 aggregate/callback calls, objects/relocations, atomics, unwind/debug, sysroot/runtime and actual ARM64 execution. | Research candidate when deployment demand exists; native Linux work stays deferred. |
| aarch64-windows-msvc | Much Windows host API code may carry over. | ARM64 ABI, COFF relocation coverage, ARM64 pdata/xdata, mixed stacks, CRT/SDK, debug and real ARM64 execution. | Research candidate; installed Hostx64 tools contain x64/x86, no ARM64 compiler. |
| wasm32-wasi | Some scalar/checked MIR may be reusable. | 32-bit layout, WASI runtime, filesystem/process differences, import/FFI model, traps and execution. | Defer until a sandbox/browser/WASI product requires it. |

There is no deployment requirement selecting among them, so target choice remains UNDECIDED.
No addition to internal/target or support/maturity flags is justified by this review.

## Local inventory and measurement conditions

PATH lookup did not resolve clang, clang-cl, llc, opt, llvm-as or qbe. Common LLVM, per-user
Scoop/program and Visual Studio LLVM locations were checked without finding clang. Visual
Studio 18 Community supplies MSVC 14.51.36231 with Hostx64 x64/x86 tools. No tools were installed.
These are the inspected locations, not an exhaustive filesystem inventory.

The official QBE remote was read with git ls-remote --heads: master
`e786f06032fefa2e3790d6b1c9e31ed138f475a6`, plus dev and loopopt branches. No unseen
branch/fork behavior is inferred. The main project page is the advertised support evidence.

The manual `tools/dev/fairyc-backend-study` harness compiles the same source into verified MIR,
checks interpreter outcomes where foreign calls are absent, measures C emission and fresh native
builds at low/optimized, and binds timings to source/runtime/compiler/image hashes. Its
[Windows baseline receipt](../ops/windows-backend-study-20261101.md) records source `769d5835`
and 42 successful native executions. Expected traps and resource-drop/C-call observers remain
distinguishable from compilation failures. QBE/LLVM adapter absence is an explicit unmeasured
result. Repeated measurement is opt-in; no automatic CI matrix was added.

Only a representative compiler/parser or real application showing a useful paired benefit,
with semantic/ABI parity, measured code size and a manageable toolchain/distribution cost, may
reopen adoption. Tiny fixtures and advertised optimizer performance cannot establish that benefit.

## Primary sources inspected

- [QBE overview and advertised targets](https://c9x.me/compile/).
- [QBE IL](https://c9x.me/compile/doc/il.html) and [official repository](https://c9x.me/git/qbe.git).
- [LLVM Windows setup](https://llvm.org/docs/GettingStartedVS.html).
- [LLVM frontend performance guidance](https://llvm.org/docs/Frontend/PerformanceTips.html).
- [LLVM arithmetic overflow semantics](https://llvm.org/docs/LangRef.html#arithmetic-with-overflow-intrinsics).
- [Clang cross compilation and sysroot requirements](https://clang.llvm.org/docs/CrossCompilation.html).
- [Microsoft ARM64 ABI](https://learn.microsoft.com/en-us/cpp/build/arm64-windows-abi-conventions).
- [Microsoft ARM64 exception handling](https://learn.microsoft.com/en-us/cpp/build/arm64-exception-handling).

These support capability/cost analysis; the direction above is a project decision, not a
vendor conclusion or a local LLVM/QBE performance result. Native Linux, new-target execution,
full repository tests, hosted CI and remote publication are outside the current evidence.
