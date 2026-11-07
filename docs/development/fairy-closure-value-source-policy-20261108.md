# Fairy closure generic source policy, 2026-11-08

Bounded worker receipt for ADR 0551 in `.cache/closure-runtime-20261108`.
The worker made no commit or push and did not write shared fairy.lock, main,
existing C tests, source-preparation files or parent project/unit IR files.
No source-of-truth contract, public ABI, DB, runner or dependency changed.

## Reproduction before production edits

Original source cases entered the Fairy scanner/parser, then
`name_resolution::resolve` (including `generic_expand`), `type_check::check`,
HIR lowering, MIR lowering and MIR verification. Go supplied only the compiler
for this bounded producer and a separate original-source CLI reference oracle;
no Go AST, inferred type or lowered target was passed into the Fairy pipeline.

The initial nine-case native comparison failed in both profiles:

```text
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueSourcePolicyNative$' -count=1 -v -timeout 10m
```

Exit 1, native test 26.21s. The verified hole was:

```text
module main;
struct Holder{fn i32(i32) action;}
fn Holder forward<T>(Holder value,T marker){discard marker;return value;}
fn i32 main(){
  let fn i32(i32) action=fn [] i32(i32 x){return x;};
  let holder=Holder{.action=action};
  let copy=forward<i32>(holder,1);
  let extracted=copy.action;
  return extracted(42);
}
```

For the exact compact fixture bytes, Go CLI `check` refused with exit 3 and
monomorphic-source-function diagnostics at 49..55, 67..79 and 112..117. Fairy
reported `phase=9 code=0` (MIR verified) in both `[run]` and
`[run --release --locked]`. Thus the source boundary was demonstrably lost for
an ordinary closure-bearing nominal signature closed with an i32 type argument.

Other initial findings were preserved rather than assumed:

- Explicit fn13 `forward<i32>` already refused (names phase 4, code 3).
- `identity<Holder>` already refused with E4502; an ordinary scalar-bearing
  `identity<Scalar>` also refused, so that restriction was not closure-specific.
- Unused generic fn13 parameters/results and unused generic Holder parameters
  are accepted by the actual Go CLI and Fairy. A blanket raw-declaration gate
  would incorrectly reject these cases.
- Used and unused scalar generic neighbors of a typed closure were accepted.

## Small production change

`closure_value_source_policy::function` inspects only the selected written
source template's result and ordinary function parameters. Structural type
children and source-owned nominal fields determine closure containment.
Nominal recursion has an active-declaration guard; lexical generic type names
are distinguished from same-named structs. Existing work/depth limits remain;
no capability or generic argument admission was broadened.

`generic_expand::instance` invokes that check after selecting the original
template and before cloning declarations/types/body into a monomorphic function.
A signature containing a safe closure returns E4502 at its original type span.
Unrequested generic templates remain untouched. The check is bypassed when the
original type arena contains no safe-function node (kind 17).

The parent's existing `private::copy` plus `assemble_private` receipt-preservation
fix in generic_expand.fy remains intact. No helper metadata is discarded.
The new policy module is compiler-private; the existing public expand API stays
unchanged. No original source bytes are rewritten.

## Final validation

```text
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueSourcePolicy(Source|Syntax|Check|Native)$' -count=1 -v -timeout 10m
```

PASS, exit 0, package 32.030s:

- Source: 0.12s. Eleven original Go CLI reference cases; observed admissions
  pinned after the initial independent reference run.
- Syntax: PASS, original source and authored producer syntax.
- Check: 3.34s. Isolated bounded producer: 162 sources, 1871 MIR functions,
  two instances. Lock changes occurred only in its temporary workspace.
- Native: 28.44s. All eleven cases agree with Go through both low and optimized
  Fairy producer profiles; accepted cases reach verified MIR. No native opt-in
  environment gate; only the Windows AMD64 platform guard remains.
- Scoped git diff --check: PASS.

Final outcomes in both profiles:

| Original source case | Go | Fairy |
| --- | --- | --- |
| Explicit fn13 forward with i32 type argument | Refuse | E4502, 15..26 |
| Holder forward result/parameter with i32 type argument | Refuse | E4502, 49..55 |
| Holder parameter only, scalar return | Refuse | E4502, 63..69 |
| identity<Holder> | Refuse | E4502, 194..210 |
| Unused generic fn13 parameter | Accept | Verified MIR |
| Unused generic fn13 result | Accept | Verified MIR |
| Unused generic Holder parameter | Accept | Verified MIR |
| Scalar generic near typed closure | Accept | Verified MIR |
| Unused scalar generic near typed closure | Accept | Verified MIR |
| Scalar generic parameter shadows closure-bearing struct name | Accept | Verified MIR |
| identity<Scalar> | Refuse | Existing E4502, 182..198 |

The direct fn13 refusal now occurs earlier with E4502 at its written result type;
admission stays closed. This unit claims admission parity, not byte-identical Go
and Fairy diagnostic text. Whole-project/unit IR design, Linux, full CLI/cache
restoration and complete C execution gates remain with the parent. The existing
full-project source-set treatment of unused generics was not changed here.

## Exact worker file scope and final hashes

- `examples/lexer_candidate/src/closure_value_source_policy.fy`
- `examples/lexer_candidate/src/generic_expand.fy`
- `examples/lexer_candidate/tests/closure_value_source_policy.fy`
- `internal/cli/candidate_closure_value_source_policy_test.go`
- `docs/development/fairy-closure-value-source-policy-20261108.md`

| Path | SHA-256 |
| --- | --- |
| `examples/lexer_candidate/src/closure_value_source_policy.fy` | `f231205de1e133efcdb338a3d7c1679c6644a3da2e712ce35bc273fde49d53ee` |
| `examples/lexer_candidate/src/generic_expand.fy` | `e66c1303b0bdb73e5f7bbfa5aeac1a227806d18b95f1fe9d6e1d5217210fac19` |
| `examples/lexer_candidate/tests/closure_value_source_policy.fy` | `4a62c846fa45f4c60ce0d08b3a699dcdc763a0935af44e1db37dfe805327e4b8` |
| `internal/cli/candidate_closure_value_source_policy_test.go` | `d547949db58c1942da7093d7ea56026420410281563ee345dfe347f4cb89d29b` |
