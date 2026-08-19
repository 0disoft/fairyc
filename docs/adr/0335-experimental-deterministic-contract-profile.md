# ADR 0335: Add a verified-MIR deterministic contract profile and bounded cost report

- Status: Accepted experimental surface
- Date: 2026-08-19
- Implementation: `internal/contractprofile/`, `cmd/fairyc-contract-profile/`
- Contract: `docs/contract/deterministic-contract-profile-v0.md`
- Output schema: `spec/cli/experimental-contract-profile-v0.schema.json`
- Depends on: ADR 0001, ADR 0005, ADR 0218, ADR 0322

## Context

Fairy C already has explicit ownership, checked arithmetic, source effects, verified MIR, deterministic
C transport, and replaceable backend boundaries. Those properties are useful for contract execution,
but they do not by themselves prove that one program terminates within a bounded amount of work or
avoids host authority that cannot exist inside consensus execution.

Starting with an EVM-shaped language surface would couple the language to one chain ABI before Fairy
C has a contract workload. Starting with a WASM or RISC-V emitter without a shared policy gate would
allow each backend to invent different rules for loops, recursion, allocation, FFI, raw pointers, and
cost accounting.

The ordinary language also must not reserve speculative blockchain syntax. Stage 0 currently treats
unknown function attributes as unsupported, and its compatibility identity must not change merely to
run an experimental contract proof.

## Decision

Add the standalone experimental command:

```text
fairyc-contract-profile <file.fy> [<file.fy>...]
```

The command reads one traversal-contained explicit source closure, parses every source with the
ordinary frontend, consumes the profile-only `@max_iter(N)` function attribute, removes that
attribute from the AST, and then runs the unchanged semantic, MIR-lowering, and MIR-verification
pipeline. The profile therefore overlays ordinary Fairy C instead of changing the Stage 0 grammar,
language revision, public interface hash, MIR codec, or stable `fairyc` command surface.

`@max_iter` accepts exactly one integer from 1 through 1,000,000 and may appear once on an ordinary
function. Its meaning is the maximum aggregate number of metered cycle-edge traversals during
one invocation of that function. It is not a per-loop bound. A future runtime meter must enforce the
same aggregate counter so the static report and deployed execution cannot disagree.

The analyzer consumes verified MIR and rejects the complete program when it contains any of the
following:

1. raw foreign declarations or foreign calls;
2. declared external effects, including clock, random, filesystem, environment, process, network,
   and console authority;
3. raw-pointer types or raw-pointer MIR operations;
4. unsafe MIR operations;
5. floating-point types or transport before every backend shares exact arithmetic and NaN rules;
6. target-dependent `c::*` scalars, `usize`, and `@repr(C)` aggregate layout;
7. Region allocation operations before a backend-specific memory-growth meter exists;
8. cyclic control flow without a valid `@max_iter`;
9. direct or mutual recursion;
10. missing internal call targets, malformed verified-MIR graph identities, orphan loop annotations,
    or abstract-cost overflow.

The first abstract cost model assigns deterministic integer weights to operations and terminators.
A function's local cost is the sum across all of its MIR blocks. Each internal call site adds the
callee's transitive bound. If the function has cyclic control flow, the complete invocation cost is
multiplied by `N + 1`, where `N` is its aggregate metered cycle-edge budget. This deliberately overestimates
many programs. It is a backend-neutral admission proof, not a chain gas price or fee prediction.

The report uses `fairyc.contract-profile-report/v0` and
`fairyc.deterministic-contract/v0`. It contains sorted source diagnostics, sorted profile
violations, per-function control-flow and call facts, local abstract cost, transitive upper bound,
and storage-growth upper bound. Persistent storage operations are not admitted in this first slice,
so every accepted function reports zero storage growth.

`BuildMeterPlan` exposes the compiler-private backend handoff only after complete acceptance. It
contains exact cyclic MIR edges, aggregate budgets, abstract cost ceilings, and storage-growth
ceilings. Public JSON contains only the number of metered edges because MIR block identities are
valid only inside one verified program instance.

The first intended executable backend is a constrained WASM contract target. A constrained RISC-V
target follows after the same programs, counters, traps, and report identities can be reproduced.
This decision adds neither emitter and makes no target-support claim.

## Evidence

Package tests construct MIR graphs directly and require bounded-cycle acceptance, missing-bound
rejection, FFI/effect/raw-pointer/unsafe/Region rejection, direct and mutual recursion rejection,
and byte-identical JSON across input map order.

Source-pipeline tests prove that a real Fairy source with `@max_iter` reaches verified MIR after the
profile attribute is consumed, while malformed, misplaced, or missing bounds fail closed. CLI tests
require stable JSON and distinct success, source-diagnostic, profile-rejection, and invalid-invocation
exit classes.

## Consequences

Fairy C gains a real backend-independent contract admission gate without turning its general
language into a Solidity clone. WASM and RISC-V work can now target one explicit verified input
contract instead of re-deciding consensus safety inside each emitter.

The current profile cannot express persistent state, events, contract calls, cryptographic host
functions, metered heap growth, or chain context. Those require separate capabilities, costs, ABI
contracts, and runtime evidence. Adding them by weakening this profile is forbidden; each capability
must extend the versioned profile and retain fail-closed analysis.
