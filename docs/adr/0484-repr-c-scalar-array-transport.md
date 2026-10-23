# ADR 0484: Exact C-scalar array transport

- Status: Accepted for prototype
- Date: 2026-10-23
- Contract: [C ABI v0](../abi/c-abi-v0.md)
- Extends: ADR 0483's bounded whole-array transport

## Decision

Admit `1..=4096` element fields of each exact `c::*` scalar in nonempty acyclic `@repr(C)`
records. The selected target must provide that scalar. In particular, `c::long` follows LP64 or
LLP64 rather than the compiler host, and POSIX `c::ssize_t` remains unavailable on Windows MSVC.
`c::bool` uses C `_Bool`; this does not admit Fairy `bool` array fields in C records.

Reuse `TransportOnlyArray`, existing array wrappers and element-copy lowering. Whole-field reads,
inferred/typed locals, internal by-value arguments/results, record reconstruction/copies, nested
reads, and direct whole-field replacement through mutable records/references follow ADR 0483.
Array identity retains the exact element and length through HIR, MIR, codecs and public interfaces:
`c::int[N]` is distinct from `i32[N]`, as are the C and Fairy floating arrays.

Element literals, indexing, mutation, views, iteration, implicit conversions, bare foreign array
parameters/results, direct private/resource/tagged payload arrays and new pointer shapes remain
outside this admission. Existing scalar conversions do not imply array conversions.

The existing layout engine selects `KindCScalar` for array elements and generates provider and
record layout assertions. This adds no runtime helper, opcode, dependency, cache format, SDK
export, CLI flag or workflow. Target maturity is unchanged.

## Evidence

- HIR shape tests cover all sixteen identities and both length boundaries.
- Compiler tests retain identity through MIR encoding, check both target layouts (including
  different `long[3]` offsets/sizes), reject Windows `ssize_t`, and preserve operation boundaries.
- Public-interface tests distinguish element identity and length, including representation-equal
  C/Fairy scalar pairs.
- The shared separate-object C oracle covers three value rotations for fifteen portable scalar
  families, including signed minima/maxima, unsigned maxima and floating extrema. It observes
  nested copies, typed/inferred locals, internal array calls, direct/referenced replacement and
  unchanged original values. Length-one records exercise small value crossings as well.
- The Linux runner additionally includes `ssize_t`; its generated C and source checks pass on
  the local Windows host, but native Linux execution is not claimed by those checks.
- MSVC 14.51.36231 passes C11/C17 at low/optimized profiles: four executions, 45 C-scalar-array
  cases each, expected exit 42 and no output. Existing aggregate probes run in the same unit.

The affected HIR, semantic, MIR, MIR-codec, module-interface, package-artifact and shared-probe
suites pass. Focused compiler and Linux-profile emission checks pass. Hosted Linux results from
F2-6 predate this change and do not attest it.
