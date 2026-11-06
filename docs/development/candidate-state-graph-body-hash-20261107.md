# Detached state graph portable body identity — 2026-11-07

This bounded slice implements the exact Go one-transaction member of
`fairyc.contract-state-package-body/v1`, including a root graph, its sorted
reachable helpers, and borrow/view/copy operations. The resulting text is
`fycsbody1-sha256-` followed by 64 lowercase hexadecimal digits. No alternate
public hash scheme was introduced.

## New authored modules and API

- `module_state_graph_hash_context.fy` validates and owns portable identity data:
  `validate(ref u8[] coordinate, ref u8[] descriptor, ref mut usize work)`
  returns `Result<Context,u32>`. `coordinate` is the nonzero raw 32-byte package
  coordinate; `descriptor` is the exact Go binary stable function descriptor.
  Read-only accessors expose `coordinate`, `descriptor`, and the storage-effect
  mask. Logical names, canonical types, ordered unique effects, nonempty result
  type, optional drop-glue type, framing, and complete consumption are checked.
  Its effects are restricted to those admitted by the state graph proof.
- `module_state_graph_body_hash.fy` exposes
  `derive(ref program::Verified proof, ref context::Context identity,
  ref mut usize work) -> Result<Identity,u32>`. It refuses descriptor effects
  that differ from the verified root. `canonical(ref Identity)` returns the
  exact preimage and `digest(ref Identity)` returns its canonical text hash.
- `module_state_graph_hash_encoding.fy` writes the exact big-endian lengths,
  storage codec strings, operations, parameters, edges, exits, helper names,
  and borrow metadata. The top-level factory is the only identity publisher.

`Context` and `Identity` own their buffers and cannot be constructed or mutated
from another module. Identity validation establishes canonical portable data;
it does **not** establish that a current source function owns the supplied
body. Current-source/Complete association remains the source-binding layer's
responsibility. Neither identity object grants cache reuse, restoration,
execution, transaction, or persistence authority.

## Exact compatibility

The preimage is domain + `00 01` + coordinate + layout-present byte + validated
layout digest + transaction count `1` + length-prefixed stable function
descriptor + the correct executable member:

- no helper/borrow: effects then `graph-v1` and graph fields;
- helpers: `helpers-v1`, root executable, helper count, sorted names and bodies;
- any borrow: `borrows-v1`, executable fields plus borrow input/result and view
  offset/length, including helper count even when zero.

Capability numbers and layout arena indexes are absent. Callees are encoded by
their canonical logical names rather than their private table indexes. Literal
arena offsets are absent; only referenced literal bytes are encoded. Field,
block, ordinary-value, and borrow IDs remain exact semantic graph fields.

The Go stable type spelling is followed literally, including its scalar slice
spelling `[u8]`, portable nominal `pkg[fypc1-sha256-...]::module::Type`, nominal
slices, fixed arrays, borrowed/raw-pointer, Option and Result forms, and its
256-level type depth limit. This is the existing hash grammar, not a statement
that every such signature is executable by the state runtime.

All phases use the existing shared 16 MiB work ledger. Failure returns no
identity: `1001` malformed/inconsistent identity, `1002` work/byte bounds,
`1003` allocation failure. Output serialization and SHA processing are charged.

## Windows verification

Executed against the final authored files:

```powershell
go test -mod=readonly -count=1 -timeout 3m -v ./internal/cli -run '^TestCandidateStateGraphBodyHash(Oracle|SealedOwner)?$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_state_graph_body_hash_test.go
git diff --check
```

Final focused test command passed in **21.735 s**. Native low and optimized
profiles each passed **127 process checks**, **254 total**:

- 65 original authored HIR fixtures are independently lowered by Go and checked
  through `ComputeHIRStateBodyHashes` and `DescribeStableHIRStateBodies`.
  An independent byte writer
  consumes the stable artifact, and its SHA must match the Go hash before the
  candidate comparison runs.
- 33 additional canonical descriptor shapes are checked by Go's stable-body
  validator and compared byte-for-byte and hash-for-hash with Fairy. These are
  identity-grammar tests, not new HIR execution claims.
- 29 malformed/inconsistent contexts are refused by both the Go stable-body
  validator and Fairy. They include logical names, type/depth/array-width and
  coordinate errors, duplicate/unsorted/foreign/mismatched effects, malformed
  result/drop-glue, trailing/truncated bytes, and impossible u64 lengths.
- Each accepted process also checks exact remaining budget, one unit too short,
  and initially exhausted budget for both context construction and hashing.
  It drops the caller's byte transport before using the owned context/proof.
- Four semantic refusal checks prevent constructing Context/Identity and
  obtaining mutable access to their descriptor/canonical bytes.

Fixtures cover typed branch joins and edge arguments, helper chains/diamonds,
helper ordering, read/write effects and borrow invalidation, all seven storage
codecs, nested views, u32 view limits, and the 1,024-operation boundary. Explicit
hash changes cover coordinate, root name/signature, literal, helper/callee name,
helper literal, borrow ID, view offset/length, exit, and trap category. Authored
helper reorder, capability renumber, and literal arena relocation preserve the
expected hash.

The final log is local ignored output:
`.cache/state-body-hash-final-20261107.log`. Earlier development failures remain
in the separate `state-body-hash-first` through `seventh` logs; they were not
counted as passes. No product timeout, compiler warning, or acceptance default
was relaxed.

## Deferred scope and changes

Linux execution stays batched. No whole compiler rebuild, current-source
association, artifact restoration, ordinary CLI selection, native state
execution/persistence, zero/multiple-transaction hashing, or allocation-failure
injection was executed by this slice. Allocation failure remains a separate
`1003` path rather than a simulated successful check.

Only these three new Fairy modules, one new Go test file, and this receipt are
owned by this subtask. Shared modules, lockfiles, ADRs, queue/index, normative
source-of-truth documents, public APIs, DB, dependencies, CI/runners and target
policies were not changed. No staging, commit, push, release, or deployment was
performed; parent integration owns the commit.
