# Next composite generic transport slice, 2026-11-07

Status: preparation plus execution links. Inspected `main` at `0bb199e7`.
The Go slice is implemented separately under
[ADR 0547](../adr/0547-flat-option-generic-transport.md); its exact scoped files,
checks and remaining integration work are in the
[Go execution record](go-flat-option-transport-20261107.md).
The coordinator owns Fairy production, commits, full-main Atlas and final acceptance.
The original preparation findings below describe the pre-implementation baseline;
the execution record owns subsequent results. This sidecar changes no queue or
release state.

## Recommendation

Implement **flat `Option<T>` value transport for i8, i16, isize, u16, u64, f32
and f64**, through both Go and Fairy, before references or other wrappers.
Cover Some/None construction, local copies, reassignment, explicit generic calls,
forwarding, return and exhaustive `switch` extraction. Keep the existing scalar
argument domain: `relay<u16>(option)` is admitted; `id<Option<u16>>(option)` is
not. This adds a signature/value wrapper, not composite generic arguments.

One Option has one payload type and no borrowed return origin, resource owner,
allocation or new representation. `Result` adds a second payload/propagation
contract; scalar references need return-origin and safe-borrow C admission;
arrays/views/pointers have their own element/representation gates. These are
reasons to start with Option, not claims that their implementations are broken.
Backend, target, trust, chain and provider choices remain outside this work.

The first production commit should close Go admission plus C emission and its
focused tests. The second should close Fairy admission plus restored-artifact
native execution. A third should establish fresh-process persistent-cache/CLI
execution. Each is independently reviewable; the family is executable end to
end only after all three. Windows checks precede each small commit. Linux
commands join the existing batch; no push or dispatch is implied.

## Current boundaries, verified from source

Read `implementation-queue-20261105.md` lines 1-110 before source exploration.
The relevant owners are ADR 0385, ADR 0387 (including its later private package
and authenticated-SDK sections), ADR 0494 and
`docs/architecture/candidate-generic-transport-v1.md`.

| Source shape | Current evidence and exact limitation |
| --- | --- |
| `fn T id<T>(T x){return x;}` instantiated with any of the seven | Go private/explicit/package compilation is admitted. Fairy tags 32-38 represent direct values. Native and CLI/cache fixture owners already exist; those historical native receipts were not rerun here. |
| `fn i32 constant<T>(){return 42;}` with `T=u16` | Admitted phantom argument; do not require materializing its value type. |
| `struct Cell<T>{T value;}` and scalar-parameter resources | Existing private aggregate family. Preserve it; do not replace it with a new Option policy. `candidate_generic_aggregate_signatures_test.go` admits `Cell<f64>` signatures and `candidate_generic_aggregate_layouts_test.go` admits `Holder<f64>` layouts. These oracle cases are not proof of complete native execution of every such aggregate. |
| `fn Option<T> relay<T>(Option<T> x){return x;}` with `T=u16` | Go generic-signature gate rejects it; Fairy early signature diagnostics and nested scalar substitution reject it. This is the proposed delta. |
| Generic signatures `ref T`, `ptr T`, `Option<T>`, `T[2]` with `T=u16` | `internal/compiler/generic_admission_test.go` asserts two deduplicated unsupported signature locations, preserving the original spelling. Repeated calls do not duplicate the reports. |
| Fairy substitution of `Option<T>`, `Result<T,i32>`, `ref T`, `ref T[]`, `ptr T`, `T[2]` for all seven | `examples/lexer_candidate/tests/generic_scalar_arguments.fy::composite_substitution_rejects_without_publishing_partial_types` requires failure with type 0 and unchanged graph length. Aggregate-enabled resolution is a distinct mode; do not describe this as a universal ban in every mode. |
| `Values<T>{T[2] elements;}` with `Values<f64>` | Existing aggregate layout oracle explicitly expects rejection code 2. Scalar fields and scalar arrays are different admissions. |

Actual Go checks run for this plan (no native consumer or broad build):

```powershell
go test -mod=readonly ./internal/compiler -run '^(TestGenericSignaturesAdmitDirectTransportScalars|TestGenericSignatureAdmissionKeepsOrdinaryTransportAndPhantomArguments|TestGenericSignatureAdmissionPreservesCompositeSpellingAndResolution|TestGenericSignatureAdmissionChecksForwardedPackageTypes|TestRestrictGenericSignaturesPreservesReportExhaustion|TestRestrictGenericSignaturesBoundsTransportUnsupportedAtCollection)$' -count=1
```

PASS, package test time 0.111s. Fairy behavior below is source-inspected, not a
new native execution claim. In particular, current negative full-pipeline
fixtures sometimes use missing call arguments: those alone cannot prove that
the wrapper admission, rather than arity, caused rejection.

## Exact new contract and examples

After substitution, the new signature root is exactly `Option<S>`, with one
direct, argument-free scalar leaf S from the seven above. A binder and a written
scalar must resolve to the same node/identity. The generic signature predicate
accepts this whole shape, not every scalar occurring at depth one. Existing
admitted types and aggregate policies keep their current behavior.

Proposed positive fixture, instantiated once per scalar and supplied real
values by the observer described below:

```text
module sample;
fn Option<T> pack<T>(T value){return .some(value);}
fn Option<T> relay<T>(Option<T> value){
    let Option<T> copied=value;
    discard value;
    var Option<T> replaced=copied;
    replaced=copied;
    return replaced;
}
fn Option<T> empty<T>(){return .none;}
fn T round_trip<T>(T value){
    let Option<T> wrapped=relay<T>(pack<T>(value));
    return switch(wrapped){
        case .some(actual):{actual}
        case .none:{value}
    };
}
fn i32 none_branch<T>(){
    let Option<T> value=empty<T>();
    return switch(value){case .some(_):{0}case .none:{42}};
}
```

These are proposed source examples, not newly verified programs. Add concrete
typed observer entries that call `round_trip<S>` and `none_branch<S>` so every
specialization is genuinely reachable. Use a separate copy probe to reuse a
Some value after passing its copy, and check both results; `discard` alone is
not enough copy-semantics evidence. Reading a None payload is forbidden.

The following remain negative **new extended-scalar generic wrapper signatures**:

```text
fn Option<Option<T>> nested<T>(Option<Option<T>> value){return value;}
fn Result<T,i32> result<T>(Result<T,i32> value){return value;}
fn ref T borrowed<T>(ref T value){return value;}
fn ptr T pointer<T>(ptr T value){return value;}
fn T[2] array<T>(T[2] value){return value;}
fn ref T[] view<T>(ref T[] value){return value;}
```

Instantiate each with valid-arity source calls and all seven S, or test selected
signature resolution directly. Also reject `ref Option<T>`,
`Option<ref T>`, `Result<Option<T>,i32>` and `Option<Result<T,i32>>` in that
new signature path. Do not revoke old-scalar wrapper support or existing
aggregate-enabled admissions. Generic arguments `Option<S>`, wrong arity,
invalid descriptors and generated-name requests remain closed.

Extended numeric literal syntax, arithmetic, numeric comparison/conversion,
direct extern-C scalar admission and Option-by-value FFI are unchanged. Deferred
capture of an Option is not part of this slice: Fairy's
`module_unit_value_terminal.fy` currently requires direct scalar deferred
arguments. Ordinary calls and returns are the executable target here.

## Production change map

All Fairy paths in this section are relative to
`examples/lexer_candidate/src/`. These are future edit instructions only.

| Owner | Concrete bounded implementation |
| --- | --- |
| `internal/compiler/generic_admission.go::unadmittedGenericScalar` | Add an exact flat-Option exception before the existing recursive transport-scalar rejection. Retain authenticated expansion origins, ordinary-function behavior, resolve-error precedence, original composite `Text`/spans, deduplication and report limits. Do not delete `containsTransportScalar`. |
| `internal/compiler/generic_admission_test.go`, `generic_admission_budget_test.go` | Replace the Option negative with its new positive matrix. Keep other wrapper refusals. Both budget tests currently use `Option<u64>` as their unsupported example; migrate them to `Result<u64,i32>` or another still-rejected shape so they continue testing exhaustion rather than accidentally requiring the old ban. |
| `internal/genericsource/clone.go::transform.typ` | Existing structural substitution already clones all type edges and retains template spans. Reuse it; do not substitute text or change planner argument domains. Add a source-immutability assertion instead of a new cloning mechanism. |
| `internal/cbackend/tagged_types.go::prepareTaggedTypes` | Go C preparation has an additional blocker: `supportedTaggedPayload` and `taggedPayloadLayout` only admit the original language scalar set. Add an exact flat-Option extended-payload candidate and an Option-specific payload layout branch in `resolveLayout`, using `layout.KindLanguageScalar` and the existing target scalar table. Reuse existing Option tag/payload emission and layout assertions. |
| Same Go C owner, boundary detail | Do **not** add all seven to shared `supportedTaggedPayload` or `copyScalarTaggedPayloadLayout`: the latter also drives Result and aggregate layouts. Keep `copyScalarTaggedAggregateLayout` unchanged. Before recursive candidate discovery, decline new tagged roots containing extended leaves unless the root is exactly flat `Option<S>`; otherwise recursive fallback can admit `Option<Option<S>>` or `Result<Option<S>,i32>`. Source signature gates and emitter candidate checks serve different entry points; test both. |
| `generic_signature_diagnostics.fy::contains` / `signature` | Add the same exact-root Option predicate, resolving either a binder through the selected concrete arguments or a written scalar. Exempt only that complete shape. Preserve the existing depth guard for all other wrappers, current-source checks, charging, sorted original spans and duplicate compaction. |
| `module_types.fy::walk` | Thread a narrowly scoped context indicating the direct payload of an allowed root Option. Permit descriptor-to-value materialization there only. Apply it to both descriptor and already-interned scalar substitutions and written scalar names. A plain `depth==1` exception opens other wrappers. Preserve independent `aggregates` behavior. |
| `module_types.fy::resolve`, `specialize_current` | Retain their graph rollback on every failed operation. No success node, interned leaf, or truncated canonical type may survive failed wrapper resolution. Handles 4097-4103 remain source-only; graph IDs remain 1-4096, scalar tags 32-38. No new kind/tag is needed: Option is kind 5. |
| `module_unit_generics.fy`, `generic_concrete_types.fy` | Keep scalar generic arguments and forwarding unchanged. `lookup_typed` remains a scalar/nominal argument lookup, not a request to support Option generic arguments. Concrete function parameter/result types may now be kind 5. |
| `module_unit_cleanup_mir_types.fy`, `module_unit_cleanup_mir_lower.fy`, `module_unit_cleanup_mir_verify.fy` | Existing detached type records accept extended scalar leaves and Option edges; existing lowering assigns Option glue kind 2. Verify no resource/drop effect is inferred for flat scalar payloads. Change only a demonstrated downstream rejection; do not broaden unrelated type predicates. |
| `module_unit_c_aggregate.fy::type`, `define`, `expression` | Already recursively emits Option/Result payloads and actual extended scalar C types; Some uses tag 1, None tag 0. Reuse the zero-initialized tagged union, assignments and extraction. Test this complete path rather than adding another C representation. |
| `module_public_type_text.fy`, `module_project_types.fy` | Reuse recursive public spelling and canonical graph joins. Assert `Option<u16>` remains distinct from u16 and from every other payload; never serialize descriptor handles as scalar tags or public names. |

The Go semantic prerequisites are already structural:
`internal/semantic/resource_types.go::canonicalNamedType` resolves Option payloads,
and `inertCopyable` accepts floats/integer transport scalars recursively through
Option. `internal/mir/verify_operations.go` checks typed Option construction and
extraction. This is evidence to reuse them, not permission to bypass semantic,
Flow or MIR verification. Non-generic Go shapes that already pass semantic
analysis but cannot emit C must not be relabeled as new executable support.

## Transport, identities and caches

No new wire layout, heap allocation, boxing, niche optimization or version bump
is needed for the flat Option value. Preserve existing Go representation and
Fairy representation independently; compare observable behavior and public
type/identity facts, not generated names or C struct bytes across compilers.
Target-owned scalar size/alignment assertions still apply. Do not widen
`StorageLayout`, repr(C) fields, vector elements or borrowed scalar operations
as a shortcut to obtain a payload layout.

Flat Option is inert/copyable. Copying it must not move/invalidate the original,
introduce resource ownership or produce a nonempty cleanup. Type, payload and
tag survive ordinary parameters, locals, reassignment and returns. None does
not authorize inspecting inactive payload storage. Native observers compare
the active scalar bytes only; aggregate padding is not a semantic result.

Reuse existing definition/body/argument specialization identities. Adding
`Option<T>` to a template changes its definition identity; replacing S changes
its concrete specialization. Alias/binder renaming and package order must retain
their existing coordinate-based invariants. Do not encode wrapper values into
generic argument IDs.

Go owners: `internal/workspacecompiler/generic_session.go`,
`generic_persistent.go`, `generic_policy.go` and `internal/packageartifact/`.
Fairy owners: `project_generic_cache_key.fy`, `project_cache_key.fy`,
`project_cache_unit_key.fy`, `module_project_artifact_encode.fy`,
`module_project_artifact_decode.fy` and `project_compiler_core.fy`.
The existing generic key contains compiler/environment authority, source
interfaces, dependencies and actual specializations. Keep that schema and
current-source reanalysis/restoration before accepting hits. Compiler identity
must reflect the actual new build; a reusable caller-supplied identity is not
fresh-compiler evidence. No admission bit may silently mix policy profiles.

Cold/warm hits must preserve flat Option descriptors, signatures, public types,
MIR verification and exact source provenance. Failed current source yields no
verified program, hit/store publication or partial provenance. Editing a helper
body and changing a selected scalar must invalidate the affected dependency
closure; a correct artifact checksum cannot authorize stale current source.

## Independent acceptance and commit boundaries

1. **Go source-to-C commit.** Add
   `internal/compiler/composite_option_transport_test.go` and
   `internal/cbackend/option_transport_test.go`. Test all seven S through private,
   explicit-file and resolved-package checks, Some/None, copy/reuse and forwarding.
   Verify MIR and emit both existing C profiles. Use an independently authored C
   observer for Windows low/optimized execution. Existing
   `internal/compiler/broader_scalar_native_test.go` demonstrates repr(C) scalar
   seed/observer records; reuse this established FFI boundary, never extern-C
   `Option<S>` or direct new scalar FFI. Target result: exit 42, empty streams,
   all seven active payload observations and all seven None branches verified.
   Preserve resolve-first and report-limit tests. For unsupported signature
   forms assert written spans and no verified MIR; for raw verified-MIR emitter
   tests assert unsupported C emission rather than claiming semantic rejection.

2. **Fairy restored-artifact native commit.** Add
   `examples/lexer_candidate/tests/module_composite_option_transport.fy` and
   `internal/cli/candidate_composite_option_transport_test.go`. Adapt
   `candidate_generic_transport_pipeline_test.go`'s private compiler harness:
   build current-source complete IR, encode, decode, restore against those exact
   sources, partition emitted C, execute the resulting product. Compile the
   authored compiler at low/optimized and use private typed scalar probe entries
   to feed real values, avoiding new FFI admissions. Go-derived public type and
   specialization expectations must be independent of Fairy's result; C observes
   the payload independently of both. Exercise signed minima, unsigned maxima,
   exact finite floats and positive/negative zero, with `memcpy` bit checks for
   zero sign. Do not require NaN payload preservation as a new language contract.
   Add direct graph tests for every seven-type Option substitution and every
   still-rejected wrapper, stale source, invalid descriptor, graph capacity and
   failure after one payload was interned. Assert unchanged graph contents,
   not just node count. Existing
   `tests/generic_scalar_arguments.fy`, `tests/generic_signature_diagnostics.fy`
   and `tests/module_generic_transport.fy` must replace only their new Option
   negatives; their old malformed-call negatives are not sufficient evidence.

3. **Persistent-cache/CLI commit.** Add
   `internal/workspacecompiler/composite_option_cache_test.go` and
   `internal/cli/candidate_composite_option_cache_test.go`, following
   `broader_generics_test.go`, `candidate_generic_cli_observer_test.go` and
   `candidate_generic_cli_cache_test.go`. Use a two-package public Option helper,
   memory cold/warm and fresh-process disk cold/warm. On a complete hit forbid
   lowering and execute restored output, observing both Some payloads and None.
   Check source/helper/payload edits, stale-source restoration and corrupted
   artifact refusal. Reuse one captured current packet per compiler profile for
   independent C observations. For CLI integration batch check/build/run plus
   one isolated native test around that workspace; do not launch the whole CLI
   suite. Only after these results amend the owning ADR/architecture receipt and
   queue in the coordinator's authorized scope; this sidecar does not do so.

Suggested **new** targeted test prefixes are `TestCompositeOption`,
`TestCandidateCompositeOption` and `TestCompositeOptionCache`. Run their owning
packages with `-mod=readonly -count=1`; they are proposed names, not existing
commands or completed tests. Reuse valid unchanged baseline results. Consolidate
the same authored native fixture into the Linux batch using the existing target
profiles and independent observer; native Linux remains unexecuted until that
batch runs. Full-main Atlas, self-build equality, distribution refresh, actual
clean-machine execution and hosted CI remain with the coordinator.

Remaining risk: Fairy tag/value and C emission paths were inspected but have not
executed the proposed Option cases. Go C has a confirmed additional admission
gate. Both must pass actual payload observations before any executable support
claim. This document adds no API/DB, runner, dependency, target or release change.
