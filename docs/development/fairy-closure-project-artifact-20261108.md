# Source-bound copied closure project artifacts, 2026-11-08

Authority: ADR0551. This continues the bounded `fn i32(i32)` family through
project transport and the current-source Complete restoration factory.

## Exact private format and admission

The explicit `module_project_artifact_encode::encode_closures` and
`restored_closures` APIs write `FYPRJ002`. Its three length-framed members are
`FYPROG02`, `FYPROV00` and `FYIDS000`. The source member retains the exact
current-source digests, package coordinates and import bindings. The identity
member retains canonical source specialization identities. The nested Program
codec independently rechecks the complete closure proof.

`preflight_closures` validates all framing before nested owners are allocated.
`decode_closures` requires an actual project with the closure feature bit;
wrapping a unit or changing only an outer marker cannot grant project admission.
Existing scalar/aggregate APIs select their original markers and reject closures.
No public language ABI or native pointer is serialized.

`project_compiler_core::restore_complete_closure_sources` then rebinds the saved
source provenance against current parsed modules, package owners, imports and
coordinates. Its explicit aggregate policy must agree with the artifact. It
derives current signatures and the source plan, rebases the saved graph and
enters the existing body/call/cleanup/metadata/whole-project coverage, state,
SDK, borrow-observation and interface gates before returning Complete.
It does not cold-lower a replacement graph to manufacture a warm result.

Rebasing preserves the closure feature in both cleanup and Program assembly.
The fresh full proof runs after assembly. A kind4 value call has zero static
provider coordinates even when module zero moves to another input position;
only ordinary static call targets are rebased. Kind36's target remains a
source helper declaration ID.

The existing byte and work limits remain unchanged. Decode, source rebind,
source recipes, reassembly and publication continue using their caller-owned
independent ledgers. No prefix or partially admitted Complete is published.

## Authored acceptance

`tests/closure_project_artifact.fy` and
`internal/cli/candidate_closure_project_artifact_test.go` use the seven original
multi-module closure controls already checked by the independent Go interpreter.
They produce Complete, encode/decode the project envelope, restore against the
current originals, then emit single and partitioned C from that restored owner.
The existing native consumer harness checks both low and optimized outcomes:
42/84 results, unused trapping helpers and trap ordering.

Additional assertions cover independent outer framing literals, scalar-family
roundtrip, old-family refusal, all 56 minimal-envelope truncations, trailing
bytes, incompatible outer/inner markers, exact decode-work exhaustion and one
unit short, stale source comments, module reorder and zero provider coordinates.
A separate splice replaces the Program with a detached-valid different helper
body while preserving the original source/identity members; source admission
must reject it despite successful detached decoding.

Windows Source passed in 21.81s (package 22.151s), following the initial 21.89s
pass before the additional framing/splice assertions. Native passed in 99.60s
(package 99.827s): the low-profile partitioned compiler fixture executed the
envelope/source assertions, then all 28 generated consumers executed in low and
optimized profiles, single and partitioned C. This is not an optimized-profile
compiler-fixture result. Log: `.cache/closure-project-artifact-native-20261108.log`.
The later body reachability-meter correction changes no valid source matching
or serialization bytes; its exact-work runtime acceptance is recorded in the
separate source-recipe receipt.

Source checking alone does not execute these assertions. Persistent project cache,
stable CLI selection, complete candidate self-host, Linux and hosted CI remain
separate delivery boundaries. No remote action is authorized by this receipt.

Only private compiler APIs and development receipts change. Normative documents,
public SDK/ABI, DB, dependencies, repository hygiene, timeouts, warning policy,
CI workflows and runner configuration are unchanged. Linux is deliberately
batched; no push, release, hosted execution or deployment was performed.
