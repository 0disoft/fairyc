# Checked package specialization membership for cache inputs

Complete package records now retain the sorted unique original-source `g0_`
membership after the existing plan/source/coordinate authentication and complete
body/Flow checking. The cache projection requires the opaque completed owner,
checks every surviving generic function/cleanup/Vec type ID against that retained
set, and returns independent owned IDs. Membership belongs to the defining
package; imported aliases do not move a provider's specialization into callers.
Types and their cleanup share one entry, and unused templates have no entry.

An initial projection scanned only lowered frames/types. Its directly executed
Go-native test failed at SDK case 2: expected 13 IDs, actual 8. Five original SDK
specializations had been lowered into intrinsics, so those identities must survive
lowering in the complete records. The failed invocation is retained in the
receipt. The final implementation preserves the original complete membership
rather than weakening the Go cache contract to the smaller lowered set.

The existing Go `AnalyzeGenericPackages` and `AnalyzeSDKStoragePackages` supply
five authored graphs and thirteen package observations with 49 IDs. The fixture
includes exact authenticated SDK source, scalar imported aliases, phantom
u16/u64 arguments, cleanup-only helper requests, scalar/owned-resource Vec,
public Vec signatures, unused templates and reversed source order. Three
authored tests match every ID in Go-native and candidate C development/release,
including detached owners, key-ready copied IDs, unknown owners and exact/
exhausted/invalid shared 16MiB work. Three selected existing complete-publication
regressions also pass all three profiles: atomic work boundaries, late body/
move/borrow failures, and stale source/alternate-coordinate rejection.

The membership API test closure contains 276 sources; its candidate emission
takes 104.76s, development execution 6.77s and release execution 0.69s. Private
candidate API images retain explicit 16MiB stacks; directly executed Go programs
retain their ordinary default stacks. These observations do not select product
timing defaults. Root and locked checks pass 457 files / 5891 MIR functions.

Regenerate the Go fixture with `tools/run-go.ts run
tools/verify-candidate-cache-specializations.go --sdk
internal/sdkstdlib/sources/optional/std/vec.fy --output <fixture>` and forward it
through the maintained private test verifier.

Receipt: [evidence](evidence/candidate-cache-specializations-20261014.json).
Next: exact workspace/SDK/source and execution-environment key inputs, then
verified reusable artifacts and actual lookup/publication. This projection
alone authorizes no cache hit. Current full fixed point, product defaults/
performance, Linux/full suite and same-head CI remain open. Public CLI/SDK ABI
and product stack/timeouts are unchanged; no push.
