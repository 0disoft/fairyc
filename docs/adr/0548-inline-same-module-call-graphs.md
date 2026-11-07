# ADR 0548: Inline same-module call graphs

- Status: Accepted for bounded prototype
- Date: 2026-11-07
- Extends: [ADR 0493](0493-cross-package-inline-bodies-and-cache-v5.md) and
  [ADR 0509](0509-inline-immutable-local-bindings.md)

## Source and execution

An admitted public safe nongeneric `@inline` i32 function may call another
admitted public safe nongeneric `@inline` function in the same source module.
Calls may occur in initializers, return expressions and arguments, including
nested and repeated calls and declarations written later in the module.
Resolution must use the actual checked declaration; a local or parameter with
the same name cannot authorize a call. The existing limits of four immutable
i32 parameters, sixteen immutable i32 locals and sixty-four typed expression
nodes per body remain. A call counts as one expression node; its callee name
is not an arithmetic value node. Each actual argument counts normally.

Every root may reach at most sixty-four distinct inline declarations, including
itself. Shared callees count once. Self/mutual recursion and an over-limit graph
are rejected before Complete publication. The existing ADR 0396 restriction
against argument labels on public functions remains; all admitted inline calls
therefore use positional arguments. Admission hashes references rather
than substituting callee bodies, avoiding exponential expression expansion.
Ordinary, private, generic, foreign, unsafe, effectful and other-module callees
remain outside this first call family. Mutation, control flow and non-i32
inline values remain separate designs; the general-inline queue stays open.

Ordinary argument evaluation executes once in source order, followed by all
local initializers in source order and the return. Existing checked arithmetic,
traps, source provenance, MIR and private C linkage govern every call. An unused
initializer still executes. C compiler inlining remains optional.

## Canonical identity and cache closure

No-call bodies retain exact v1/v2 canonical bytes. A body containing a call uses
`fairyc.inline-function/v3` with the existing `Version`, `Arity`, `Root` fields.
Locals retain the ordered `bindings` root. A call node has `Op=call`, the bare
callee declaration name in `Value`, and ordered argument nodes in `Children`;
zero arguments encode JSON null. Callee names intentionally participate;
parameter/local renames, source spans and equivalent literal spellings retain
their existing rules. The digest carrier remains `fyinlinefn1-sha256-`.

Every admitted target is an exported inline declaration in the same module
and therefore belongs to the existing owner's all-export inline package class.
Changing any leaf body changes that class even when callers' source and
function marker remain identical. Existing owner/direct-dependent cache keys
already include that class. No hidden ordinary-helper or transitive package
body dependency is admitted, and no cache-key/wire version is changed. Extending
calls across modules/packages requires a separately specified dependency closure.

Source checking validates actual targets and graph closure. Detached body
verification reconstructs v3 identities and validates argument records; complete
program verification rejects unmarked/invalid targets and cyclic/over-limit
graphs. Restoring individual bodies does not bypass whole-program verification
or current-source marker checking. Missing/changed/extra markers retain their
existing rejection and healing rules.

## Delivery

Go implementation, Go workspace-cache proof, Fairy source admission and Fairy
detached/program artifact verification have disjoint owners. Windows evidence
is recorded after actual execution. Linux remains in the batched handoff;
final full-source self-build/fixed point, hosted CI and distribution refresh
remain separate. No public API/ABI, DB, dependency, runner, target promotion,
default cache-policy or remote action is introduced.
