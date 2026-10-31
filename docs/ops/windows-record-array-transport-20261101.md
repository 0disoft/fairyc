# Windows repr(C) record-array transport receipt, 2026-11-01

- Source implementation: `e5b03f3f` (local commit).
- Probe delta: the commit containing this receipt adds the final 4,096-element family and
  records the Windows validation below.
- Host: native Windows amd64, MSVC 14.51.36231.
- Contract owners: [ADR 0488](../adr/0488-repr-c-record-array-transport.md) and the
  [C ABI v0](../abi/c-abi-v0.md#c-representable-types) record-array paragraph.
- Current scope: this receipt supersedes the record-element-array portion of the historical item
  3 aggregate matrix only. The existing item 3 documents and their historical 116/122 inventory
  remain unchanged.

## Final executed evidence

| Check | Final observed result |
| --- | --- |
| Complete foreign-struct probe package | Pass in 19.670s. Only `TestBidirectionalAggregateLinux` was skipped by the user-requested Linux deferral. |
| Windows and POSIX emission inventory | 133 portable and 139 POSIX record shapes; 428 and 448 generated functions respectively. |
| Native bidirectional Windows matrix | Four passes: MSVC C11/C17 at low/optimized, with separate `caller.obj`/`callee.obj` links. |
| Native record-row volume | 133 records x 3 independently varied value sets = 399 record-case entries per execution. |
| Forward record-array entry | `TestReprCRecordFieldNativeCrossing` passes in 5.524s. Each of four executions runs 12 record-array cases per compilation profile, including the 4,096-element family. |
| C-to-generated-Fairy and trap path | The fixture bridge enters the actual generated Fairy body. Two assertion-trap executions pass, with status 134, trap-channel control byte 6 and empty stdout/stderr. |
| Compiler/package suites | Full tests pass for seven packages: HIR, semantic, MIR, MIR codec, module interface, package artifact and C backend. Compiler coverage is limited to the `(Array|Struct|CEnum|Transport)` plus `^TestCRecordArray` filter; no full compiler-package run is claimed. |
| Scoped vet | `go vet -mod=readonly ./internal/foreignstructprobe ./tools/dev/fairyc-windows-host-smoke` passes. |

The four final record-array families are a small record, a mixed `c::long`/`f64`/nested-leaf
record, an element with `c::int[2]`, `f32[2]`, explicit C enum `[2]` and nested `@repr(C)` leaf
`[2]`, and a 4,096-element record array. Each family rotates three independently produced value
sets. The large case covers real native structure-array stack transport and copy behavior. It does
not create a source export or admit general aggregate callbacks.

## Failed expectation discovery and bounded reruns

The first package run after adding record arrays was not a clean pass: stale tests still rejected
record arrays and expected the old 116/122 inventory. After changing the rejection to a
`non-repr(C)` element and admitted element/container records to `@repr(C)`, the interim expected
inventory became 129/135 and the limited emission/negative rerun passed. Adding the 4,096-element
family produced 133/139; the final complete package pass supersedes that transient expectation
failure and is the evidence used above.

## Compiler, layout and identity checks

The record-array tests cover lengths 1, 3 and 4,096; elements containing exact C scalars, Fairy
floats, explicit C enum arrays, nested record arrays and admitted field arrays; and target-specific
`c::long` layout. The Packet assertions expect 20 bytes under Windows LLP64 and 40 bytes under
Linux LP64. MIR mutations for `non-repr(C)` elements, empty elements, cycles and element
operations are rejected. Public-interface checks distinguish nominal element types, lengths and
field shapes in identity hashes. The package cache-context rebind changes the package identity from
3 to 19 and passes.

## Reproduction

From the repository root on Windows:

```powershell
go test -mod=readonly ./internal/foreignstructprobe -count=1 -v
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestReprCRecordFieldNativeCrossing$' -count=1 -v
go vet -mod=readonly ./internal/foreignstructprobe ./tools/dev/fairyc-windows-host-smoke
git diff --check
```

## Boundaries and omissions

Native Linux, WSL discovery and execution, a full `go test ./...`, hosted CI, push, publication
and deployment were not run. The POSIX 139-shape inventory and 448 generated functions are
emission-only evidence on Windows. There is no new SDK or source-export API; serialized format,
dependency, workflow and runner surfaces are unchanged. The language and C ABI subset is extended
by this record-array slice; bare foreign arrays, direct array-of-array elements, new nonplain
element operations and nested stores for the new nonplain shape remain closed.
