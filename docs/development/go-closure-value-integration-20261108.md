# Copied closure values: Windows integration, 2026-11-08

Contract: [ADR 0551](../adr/0551-copied-closure-values.md). Source, MIR tests,
backend/interpreter, and Fairy type retention used disjoint writer scopes;
the codec followed the completed Fairy type worker. Coordinator integration
and commits occurred serially on main. Workers' frozen receipts remain separate.

## Delivered boundary

The Go source pipeline creates and invokes `fn i32(i32)` values with zero through
four copied i32 captures. Returned factories, different targets through the same
parameter, immutable local/ordinary-struct copies, snapshot semantics, once-only
argument evaluation and checked traps execute through verified MIR, interpreter
and generated C. No creator-frame pointer survives construction.

The target marker cannot authorize an arbitrary body. The verifier checks exact
i32 signatures/operands/results, capture arity, pure arithmetic, immutable locals,
acyclic control flow, initialized callable operands and one normal successor.
Ordinary direct struct fields are the only admitted aggregate storage. Wrappers,
resources, nested aggregates, pointer/foreign transport and private environment
field access remain closed. Reserved closure identities cannot be shadowed by
aggregate definitions.

Whole-program MIR selects `fairyc-mir-closure-values/1` only for the structured
feature; feature-free ordinary/storage bytes retain their existing versions.
Legacy package artifacts reject closure records with the inspectable
`ErrClosureValuesUnsupported` cause. Extraction visitors reject before publishing
any package. Process encode/decode, stable-body validation/projection and type
contexts retain this gate. No native pointer enters the MIR codec.

This is source/MIR/native support, not package-cache or complete workspace CLI
acceptance. Stable lifted-target IDs, source-owned restoration, new-process
package-cache tests, Fairy executable admission and final compiler acceptance
remain separate work. Fairy's type/JSON retention is already integrated in
`61d940e3`; semantic preparation is committed separately in `6666fed7`.

## Integrated checks

Commands ran on Windows AMD64 with `GOMAXPROCS=1`, `-p=1` and unchanged limits.

| Scope | Result |
| --- | --- |
| Complete closureprep, semantic, AST, HIR packages in main | PASS: 0.057 / 0.687 / 0.044 / 0.101s |
| Complete MIR package after all opcode consumers integrated | PASS: 0.527s |
| Complete MIR codec, packageartifact, interpreter packages | PASS: 0.104 / 1.967 / 0.080s |
| Backend layout, MIR emission and five original-source execution/emission cases | PASS: 0.082s |
| Compiler and genericsource closure regressions | PASS: 8.390 / 0.043s |
| Original-source returned-factory/struct/snapshot native recheck in main | PASS: 3.406s, actual low and optimized Windows products |
| MIR, MIR codec, packageartifact and interpreter vet | PASS |
| Final vet of all nine changed source/runtime packages | PASS, no diagnostics |
| Final main CLI structural-type oracle | PASS: 0.189s |

Retained main logs are `.cache/closure-value-semantics-main-attempt1.log`,
`closure-value-mir-main-attempt1.log`, `closure-value-mir-main-attempt2.log`,
`closure-value-mir-main-vet-attempt1.log`, `closure-value-cbackend-main-attempt1.log`,
`closure-value-compiler-main-attempt1.log` and
`closure-value-native-main-attempt2.log` under `.cache/`.
The final nine-package vet is retained in
`.cache/closure-value-integration-main-vet.log`. Integration compared all 81
changed/new paths from the isolated closure checkout: all bytes matched main
except the Fairy type receipt, whose coordinator integration section was
deliberately appended after copying. No implementation path was omitted.
The final CLI oracle log is `.cache/closure-value-oracle-main-attempt1.log`.

The first main MIR run caught dispatch-coverage failures while the C consumers
were not yet copied. The same immutable opcode must reach MIR, interpreter,
codec and C consumers together; the runtime is therefore one atomic commit.
The original failed log remains. A first native filter selected no tests
(`closure-value-native-main-attempt1.log`); it is not a native pass. The corrected
filter explicitly selected `TestClosureValueSourceNative/factories-struct-snapshot`.

The isolated runtime receipt supplies 28 actual Windows executions: C11 eight,
C17 eight, partitioned C two and original-source C ten (five cases times two
profiles). Total is exactly 28.
The final main native check repeats only the representative factory case; the
other unchanged native results were reused. The MIR worker supplies 96 leaf
source/forgery cases; four additional reserved-identity/private-field cases and
dormant feature checks pass in the final complete MIR run. The codec worker
supplies 440 structural version-gate combinations and ordinary/storage 0/4-capture
round trips; these are not package-cache restoration evidence.

The source contract ADR was not changed in this integration. Internal AST/HIR/MIR
APIs and private codec selection change deliberately. Public SDK/FFI ABI, DB,
dependencies, runner/workflow and repository-hygiene configuration do not change.
No complete `go test ./...`, new full compiler self-build/distribution, Linux,
push, release or hosted CI was run. Linux remains batched. Earlier release and
fixed-point receipts are historical snapshots, not acceptance of this new image.
