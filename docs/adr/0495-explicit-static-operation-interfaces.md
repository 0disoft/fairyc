# ADR 0495: Explicit static operation interfaces

- Status: Accepted for Go Stage 0 and bounded Fairy candidate Windows prototypes
- Date: 2026-11-01
- Supersedes ADR 0223's deferral for this user-requested bounded family only.

An interface is one explicitly declared generic operation, written using existing attributes:

```text
@interface pub fn T transform<T>(T value) {}
@implements(transform, i32) fn i32 twice(i32 value) { return value * 2; }
pub fn T algorithm<T>(T value) { return transform<T>(value); }
```

The operation has exactly one type parameter, a T result, zero through four T value
parameters and an empty safe body. Each implementation is one safe ordinary function
with the exact builtin scalar signature, owned by the same module. Imported public
operations may be consumed by explicit generic calls; orphan/blanket/implicit providers,
method groups, defaults, inheritance, associated types, dynamic values and tables remain
closed. No interface value, allocation or runtime dispatch is introduced.

The existing planner capability kernel selects one exact provider per scalar tuple.
Missing providers fail at the original root request with E4502; malformed declarations,
wrong signatures and duplicate tuples fail with E4501. Implementations still pass ordinary
name, typing, effect, Flow and MIR checks. Effect propagation follows the selected ordinary
calls; an operation is not a promise of an implementation-independent effect ceiling.

One bounded wrapper is emitted per specialization. Provider bodies and operation metadata
enter both specialization constraints and exported template definition closure, after import
binding. Provider edits invalidate owner/dependents in generic cache; unrelated private
bodies retain dependent identity. Caller AST remains detached and source ownership exact.

The user explicitly requested interfaces after generic support. This prototype replaces
ADR 0223's earlier corpus-demand precondition for the named operation family; it does not
claim that the old three-production-module workload requirement was measured or fulfilled.
Source grammar, SDK/candidate promotion, runtime ABI, CLI flags and runner policy do not change.

Evidence: `TestStaticInterfaceExecution` runs i32/i64 provider selection in the interpreter
and MSVC 14.51.36231 C11/C17 x low/optimized; negative declarations and missing operations
reject. `TestStaticInterfaceCacheInvalidation` crosses a package boundary through public
operations/algorithms and verifies memory/fresh-session disk cold/warm, provider edits,
unrelated private edits and missing-provider rejection. Existing generic source/planner tests
and scoped vet pass. Native Linux uses the same GCC/Clang helper later; it remains unexecuted.

Fairy candidate evidence is recorded in
[the Windows static operation receipt](../ops/windows-candidate-static-operations-20261102.md)
and [candidate static operations v1](../architecture/candidate-static-operations-v1.md).
Native low/optimized compilers select all thirteen scalar families, restore whole
project artifacts and produce four passing C11 executions. Independent Go root,
specialization and public closure identities agree across selected/unused provider
edits and unrelated private edits. Native Linux, candidate persistent-cache
acceptance and candidate promotion remain deferred.
