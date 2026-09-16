# Nominal scalar prototype v0

- Status: Proposed experiment, not accepted language behavior
- Date: 2026-09-17
- Reviewed source: `080b312fd12ebd4c457323873dc43597b62d657b`
- Decision owner: [ADR 0254](../adr/0254-defer-nominal-scalars-pending-user-domain-evidence.md)
- Evidence: [token index experiment](../ops/token-index-struct-experiment.md)

## Purpose and admission boundary

Test whether nominal value identity can protect the real scanner's token index
boundary while emitting the same executable operations as its `usize` baseline.
The existing struct experiment rejects accidental ByteOffset/TokenIndex interchange
but increases generated C by 1014 bytes and the low-profile EXE by 512 bytes.
That motivates a prototype; it does not establish that a new language feature is
necessary or free. No accidental production bug was discovered in that experiment.

Do not migrate production scanner/parser code or promote ADR 0254 from candidate
status in this step. No spelling is reserved. Examples below use metalanguage
`wrap(N, value)` and `unwrap(value)`, not Fairy syntax or ordinary function calls.

## First semantic slice

| Concern | Proposed prototype rule |
| --- | --- |
| Representation | Only `usize`, directly; no nested nominal underlying types |
| Identity | Declaration owner plus declaration name; TokenIndex differs from ByteOffset and usize |
| Scope | Declarations and ordinary direct calls within one resolved package; cross-module calls allowed |
| Values | Local bindings, assignment, ordinary value parameters and results |
| Construction | `wrap(TokenIndex, usize-expression)`; evaluate the operand exactly once |
| Extraction | `unwrap(TokenIndex-expression)` produces usize; evaluate once |
| Conversion | No implicit wrap, unwrap, literal coercion or nominal-to-nominal conversion |
| Constants | Underlying literals use existing usize range checking before explicit construction; nominal named constants/const-eval deferred |
| Comparison | `==`, `!=`, `<`, `<=`, `>`, `>=` only for exactly the same nominal type; result bool |
| Arithmetic | Reject direct nominal arithmetic/compound assignment; unwrap, use existing checked arithmetic, explicitly wrap |
| Copy/drop | Copy value with no cleanup or runtime tag; not a resource or permission token |
| Pattern binding | Ordinary local binding only; nominal switch/pattern matching rejected initially |
| References/pointers | Reject ref, mutable ref and pointer forms involving nominal types initially |
| Containers | Reject arrays, aggregate fields, Option, Result, Vec and other generic arguments involving nominal types initially |
| FFI/state | Reject nominal types in extern signatures, repr-C storage and persistent state |

Every deferred position must give a deliberate diagnostic, not silently become
usize. This is an intentionally incomplete experimental feature, not the eventual
public language contract. A caller may explicitly unwrap and wrap another domain;
this expresses intent but proves neither valid range nor association with a stream.
Bounds checks in `token_info` remain unchanged.

The representative conceptual client is:

```text
token_index: TokenIndex = wrap(TokenIndex, 2usize)
byte_offset: ByteOffset = wrap(ByteOffset, 7usize)
token_info(stream, token_index)              // accepted
token_info(stream, byte_offset)              // rejected
token_info(stream, 2usize)                   // rejected
token_info(stream, wrap(TokenIndex, unwrap(byte_offset))) // explicit, accepted
```

The callee explicitly unwraps its argument for existing `v::at<Token>` access.
The generic call still receives an ordinary usize index: the prototype does not
admit nominal generic arguments or share specializations by storage size.

## Semantic identity must survive until validation

Current `internal/hir/hir.go` defines `Type` as a canonical string and PackageID as
a build-local resolved-instance identity that must not be serialized as a public
package identity. Add a distinct nominal identity and a declaration table with
underlying type and source span; do not globally redefine scalar predicates to
accept these types. Use existing declaration ownership resolution rather than
matching bare spelling. Reject duplicate/unknown owners and non-usize bases.

`internal/semantic/nominal_bindings.go` handles concrete struct/resource targets for
checked generic expansion. It is not already a scalar-newtype facility and must
not be broadened merely because its name contains "nominal".

Typed HIR retains nominal identity on bindings, expressions, calls and results,
and explicit wrap/unwrap operations. Before erasure validate exact operand/result
types, comparison operands, function arguments/results and every supported use
position. A malformed typed-HIR caller must not bypass those checks by entering
`mir.Lower` directly. The lowering boundary must reject invalid input even when
the source analyzer did not produce it; mutation tests cover this entry point.

## Erasure boundary and backend invariant

After validation, lower nominal value slots, parameters/results and comparisons to
the existing usize MIR representation. Wrap/unwrap reuse the lowered operand:
no helper call, record construction, tag, memcpy, allocation or extra local is
introduced solely for conversion. Preserve side-effect order and existing trap and
cleanup paths. Lowering is a transformation, not a mutation of the retained HIR.

`internal/mir/lower.go` currently collects aggregate declarations and lowers each
function. Add an explicit checked boundary there or an owned adapter invoked by
every entry into it; do not perform textual type replacement in generated source.
Keep C backend, interpreter and MIR scalar rules unchanged for the first slice.
An escaped nominal type or conversion operation is a lowering failure, not a new
backend fallthrough. Nominal declarations must not create C typedefs/structs,
symbols, RTTI or default formatters in the executable output.

Once erased, MIR cannot prove domain separation: that obligation belongs to the
pre-erasure validator. Debug/source information may preserve names separately and
therefore may grow. Same-layout types alone are insufficient grounds for generic
code sharing because operations, cleanup and ABI can differ.

## Public interfaces, caches and later admission

The prototype permits same-package public visibility only; exporting a nominal
type through another package is rejected until the following contract is designed
and tested. Current `internal/moduleinterface/interface.go` uses public-interface
encoding v5 and includes parameter/result type strings. A future export must
encode declaration kind, stable resolved package ownership, module/name and base
type. Encoding only usize would let incompatible changes reuse cached callers.
Internal numeric PackageID and machine layout must never substitute for identity.

Before persistent prototype caching is enabled, version its semantic schema/key
domain and verify rejection of old records. A bounded first experiment may disable
persistent reuse for packages containing nominal declarations, and must report
that limit rather than compare its timing as normal cached compilation. Existing
packages should retain their current cache path. No persistent schema changes are
made by this design note.

Future public admission must resolve ref/aggregate/Option/Result composition,
constants, pattern bindings, public hashes, Atlas identity, debug representation
and explicit FFI scalar conversion before claiming general-purpose newtypes.
No niche compression, nominal arithmetic, trait solving, generic sharing, or
resource typestate is bundled into this proposal.

## Bounded implementation sequence

1. Add compiler-private typed-HIR nominal declarations/conversions, exact validation
   and erasure for locals, comparisons and direct value calls. Use existing HIR
   scalar-call fixtures plus invalid-HIR mutations; no parser syntax yet.
2. Route a copy of the real scanner workload through that boundary and compare the
   baseline/struct/nominal variants. A manually constructed HIR probe proves only
   lowering behavior, not source-level safety or frontend compilation cost.
3. Only if the lowering experiment succeeds, propose source spelling and build an
   explicitly experimental frontend path with diagnostics and same-package source
   identity tests. Keep unsupported interactions rejected. Measure frontend cost
   before any stable language revision or SDK migration.

Each item is a separate reviewable commit. Stop on a failing cost gate and report
the first stage that differs rather than broadening optimization to hide it.

## Acceptance oracles

- Semantic: wrong domain/raw value rejected with expected/actual names and source
  declaration spans; valid explicit conversions accepted; same spelling in separate
  modules remains distinct; no new implicit conversions in existing code.
- Boundary: mutate HIR call/result/comparison/conversion types and verify rejection
  before erasure, including direct lowering and any future cache-loading entry.
- Execution: side-effecting conversion operands execute once; checked arithmetic
  and token bounds retain traps; valid scanner output and cleanup remain unchanged.
- Artifact: compare normalized executable MIR (exclude only source metadata),
  emitted C, function/runtime-call counts, object sizes, EXE sections and total size
  under low and optimized MSVC. Explain nondeterministic metadata separately; do
  not infer unchanged code from equal aligned EXE sizes.
- Cost: measure both feature-using and unchanged corpora, fresh and warm caches,
  check and full build separately. Predeclare repetitions and report distributions;
  the prior two-sample timings establish no speed claim. Do not invent an acceptable
  regression percentage after seeing results.
- Storage: report generated C, objects, EXE, debug/Atlas output and caches separately.
  If "all artifacts must never grow" includes semantic/debug identity records,
  general nominal types cannot be promised cost-free. Adoption requires actual
  measurements or an explicitly revised budget, not a hidden exemption.

This change adds a design note only. Existing experimental measurements are linked,
not rerun or presented as verification of an unimplemented nominal scalar.
