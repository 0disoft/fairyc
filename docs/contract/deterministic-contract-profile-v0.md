# Deterministic Contract Profile v0

- Status: Experimental executable contract
- Profile identity: `fairyc.deterministic-contract/v0`
- Report identity: `fairyc.contract-profile-report/v0`
- Cost identity: `fairyc-abstract-step/v0`
- Implementation: `internal/contractprofile/`
- CLI: `tools/dev/fairyc-contract-profile/`
- JSON Schema: `spec/cli/experimental-contract-profile-v0.schema.json`

## Purpose

This profile answers one narrow question: can a verified Fairy C MIR program execute under a
consensus runtime without ambient host authority and with a finite backend-neutral work bound?

It does not define a blockchain ABI, persistent storage model, token type, event format, gas price,
fee, bytecode format, chain context, cryptographic host interface, or upgrade protocol. Those
surfaces remain closed until they have separate versioned contracts.

## Invocation

```text
go run ./tools/dev/fairyc-contract-profile main.fy helper.fy
```

Every argument is a source-root-relative `.fy` path. The command opens the supplied root once,
prevents traversal outside it, caps each source at 8 MiB, caps the explicit closure at 64 files, and
prints one JSON report to stdout.

Exit status `0` means accepted. Status `3` means ordinary source diagnostics prevented verified MIR.
Status `6` means valid Fairy C reached the contract policy but failed it. Status `2` is an invalid
invocation, and status `4` is an internal failure.

## Profile overlay

A cyclic function must declare one profile-only attribute:

```fy
@max_iter(10_000)
fn i32 execute() {
    var i32 index = 0;
    while (index < 100) {
        index += 1;
    }
    return index;
}
```

The frontend parses the attribute through the existing generic attribute grammar. The profile reads
it, validates it, and removes it from the AST before ordinary semantic analysis. It therefore does
not become a Stage 0 language attribute, public interface member, or MIR field.

`@max_iter(N)` obeys all of these rules:

1. it is valid only on an ordinary function;
2. it appears at most once;
3. it contains exactly one integer;
4. `N` is in `1..1_000_000`;
5. `N` bounds the aggregate number of metered CFG cycle-edge traversals during one invocation.

The aggregate rule is deliberate. The analyzer computes strongly connected components and marks
every CFG edge whose source and destination belong to the same cyclic component. The report exposes
that count as `meteredCycleEdges`. A future selected runtime increments one per-function
counter on every such edge and traps before the aggregate budget is exceeded. This cannot silently
multiply independent per-loop promises or miss irreducible control flow.

## Closed authority rules

The profile rejects every Stage 0 external effect. A contract cannot read a host clock or random
source, use files, environment variables, processes, sockets, console streams, or any raw foreign
declaration. Later chain capabilities must enter through a separate deterministic host contract with
explicit prices and replay semantics; reusing the current host effects is forbidden.

Raw pointers, unsafe MIR operations, foreign calls, raw-pointer-bearing aggregate fields,
target-dependent `c::*` scalars, `usize`, `@repr(C)` layout, and floating-point types are rejected. Floating point remains closed until every admitted backend,
constant folding, traps, NaN canonicalization, and serialization share one exact contract. Region
enter, placement, and reset operations are also rejected until a backend owns a measurable
memory-growth ceiling. Ordinary move-only resources remain
admissible when their cleanup is effect-free.

Direct recursion and mutual recursion are rejected. First-class recursive metering is not inferred.
An internal call whose target is absent from the verified program and any malformed block identity
fail closed even though the normal compiler verifier should already make them unreachable.

## Abstract cost model

The model is intentionally small and deterministic.

| MIR form | Abstract weight |
| --- | ---: |
| ordinary operation | 1 |
| checked arithmetic, division, index, or slice guard | 2 |
| resource drop | 3 |
| Region operation | 4 and profile rejection |
| aggregate construction | 2 plus operand and field counts |
| ordinary terminator | 1 |
| switch terminator | 2 plus successor count |
| internal call terminator | 5 plus the callee's transitive bound |

For an acyclic function:

```text
transitive = local block cost + sum(callee transitive cost at each static call site)
```

For a cyclic function with `@max_iter(N)`:

```text
transitive = (local block cost + sum(callee transitive cost at each static call site)) * (N + 1)
```

Summing every MIR block and multiplying the complete body deliberately overestimates many CFGs. The
report is an admission ceiling, not an optimizer estimate. Any `uint64` overflow rejects the program.

`storageGrowthUpperBoundBytes` is zero because profile v0 admits no persistent-storage instruction.
That value must not be reused after storage is added; the profile and report schema must advance.

## Violation codes

| Code | Rejection |
| --- | --- |
| `CP0101` | duplicate `@max_iter` |
| `CP0102` | malformed or excessive loop budget |
| `CP0103` | `@max_iter` placed on a non-function declaration |
| `CP1001` | foreign declaration or call |
| `CP1002` | external function or cleanup effect |
| `CP1003` | raw-pointer type, transport, or operation |
| `CP1004` | unsafe MIR operation |
| `CP1005` | unmetered Region allocation |
| `CP1006` | missing internal call target |
| `CP1007` | cyclic CFG without a loop budget |
| `CP1008` | direct or mutual recursion |
| `CP1009` | abstract-cost overflow |
| `CP1010` | loop annotation without a matching MIR function |
| `CP1011` | malformed CFG identity or successor |
| `CP1012` | duplicate verified function symbol or name |
| `CP1013` | floating-point type or transport |
| `CP1014` | target-dependent `c::*` scalar |
| `CP1015` | target-width `usize` |
| `CP1016` | target-dependent `@repr(C)` aggregate layout |
| `CP1017` | compiler-published contract-state transaction not admitted by profile v0 |

Violations sort by function, source range, code, and message. Reports never contain source text,
absolute host paths, build-local symbol IDs, bytecode, toolchain paths, or secrets.

## Backend handoff

A future contract backend may consume a program only after both ordinary MIR verification and this
profile return success. It must preserve checked Fairy arithmetic and trap behavior and enforce the
same aggregate metered cycle-edge counter represented by `declaredLoopBudget`.

`BuildMeterPlan` is the compiler-private backend handoff. It refuses every rejected report and
returns each function's exact cyclic MIR edges, aggregate loop budget, abstract cost ceiling, and
storage-growth ceiling. Block identities remain out of the public JSON because they are valid only
for the current verified MIR instance.

The first backend remains `UNDECIDED`. Its target ADR must preserve this profile report and runtime
counter contract, and it may not reinterpret `fairyc-abstract-step/v0` as chain gas without a
separately versioned schedule and native execution evidence.
