# ADR 0387: Retain generic aggregate declarations

- Status: Accepted frontend preparation; aggregate execution not admitted
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

This is the first dependency of issue 44, not a `Vec<T>` implementation or a
language/SDK promotion. Language 14 and the runtime ABI are unchanged. Generic
aggregate construction expressions, concrete type expansion, layout ownership,
package identities, cache/interface and Atlas integration, native cleanup proof,
and container operations remain to be implemented before stable admission.

## Evidence

AST tests retain ordered parameters, fields, visibility, declaration provenance,
and cleanup, preserve ordinary declarations, and reject malformed parameter lists.
Compiler tests reject unused private/public struct and resource templates even
when all fields are concrete. Native tests cannot establish generic aggregate
execution yet because this preparation deliberately produces no verified MIR.
