# ADR 0493: Cross-package inline bodies and cache key v5

- Status: Accepted for prototype
- Date: 2026-11-01
- Windows command/cache update: 2026-11-03
- Owners: [Cache key v5](../package/whole-package-cache-v5.md),
  [Phase 3 reopening](../development/phase-status.md#phase-3-inline-reopening-2026-11-01)
- Extends: ADRs 0173 and 0349; supersedes the no-inline portion of the 2026-10-21 Phase 3 decision.

## Bounded source contract

Admit exactly one argument-free `@inline` on a public safe nongeneric ordinary function.
The result is i32 and zero through four immutable parameters are i32. Its entire body
is one returned i32 expression (explicit return or tail), with at most 64 typed nodes:
integer literals, parameter paths, parentheses, unary +/- and binary +,-,*,/,%. Calls,
effects, locals, mutation, control flow, recursion, unsafe bodies, owner methods,
captures, aggregate/resource/borrow values and generic inline functions remain closed.
Existing attribute syntax is reused; no grammar production changes.

The typed body is exported to the backend with a versioned identity. C emission uses
`static inline` for its prototype and definition. Partitioned emission puts one definition
in the private shared header and omits it from separately compiled function sources,
so callers in other package partitions see the body. Linkage remains private. This is
an optimization hint: the C compiler may retain a call, particularly at low optimization.
It is not forced substitution, a public C export, or an assembly-level speed guarantee.
Argument evaluation remains ordinary MIR evaluation: once, in source order, before calling.
Existing checked arithmetic, traps and source provenance remain intact.

## Identities and cache contract

`fairyc.inline-function/v1` hashes a canonical typed expression and parameter arity;
`fyinlinefn1-sha256-` carries its full digest in HIR and MIR. Parameter names, HIR IDs,
source spans, whitespace, comments, grouping and equivalent integer spelling are excluded.
Operator tree, parameter positions, literals and arity participate. The parser's admitted
integer spellings retain their existing grammar restrictions.

`fairyc.inline-bodies/v1` binds sorted module/function names and these IDs to a stable package
coordinate. `fyinline1-sha256-` is the full package class digest; zero denotes no inline
exports. Missing owners, duplicate names and malformed IDs fail closed. Calls are closed
in inline bodies, so no hidden helper/transitive body closure is omitted from this class.

Public-interface v5 stays unchanged and excludes ordinary and inline bodies. The explicit
successor cache-key v5 adds the owner's inline class and every sorted direct dependency's
inline class, while retaining all v4 compatibility and exact contract-state body members.
An inline-only edit invalidates owner and direct dependents; ordinary/private body edits
retain existing dependent stability. Every inline export of a dependency participates,
even when unused: the initial whole-package rule is intentionally conservative.

Process and persistent ordinary/generic cache paths compute the class before lookup.
After reassembly they compare inline markers against freshly checked HIR before exposing
an all-hit result. Missing, changed or extra markers force verified lowering and cache
healing. Stable artifacts preserve the marker through renumbering and reassembly; MIR
rejects malformed IDs, unsupported function state, calls and cyclic/excessive inline graphs.
The marker is not an authenticity proof for hostile executable cache contents; normal
artifact bounds, checksums, context/function identities and verifier re-entry still apply.

V5 keys use `fycache5-sha256-`; old prefixes are incompatible disposable misses. The v4
byte tuple stays frozen in EncodeLegacyV4 and its original digest test, and refuses inline
facts. Existing no-inline public interface and C-emission fixtures remain unchanged.
The package-cache fixture format remains v8 (same JSON shape) with its baseline/schema key
expectation deliberately advanced to v5. The private key-ledger verifier follows v5.

## Delivery and evidence

Implement core identity/emission/cache behavior first, then record Windows native and
partition observations in a separate receipt. Reopen Phase 3 only for this bounded family;
Linux native runs remain in the maintainer's batched queue. No target promotion or default
cache-policy change is implied. SDK/CLI API, DB, dependencies, runtime ABI, repository hygiene
and CI/runner configuration are unchanged; no push or hosted dispatch is authorized here.

## Windows command/cache update

The [CLI/cache receipt](../ops/windows-candidate-inline-cli-cache-20261103.md)
adds full authored Windows command acceptance and 26-command fresh-process
cache replay under both compiler profiles.
The nongeneric caller/dependency graph verifies current inline bodies after
source and annotation changes, canonical-body mixed reuse, corruption repair
and invalid-source rejection. Whole-project class invalidation permits separate
[current-source rechecked caller IR reuse](../architecture/candidate-package-unit-cache-v0.md);
it does not require zero reused units. Fresh C uses the current inline provider.

The [generic-caller receipt](../ops/windows-candidate-generic-inline-cli-cache-20261103.md)
extends command evidence to `evaluate<i32>` and `evaluate<bool>` callers of the same
nongeneric exports. Both specializations execute the ordered four-argument call,
i32 minimum and division, then agree on 42. Both persistent-cache profiles pass
all 26 commands. Generic use of these
exports is admitted without admitting generic inline declarations or changing the
source contract.

This evidence changes no admitted forms, public API, cache wire format, ABI,
dependencies, repository policy or CI/runner configuration. Generic-wrapper
full-CLI and persistent-cache acceptance are covered; native Linux,
full-suite/hosted CI and distribution remain separate. No push or promotion is implied.

## Windows inline trap follow-up

Windows release compilation exposed an MSVC-only consequence of constant actuals reaching a
parameterized inline checked operation: the compiler could treat the inline call as unconditionally
trapping and reject the caller's valid continuation before execution. Go commit `f838dc5c` reads each
private i32 parameter once through `volatile int32_t fy_inline_arg_N` and back, only for marked
inline bodies with nonzero parameters and only under `_MSC_VER`. Other compilers preprocess the barrier
away. The [trap receipt](../ops/windows-candidate-inline-traps-20261103.md) bounds evidence to
division-by-zero, signed-division-overflow and addition-overflow. This adds no warning suppression,
public contract, MIR, ABI or cache-wire change, and does not establish zero-parameter or full
constant-call behavior.

The [zero-parameter follow-up](../ops/windows-candidate-inline-zero-traps-20261103.md) extends the
same three categories to literal-only bodies. Go commit `b80b01f8` reads the existing private
volatile divisor in marked zero-parameter inline i32 division guards under `_MSC_VER`. Its
addition-overflow non-returning proof stays unchanged. The Fairy emitter passes these named
cases without a new production change. No warning suppression, public contract, MIR, ABI or
cache-wire change is introduced; full constant-call behavior remains unproven.

The [remainder/multiplication receipt](../ops/windows-candidate-inline-remainder-multiply-20261103.md)
extends the Windows observations to parameterized and zero-parameter `% 0`, signed-minimum
remainder by negative one, and `46341*46341`, through ordinary and generic bool-marker callers.
The existing Go/Fairy guards handle these cases without another production correction. This adds
bounded regression evidence, not a new language, cache or ABI contract.
