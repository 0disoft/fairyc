[Candidate guide index](../README.md)

## Cleanup body ownership checks (unit 21)

Version 0.52.0 checks nonempty cleanup bodies in the declaring resource's
scope. The implicit `this` is a distinct compiler-owned dropping receiver.
Field reads, direct scalar field updates, local values, ordinary calls, control flow and
void return use the existing type and flow checks. Transferring or replacing
the complete receiver rejects E1102; moving a resource field still rejects
E1103. Cleanup cannot return a value or propagate a fallible Result with `?`.
Local resources and Results keep their usual move and handling obligations.
Receiver shadows refer to their own bindings and `this` does not leak outside
cleanup. Nested field writes through the resource receiver remain rejected,
matching the existing plain-struct-only store contract. Declaration/cleanup
attributes remain outside this milestone.

`tests/resource_cleanup.fy` executes four native compiler API tests and
`verify_resource_cleanup.py` observes 17 source fixtures with UTF-8 output
readers. At unit 21 nonempty cleanup failed closed in owned HIR/MIR/C. Unit 22
adds the owned glue and normal-exit execution described at the top of this file.
Evidence is `docs/ops/evidence/candidate-resource-cleanup-20261007.json`.

The Go reference previously evaluated an ownership-consuming `while` condition
only once during analysis. It now checks the next condition evaluation on
fallthrough/continue edges after loop-state validation, excluding exited locals.
Break and return paths do not perform that extra evaluation. Focused regressions
and all 12 Resource / 24 Result fixtures pass; the newly hashed reference agrees
on acceptance/rejection and the required ownership code for all 33 controls.
Full diagnostic lists need not be identical. Pre-fix observations stay separate.

The requested units 1..20 are complete in the progress ledger. The final audit
retains their individual commit/evidence identities and a bounded native smoke
using the final images. Those historical API tests do not themselves claim
resource execution; unit 22 supplies separate generated-C native evidence.
Linux, performance and stage1/stage2 promotion remain later work.

Unit 24 connects body and nested-scope cleanup before nearest-loop break/continue
and before the for post statement, preserving enclosing roots. Repeated body
construction, zero iterations, nested loop targets and return execute in both
profiles; evidence: `candidate-resource-loop-exits-20261007.json`.

Unit 25 evaluates and freezes a replacement before dropping the old generation.
Self moves and transferred donors discharge once; failed scalar propagation
leaves the previous generation for the function-exit cleanup. Owned tagged
factory propagation remains the unit-27 integration test.

Unit 26 tracks completed owned field and argument temporaries until installation.
On propagation, only completed temporaries clean in reverse completion order.
Complete nested resources run outer cleanup before reverse recursive field cleanup.
A function admits at most 4096 MIR locals (16384 program-wide); earlier larger
recursive snapshot stack failure is resolved, not treated as passing evidence.

Unit 27 carries owned Option/Result roots through construction, moves, pattern
captures, calls, returns and propagation. Structural drops visit only the active
payload. Ignored owned captures live until their arm exits, including return and
nearest continue; later declared arm resources clean first. Source-order temporary
metadata uses bounded u32 storage, avoiding the development stack overflow found
in the larger representation. Direct `old=make()?` success/failure now execute.
Partial field moves and resource fixed arrays keep their existing rejection.

Unit 28 admits the closed drop/store/key attribute and explicit bare-local burn.
No-drop obligations follow nested resources and Option/Result through normal,
return, loop and overwrite edges. Burn has a distinct verified MIR operation 27
and emits no cleanup call. Direct no-drop resources with owned fields, wrappers,
temporary operands, droppable operands and repeated consumption remain rejected.
External-module resource identity stays closed until module integration.

Unit 29 shares one cleanup-edge policy for lexical fallthrough, return,
propagation and nearest loop exits: freeze the value, run the exiting defer
suffix in reverse order, then discharge resources. Registration captures only
copy scalars immediately and in argument order; resource/aggregate captures and
value-returning calls remain rejected. Defer in a cleanup body follows the same
policy without owning the compiler's `this` receiver.

Unit 30 verifies the existing non-unwinding trap policy under both native profiles.
Arithmetic/assertion traps skip pending defer and owned cleanup, including partial
construction. A trap inside defer or cleanup preserves effects already executed
and stops the remaining chain. A no-drop value may terminate in a trap; this does
not turn trap into a destruction permission or a normal return.

Unit 31 admits explicit scalar/struct/resource references, dereference reads and
copyable field reads through references. Canonical reference IDs occupy a separate
high-half namespace; MIR address/load validation and C pointers retain the
original storage. Local/by-value parameter escape, owner mutation/move during
a retained loan, and borrowed fields in ordinary records remain rejected.

Unit 32 projects shared addresses through nested records and named fixed-array
elements (including struct-element fields), without copying intermediate records.
Checked index dominance and whole-root origins are verified before C emission.
Temporary roots, field-array addresses and multi-dimensional element addresses
remain explicitly unsupported in this bounded lane.

Unit 33: lower mutable reference stores and projections. See `docs/ops/evidence/candidate-resource-reference-stores-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 34: preserve shared reborrow generations through calls. See `docs/ops/evidence/candidate-resource-shared-reborrows-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 35: retain mutable reborrow ancestry and last use. See `docs/ops/evidence/candidate-resource-mutable-reborrows-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 36: hold reference arguments through later evaluation. See `docs/ops/evidence/candidate-resource-argument-loans-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 37: substitute return origins across calls and switches. See `docs/ops/evidence/candidate-resource-return-origins-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 38: prove bounded recursive return origin summaries. See `docs/ops/evidence/candidate-resource-recursive-origins-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 39: carry safe borrow leaves through Option and Result. See `docs/ops/evidence/candidate-resource-tagged-borrows-20261007.json` for the bounded executable fixtures and current profile identities.

Unit 40 joins safe-borrow holds across reachable branches and loop exits. Shared
and exclusive loans are released on paths whose last use is in another arm or
that return before the access. A use after the join retains the whole owner.
Loop fallthrough and continue retain external-owner loans across the backedge;
break/return cannot turn a later unreachable break into an exit proof. Dead
returns and uses contribute no origin or last-use hold. Path depth is bounded
at 32 with the shared 1,048,576-work budget; unknown paths stay conservative and
work exhaustion rejects the input. Mutable sibling loans remain exclusive.

This is conservative source-span/outcome analysis for the candidate's admitted
immutable borrow bindings, not the Go frontend's general CFG fixed point.
Predicate correlation, mutable borrow binding reassignment and safe-borrow
storage inside named aggregates/arrays remain closed. Unit 40's 18 fixtures and
24 native runs cover both final candidate profiles. Units 33-39 were also
replayed with those profiles (60 fixtures, 64 native runs). See
`docs/ops/evidence/candidate-resource-borrow-flow-20261007.json` and
`docs/ops/evidence/candidate-resource-borrow-profile-regression-20261007.json`.

The candidate release build required the product COFF inspector's bounded
section fanout adjustment in ADR 0473. The validation bootstrap still uses the
explicit 120-second command-timeout overlay recorded in the receipt; integrating
production timeout policy remains unit 75. Linux and self-hosting stage proofs
remain later work.

Unit 40 completion audit additionally retains all incoming exclusive generations
for if/switch results rather than rejecting a same-owner join as ambiguous.
Synthetic DAG nodes preserve every possible parent until the joined descriptor's
last use; parent operations resume only after all retained children expire.
Mutually exclusive creators are compared by their creation spans, separately
from the extended lifetime of the joined result. A payload-free Option arm has
no incoming generation. There are 30 final fixtures and 40 native runs for unit
40, plus 60 fixtures/64 native runs for final units 33-39 regression. The final
unit-24 success replay has six fixtures/12 native runs, repairing the old receipt's
missing post-correction successful harness row. Earlier temporary admission
rejections remain historical, since later units implement those operations.
See `docs/ops/evidence/candidate-resource-borrow-flow-joins-20261007.json`,
`candidate-resource-borrow-joins-regression-20261007.json`,
`candidate-resource-loop-exits-final-20261007.json` and
`candidate-units-21-40-completion-audit-20261007.json` in that evidence directory.

### Stored byte-array field views (unit 42)

`ref value.field` and `ref mut value.field` borrow stored rank-one `u8[N]` fields, including nested plain struct fields. MIR projects an address into original storage before constructing the descriptor; writes are visible in the owner. Shared struct-reference parameters can return a field view with the parameter origin. Temporary roots remain rejected, and retained mutable siblings keep the ADR0415 whole-owner restriction. SDK text conversion is still pending its unsafe/cast and module dependencies.

### Explicit scalar function specialization (units 44–45)

Same-module `fn T id<T>(T n)` and `id<i32>(7)` specialize in Fairy code before ordinary name/type/HIR/MIR checks. Ordered bool/u8/i32/i64/u32/usize arguments and forwarding to another template are supported. The key is the original declaration and exact ordered types; repeated calls execute even when the function is emitted once. Cloning preserves the original source/token stream and spans in independent arenas. Unused templates remain catalog entries and are not claimed as checked executable bodies. Inference, template-name shadowing, user nominal arguments, recursive expansion and duplicate type parameters are rejected. Expansion is bounded by depth 32, 16 instances per declaration, 64 total, a 1048576-work budget, and existing arena limits. Cross-module/package identity and the authenticated SDK Vec nominal exception remain pending their dedicated units.

### 후보 60: 값 보존 스칼라 캐스트

`as`는 u8→i32/i64/u32/usize, i32→i64, u32→i64/usize의 기존 값 보존 조합만 허용한다. 피연산자는 한 번 평가하고 원래 정수 trap 의미를 유지한다. HIR kind 26/MIR kind 35는 원본·결과 타입을 독립 검증하며 위조 타입과 operator를 거부한다. 좁히기·부호 변경·같은 타입·bool/enum/포인터 cast는 허용하지 않는다. 실행 증거는 `docs/ops/evidence/candidate-resource-scalar-casts-20261007.json`에 있다. 전체 자기 소스 분류는 아직 완료로 치지 않는다.

### 후보 53a: 함수 효과 권한

`@effects`는 language/11의 12개 효과를 중복 없는 bitset으로 보존한다. 함수와 scalar generic 특수화의 직접 호출은 caller가 callee의 효과를 모두 선언해야 하며, forward/재귀·실행되지 않는 short-circuit 호출도 검사한다. HIR binding과 MIR target이 권한을 소유하며 MIR 검증은 caller 권한 제거와 미지 비트를 거부한다. 자원 cleanup은 53b, scalar import 전파는 63번 부분 단위에서 연결했다. 검증은 `docs/ops/evidence/candidate-resource-effects-calls-20261007.json`에 있고 원시 C/실행 기록은 `.cache/candidate-unit53`에 보관한다.

### 후보 53b: 자원 정리 효과

자원 선언의 `@effects`는 비어 있지 않은 cleanup에만 붙인다. 실제로 정리되는 root·임시값·중첩 자원·Option/Result payload의 권한 union을 검사한다. 반환/전달하는 sender, 비-unwind trap과 no-drop burn에는 자동 정리 권한을 요구하지 않는다. `@abilities`와 `@effects`는 순서와 무관하게 해석한다. owned HIR의 resource와 cleanup 함수 권한이 일치해야 하며, MIR은 직접 cleanup call과 tagged/중첩 drop op의 권한을 독립 검증한다. source flow는 root의 Live/Dead 상태를 쓰고 최종 MIR은 부분 임시값까지 확인한다. 증거는 `docs/ops/evidence/candidate-resource-effects-cleanup-20261007.json`. 최초 assertion trap 기대값 1은 실제 계약의 5로 수정했다. 고정 Go 기준 도구는 literal assert(false) 뒤에도 정리 효과를 요구하는 차이가 있다. 후보의 기존 30번 trap 규칙은 유지했고 이를 일치로 보지 않았다.

### 후보 56a: must-use 호출 계약

단일 무인자 `@must_use`는 inert 반환 함수에만 붙인다. 직접 무시한 호출 및 괄호·블록·if/switch의 무시된 끝값은 E4202로 거부한다. binding/return/argument/operator/cast/condition/aggregate 또는 명시 `discard`는 호출 값의 처리로 인정한다. 일반 함수와 scalar generic 특수화의 HIR binding이 marker를 소유하며 MIR/C에는 실행 비용을 추가하지 않는다. resource/Result/safe borrow/void 반환과 malformed 선언은 E4201이다. scalar import marker는 63번, extern-C marker는 55번에서 연결했다. @test/@const_eval 조합은 남았다. 증거: `docs/ops/evidence/candidate-resource-must-use-20261007.json`. tagged .some 생성자를 함수로 잘못 분류하던 초기 walker 오류는 수정했고 정상 Option 처리 fixture로 검증했다.

### Imported scalar call contracts (unit 63 partial)

Registry items own effects and must-use markers; copied external signatures and HIR bindings preserve them after registry cleanup. Provider declaration errors and caller authority/ignored-value failures are checked at the actual imported target. Both native profiles pass two API tests; see `docs/ops/evidence/candidate-resource-module-contracts-20261007.json`. Imported nominal types, generic instances, borrow origins and whole-program linking remain pending.

### Unsafe lexical call scopes (unit 54 partial)

Unsafe Fairy functions and explicit blocks authorize unsafe calls only within their lexical body. Effects, must-use, moves and borrows remain independently checked. Generic cloning remaps unsafe blocks; defer retains permission from registration and cleanup requires its own explicit block. Owned HIR checks the actual block tree, and MIR rejects stripped or mismatched call provenance before C output. Two candidate profiles, 17 fixtures, 18 generated-C native runs and two mutation API tests passed. See `docs/ops/evidence/candidate-resource-unsafe-scopes-20261007.json`. Extern-C imports and raw-pointer values remain pending the next unit.

### Scalar extern-C imports (units 54–55 partial)

`unsafe extern C fn` accepts exact i32/i64/u8/u32/usize parameters and results,
plus void results. A single `@link_name("symbol")` selects an unescaped ASCII
identifier of at most 256 bytes; invalid/reserved C names and incompatible
aliases fail before output. Ordinary Fairy names retain numeric C symbols.
Owned HIR/MIR retain the validated linker span, foreign target, effects,
must-use and unsafe-call facts. Defer captures permission at registration;
resource cleanup still checks its own effects and explicit unsafe block.

The test consumer optionally appends a bounded foreign fixture to its separate
runtime translation unit and hashes those bytes into the build identity.
The generated product translation unit is unchanged by that fixture. Both
candidate profiles pass 24 source fixtures; two C profiles execute 18 native
calls across the object boundary. One native API test rejects four linker/ABI
mutations before C output. Eight earlier unsafe CLI cases still match.
See `docs/ops/evidence/candidate-resource-scalar-foreign-20261008.json`.

Raw pointers, closed C scalar transport, inert repr(C) results and imported
nominal signatures now have separate implementation and native evidence. Escaped
linker names and authenticated SDK fy_ symbols remain pending.
This does not complete SDK Vec/Buffer/File integration or self-hosting.

Raw-pointer candidate FFI progress and exact remaining boundaries: [candidate FFI progress](../../../docs/ops/candidate-ffi-progress-20261008.md).

The candidate also lowers unsafe `str::from_raw_parts(owner,data,length)`,
`byte_slice::from_raw_parts(owner,data,length)` and
`byte_slice::from_raw_parts_mut(owner,data,length)`. The owner must be a shared
safe borrow for text/shared bytes, or an exclusive safe borrow for mutable bytes;
data is an admitted raw u8/void pointer and length is usize. Returned origins and
loans follow the owner, including safe wrapper calls. Unsafe callers must supply
valid accessible storage and valid UTF-8 for text; construction does not scan or
allocate. Text bracket indexing stays closed. CStr representation and authenticated
SDK Vec/Buffer/String/File integration remain pending. See the
[module progress](../../../docs/ops/candidate-module-progress-20261008.md) and
[remaining work](../../../docs/ops/candidate-remaining-work-20261007.md).

The authenticated scalar Vec lane now executes empty/len/capacity/reserve,
push/pop/clear and owner-bound `at/at_mut` for all six scalar types. Vec<u8>
also supports shared/mutable `view` and the named `view_u8`/`view_mut_u8` exports.
Safe aliases retain the vector through their last use, including returned
subviews. Non-byte and nominal views and resource/nominal element storage remain
pending; current evidence and precise boundaries are in the module progress
linked above.
