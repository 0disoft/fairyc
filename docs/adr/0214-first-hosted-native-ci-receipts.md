# ADR 0214: Record the first hosted native CI receipts

- Status: Accepted
- Date: 2026-07-29
- Evidence commit: `0bb41990c09edf87f8fa2ffa654e0b8b0b4aab99`
- Workflow: `Frontend`
- Workflow run: <https://github.com/0disoft/fairyc/actions/runs/30442310794>

## Context

The Linux and Windows target descriptions deliberately retained `hosted-linux-receipt` and
`hosted-windows-receipt` as missing evidence after the native jobs were configured. Workflow text,
local execution, and a queued run were not accepted as hosted proof. The markers could be removed
only after GitHub recorded exit-zero native jobs for an exact commit.

## Observed evidence

GitHub Actions run `30442310794`, created at `2026-07-29T10:05:48Z` and completed at
`2026-07-29T10:10:54Z`, records success for exact commit
`0bb41990c09edf87f8fa2ffa654e0b8b0b4aab99`.

- job `90544257942`, `Go frontend (ubuntu-latest)`, passed vet and the complete Go test suite;
- job `90544258009`, `Go frontend (windows-latest)`, passed vet, the complete Go test suite, and
  the configured Windows native target smoke using the hosted runner's MSVC and SDK;
- job `90545026655`, `C differential (Linux)`, passed the configured generated-C differential,
  sanitizer, layout, ABI, and product-build checks with Clang and GCC.

The run uploaded no product artifact and performed no release or deployment.

## Decision

Remove `hosted-linux-receipt`, `c-layout-probes`, and `bidirectional-c-abi-fixtures` from the Linux
TargetSpec missing-evidence list because the exact hosted job executed all three. Remove only
`hosted-windows-receipt` from the Windows list: its current layout coverage is incomplete and its
ABI crossing is same-MSVC rather than an independent bidirectional probe. Keep `emittable`,
`conformant`, and `supported` false for both targets.

The receipt does not close the remaining runtime, standard-library, Windows-independent ABI,
object, unwind, toolchain-baseline, or provisioning gates. The `*-latest` runner labels and observed local
tool versions are execution evidence, not a stable toolchain baseline. A future change to native
runner behavior, backend/runtime code, workflow steps, or runner image must obtain a fresh green
hosted run before making a current-revision receipt claim.

## Consequences

The stable `fairyc targets` report no longer lies about the observed hosted receipts or the completed
Linux layout and bidirectional ABI probes, but it still reports both targets as unsupported and lists
every unresolved gate. This is evidence maintenance, not target promotion, ABI stabilization,
distribution, or release.
