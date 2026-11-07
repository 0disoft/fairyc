# Fairy closure value source execution, 2026-11-08

ADR 0551 source/HIR/lowering/C stage, following source preparation 54d81886
and explicit MIR admission 47f2698c. The coordinator integrates the authored
bytes from `.cache/closure-runtime-20261108`; Linux remains deferred.

## Implementation

The direct structural signature `fn i32(i32)` resolves to private native type
13, distinct from C callbacks and graph tag 23. A prepared source construction
must name its real source-owned helper and retain exactly its ordered captures.
Invocation accepts an immutable local or parameter and evaluates the callable
before its one i32 argument. HIR retains explicit create/call expressions 31/32;
MIR retains construction 40, invocation terminator 8, and checked helper metadata.

The callable is copied before argument lowering. Capture values are stored
inline, with no pointer into the factory frame. Metadata is selected from actual
emitted construction operations, so a literal in a removed continuation leaves
no orphan target row. Ordinary generic expansion preserves the owned private
name/capture receipts, and receipt-copy failures become source problems.

C emission defines a private invoke pointer plus four i32 slots, zeroes unused
slots, and uses an adapter for each verified helper. Function transport,
immutable copies, direct ordinary struct fields, returned factories, checked
body locals and trap order use the ordinary pipeline. The separate C gate in
the preceding MIR-only commit is replaced by verified closure C admission.

## Executed Windows acceptance

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1' # temporary coordination gate, removed afterward
go test -mod=readonly ./internal/cli -run '^TestCandidateClosureValueC$' -count=1 -v -timeout=240s
```

PASS: package **64.562s**, acceptance **64.44s**. Original command output is
retained locally in `.cache-closure-value-c-attempt2.log` in the isolated checkout.
No compiler flag, warning policy, timeout, source body or expected outcome was
relaxed. The coordination environment gate is removed from the final test.

Both low and optimized Fairy producers execute the original scanner/parser,
name resolution, type checker, HIR and MIR lowerers, verifier and C emitter.
Go builds these producers and observes their output; it supplies no synthetic
AST, lifted body, type fact or MIR target to the source cases.

- Fourteen original-source cases, compiled and executed with both C profiles
  from both producer profiles: **56 emitted-program executions**.
- Cases cover ordinary scalar output, signature-only consumers without a
  literal, a dead typed literal, an unrelated scalar generic near a closure,
  four ordered captures, factory snapshots, returned/copied direct struct
  fields, different helpers through one parameter, repeated calls and argument
  evaluation, unused trapping bodies, division traps and signed overflow.
- Each producer also executes **28 forged-MIR refusals and 14 source refusals**.
  These remain separate from emitted-program execution counts. Invalid helpers,
  captures, callable operands, unsafe/effectful targets and old-family facts
  cannot grant C emission authority. Nested carrier records remain refused.
- Emitted C bytes match across producer profiles. Normal programs return 42;
  a minimal trap observer preserves the exact candidate trap category through
  exit status `80 + category` and requires empty output streams.

The first coordinated attempt stopped honestly at MIR local type-13 admission.
The coordinator added that missing lowering admission, fixed the callable copy
owner type, preserved generic receipts, and selected metadata after dead-code
lowering before the successful final run.

## Remaining scope

This proves the authored source-to-C execution path, not complete project/unit
artifact formats, source-bound restoration, persistent cache execution, the
whole compiler CLI image, self-build, distribution or hosted CI. Old executable
artifact families must keep refusing the new records until their versioned
consumers are implemented. Linux remains batched. No normative contract, public
SDK ABI, dependency, database, runner setting or remote state changed.
