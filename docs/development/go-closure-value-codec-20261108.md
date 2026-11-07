# Go closure value MIR codec, 2026-11-08

Isolated checkout:
`C:/Users/cherr/Documents/workspace/zerodi-wd1/projects/hobby/opensource/fairyc/.cache/closure-values-20261108`,
baseline `b4ac33904430281b81f366db21154570496f5165`.

## Exact private format gate

`mircodec.ClosureMIRVersion` is exactly `fairyc-mir-closure-values/1`.
Encoding first runs the existing MIR verifier, then selects this version only
when the parent-owned `mir.HasCopiedClosureValues(program)` reports a structured
feature. There is no codec-local detector or source-text heuristic. Feature-free
MIR keeps `fairyc-mir/2`, its existing schema and canonical JSON bytes.

Decoding selects the expected version from the decoded structured program before
verification. Closure records under any legacy version, unknown closure versions,
and the closure version on a feature-free program are rejected. It then verifies
the program and compares the entire input with its canonical re-encoding. Unknown
fields, trailing values and noncanonical bytes retain their existing rejection
paths. Caller-supplied positive byte limits apply unchanged.

Both ordinary and storage-transfer APIs use this shared version gate while keeping
their distinct schemas and existing verifier capabilities. The public addition is
the exact version constant; existing encode/decode function signatures do not
change. This worker does not edit MIR models, lowering or verification.

The shared detector follows `hir.Type.Descriptor().Element` and `.Failure`, not
reparsed spelling. Tests exercise direct types, Option, both Result branches,
shared/mutable borrows, pointers, slices, arrays, nested compounds and a compound
with noncanonical wrapper whitespace. Coverage includes aggregate identities and
fields/payloads, foreign/internal signatures, function resource locals/drop glue,
operation result/owner/index/operand types, and terminator return/switch/call types.
Unused targets and type-only signatures also require the closure version.

## Portable records and separate artifact admission

The MIR wire retains source-independent target symbols, exact function metadata,
capture operands and invocation operands/control flow. No native invoke pointer,
environment address, process-local closure handle or HIR intern handle is encoded.
The 0/4-capture tests preserve all original records, including i32 minimum/maximum
capture values, and invoke the decoded MIR through the actual Go interpreter.
Unknown native-pointer fields and hidden pointer bits in capture operands fail
decoding/verification respectively.

The parent-owned packageartifact legacy gate explicitly rejects copied closure
values. This new private MIR version does **not** admit them to the existing
package/cache format. Future stable helper-id cache records, fresh-process
rebinding, source-body validation and deliberate package format versions remain
separate parent work. There is no packageartifact edit or cache-delivery claim
in this sidecar.

## Owned changes

- `internal/mircodec/codec.go`
- `internal/mircodec/closure_values_test.go` (new)
- `internal/mircodec/closure_value_versions_test.go` (new)
- `docs/development/go-closure-value-codec-20261108.md` (new)

The source/type-retention handoff remains frozen. No Fairy, MIR production,
packageartifact, main-tree, ADR, lock, queue, database, dependency, runner or
repository-hygiene configuration is changed by this sidecar. No commit, native
full build, Linux or remote action is performed.

## Validation receipt

Windows AMD64, `GOMAXPROCS=1`. All test attempts retain their original logs in this
checkout's `.cache/` directory.

- `closure-value-codec-attempt1.log`: initial round-trip and forged-MIR subset
  **PASS**, package 0.080s. Both ordinary/storage codecs round-trip 0/4 captures,
  invoke decoded MIR with result 42, enforce exact byte-size/canonical limits,
  reject native-pointer JSON fields and reject 15 forged shapes on encode/decode.
- `closure-value-codec-attempt2.log`: expanded suite **FAIL**, package 0.186s.
  The new version-gate matrix exposed missing aggregate identity, function
  resource-local and drop-glue traversal in the initial parent detector. The
  first C callback byte control also incorrectly used an internal callback
  parameter, rejected by the existing verifier (V1020). Neither failure was
  treated as a pass or bypassed.
- `closure-value-codec-legacy-attempt3.log`: corrected legacy byte controls
  **PASS**, package 0.040s. The callback control uses its existing final foreign
  callback/user-data pair. Independently reconstructed legacy envelopes omit new
  function metadata explicitly and compare complete bytes for scalar, pointer
  and C callback programs in both schemas.
- `closure-value-codec-final.log`: complete `internal/mircodec` package
  **PASS**, 10 top-level tests, package 0.158s, after the parent fixed the shared
  detector. The structural gate matrix covers 20 locations times 11 type
  shapes times both codecs (440 cases), plus four feature markers per codec.
  These deliberately malformed/dormant records must fail at the version gate
  under the legacy version; the test does not grant wrapper execution. Existing
  deterministic, size/invalid-input, incompatible/unverified, signed-i64 extreme
  and all-256-literal-byte regressions pass in the same final run.
- `go vet -mod=readonly ./internal/mircodec`: **PASS**, no diagnostics.
- `gofmt -l` on the three owned Go files and scoped `git diff --check`: **PASS**.

Exact commands (the first two restricted runs precede the complete final run):

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/mircodec -run '^TestClosureValueCodec' -count=1 -timeout=3m -v
go test -mod=readonly ./internal/mircodec -run '^TestClosureValueCodecLegacyBytesStayExact$' -count=1 -timeout=3m -v
go test -mod=readonly ./internal/mircodec -count=1 -timeout=3m -v
go vet -mod=readonly ./internal/mircodec
gofmt -l internal/mircodec/codec.go internal/mircodec/closure_values_test.go internal/mircodec/closure_value_versions_test.go
git diff --check -- internal/mircodec docs/development/go-closure-value-codec-20261108.md
```

Frozen owned Go source SHA-256:

| File under `internal/mircodec/` | SHA-256 |
| --- | --- |
| `codec.go` | `B2C91246AB7B868BC51A72119E1395E4A6198CA45C4C3F8AC17DA5A805797A8D` |
| `closure_values_test.go` | `4DC370713AFEDFE2FC21803FB095F8A9FD07E1D50C01A53DEE2C0DC54C6E2403` |
| `closure_value_versions_test.go` | `921B6AFAF7888B08D78AD9DE1B1E517CC8BCD1F8491AF5EB8843AC8249A3BE6D` |

The passing shared detector dependency `internal/mir/closure_value_features.go`
has SHA-256 `B2A144D22736F572B8B7A3D74B9E03827CAF10102C5F371ADCBBB6ACA202CD54`.
The closure verifier dependency `internal/mir/closure_values.go` has SHA-256
`B7A99F8976C9EF9A8BEBBA135DD50E346119624AE6D69951159071B5D2B9E8E6`.
Both are parent-owned and were read, not edited, by this worker. Final tests
have completed; no process/session remains active and no codec blocker remains.
Full CLI/native builds and package/cache execution are outside this codec-only
assignment and were not run.
