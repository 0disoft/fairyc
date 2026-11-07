# Fairy candidate static operation interfaces v1

- Owner: [ADR 0495](../adr/0495-explicit-static-operation-interfaces.md).
- Status: Bounded Windows prototype with full-CLI and persistent-cache evidence
  under both compiler profiles; native Linux remains deferred.
- Evidence: [source/artifact and C receipt](../ops/windows-candidate-static-operations-20261102.md)
  and [full CLI and persistent-cache receipt](../ops/windows-candidate-static-operations-cli-cache-20261103.md).

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
proof covers identities and artifact restoration. The full authored candidate
CLI now exercises cold/locked checks, development/release builds and runs, JSON
tests and interpreter/native tests under both compiler profiles. Both profiles'
persistent-cache replay also passes generation and lock stability, provider and
private-body invalidation, mixed-unit reuse, corrupt-generation repair and
missing/wrong/duplicate provider rejection.
Windows `test` remains a non-cache command, so no test result is cached.

All thirteen provider tuples are admitted in the same package. The ordinary
program/tests execute the original six scalar families and zero/four-argument
operations. Seven extended wrappers are retained through a literal-false branch
without execution; a separate test-only low C observer invokes those packet
entries with boundary values and compares object bytes, including signed zero
and `UINT64_MAX`. This adds no source literals or arithmetic and does not admit a
full seven-scalar interpreter runtime path.

Implementation owners are `static_operation_declarations.fy`,
`static_operation_syntax.fy`, `static_operation_identity.fy`,
`generic_source_catalog.fy`, `generic_source_plan.fy`,
`generic_source_dependency_json.fy`, `generic_source_dependency_walk.fy` and
`module_unit_{generics,bodies,body_ast,name_model,calls}.fy` under
`examples/lexer_candidate/src/`. Native fixtures and Go oracles live under
`examples/lexer_candidate/tests/static_operation*.fy` and
`internal/cli/candidate_static_operation*_test.go`.

## Grouped static operations

[ADR 0549](../adr/0549-grouped-static-operation-interfaces.md) adds
`@interface(Label)` groups of 2..16 public operations, with a 1..64-byte ASCII
identifier label scoped to the original module. Existing per-operation provider
syntax remains. Every scalar tuple supplied for any member must be complete for
all members, including unused tuples. Operation declarations precede provider
errors, then size precedes completeness. Group errors retain the original full
declaration span; a partial tuple selects its earliest provider.

Members share `source-static-operation-group-v1` over all original member ASTs
and all providers, sorted by member then scalar name. Distinct capability and
provider identities and bound-operation domains remain. Ungrouped identity bytes
are unchanged. There is still no interface value or dynamic dispatch.

Go source/native and memory/new-process disk-cache checks pass. Fairy admission,
original-byte diagnostics, Go metadata/planner comparisons and restored-artifact
C execution pass both Windows profiles; see
[admission](../development/candidate-grouped-static-admission-20261107.md) and
[metadata/native](../development/candidate-grouped-static-metadata-20261107.md).
The [grouped CLI/cache receipt](../development/candidate-grouped-static-cli-cache-20261107.md)
adds actual baseline commands, mutation/invalidation, rejection and project
repair. Low evidence composes completed steps from an interrupted run with its
authenticated resumed tail; optimized runs the full matrix. The reduced low
default was not rerun from cold. The workspace CLI retains its existing
module-wide E4501 presentation; exact original declaration spans above belong
to the source admission API. Native Linux and final whole-source gates remain
separate.

Dynamic interfaces, inference, nominal/aggregate providers,
associated types, default/blanket/orphan implementations, boxing, allocation and
retained/asynchronous callbacks remain outside this family. Copied local closure
candidate parity remains a separate task. Native Linux, full `go test ./...`,
hosted CI, self-hosting and clean-machine distribution remain deferred.
