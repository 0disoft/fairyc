# Detached closure Program artifact assembly, 2026-11-08

This slice adds explicit `encode_closures` / `decode_closures` entry points for
`FYPROG02`. Existing `encode`, `decode` and `decode_aggregates` do not acquire
closure admission; ordinary `FYPROG00` and aggregate `FYPROG01` retain their
existing component ordering and record bytes.

The closure envelope accepts exactly `FYCLEAN2`, `FYSDK000`, `FYMETA00`,
`FYSTATE0` and frame-ordered `FYBODY01` members. Whole-envelope preflight checks
every marker, member size, count and trailing boundary before allocation.
There are no additional envelope flags. `FYCLEAN2` retains the independent
`project` and `aggregates` scope flags as canonical 0/1 words. They are preserved,
not normalized or inferred from table contents. Cleanup verification checks
scope-dependent type/frame validity, and closure artifact status requires an
actual canonical kind23 leaf. This marker alone cannot admit scalar-only data.

`decode_closures_component(raw, cleaned, frame, work)` is the frozen component
API supplied by Noether. It restores a bounded structural `ir::Body`; it does
not return an executable Program or a source-bound owner. The metadata model's
new explicit `restore_closures` path copies retained metadata and invokes
`ir::assemble_artifact_closures` only after all bodies are present. The complete
fresh owner must pass ordinary verification plus helper identity/body/signature,
initialized named callee, capture, invocation and closure storage proof. An
error returns no Program. Ordinary `restore` still rejects the closure family.

Producer and decoder work histories retain the existing aggregate ledger and
component limits. Measured nested verification is charged on both success and
failure. Fresh reconstruction retains zero producer work. The existing component
encoder may recheck the full closure Program for each body; no unchecked shortcut
or component API change is introduced by this assembly slice.

## Evidence and boundary

`closure_program_artifact.fy` authors detached roundtrip/canonical re-encoding
assertions for the accepted original-source closure controls. It also covers
old-family refusal, mixed top/cleanup/body markers, truncation, trailing bytes,
canonical scope flags, forged constructor targets, uninitialized local callees
and exact aggregate work boundaries. A 1,266-byte old `FYPROG00` oracle is built
from explicit Go record words and the existing independent scalar body oracle;
the Fairy fixture compares its entire old-family output to those bytes.

The Go Golden test checks the independent oracle width. The Source test compiles
and checks the authored Fairy fixture in a temporary workspace. Source checking
does not execute its roundtrip/forgery assertions. Native execution was reserved
to the parent; the executed result is recorded below. No source-bound Complete,
current-source identity admission, cache hit, restored execution or publication
claim follows from this detached structural codec.

Write set is limited to `module_program_artifact_{wire,encode,decode}.fy`,
`module_metadata_artifact_model.fy` and the new fixture, Go harness and this
receipt. Frozen C files, component codecs, source admission/identity modules,
staged files, dependency locks, language/SDK contracts, DB, runners and remote
state are not modified. The parent owns integration and commits.

Final checks: `GOMAXPROCS=1 go test -mod=readonly ./internal/cli -run
'^TestCandidateClosureProgramArtifact(Golden|Source)$' -count=1` passed in
13.053s (the first source check also passed in 14.463s). Scoped
`git diff --check` passed. Roundtrip, forged artifact and byte-equality assertions
inside Fairy remain authored but not executed under the parent Native reservation.
The frozen component API and independent scope-flag contract were coordinated
with Noether; the public Program codec signatures are now frozen for consumption.

## Authored Native execution entry point

`TestCandidateClosureProgramArtifactNative` now prepares the same fixture
workspace, uses `candidateParityBuildParts`, and builds both low and optimized
profiles with `BuildWindowsMSVCCompilerHostPartsWithStackReserve`. The existing
two-CL compiler-host bound, one-minute compiler command timeout and 16MiB stack
reservation remain unchanged. Windows AMD64 is required; other hosts skip.

Each image executes the generated main, including the full old-family byte
oracle and every accepted closure control's roundtrip, forgery and exact work
assertions. `candidateAtlasProcess` requires exit 0 and empty stdout/stderr.
Native execution success is logged only after that process returns. The worker
authored the test and left execution to the parent.
Only the Go harness and this receipt changed in this follow-up. Product sources
and the Fairy fixture remain frozen.

Harness follow-up validation: `GOMAXPROCS=1 go test -mod=readonly ./internal/cli
-run '^TestCandidateClosureProgramArtifactGolden$' -count=1` passed in 0.167s,
compiling the new Native test without running it. `gofmt` and scoped
`git diff --check` passed. Neither Source nor Native was rerun in this follow-up.

## Parent Windows execution

`TestCandidateClosureProgramArtifactNative`: PASS, 155.51s. Both low and
optimized fixtures compiled 68 parts with the existing two-CL host builder,
unchanged one-minute per-command limit and 16MiB stack. Eleven closure controls,
complete old-family golden bytes, FYPROG02 roundtrips, malformed marker/flag/
target/callee refusals and exact work boundaries executed with exit 0 and empty
stdout/stderr. Log: `.cache/closure-artifact-project-native-20261108.log`.
The stable detached validation worktree used the frozen integration sources.
Linux, whole CLI/self-host, source-bound project envelopes, cache and hosted CI
remain separate work. No normative document, public API/ABI, DB, dependency,
runner or workflow changed.
