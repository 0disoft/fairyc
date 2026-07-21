# ADR 0133: Add deterministic Arena allocation-failure probes

- Status: Accepted
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Related decisions: `docs/adr/0042-single-active-region-arena-c-slice.md`,
  `docs/adr/0086-windows-msvc-arena-region-lifecycle.md`,
  `docs/adr/0132-deterministic-owned-text-allocation-failure-probe.md`

## Context

One nonempty Arena placement needs two fallible SystemHeap allocations: payload storage followed by
private tracking-node storage. The implementation promises not to publish either allocation until
both succeed, and it must release the payload if the node allocation fails. Ordinary execution
cannot reliably force either exact branch. The generated Fairy Windows example also cannot
dereference its returned raw pointer, so it proves lifecycle transport but not zero-filled bytes.

The existing raw Arena probe already observes bytes and lifecycle state. Extending that closed probe
with the generic SystemHeap probe adapter gives deterministic local evidence without changing the
product allocator, Arena layout, public source surface, or standard-library ABI.

## Decision

1. The shared `stage0_system_heap_probe_allocator.c` adapter may fail the next allocation immediately
   or after exactly one successful allocation and reports its live allocation count. It remains a
   probe-only prefix and is never selected by product support resolution.
2. `ArenaProbeImplementationSource` and `WindowsMSVCArenaProbeImplementationSource` prepend that
   adapter to the portable or target-specific Arena dependency closure. The Windows closure retains
   the x64 MSVC 16-byte fundamental-alignment contract.
3. The raw probe forces payload-allocation failure and requires the Arena head plus caller output to
   remain unchanged with zero live allocations.
4. It then forces tracking-node allocation failure and requires the already-created payload to be
   released, the Arena head plus caller output to remain unchanged, and the live count to return to
   zero.
5. A following unforced placement must succeed, publish two live allocations, expose eight zero
   bytes, and release both allocations on Region reset. Active drop, stale Region rejection,
   generation transitions, copied placement, and final drop remain in the same probe.
6. The native Linux runner consumes the instrumented portable Arena implementation when its glibc,
   Clang, and GCC lane runs. The native Windows smoke compiles the probe caller and target-specific
   implementation separately and executes both low and optimized x64 MSVC profiles without WSL.
7. The exact `fy_std_region_copy_bytes_v0` symbol joins the closed Linux and Windows Arena resolver
   contracts. This admits the existing implementation entry point; it does not stabilize a public
   Fairy API or representation.
8. `fairyc.native-c-report/v31` is not bumped. The existing `arenaExecutions` success condition is
   strengthened without adding or changing a report field or counter.

## Consequences

- Windows now has direct byte-level Arena zeroing, two-stage allocation rollback, leak-count,
  simultaneous/stale Region rejection, and active-drop preservation evidence at both local MSVC
  profiles.
- The generated Fairy lifecycle example still cannot inspect payload bytes and remains a separate,
  narrower source-transport proof.
- The adapter is single-process test instrumentation, not thread-safe fault injection, a general
  heap leak detector, allocator substitution, or production behavior.

## Verification

- Go source-contract tests require exact symbol admission, dependency closure, deterministic source
  construction, adapter-before-fallback ordering, target alignment injection, and absence of probe
  controls from product source;
- the raw C caller verifies both forced failure points, unchanged outputs, live-allocation counts,
  recovery zeroing, reset release, stale-use rejection, active-drop preservation, and final cleanup;
- the Windows host smoke inspects caller and callee COFF objects plus both linked PE/unwind images and
  executes low and optimized profiles with native x64 MSVC;
- the configured Linux native lane performs the same probe when a glibc Clang/GCC environment is
  available; it is not executed on this Windows-only development host.
