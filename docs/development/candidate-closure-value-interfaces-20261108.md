# Original-source copied closure signatures and interfaces, 2026-11-08

Baseline: `329ebcf5c0cd613460fa2863e02c96cfadbb6d26`, isolated checkout
`.cache/closure-interfaces-20261108`. ADR 0551 remains a staged prototype.

This worker owns only these three new files:

- `examples/lexer_candidate/tests/closure_value_interfaces.fy`
- `internal/cli/candidate_closure_value_interfaces_test.go`
- this document

Production belongs to the coordinator: `closure_value_source_set.fy`, its
`module_signatures.fy` hook, and `module_unit_shape_contracts.fy` changes.
Public type text and Go CLI acceptance belong to other workers. This worker
does not edit those files, the shared lock, existing tests or existing receipts.
No commit, push, CI, distribution, release or main-checkout mutation is requested.

## Source authority and test route

The Go oracle parses the original UTF-8 sources, preserving comments and CRLF.
It checks semantic classification and, for semantic positives, lowers and
verifies actual Go MIR. It compares original AST JSON before and after analysis.
Generic cases use the real package-aware generic compiler pipeline, with package
coordinates hashed from the exact source bytes and imports resolved from the
parsed modules. The Vec case includes the complete original bundled
`internal/sdkstdlib/sources/optional/std/vec.fy`, with that exact provider selected
as the SDK storage owner. No fabricated SDK declaration is substituted.

The binary input contains source bytes, case names, acceptance classification
and exact expected original refusal intervals. It contains no fabricated
catalog, nominal-containment table, inferred graph record or stamped tag-23 ID.
Fairy parses each original file and builds its own real owners:

1. `signatures::build`, and separately real `catalog::build` followed by
   `signatures::build_catalog`.
2. `binding::bind` for the caller range, with a preceding unrelated module.
3. Both `interfaces::build` and `interfaces::build_unit_seeded` with empty seeds.

The positive checks inspect the produced canonical closure leaf (tag 23 with
all payloads zero), signatures, source identity, nominal provider identity,
shape/field source indices and spans, and the existing 1 MiB interface work
bound. Equal Holder spelling and equal closure fields in distinct providers
must retain different nominal IDs. Neither this fixture nor the oracle calls
the raw `closure_value_source_set::validate_current` policy API as authority.

Source-authority controls replace a provider with trivia-only changed bytes,
reorder modules, and replace the caller while retaining old bindings. A stale
catalog must refuse `build_catalog`; old Programs must refuse binding and both
interface routes. Fresh owners admit the changed original source. These checks
use real rebuilt catalogs and Programs, not caller-supplied source digests.

## Coverage

31 original-source cases (8 positive / 23 negative), each through both signature
owner entry points:

- Direct explicitly typed closure parameter/result/local copy; ordinary struct
  direct closure field; imported Holder parameter/result/local copy; two
  distinct providers declaring the same Holder spelling.
- Imported carrier ptr/ptr-mut/ref/ref-mut, Option, both Result positions, fixed
  array, generic aggregate argument, explicit generic call argument, and Vec.
- Direct and imported resource, enum payload and repr(C) storage.
- Nested carrier storage; self cycles with Option before the closure and raw
  pointer after it; mutual Result cycle; provider chain and provider cycle with
  closure containment behind wrappers.
- Closure-free scalar, recursive raw-pointer nominal with Option transport,
  resource/enum/repr(C), and imported ordinary wrapped nominal controls.

Refusals compare an exact code, original module index and original type byte
interval. Existing depth/node/work/type budgets are unchanged. All inputs are
retained if a production refusal differs; such a failure must be reported.

## Go semantic and MIR boundary

The nested source `struct Outer{p::Holder value;}` with ordinary argument,
result and immutable copy is accepted by Go semantic analysis and `mir.Lower`.
`mir.Verify` then refuses it with **V1032**:
`copied closure transport: only direct fields of ordinary structs are admitted`.
Fairy intentionally refuses nested storage at source-set policy time. This is
an earlier refusal at the existing verified-MIR boundary, not semantic parity.
The test records all three Go stages rather than asserting that lowering alone
rejects it. The remaining forbidden source families are rejected earlier by
Go semantics or the actual generic compiler source pipeline.

## Validation status

The coordinator announced frozen production after its full Go source check
passed (761 sources / 9,613 MIR functions). This worker's native probe uses only
the authored transitive import closure in an independent temporary project.
Its `fairy.lock` is private to that temporary project.

Results:

- Go original-source oracle: 31 classifications passed; final standalone run
  passed in 0.145 s. The fixture syntax check passed with it.
- First native attempt: source `check` refused harness switch expression with
  E1006 (resource-valued success arm versus `continue` error arm). Repaired by
  moving one case check into a function with an early return. No production,
  inputs, assertions or budgets changed.
- Repaired native probe: passed in 35.992 s, including source `check` and both
  real Windows AMD64 low and optimized/locked compilation/execution. All 31
  inputs, exact refusal codes/module indices/spans and source-authority controls
  passed. Both native runs returned exit 0 with empty stdout/stderr.
- Go formatting: no unformatted paths. The three owned files have no trailing
  whitespace and retain final newlines.

Earlier harness drafting errors were corrected in owned files: source display
paths must match declared module paths; a stale unused Go loop index was
removed; generic original-source imports need explicit package bindings. None
was treated as a production pass or repaired by dropping cases. `mir.Lower`
alone initially proved insufficient to classify nested storage; the oracle now
requires the actual `mir.Verify` result. The initial native E1006 and the repaired
pass are the complete native attempts. After that pass, only a Go fail-fast guard
was added to prevent writing/running a partial oracle after failed subtests;
the final standalone oracle checked that guard. Native inputs and Fairy harness
were unchanged, so their successful native result was reused.

Commands (repository root):

```powershell
go test ./internal/cli -run '^TestCandidateClosureValueInterfacesOracle$' -count=1 -v
go test ./internal/cli -run '^TestCandidateClosureValueInterfaces$' -count=1 -timeout=15m -v
```

The native Go test runs source `check`, `run -- <oracle>`, and
`run --release --locked -- <oracle>`. Success requires exit 0 and empty native
stdout/stderr. Linux native validation is reserved for the coordinator's batch.
No whole compiler rebuild, executable closure factory/call, cache restoration,
distribution or hosted CI claim follows from this probe. **Fairy closure runtime
execution support is not established by source-set/signature/interface coverage.**

No normative ADR/grammar/specification, public API, DB, runner, dependency or
repository-hygiene configuration is changed by this worker.

## Coordinator integration

The source-policy, interface/layout and public-text slices were integrated
together on main. All ten authored implementation/test/receipt files matched
the isolated checkout byte for byte before adding this integration note.
The existing small Windows low/optimized receipts above and the 17-source
public descriptor byte/hash comparisons were reused for identical inputs.

The signature owner now runs `closure_value_source_set::validate_current`
after its existing catalog/source/signature checks. The new policy scans
structural safe-function nodes, computes nominal containment to a fixed point
over current catalog identities, then checks original declaration, entry,
field and explicit generic-argument roots. Nominal cycles cannot hide a closure.
Imported wrappers, nested storage, resources, repr(C), foreign and generic
transport remain closed. Feature-free sources avoid the containment pass.
Closure-bearing work has its own unchanged 1 MiB bound; this is additional
validation work, not a performance improvement claim. This policy helper does
not independently authenticate a catalog or confer artifact/cache authority.

Main checks passed:

```powershell
# From examples/lexer_candidate
go run -mod=readonly ../../cmd/fairyc check
go run -mod=readonly ../../cmd/fairyc check --locked
# From repository root
go vet -mod=readonly ./internal/cli
```

Both source checks covered 2 instances, 761 files and 9,613 MIR functions. Only
the root source identity/references in `fairy.lock` were refreshed; dependency
and SDK versions did not change. Vet passed with no diagnostics. Main logs:
`.cache/closure-interfaces-main-check.log` and
`.cache/closure-interfaces-main-locked.log`. Formatting/whitespace checks pass.

The first isolated source check failed on an inferred Option-pop switch arm;
the production helper now uses an explicitly typed usize zero sentinel and
checks that sentinel before access. Its passing bytes are included in the
snapshot below. A first logging redirection could not start the source command
because the isolated `.cache` directory did not exist; it is not a pass.

Fairy construction/invocation, executable MIR, project/artifact/cache adoption,
full compiler self-build/distribution and full CLI acceptance remain open.
The unrelated Go CLI optimized native-test compiler timeout is not covered or
relabelled by this interface commit. Linux remains batched; no push, hosted CI,
public SDK ABI, DB, runner, dependency or normative contract change occurred.

## Snapshot hashes

SHA-256 of the final authored test files and coordinator-owned production bytes
used by the passing native probe. Production hashes were unchanged across the
native run. This receipt is excluded from its own hash list.

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/tests/closure_value_interfaces.fy` | `9AE77902ED6261927650660FFE455A9D71F5722CAF406B544DFBEE312ABD9FA0` |
| `internal/cli/candidate_closure_value_interfaces_test.go` | `B74C5901010F1C75741C9296C92F8A7799771543CD723032945BC904D1EC0ABA` |
| `examples/lexer_candidate/src/closure_value_source_set.fy` | `3436075E0FFD4F820391654CA33FB72E4FA8E83FD297F8D04695952D2731B257` |
| `examples/lexer_candidate/src/module_signatures.fy` | `6914120747778CADCF31878EC5891C6E17E45EFECFC7FADE406FED0BA6B58A86` |
| `examples/lexer_candidate/src/module_unit_shape_contracts.fy` | `3C90894F1C5BC7AB742F901E235DEEEEDF5C93963BB6338E2BC8C9B9934BC186` |
