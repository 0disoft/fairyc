# Go copied-closure package formats (2026-11-08)

Implementation receipt for the private format gates required by
[ADR 0551](../adr/0551-copied-closure-values.md). This does not change source
syntax, native ABI, closure limits, or executable admission rules.

## Exact format selection

Existing default APIs choose the legacy schema/version when their value has no
copied-closure feature. Values containing copied closures choose these separate
envelopes; each closure family starts at version `/1`:

| Value | Schema | Version |
| --- | --- | --- |
| Process-local package | `fairyc.package-mir-closure-process/v0` | `fairyc-package-mir-closure-process/1` |
| Storage process-local package | `fairyc.package-mir-storage-closure-process/v0` | `fairyc-package-mir-storage-closure-process/1` |
| Stable function body | `fairyc.package-function-closure-body/v0` | `fairyc-package-function-closure-body/1` |
| Stable package | `fairyc.stable-package-closure-mir/v0` | `fairyc-stable-package-closure-mir/1` |
| Stable storage package | `fairyc.stable-package-storage-closure-mir/v0` | `fairyc-stable-package-storage-closure-mir/1` |
| Stable type context | `fairyc.package-type-closure-context/v0` | `fairyc-package-type-closure-context/1` |

Both readers and writers select the expected format from the decoded value.
Legacy envelopes containing closure features fail. Closure envelopes containing
no closure features also fail. Mixed schema/version pairs and unknown versions
fail. Storage and ordinary packages retain separate formats.

The existing process identity prefixes and identity values are unchanged by
closure selection. The existing ContextID representation remains unchanged;
its digest covers the selected canonical context envelope. No-closure contexts
therefore retain their bytes and IDs. Existing no-closure callback records
retain the legacy envelopes and their exact canonical bytes.

## Integration helpers

`internal/packageartifact/closure_value_formats.go` provides:

```go
func bodyFormat(body StableBodyArtifact) (schema, version string)
func stableArtifactFormat(artifact StableArtifact, transfers bool) (schema, version string)
```

The latter delegates to the existing `stableArtifactContract(transfers)` for
feature-free values. Body and stable-package readers/writers call these
helpers. `contextFormat` stays internal to context encoding/decoding.
`ProcessCodec.artifactContract(artifact)` selects the exact process format and
the existing ordinary/storage verifier; `contract()` retains its previous
feature-free behavior for existing internal callers.

Process selection examines owned functions, imported signatures (including
`ClosureTarget` and `ClosureCaptures`), signature descriptors, MIR context, and
stable context. Stable package selection examines bodies and stable context.
Body selection uses the structured MIR feature detector.

ContextDescriptor holds private stable type strings. Its detector recognizes
the exact `fn i32(i32)` HIR leaf, parsed compound children, private `ref`/`ptr`
indirection, and private `T[N]` arrays with the existing positive uint64 length
rule. HIR normally uses `[T; N]`, so private arrays need explicit structural
unwrapping. The detector never grants a feature based on a substring, logical
function name, or C callback spelling. Wrapper detection only selects an
envelope; it does not admit wrappers into executable MIR.

Artifact, stable identity, context, and MIR validation still run. Selecting a
closure version does not authorize native pointers, targets, captures, FFI, or
unsupported transport. Stable closure-create records use the parent
implementation's `NoSymbol` plus stable FunctionID in `Variant`; restoration
and target/source validation remain with the body and compiler owners.

## Windows verification

Executed with `GOMAXPROCS=1`, `-p=1`, `-parallel=1`:

```powershell
go test -p=1 -parallel=1 ./internal/packageartifact -run '^TestClosureValueFormats' -count=1 -v
go test -p=1 -parallel=1 ./internal/packageartifact -count=1
```

The first focused run found the missing private `fn i32(i32)[2]` array
unwrapping. After repair, the complete package suite passed (`2.215s`). Its log
is `.cache/closure-format-package-suite-attempt2.log` in the shared detached
worktree. That suite includes the new format tests and the parent's updated
closure projection/rebind tests.

New format tests cover ordinary functions, C callbacks, actual source-created
closures, all six codec families, exact canonical bytes and size boundaries,
zero limits, whitespace, unknown fields, trailing JSON, downgrade/upgrade,
schema/version mismatches, empty-feature closure envelopes, dormant context
types, owned/imported target/capture metadata, wrapper detection, maximum and
invalid private array lengths, process identity isolation, storage isolation,
and retained MIR validation for forged capture metadata.

`gofmt` and scoped `git diff --check` passed. No native compiler host, Fairy
source/lock, main checkout, queue, budget, runner, or CI change was made by this
format sidecar. Native execution and workspacecompiler cache integration are
separate parent/worker lanes. Commits, push, and publication are parent-owned.
