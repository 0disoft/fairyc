# Windows aggregate crossing receipt, 2026-11-01

- Source code: `d558eb1f` (local commit; no push or hosted execution).
- Host: native Windows amd64, MSVC 14.51.36231.
- Owners: `internal/foreignstructprobe/bidirectional.go`, its execution tests, and the
  [C ABI contract](../abi/c-abi-v0.md#windows-aggregate-crossings-2026-11-01).

## Executed evidence

| Check | Observed result |
| --- | --- |
| Portable probe source, MIR verification and Windows C emission | Pass: 116 records, 377 generated functions. |
| POSIX probe source, MIR verification and Linux C emission | Pass on Windows: 122 records, 397 functions. No Linux execution. |
| Native MSVC C11 low / optimized, C17 low / optimized | Four passes; each processes 116 records with three C-produced value sets. Expected exit 42 and empty stdout/stderr. |
| Separate compilation | Two independent source/object receipts per native execution (`caller.obj`, `callee.obj`); final PE and unwind inspection pass. No LTO or C-body replacement. |
| C-to-generated-Fairy assertion failure | Two passes, low/optimized C11; second call marker forces the real Fairy assertion, expected status 134 and trap-channel category byte 6, empty streams. |
| Unsupported aggregate shapes | Six rejections: empty repr(C), Fairy bool, safe borrow, owning tag, record-element array and bare foreign array result. Rejection occurs in semantic checking or target C admission. |
| Generator repeat and missing signature/metadata | Deterministic repetition; missing generated Fairy function identity or prototype fails closed. |
| Focused backend regression, scoped Go vet, diff whitespace | Pass. |

Each record is constructed by independent C, passed to Fairy by value, reconstructed in typed
Fairy, passed back to the C observer, and called again from C through a fixture-only bridge to
the same generated Fairy body. A per-record marker confirms that body executed; the C observer
compares every leaf field and every array element for original, Fairy-produced and C-invoked
results. Values rotate by field and array position across the three runs. Scalar cases include
signed minima/maxima, unsigned maxima and finite floating extrema. Raw pointers include null
and nonnull addresses to static C objects. Padding bytes are not compared.

The records cover small-register and indirect/hidden-result paths, acyclic nesting, all ten
Fairy integer families, Fairy floating fields, all fifteen Windows C scalar families as fields,
integer/floating/C-scalar/explicit-enum arrays, admitted raw-pointer field targets and a 4096-byte
array. Existing forward probes also run, preserving their independently observed value changes,
argument pressure, callback and copy-isolation checks. Resource and tagged ownership remain
outside the public aggregate ABI; pointer lifetime is not transferred.

The bool rejection fixed in this code occurs before private bool-to-integer canonicalization.
Private Fairy bool fields remain unchanged. Exact C `_Bool` transport uses `c::bool` and remains
inside the positive matrix.

## Reproduction on Windows

From the repository root:

```powershell
go test -mod=readonly ./internal/foreignstructprobe -run '^Test(BidirectionalAggregate(Windows|WindowsTrap|Emission|RejectsClosedShapes)|Generate|CScalar)' -count=1 -v
go test -mod=readonly ./internal/cbackend -run 'Struct|Foreign|ReprC|Bool' -count=1
go vet -mod=readonly ./internal/foreignstructprobe ./internal/cbackend
git diff --check
```

## Remaining boundaries

The bridge is fixture C appended to unmodified generated Fairy definitions using emitted function
metadata. It does not expose a source-level export, add an aggregate callback, or stabilize Fairy
function names. This is same-MSVC native transport evidence, not an independent register oracle,
new target support or a hosted CI receipt. No foreign exception or longjmp crosses Fairy frames.
The float cases do not establish every NaN payload or signed-zero bit pattern.

Native Linux tests were not invoked, including WSL discovery. The [batched Linux queue](../development/ordered-work-20261031.md#batched-native-linux-validation-2026-11-01)
owns the future Clang/GCC cross-compiler runs. Full repository tests, sanitizers, hosted CI,
clean-machine installation, distribution and self-hosting were not run in this slice.

No public API expansion, DB, dependency, CI/runner or repository hygiene configuration changed.
No push, publication or deployment occurred.
