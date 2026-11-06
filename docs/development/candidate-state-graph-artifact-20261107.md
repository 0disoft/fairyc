# Detached state graph program artifact — 2026-11-07

## Implemented scope

`module_state_graph_artifact.fy` adds a private canonical detached program codec.
Its only factories are:

```fairy
pub fn Result<wire::Payload,u32> encode(ref program::Verified proof,ref mut usize work)
pub fn Result<program::Verified,u32> decode(ref u8[] input,ref mut usize work)
```

Callers access encoded bytes through existing `candidate_artifact_wire::content`.
The returned proof is the existing immutable `program::Verified`, not a new
constructor or authority type. Decode reconstructs independent owned graph data,
parameters, arguments, operations, blocks, helper names and layouts, then calls
`program::verify`. That factory independently copies validated layouts. No
current-source Complete, cache admission, artifact restoration, executable
transaction, chain or storage authority is manufactured.

The implementation and test are each below the workspace's 40KiB refactoring
candidate threshold. Shared modules, ADRs, lockfiles, queue and indexes are not
edited by this slice.

## Private canonical format and bounds

All integers are unsigned fixed-width eight-byte big-endian fields. Numeric u32
facts reject nonzero high words; no narrowing/truncation accepts a forged value.
The top-level marker is `FYSGRPH0`, disjoint from `FYSTATE0`, `FYSTATE1` and
ordinary `FYPROG00`. Its ordered payload is:

1. A length-framed canonical `FYSTATE1` layout artifact.
2. Helper count, name byte count and densely packed helper-name spans.
3. One length-framed root graph and the ordered length-framed helper graphs.
4. Exactly the helper name bytes, with no trailing content.

Each graph starts with nine words: effects, capability, layout index, entry,
parameter count, argument count, operation count, block count and literal-byte
count. Parameters use two words, arguments one word, operations nineteen words,
and blocks sixteen words (five block facts plus eleven terminator facts).
Operations preserve both operands' input/codec/spans and every result, found,
cost, callee, borrow and view field. Graph literal bytes, including unused bytes,
are preserved exactly. Canonical name packing is rechecked and regenerated from
immutable helper-name accessors without projecting private resource fields.

The existing shared ledger is 16MiB. Payload/frame size is capped at 8MiB,
individual graph literal/name buffers at 1MiB, helpers at 32, blocks per function
at 64, and combined root/helper operations at 1,024. Parameter and argument
counts are additionally bounded by the payload size divided by their wire row
width. Each entire graph section sum must fit its declared frame before the
first vector push. Every operand, block and terminator span is bounded before
its decoded row is pushed. Total operations are deducted across functions
before allocating that function's rows. The verifier then rechecks packed
spans, types, topology, provenance, helper effects, calls and borrow lifetimes.
All parse/copy/output work uses the existing ledger and allocation outcomes.
Errors retain private 1001 malformed, 1002 bound/work, and 1003 allocation.

## Windows validation

Final command:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphArtifact(Oracle|Owner)?$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/candidate_state_graph_artifact_test.go
git diff --check
```

Final focused test suite passed in **29.455s**. Its local diagnostic log is
`.cache/state-graph-artifact-final-20261107.log` (ignored execution output, not a
source of truth or a tracked artifact).

- **51 independently authored Go oracle admissions** pass.
- **416 Windows native processes** pass: 208/profile, low and optimized.
  Each profile executes 51 encode/roundtrip requests, 51 independent-owner
  replay requests, 105 corruption/boundary requests and one decode-cost query.
- **2 semantic owner-refusal probes** pass, rejecting proof construction and
  mutable access to the independently owned layout.
- Vet passes. Gofmt reports no paths. Diff whitespace and explicit nonempty
  UTF-8 without BOM, LF and final-newline checks pass for this slice.

Successful cases preserve the independent test writer's entire graph/name
payload byte for byte, with Go-computed layout hashes and field prefixes checked
separately. Fixtures include typed joins, root/helper branches, transitive and
diamond helpers, backward lexical calls, empty helpers, 32 helpers, 1,024 total
operations, 1,024 borrow handles, all seven codecs, views/copies, map keys and
root-borrow/helper-call combinations. Encode and decode are each exercised at
the exact measured remaining work, one unit short, and an already invalid
ledger. Owner replay drops the caller's original wire buffer before re-encoding
the decoded proof; original caller layouts are also replaced before encoding.

Corruptions retain complete structural framing when testing semantic refusal.
They include altered callee/effects/capability/metadata/cost, recursion,
unreachable or forbidden helpers, helper commit/root return, and a write helper
invalidating a live borrow. Marker refusal is checked in both directions:
`FYSGRPH0` cannot enter either existing state-layout decoder, and graph decoding
refuses state and ordinary-program markers and a state body with a forged graph
marker. Representative truncations, section sums, u32 overflow, out-of-range
spans, count bounds, 1MiB literal data, exact 8MiB input with trailing data
(malformed 1001) and 8MiB+1 input (bound 1002) also pass.

Development failures were resolved before the final command: an initial Go test
used method syntax for MIR layout fields; an initial native probe omitted the
semicolon separating an `if` expression from a dereference assignment. Neither
failed command is counted as a passing verification. The final suite reran both
profiles after the reverse marker checks were added.

## Deferred gates

Linux execution remains batched as requested. Allocation-error 1003 fault
injection, source binding, cache/Complete admission, ordinary command selection,
transaction execution and persistence are not validated or implemented here.
The parent owns any separate exact-source envelope. No existing public CLI,
SDK ABI, DB, dependency, runner policy, source-of-truth document or product limit
is changed. No stage, commit, push, workflow dispatch or deployment is performed
by this slice; the parent integrates its three new files.
