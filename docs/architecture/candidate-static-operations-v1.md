# Fairy candidate static operation interfaces v1

- Owner: [ADR 0495](../adr/0495-explicit-static-operation-interfaces.md).
- Status: Bounded Windows prototype; native Linux remains deferred.
- Evidence: [Windows receipt](../ops/windows-candidate-static-operations-20261102.md).

The Fairy-authored compiler admits the same explicit single-operation family as
Go: exactly one type parameter, a direct T result, zero through four immutable T
value parameters and an empty safe `@interface` declaration. A safe, ordinary,
same-module `@implements(operation, scalar)` function provides one exact tuple.
All thirteen builtin scalar families are recognized. Duplicate/malformed/orphan
providers fail with E4501; a missing tuple fails with E4502 at the originating
explicit root request. There is no interface value or runtime dispatch.

The syntax owner installs compiler-private wrapper rows without changing source
bytes. A private virtual path token retains the operation's diagnostic span while
naming the ordinary provider. Token copying and module joins preserve that tag.
The body owner selects the actual provider for each concrete specialization;
ordinary call resolution, value typing, effects, Flow and MIR still check it.
Zero/four parameter wrappers retain argument order and issue one ordinary call.
Original-source JSON and dependency collection retain the written empty body,
not the compiler-installed executable wrapper.

The original operation AST and every explicitly supplied provider, sorted by
scalar name, enter `source-static-operation-v1`. Each provider also enters
`source-static-provider-v1`. The operation definition binds its package/import
identity with `source-static-bound-operation-v1`; its planner constraint contains
the exact capability definition, concrete argument and selected provider digest.
These facts use Go's canonical encodings. Public template closures include the
bound operation, so even an unused provider edit invalidates dependent identity.
An unrelated private body edit with unchanged public/template spans does not.
Metadata keeps its local work limit and charges the enclosing project ledger;
accumulated project work does not become a spurious local metadata limit.

Whole-project artifact encode/decode and current-source restoration precede the
native C replay. Cache wire versions and public CLI flags do not change. The
proof covers identities and artifact restoration; it does not newly establish a
candidate whole-package persistent cold/warm cache acceptance run.

Implementation owners are `static_operation_declarations.fy`,
`static_operation_syntax.fy`, `static_operation_identity.fy`,
`generic_source_catalog.fy`, `generic_source_plan.fy`,
`generic_source_dependency_json.fy`, `generic_source_dependency_walk.fy` and
`module_unit_{generics,bodies,body_ast,name_model,calls}.fy` under
`examples/lexer_candidate/src/`. Native fixtures and Go oracles live under
`examples/lexer_candidate/tests/static_operation*.fy` and
`internal/cli/candidate_static_operation*_test.go`.

Dynamic or multi-method interfaces, inference, nominal/aggregate providers,
associated types, default/blanket/orphan implementations, boxing, allocation and
retained/asynchronous callbacks remain outside this family. Copied local closure
candidate parity remains a separate task.
