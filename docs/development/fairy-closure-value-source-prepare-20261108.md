# Fairy closure value source preparation, 2026-11-08

ADR 0551 source-preparation worker receipt. Work is isolated in
`.cache/closure-runtime-20261108`. No commit or push was made by this worker.
No shared fairy.lock, main, parent HIR/type/MIR files, or other worker files
were written. No source-of-truth document, DB, runner or dependency changed.

## Private API and behavior

- `syntax::closure_value_captures(p, declaration)` returns zero for an ordinary
  or old direct-call target, or captures + 1 (1..5) for a typed value helper.
- `copied_closure_tables::Row.value_captures` owns that encoded fact. Existing
  `add` stores zero; `add_value(..., captures)` accepts only 0..4 with a real
  helper name. `copy` and `append` preserve it; the existing module join already
  uses `append` with its declaration base.
- Explicit immutable outer-block bindings with the exact structural
  `fn i32(i32)` type retain a real kind-2 binding with a private kind-36 creation
  initializer. Its left is a helper kind-2 reference carrying
  `helper_flag + declaration`; its right is the written-order kind-7 capture
  argument list, each value carrying `snapshot_flag + entry`. These are capture
  arguments only; the lifted helper takes captures followed by one i32 value.
- The typed environment binding has helper zero, so named value copies,
  returns and invocations retain ordinary references. Closure parameters,
  literals and direct local aliases cannot be captured as i32 snapshots.
- Existing untyped literal admission/direct rewriting remains selected by
  type_root zero. Both paths share the existing pure body admission, 0..4 i32
  captures, 16 immutable locals and 64-node budget. Typed values add exactly
  one i32 parameter. No source bytes or generated helper text are inserted.
- Call-argument queries accept kind 36 only after checking helper metadata,
  bounded argument lists and private snapshot entries. Name resolution binds
  helper/snapshot references by exact private IDs, hides snapshots from normal
  name lookup, and does not confuse repeated written literal binding names
  across different source functions with duplicate generated helpers.
- Original source entries and written function JSON remain intact. The focused
  fixture compares JSON before/after owned copy, then compares its definition
  hashes with independently encoded Go source AST JSON.

## Executed validation

Final command, with no native opt-in environment variable:

```text
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueSourcePrepare(Syntax|Check|Native)$' -count=1 -v -timeout 10m
```

PASS, exit 0, package 14.559s:

- SourcePrepareSyntax: PASS, 0.04s, owner/probe syntax, 11 accepted source
  shapes and 22 syntax-valid source-preparation refusal controls.
- SourcePrepareCheck: PASS, 1.62s, Go source check of only the isolated authored
  import closure. This does not mutate the shared candidate lock.
- SourcePrepareNative: PASS, 12.70s, both `[run]` and
  `[run --release --locked]`, all 11 source shapes and 22 refusal controls in
  each profile. Also checks exact capture order and snapshot binding order,
  copied/append-remapped helper metadata, original source and entries,
  name resolution, no-capture and four-capture values, distinct helpers through
  one signature, legacy direct-call coexistence, unused trapping body locals,
  16/64 bounds, and original JSON definition hashes.
- Temporary native environment opt-in removed after the first successful
  two-profile run; final test above genuinely executes native on Windows AMD64.
  Only the Windows AMD64 platform guard remains.
- Scoped `git diff --check`: PASS.

Corrected attempts: the inferred `if` expression's zero branch needed explicit
`usize` context; the first raw-JSON native output exceeded the existing 65536-byte
execution output bound, so the probe retained its internal JSON equality checks
and emits the existing definition hashes instead. A temporary Go Digest-to-string
compile mismatch was corrected before the final successful command.

## Parent handoff and limits

The source-preparation files are stable. The parent may resume its native C
checks now. Whole-candidate checks, HIR/MIR execution acceptance, fresh-process
cache restoration and Linux are parent/integration gates, not claimed here.

One static integration risk outside this worker's write scope was reported:
`generic_expand::expand` currently finishes with `syntax::assemble`, which drops
private names/metadata when expansion is needed. An unrelated generic in the
same source set could therefore lose closure helper metadata; preserve the
private table when the parent handles that composition path. This worker did
not change generic_expand.fy and did not claim a native reproduction of that
combination.

## Exact worker file scope

- `examples/lexer_candidate/src/copied_closure_lower.fy`
- `examples/lexer_candidate/src/copied_closure_tables.fy`
- `examples/lexer_candidate/src/syntax.fy`
- `examples/lexer_candidate/src/name_resolution.fy`
- `examples/lexer_candidate/tests/closure_value_source_prepare.fy`
- `internal/cli/candidate_closure_value_source_prepare_test.go`
- `docs/development/fairy-closure-value-source-prepare-20261108.md`

## Final worker input hashes

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/copied_closure_lower.fy` | `5d4727e3c234da0065b86dbce4af7c2f3744609baacb63341cb3c68f7e81d091` |
| `examples/lexer_candidate/src/copied_closure_tables.fy` | `842e307207fc85ff62dbd08a6f76a07b54be58bf3a60891100fc9eb3d5235783` |
| `examples/lexer_candidate/src/syntax.fy` | `497456ef9b791a5834dc7b7bf47002760490b9dff6fe04a27b5eb0cb9094c81f` |
| `examples/lexer_candidate/src/name_resolution.fy` | `012cc8b165ab8e6af37b6be41ec95267f8f5996e8c0a3613d3b3b858b2a0e774` |
| `examples/lexer_candidate/tests/closure_value_source_prepare.fy` | `4a355a572fb2883e65bea240f181ce1ff7abe17fe881770d4959427d518e0620` |
| `internal/cli/candidate_closure_value_source_prepare_test.go` | `498290bd993d3d9bf2910e9462e61b89e6926086f396d23a1cc8068af4f14362` |
