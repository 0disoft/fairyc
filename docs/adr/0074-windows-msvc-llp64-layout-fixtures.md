# ADR 0074: Windows MSVC LLP64 layout fixtures

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Target source: `internal/target/`
- Layout source: `internal/layout/`
- Fixture source: `spec/fixtures/layout/manifest-windows-msvc-v0.json`
- Implementation source: `internal/layoutfixture/`

## Context

ADR 0073 records Windows LLP64 facts, but a table plus unit assertions is weak evidence. The existing
layout engine and fixture runner are already host-independent and target-bound. The Linux manifest,
however, is also consumed by a Linux-only native Clang/GCC probe. Mixing Windows cases into that
file would either feed the wrong compiler path or force platform branching into one evidence ledger.

## Decision

1. `fairyc.layout-fixtures/v0` admits both declared target IDs while each manifest names exactly one
   target. The existing `manifest-v0.json` remains the Linux ledger.
2. `manifest-windows-msvc-v0.json` is a separate strictly decoded Windows ledger. Its nine cases
   prove four-byte `c.long`, four-byte `c.ulong`, eight-byte `c.size_t`, eight-byte pointers and
   `isize`, a four-byte explicitly based C enum, rejection of unavailable POSIX `c.ssize_t`, and a
   mixed LLP64 structure with exact field offsets and padding.
3. The mixed record places `u8`, C `long`, a pointer, `size_t`, and C `unsigned long` in declaration
   order. The expected result is size 32, alignment 8, internal padding at bytes 1 through 3, and
   trailing padding at bytes 28 through 31.
4. `internal/layoutfixture` executes both repository manifests through the same strict JSON and
   target lookup path. Pass-case extraction remains isolated and deterministic.
5. No Windows native probe consumes the extracted cases yet. The TargetSpec keeps
   `c-layout-probes`, `hosted-windows-receipt`, ABI, object, and unwind evidence missing.

## Consequences

- The Go host no longer needs to guess whether the Windows layout table actually drives the shared
  layout engine.
- Linux native probe identity remains unchanged and cannot accidentally absorb Windows cases.
- Agreement with actual MSVC `sizeof`, `_Alignof`, and `offsetof` remains the next layout gate.
