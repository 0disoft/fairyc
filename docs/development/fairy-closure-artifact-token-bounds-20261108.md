# Closure artifact token bounds, 2026-11-08

The structural closure proof formerly compared a stored token with
`helper_flag + declaration` or `snapshot_flag + local.source_index`. Those IDs
can come from decoded body records. A malformed usize near its maximum could
overflow that checked addition before the decoder returned error 1001.

The proof now checks that the token is at least the relevant flag, then compares
`token - flag` with the supplied ID. Short-circuit evaluation guards subtraction.
This preserves every valid token/ID pair without adding an arbitrary source-ID
cap, changing a format, or weakening helper/capture verification.

`closure_unit_artifact.fy` mutates the actual FYBODY01 constructor target and
snapshot-local source-index words. It checks the last nonoverflowing sum, first
overflowing sum and maximum usize for both ID classes. Contextual body decoding
must return 1001 rather than terminating the fixture. Existing thirteen closure
controls, scalar bytes, roundtrips, forged records and work boundaries still run.

Executed Windows validation in the stable detached worktree:

- `TestCandidateClosureUnitArtifactSource`: PASS, 14.60s.
- `TestCandidateClosureUnitArtifactNative`: PASS, 180.21s; both low/optimized
  fixtures compiled 67 parts with the existing two-CL host builder, one-minute
  command limit and 16MiB stack. All assertions executed with exit 0 and empty
  stdout/stderr. Package total: 195.004s.
- Log: `.cache/closure-artifact-overflow-final-20261108.log`.
- Final main candidate `check` and `check --locked`: PASS, 775 sources and
  9,745 MIR functions. `go vet -mod=readonly ./internal/cli`: PASS.

The pre-fix arithmetic risk was identified by code inspection; a separate
pre-fix native crash reproduction was not run. The recorded runtime result is
the corrected decoder rejecting the malformed boundary inputs.

No normative document, public API/ABI, DB, dependency, runner, workflow or
timeout policy changed. Whole CLI/cache, Linux and hosted CI remain separate.
