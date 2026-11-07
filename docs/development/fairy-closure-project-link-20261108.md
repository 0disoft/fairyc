# ADR0551 source-backed closure project linking, 2026-11-08

Parallel worker slice, starting HEAD `7b513405`, final source-check HEAD
`96db6ff1` after parent commits. Only these files belong to this slice:

- `examples/lexer_candidate/src/module_project_body.fy`
- `examples/lexer_candidate/src/module_project_cleanup.fy`
- `examples/lexer_candidate/src/module_project_calls.fy`
- NEW `examples/lexer_candidate/tests/closure_project_link.fy`
- NEW `internal/cli/candidate_closure_project_link_test.go`
- This NEW receipt.

The optional `module_project_types.fy` change was unnecessary: its existing
signature-owned graph validation and canonical interning already handle kind23
without payload edges. Its source was not changed. The preceding cleanup-lower
slice is frozen; no file from that slice was edited here. The parent owns native
CPU, shared locks, body/CFG/C changes and commits. No commit or lock update was
made by this worker.

## Production changes

`module_project_body` preserves `Node.value.target` for construction kind36.
That target is a source declaration ID, qualified by the frame's original module
index, not a type ID. All other node targets keep the old canonical type mapping.
Expression/statement IDs, capture links, helper/snapshot syntax tokens, source
spans, declaration IDs and module indices remain unchanged; ordinary type,
expected-type, layout and argument mappings still run.

`module_project_cleanup` detects kind23 while copying the already validated
project descriptors and selects the existing private `assemble_closures` family
with `project=true` and the independent `types::aggregates(graph)` bit. With no
kind23 it still selects the exact old project/project-aggregate constructor.
No cleanup model, verifier, wrapper, resource-field, producer work, count or
capacity policy changes. The closure leaf retains trivial glue and no drop.

`module_project_calls` handles invocation kind4 before looking up provider
contracts or ordinary/specialized targets. The existing owned closure body proof
checks the complete caller once on its first value call: canonical leaf, checked
flag, named immutable callable, exact i32 signature/argument and zero fabricated
provider/instance coordinates. Its independent bounded work is charged to the
project ledger. Subsequent value calls reuse that immutable body proof. Ordinary
and generic declaration-call, foreign, argument, effect and unsafe checks remain
on their previous path. No value call gains provider authority by name collision.

No new public signature, language/ABI contract, transport family, project source
identity, codec or CLI feature was added. Raw unit/project constructors remain
private checked IR boundaries; this slice confers no Complete/cache authority.

## Authored original-source producer

Go passes only original module texts and independent expectations. It parses
each original using its written module path and checks it with the Go compiler;
no Go type graph, helper target, frame, unit or MIR is fed to Fairy.

The Fairy fixture independently parses the original modules, builds signatures,
calls `module_project_compile` and then the ordinary raw `module_project_link`
API. Its first control is a single original module/unit with a returned copied
closure. Each small admitted input is then checked through four compositions:

1. One compiled range containing all modules, linked as one unit.
2. One compiled unit per original module.
3. The same module units supplied in reverse order, preserving source indices.
4. The actual automatic `module_project_compile::build` grouping and linking.

A separate original-source case contains six ordinary 96-parameter functions in
each of two modules. Each module stays below the catalog's 128-declaration cap
and the unit's 1024-entry cap, but together
they exceed the existing unit cap. It requires two actual automatic batches;
the impossible full-range composition is omitted for this case, while the two
per-module orders and real automatic batching are all retained. No compiler
capacity or fallback result is forged or relaxed.

The 23 original-source controls comprise 15 admitted and 8 refused cases:

- Provider-owned returned factories, escaped local copies, parameters and
  ordinary struct fields across three modules; different helpers with matching
  declaration numbers in separate providers; one and four i32 captures; a
  created but never invoked value; ordinary resource cleanup alongside a closure.
- A padded provider's helper declaration exceeds its type graph size, directly
  exposing accidental declaration-to-type mapping.
- Signature-only value calls, scalar generic struct/resource neighbors and old
  scalar/project-aggregate controls preserve all four cleanup mode combinations.
- Option, both Result alternatives, pointer and borrow around imported closure
  carriers, direct resource storage, FFI and generic closure argument transport
  remain closed. Budget/allocation/stale-record failures cannot stand in for
  language refusals.

Scalar aggregate neighbors retain the existing source-owned planning prerequisite
(`build_aggregate_sources -> generic_aggregate_types::build -> build_aggregates`).
Those inputs are explicitly single modules with no closure literals or generic
closure transport. Their package coordinate is derived from actual source bytes.
No identity or generic preparation production owner is modified.

Linked-program assertions compare frame/body/source identity and every kind36
target/token/link with its original unit. They re-enter owned body and project
call proof, build each frame's CFG, and use the existing callable/frame indexes
and reachability fixed point. Every construction in a selected frame must keep
its module-qualified helper selected, even when the resulting value is never
invoked. Empty, missing-provider and duplicated raw inputs must still refuse.
Frame counts, helper/value-call counts and cleanup drop counts must agree across
unit compositions. The producer does not emit or execute the input program's C.

## Validation and parent handoff

The worker runs only:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectLinkSource$' -count=1 -timeout=4m -v
gofmt -l internal/cli/candidate_closure_project_link_test.go
git diff --check -- examples/lexer_candidate/src/module_project_body.fy examples/lexer_candidate/src/module_project_cleanup.fy examples/lexer_candidate/src/module_project_calls.fy
```

- Final source Go test: **PASS**, test 17.73s, package 18.065s. All 23 original
  Go source controls match the declared admissions (15 accepted, 8 refused).
  The actual CLI `check` compiled the authored producer and import closure in a
  temporary workspace. Earlier checks exposed original-file path mismatch, a
  private resource field access and an inferred i32/usize conditional; the
  fixture/harness was corrected before this pass, retaining all intended cases.
- Go formatting and scoped tracked diff whitespace: **PASS**.
- All six owned files: LF, final-newline and trailing-whitespace checks **PASS**.
- `module_project_types.fy`: unchanged; no file outside the authorized slice
  was edited by this worker.

Source checking compiles the producer and its authored import closure; it does
not execute any of the Fairy linking, CFG, reachability or refusal assertions.

Parent-only native reproduction, not run by this worker:

```powershell
$env:GOMAXPROCS='1'
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureProjectLinkNative$' -count=1 -timeout=12m -v
```

The native harness uses the existing `candidateParityBuildParts` and
`BuildWindowsMSVCTargetPartsWithStackReserve` private producer path, low and
optimized host profiles, unchanged one-minute per-part compiler limit and 16MiB
stack reserve. It requires one ordered receipt per original-source case. This
avoids making monolithic compiler-producer compilation a prerequisite and does
not establish stable CLI optimized execution parity.

Parent Source/Native validation in the stable detached worktree passed:
Source 19.93s; Native 196.08s; package 216.201s. Each native profile compiled
82 private producer parts and executed all 23 ordered original-source receipts.
This executes the linked body/CFG/helper reachability and refusal assertions,
not generated input-program C. Log: `.cache/closure-project-link-final-20261108.log`.
The earlier native log preserves a test-input failure: 258 nominal declarations
exceeded the actual per-module catalog cap of 128. Replacing that padding with
valid parameter entries exercises the intended automatic batch boundary without
changing product limits or removing the case.

**Not run in this slice:** input closure program execution, whole compiler
or self-build, source identity, codec/cache round trips, Linux, CI or release.
No normative document, DB, runner, dependency, manifest or hygiene config changed.

## Final worker code SHA-256

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/module_project_body.fy` | `ee668504360f4f274b77ecdfe77d056cd5badb6bbe8e7429f4f9d62c1dd170f4` |
| `examples/lexer_candidate/src/module_project_cleanup.fy` | `71e4386079aa20fc0da489ce8a1c29efd8befe8ef1beb52de19318b97bbac17a` |
| `examples/lexer_candidate/src/module_project_calls.fy` | `9b59a26032ebefd348bd91beea0d114d0e8a02775cba658bc1a13e0fda28a743` |
| `examples/lexer_candidate/tests/closure_project_link.fy` | `d0b3939e7ac3f401452a463b6e1d1aa7b0089ab2873d67fc6ab76ab23e2d2aa1` |
| `internal/cli/candidate_closure_project_link_test.go` (parent-corrected) | `65b4d46da7263cf7296ea0cb2a0707ada22497258f6b97fc083777f1bc0efbaf` |
