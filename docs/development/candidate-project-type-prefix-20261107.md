# Project type prefix mapping

Root code commit `acd2cda3` integrates worker commit `2bfd4575`.
This implementation/evidence note changes no source-of-truth contract, source
policy, public function signature, C/SDK ABI or IR wire format. The existing
256-input, 4096-canonical-type and 1048576-work limits remain unchanged.

## Retained mapping and validation

The compiler-private Plan records the shared signature-prefix length and
retains map entries only for each input's suffix. Every input still undergoes
the complete prefix comparison and `signatures::own_types` validation before
any descriptor edge is used as an index. The first fully validated unit seeds
the global canonical graph in order, so prefix IDs equal global IDs. Later
units can return those IDs directly while interning and mapping their suffixes.
The final global graph still passes `signatures::own_types`; nominal and SDK
coordinates, edge handling and malformed-input refusal remain checked.

## Recorded Windows regressions

Both Windows low/optimized profiles pass `TestCandidateProjectTypePrefix`;
the measured package duration is 23.350s (test duration 23.25s). The existing
`TestCandidateOrdinaryProjectTypes` passes both profiles in 61.337s at package
level. These are regression replay times, not normal compiler throughput.

| Units | Prefix | Charged work, old -> new | Retained maps, old -> new |
| ---: | ---: | ---: | ---: |
| 8 | 0 | 8408 -> 8408 | 72 -> 72 |
| 1 | 33 | 8315 -> 8315 | 41 -> 8 |
| 256 | 33 | 33536 -> 16706 | 8448 -> 0 |
| 64 | 129 | 34535 -> 18281 | 8768 -> 512 |
| 64 | 513 | 108234 -> 43596 | 33344 -> 512 |

The final case saves 262656 bytes of logical usize map payload on amd64,
not measured allocator memory or RSS. Full mapped-ID and descriptor-edge
equality against the full-map baseline, malformed refusal, repeated-build
results and unchanged borrowed inputs are covered. The zero-prefix case is
unchanged; a single unit saves map storage without reducing charged work.

Evidence is under
`.cache/parallel-project-type-prefix-20261107/.cache/project-type-prefix/`:
`windows-native.log` and `windows-existing-types.log`. Tested production SHA-256:
`7c538da3beaec97095a81ff7eab237c36fc3e9417dff9c88100df43c03814af9`;
fixture SHA-256:
`e56d1096e684beff3bee37a511d20caa465511c560f855e932f4a4aab79aa04c`.

## Normal build failure and unresolved diagnosis

The preceding cleanup-index normal 646-source self-build exits 2 after
551.636s at phase 3, stage 0, code 1002 (bytes 0..0), producing zero output.
Its invocation remains in
`.cache/current-selfbuild-cleanup-index-20261107/invocations.json`.
A separate complete-source diagnostic takes 527.082s, preserves all 646 inputs
without drift, produces 157 input units and reaches the same link refusal.
Its 21157 input frames include duplicates before linking; this is not a count
of unique linked frames. Evidence is
`.cache/current-project-link-focus-20261107/run.json` and adjacent stdout.

The 256-input limit is not the observed cause, and no canonical 4096-type cap
hit has been observed. Parent stage instrumentation mistagged diagnostic
31002 as `from_ir`; TypeBuild versus programInputs remains unresolved.
The generic 1002 refusal does not establish the 1048576-work cap as root cause.
This prefix optimization is not evidence that the normal failure is fixed.

The parent A/B probe `.cache/current-project-link-focus-20261107/run-ab.*`
finishes in 577.746s with no source drift. It keeps the original 646 source bytes
and order fixed, with exactly one frozenPathOverride for the old
`module_project_types.fy` data input (index 209), allowing the engine's root
implementation to advance independently. Its `ab-source-snapshot.json`
authenticates original snapshot SHA-256
`249cfdf022ff00f87185a8b4464c3fb8a80a53cd1cc7853368c11e61f5c56b45`.
The same 157 input graphs contain 345531 type rows and a 2163-row signature
prefix. The full-map baseline refuses at work 1048576 (diagnostic 22002);
the shared-prefix plan succeeds with work 373752, 5940 retained map cells
and 3359 canonical types. Linking then refuses in metadata preparation
(diagnostic 33002). That stage label does not yet distinguish metadata work
exhaustion from its independent row/name limits.
This is an explicit diagnostic comparison, not current normal acceptance.

## Integration status

Integrated source check/locked pass: 755 files, 9528 MIR functions,
13.2314665s / 13.5183886s. Evidence is
`.cache/current-project-type-prefix-source-check-20261107.log`.
The refreshed lock is integrated with this note. A fresh optimized build-only
seed and help smoke pass in 132.660s (205 C parts), recorded at
`.cache/current-project-type-prefix-seed-20261107/build.log`. The new normal
646-source retry exits 2 after 561.174s at phase 3/stage 0/code 1002, with
zero output. Its invocation is recorded in
`.cache/current-selfbuild-project-type-prefix-20261107/invocations.json`.
Normal self-build acceptance,
fixed points and full current-source parity remain pending. Native Linux is
unexecuted and batched separately; CI and clean-machine validation are also
unexecuted for this change. API, DB, SDK ABI, CI and runner behavior are unchanged.
No push or broader queue completion is claimed.
