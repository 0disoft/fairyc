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
The function-only Atlas adapter explicitly rejects aggregate provenance until its
aggregate declaration/signature support is implemented. This is not package
aggregate admission, an Atlas promotion, or a serialized interface/cache revision.

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

The compiler's `AnalyzeGenericPackages` entry point still rejects aggregate
templates before publishing interface hashes or cache specializations. Its gate
now belongs to compiler package admission rather than the source expander.
Interface/cache/Atlas integration and stable language/SDK promotion remain pending.
