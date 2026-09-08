# ADR 0387: Retain generic aggregate declarations

- Status: Accepted compiler-private aggregate execution; stable admission pending
- Date: 2026-09-07

## Decision

Retain ordered type parameters on C-shaped aggregate declarations, using the
same parameter-list parser as generic functions:

```text
struct Pair<T, U> { T first; U second; }
resource Holder<T> { T value; cleanup {} }
```

The syntax tree retains parameter nodes and source spans. The AST retains
`Struct.TypeParameters` and `Resource.TypeParameters` independently from fields,
attributes, visibility, documentation, and resource cleanup. Field types remain
structural type references. No source string substitution is used.

Ordinary semantic analysis rejects every unexpanded generic aggregate, including
unused declarations and declarations whose fields do not mention a parameter.
Otherwise a phantom parameter could be silently erased and an ordinary type
published before specialization. Public declarations follow the same gate.

This is a dependency of issue 44, not a `Vec<T>` implementation or a
language/SDK promotion. Language 14 and the runtime ABI are unchanged. Resolved
package identities, cache/interface and Atlas integration, and container operations
remain to be implemented before stable admission. The private execution section
below supersedes the preparation-only boundary for its named internal entry point.

## Evidence

AST tests retain ordered parameters, fields, visibility, declaration provenance,
and cleanup, preserve ordinary declarations, and reject malformed parameter lists.
Compiler tests reject unused private/public struct and resource templates even
when all fields are concrete. The initial source-preparation commits deliberately
produced no verified MIR for generic aggregates.

## Construction syntax preparation

`Holder<i32> { .value = 42 }` retains its type arguments directly on the record
AST, separately from the constructor path and value-field children. Qualified
paths, nested structural types, and trailing commas use the existing type parser.
The 256-token lookahead recognizes either a call or a record continuation; record
recognition still requires a context that allows record expressions. It does not
reinterpret ordinary comparisons or argument-free records.

An unexpanded typed construction is rejected even if its constructor names an
ordinary non-generic aggregate. This prevents type arguments from being ignored
while an otherwise valid record is published as verified MIR. This source-shape
preparation did not change the execution or version boundary.

## Compiler-private concrete execution

`CheckGenericFunctions` and its coordinate-bound module-local variant now admit
private scalar-parameter struct and resource templates. Function and aggregate
requests share one planner, expansion budget, and dependency graph. Field types,
function signatures, local bindings, constructors, and cleanup blocks are cloned
and substituted structurally. Dependency IDs include complete aggregate definitions,
so a cleanup change invalidates the aggregate and factories that use it.

The resulting concrete declarations pass ordinary semantic, Flow, MIR lowering,
and verification. A resource remains move-only; specialization does not change
its cleanup protocol. Recursive dependencies, bad arity, duplicate parameters,
unsupported arguments, implicit type arguments and generated-name access fail
closed. The resolved-package path rejects aggregate templates pending its identity,
interface, cache and Atlas integration. Generic methods and nominal/composite
type arguments remain outside this private slice.

This work also extends ADR 0119's private C resource transport to fields whose
inert struct layout has already passed the aggregate resolver. Struct definitions
precede resource definitions, and target-derived size, alignment and offset asserts
cover the nested field. The distinct feature is `nested-copy-resource`; old flat
features retain their meaning. This does not admit a struct containing a resource
or general resource-to-resource nesting. Interpreter cleanup verifies inert runtime
fields before skipping their drop, rather than silently skipping hidden ownership.

The generic test has a factory returning a resource with a generic inert-struct
payload. It checks shared specialization, source immutability, combined budget,
cleanup-dependent identities, interpreter drop/output traces, and a real native
counter after return and move through a consuming function. Low and optimized
MSVC must both observe exactly one cleanup before returning 42. No allocation,
vector growth, allocation-failure ownership or resource-element container support
is claimed by this test.

## Generic cleanup helpers

Private module-local cleanup blocks may call generic helpers with explicit scalar
arguments or their aggregate's directly forwarded type parameters. Calls use the
same registration and rewriting routines as ordinary function bodies. They create
real planner dependencies, count toward the combined expansion budget, and retain
original source spans. A helper change invalidates the resource and its factories.
Local bindings may not shadow generic helper or aggregate names.

The private native/interpreter fixture also calls `finish<T>` on a copied payload
from cleanup after return, move and consumption. It requires one cleanup and one
observed marker, preserves the original AST, and rejects unknown/unbound/recursive
calls, bad arity, shadowing, and a budget that omits the helper. Resolved-package
cleanup specialization remains closed until its package/interface/cache evidence
is integrated. All dependency cycles remain rejected, including cycles formed by
a helper signature that refers back to the resource being specialized.

## Kind-aware source provenance

The private module path now retains the same current-source provenance boundary
as package function specialization. Original declarations explicitly identify
function, struct, or resource kind. Aggregate signatures retain ordered written
fields and cleanup presence, not function arguments or a synthetic return type.
Unused templates and their documentation locations remain source declarations,
not claims that their bodies have been checked. Type edges and field slices are
detached from both source ASTs and getter results. Failed compilation publishes
no partial sidecar.

Each specialization joins its original declaration ID to the appropriate concrete
MIR function, struct or resource. No generated-name parsing supplies the kind.
The initial function-only Atlas adapter rejected aggregate provenance. The header
adapter described below now handles original aggregate declarations, while full
snapshot implementation joins remain closed. This is not an Atlas promotion or
a serialized interface/cache revision.

## Resolved-package source expansion

`genericsource.PreparePackages` resolves aggregate references through exact import
aliases and package owners. Cross-module access requires a public template;
generated declarations stay in the defining module and rewritten references keep
the caller's alias. Functions, aggregate fields, constructors and cleanup helper
calls share the same dependency graph. Public aggregate closures include unused
templates and exact import bindings, so rebinding a cleanup dependency invalidates
both the resource and its factory. Source ASTs remain unchanged on expansion.

The three-package fixture returns a resource from its owning module's factory,
uses it in the consumer and runs a generic cleanup helper from another package.
Ordinary package semantics, MIR verification, interpreter execution, and low and
optimized MSVC execution return 42. This checks executable package transport, not
an exact cleanup count or allocation-failure behavior. Private access, wrong arity,
missing aliases and generated-name requests publish no partial expansion.

## Compiler-private package interfaces

`AnalyzeGenericPackages` and `CheckGenericPackages` now admit the resolved
aggregate source graph. Interface extraction removes generated functions, structs
and resources rather than treating a specialization as a source export. Original
template closures remain the interface facts, including unused public templates
and cleanup dependencies. Concrete public signatures still use the ordinary
typed visibility and coordinate checks before generated items are removed.

Tests compare used and unused template interfaces, propagate helper edits through
the resource and factory closure, and preserve interface hashes when package
numbers or input ordering change. Verified lowering retains kind-aware original
declarations and specialization provenance.

## Compiler-private aggregate cache reuse

`GenericSession` reuses the existing generic key and verified aggregate artifact
transports for memory and persistent storage. Current-source analysis and
kind-aware provenance are still rebuilt before cache lookup; templates are not
recovered from serialized MIR. Source interfaces and owner specialization sets
participate in keys, and all-hit results still require whole-program reassembly.
No cache encoding or stable artifact format is changed by this admission.

The cache fixture owns a nested `Cell<T>` struct, a `Box<T>` resource, its factory
and a private generic cleanup helper. Memory and disk cold checks store two
package artifacts. Warm checks hit both with no additional lowering, join all
four current origins to the right MIR kinds, and return 42 in the interpreter
and low/optimized MSVC. A separate process reads the persistent artifacts with
lowering disabled and repeats those provenance and execution checks. This is
cache-reuse evidence, not a measurement of exact cleanup counts.

Adding a bool specialization or changing the cleanup helper invalidates both
package artifacts. Invalid current source publishes no hits, stores, or partial
provenance even when prior valid artifacts remain available. Stable `Check` and
`SourceSession` still reject aggregate declarations through ordinary unsupported
source analysis. Atlas integration and stable language/SDK promotion remain
pending; language 14 is unchanged.

## Original aggregate headers for Atlas

Compiler provenance now retains detached written-name bindings from each source
scope to visible original aggregate templates. Local declarations and exact
public import providers are included; private imported names are not. These facts
identify templates and arity, not generated nominal types or checked unused bodies.

`GenericDeclarations` uses those facts to represent struct/resource headers with
their original kind, ordered field names/types/spans and a coordinate-bound self
type. Nominal template applications retain recursively converted type arguments.
Binders use canonical positions rather than display names. Cleanup presence is
the `source-declared` marker, not a claim about empty/non-empty inferred drop glue
or effects. Aggregate attributes still require separate adaptation and fail closed.

The existing SymbolId encoder can encode these generic aggregate headers without
changing ordinary identities or its format. Tests preserve IDs across binder and
alias renaming and package renumbering, distinguish actual provider UIDs, and
check detached getters and header results.

## Aggregate implementations in Atlas snapshots

`BuildCompiler` joins concrete struct/resource types and fields to original
templates, substituting concrete arguments without exposing generated type names.
Aggregate specialization summaries retain inferred drop kind separately from
source-declared cleanup presence. Resource cleanup bodies have separate execution
summaries keyed by the original resource; calls identify the exact helper
specialization. Borrow analysis also runs for cleanup bodies and retains their
private function identity, even when specializations share source spans.

Ordinary function summaries continue to exclude private drop bodies and their
borrow facts. A private borrow must match its actual cleanup body's span; it is
not attributed to a neighboring source function. Generic cleanup uses a separate
summary view without modifying executable MIR.

Tests cover two resource specializations, nested fields, cleanup helper calls and
borrows, documentation attachment, detached getters, inconsistent concrete fields,
and ordinary cleanup ownership. The v1/v2 planners still explicitly reject
aggregate specializations. No stable language, SDK or ABI promotion is made.

## Opt-in aggregate map codec

The v3 logical planner and codec preserve aggregate declarations, concrete fields
and separately owned cleanup summaries. The exact contract and mutation evidence
are in `docs/atlas/fymap-transport-v3.md`. Existing v1/v2 bytes and explicit entry
points remain isolated. The opt-in v3 store and current-source expectations are
implemented as documented in `docs/atlas/fymap-store-v3.md`; they preserve the
existing opened-root I/O and version-local retention boundary. Package-local v3
queries now expose concrete aggregate fields and separately attributed cleanup
relations under the existing selection budgets; see `docs/atlas/fymap-query-v3.md`.
The opt-in v3 workspace join now preserves original resource ownership and exact
source/target specialization IDs for cleanup calls across direct dependencies.
Codec-roundtripped two-package tests cover both selector directions, deterministic
ordering, detached output, call budgets, stale source and missing concrete targets.
It does not model implicit drop scheduling. The opt-in call catalog v2 encoder
now preserves this cleanup source kind and both concrete endpoints with a separate
checksum domain version and atomic byte bound; see
`docs/atlas/workspace-call-catalog-v2.md`. CLI consumers must adopt v3 before stable
aggregate admission.
