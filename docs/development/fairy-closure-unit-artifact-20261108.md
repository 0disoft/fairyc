# ADR0551 closure component artifacts, 2026-11-08

Authority: ADR0551. This slice adds detached structural cleanup/body components.
It does not extend Program/project/source-bound envelopes, Complete, source
identity, cache admission, executable publication, or native pointer transport.

## Private formats and APIs

- `FYCLEAN0` and `FYCLEAN1`: unchanged project bit and existing record tables.
  Old `encode`, `decode`, `decode_aggregates` and artifact constructors remain
  closed to closure mode. No-closure records use their original serializer.
- `FYCLEAN2`: project bit, aggregate bit, then the existing type/glue/frame/field/
  operand/drop records, in the same order and widths. Both bits are canonical
  eight-byte 0/1 words and independent. Closure mode is implicit and requires
  a real canonical kind23 leaf. A scalar-only table cannot select this family.
- `FYBODY00`: unchanged fields, tables, text and existing inline suffix policy.
  Its old APIs reject any closure cleanup context.
- `FYBODY01`: exactly the existing body fields and record layout under the new
  magic. Its cleanup context must be closure mode with a canonical kind23 leaf.
  Signature-only consumers are allowed; a literal construction is not required.

Both codec modules expose explicit `encode_closures` and `decode_closures`.
Cleanup decode restores independent project/aggregate flags and resets producer
work. `clean::assemble_closure_artifact(..., project, aggregates, work)` is the
explicit fresh cleanup constructor; the old constructors do not gain admission.

Body `decode_closures(raw, context, frame, work)` takes an immutable structural
`ir::Program` context and checks the decoded body's helper/capture/invocation
proof against it, including leaf-local initialization. This context is not a
source authentication claim. Frame mismatch, old/new magic cross-use,
noncanonical words, trailing data and the existing bounds/work failures reject.

Whole-Program restoration may instead call
`decode_closures_component(raw, cleaned, frame, work)` to break the dependency
cycle between individual bodies and their helper bodies. This staging API
returns only a Body after the existing local structural body check. It does
**not** establish cross-body helper, callable permission, storage, or source
proof. The consumer must collect all decoded components and call
`ir::assemble_artifact_closures(...)`, which resets producer work and runs the
full existing Program verifier, including helper purity, callable contracts,
construction/capture identity, invocations, initialized leaf locals and storage.
No partially assembled Program is returned on failure. The strict contextual
decoder remains available when a verified structural context already exists.

Serialization and record parsing loops are shared between format versions.
There is no duplicated expression/statement serializer, raised arena/byte/work
limit, warning change, timeout change, dependency or runner change.

## Acceptance fixtures

`tests/closure_unit_artifact.fy` and
`internal/cli/candidate_closure_unit_artifact_test.go` define:

- Thirteen original-source closure cases: the eleven existing accepted controls,
  a return-only factory and a return-only leaf copy. An additional scalar source
  control follows the same parser -> checked bodies -> lowering pipeline.
- Fresh cleanup decode, staged body decode and full Program reconstruction,
  zero producer histories, structural verification, and exact component reencode
  equality. The old full constructor rejects closure reconstruction; the new
  constructor rejects scalar cleanup mode.
- Independent cleanup byte oracles for all eight scalar/closure x project x
  aggregate combinations. Independent 754-byte FYBODY00 return-42 golden words
  preserve the existing scalar body bytes, alongside scalar roundtrip checks.
- Serialized helper-target, snapshot-token, unchecked invocation, invented
  provider identity and uninitialized leaf-local mutations. Return-only controls
  exercise initialization refusal without an invocation. The strict decoder is
  checked independently before the staging/full-assembly rejection path.
- Unsafe callable metadata is introduced after body staging, so the final
  Program constructor must independently reject the forged helper contract.
- Canonical leaf/feature mismatches, bit/word corruption, old/new cross-use,
  truncation, trailing bytes, frame mismatch, 4097 types, 16385 nodes and exact/
  one-over encode/decode shared-work boundaries. Existing byte and arena limits
  continue to belong to the shared wire/parser/proof implementations.

## Validation status

With `GOMAXPROCS=1`:

- `go test -mod=readonly ./internal/cli -run '^TestCandidateClosureUnitArtifactSource$' -count=1 -v -timeout=240s`:
  final staging API revision PASS, 14.79s (package 15.026s), including thirteen
  closure source controls and the scalar control. The earlier complete fixture
  source check also passed, 12.97s (package 13.176s).
- The source acceptance parses/types/checks the authored test program and its
  import closure. It does not execute the Fairy byte/forgery assertions.
- Static comparison against `96db6ff1`: cleanup/body record serialization and
  parsing regions are byte-for-byte unchanged in all four codecs.
- `git diff --check` on the six tracked assigned files: PASS. Explicit trailing
  whitespace/LF/final-newline checks on the three new files: PASS. All nine
  assigned files use LF and a final newline; the Go test file is gofmt-clean.
- Two earlier source attempts failed: unsupported Go-produced string escapes
  and inferred i32 oracle offsets. Both fixture issues were corrected without
  changing compiler policies.

Parent native validation: `TestCandidateClosureUnitArtifactNative` PASS,
151.18s. Both low and optimized compiler fixtures used the existing two-CL
partitioned host builder, 67 parts, one-minute per-command limit and 16MiB stack.
All thirteen closure controls plus scalar bytes, exact roundtrips, forged-record
and work-bound assertions executed with exit 0 and empty stdout/stderr.
Log: `.cache/closure-unit-artifact-restore-parts-20261108.log`.
Final Source recheck also passed in 12.22s.

The earlier CLI `run` fixture passed, while its monolithic
`run --release --locked` compiler build exceeded the unchanged 60-second limit.
That failure remains in `.cache/closure-artifact-project-native-20261108.log`;
the final harness uses partitioning to test component semantics independently of
monolithic optimization time. It does not establish optimized whole-CLI or
cache acceptance. Fresh-process transport, generated input-program execution,
Linux, whole-candidate/self-host and hosted checks remain separate work.

Only the assigned six model/codec files, the new Fairy/Go fixtures and this
receipt are edited. Existing staged closure guards/proof are parent-owned;
this slice performs no staging, commit, push or other remote action. No
normative source-of-truth document, public API/ABI, DB, dependency, runner,
workflow, lockfile or repository-hygiene policy is changed.
