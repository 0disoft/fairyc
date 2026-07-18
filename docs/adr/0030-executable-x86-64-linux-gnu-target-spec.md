# ADR 0030: Executable x86_64 Linux GNU target specification

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- ABI source: `docs/abi/c-abi-v0.md`
- Compatibility source: `docs/compatibility/versioning-v0.md`
- Implementation source: `internal/target/`
- CLI schema: `spec/cli/experimental-target-spec-v0.schema.json`

## Context

The first C backend and C driver agreed on the target name only through repeated string literals.
That was enough for scalar C emission, but it was not an executable `TargetSpec`: no shared object
owned endianness, pointer layout, scalar layout, stack rules, calling convention, object format, or
target-maturity evidence. Extending the backend to aggregates or resources in that state would let
the Go host layout or a locally installed C compiler become an accidental semantic source.

Target semantics and local toolchain discovery are different contracts. A Windows Go host may
describe `x86_64-linux-gnu` without being able to build it, while a Linux host still needs an
eligible glibc, Clang, and GCC installation before it can produce native evidence.

## Decision

1. `internal/target` owns the first executable target description under schema
   `fairyc.target-spec/v0`. It records the canonical target identity, x86-64 architecture, Linux GNU
   environment, little endianness, LP64 data model, pointer and function-pointer layout, language
   and C scalar layouts, SysV AMD64 stack and calling identities, ELF object and symbol identities,
   DWARF unwind/debug contracts, and the permitted C-layout feature vocabulary.
2. Target data is immutable at the package boundary. Lookup returns isolated copies, validates
   sorted and unique closed lists, power-of-two natural alignment, nonempty target identities, and
   monotonic maturity states.
3. The target description contains no compiler path, SDK path, sysroot path, environment variable,
   or host filesystem location. Those values remain local `ToolchainSpec` resolution results.
4. The C backend profile carries the typed target ID. Emitted-unit metadata remains a string JSON
   value. The C driver binds every resolved toolchain to the same target ID and rejects unknown
   targets before host probing.
5. `fairyc experimental target-spec x86_64-linux-gnu` emits deterministic indented JSON matching
   `spec/cli/experimental-target-spec-v0.schema.json`. Missing or unknown target IDs are invocation
   failures and emit no JSON.
6. Target maturity remains fail-closed. The descriptor reports the existing
   `scalar-and-flat-i32-resource-c11` emission subset but keeps `emittable`, `conformant`, and
   `supported` false. ADR 0033 expands the reported implementation subset without promoting target
   maturity.
   It lists missing C layout probes and bidirectional ABI fixtures, complete runtime and standard library,
   hosted Linux receipt, toolchain baseline, and provisioning evidence.
   A scalar layout or `@repr(C)` vocabulary entry does not by itself make the corresponding Fairy C
   source type, aggregate, foreign call, or ABI fixture implemented.
7. This target description is a prototype compatibility member. Changing its layout or classifier
   meaning requires a target-schema or ABI-profile decision; fixing code to match unchanged data is
   an implementation change.

## Consequences

- Backend, driver, CLI, and future layout work share one target identity instead of parallel
  literals.
- Windows can inspect the intended Linux target without pretending that a local Linux toolchain or
  native-conformance receipt exists.
- Aggregate and resource lowering now has a target-owned place to obtain layout facts, but the
  descriptor alone is not an aggregate layout engine or ABI conformance proof.
- The experimental JSON surface adds a machine-readable contract that must change together with
  its schema, tests, ABI documentation, and compatibility rules.

## Rejected alternatives

- Ask the host C compiler for semantic layout on every build: makes target meaning depend on a
  local installation and prevents deterministic cross-host inspection.
- Use Go `unsafe.Sizeof`: measures the bootstrap compiler host, not the Fairy C target.
- Report the target as emittable because eight programs generate C: confuses a narrow backend
  subset with the complete declared target.
- Store resolved compiler and sysroot paths in `TargetSpec`: leaks host state into compatibility
  identity and makes receipts non-portable.
- Delay all target data until resource lowering: leaves every intermediate backend change free to
  invent a private layout source.
