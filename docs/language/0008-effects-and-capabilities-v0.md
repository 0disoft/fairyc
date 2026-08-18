# Effects and External Capabilities v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision sources: `docs/adr/0116-stage-0-source-effects.md`, `docs/adr/0121-resource-drop-effects.md`, `docs/adr/0337-onchain-effect-capabilities.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Purpose

Effects make externally observable authority visible at the function boundary. They do not predict
every instruction a function executes and they are not an optimizer purity claim. A function that
can touch a file, console, environment, process, network, clock, random source, chain storage,
consensus event log, another contract, or block context declares that capability even when one
particular execution path does not use it.

Stage 0 call effects are checked before Flow; compiler-scheduled resource cleanup effects are
checked by Resource Flow. Both survive in typed HIR. They do not by themselves grant a backend,
interpreter, host, or chain implementation. Real external authority remains unavailable until the
corresponding safe API, typed errors, runtime transport, target profile, and executable evidence
exist.

## Declaration syntax

An ordinary function, raw foreign function, or resource with a nonempty cleanup body may carry one
`@effects(...)` attribute:

```fy
@effects(filesystem)
fn Result<Config, IoError> load_config() {
    // later safe standard-library call
}

@effects(console, filesystem)
fn i32 main() {
    discard load_config();
    return 0;
}

@effects(filesystem)
resource File {
    ptr mut void handle;

    cleanup {
        // later safe runtime release
    }
}
```

The same declaration form represents target-neutral on-chain authority:

```fy
@effects(storage_read)
fn u32 balance(ref TxContext context) {
    // a future chain SDK reads through the explicit context value
    return 0;
}

@effects(block_context, emit, storage_read, storage_write)
fn void transfer(ref mut TxContext context) {
    // a future chain SDK performs checked state access and event emission
}

@effects(external_call, storage_write)
fn void settle(ref mut TxContext context) {
    // external_call remains conservatively mutation-capable
}
```

`TxContext` is illustrative and not admitted by this contract. A future target profile must define
its exact owner, lifetime, construction, borrowing, storage, and ABI rules before providing such a
type.

Arguments are identifiers from the closed vocabulary. Source order is not semantic; typed HIR
stores a sorted duplicate-free set. An empty attribute, a duplicate attribute or argument, a
literal argument, and an unknown name are `E7101`. Omitting the attribute declares the empty set.
There is no separate `pure` spelling. An empty resource cleanup body cannot claim effects because it
has no user cleanup operation to authorize.

## Closed vocabulary

| Effect | Authority represented |
| --- | --- |
| `block_context` | Observe target-supplied block, transaction, caller, value, height, timestamp, or chain context through an admitted explicit API |
| `clock` | Observe wall, monotonic, or CPU time |
| `console` | Read or write an attached program console or standard stream |
| `emit` | Append a consensus-visible event or log record |
| `environment` | Read or mutate process environment state |
| `external_call` | Invoke another contract or chain program through a target-defined call boundary |
| `filesystem` | Observe or mutate files, directories, or filesystem metadata |
| `network` | Use sockets, name resolution, or other network transport |
| `process` | Start, signal, wait for, or inspect another process |
| `random` | Observe an operating-system or external randomness source |
| `storage_read` | Observe persistent chain state owned or addressed by the executing contract |
| `storage_write` | Create, replace, or remove persistent chain state |

`unsafe` is a proof boundary, not an effect. It does not satisfy or erase an effect requirement.
Allocation is also not an effect: Fairy C has no hidden default allocator, and explicit allocator or
Region operations already expose their cost and ownership. Traps are defined language control flow
rather than external authority.

## On-chain authority classification

The target-neutral on-chain subset is exactly `block_context`, `emit`, `external_call`,
`storage_read`, and `storage_write`.

`block_context` and `storage_read` are read-only with respect to consensus state. They remain
observable authority and therefore must still be declared. `storage_write`, `emit`, and
`external_call` are mutation-capable. Event emission changes consensus-visible output even when it
does not alter key-value storage. `external_call` is conservatively mutation-capable because an
unproven callee may mutate its own state, emit output, transfer value, or re-enter the caller. A
later target may add a separately proven static-call operation without weakening this effect.

`storage_write` does not imply `storage_read`, and neither implies `block_context`. The source
contract lists each authority actually available to the body. This prevents a broad write marker
from silently granting unrelated reads and keeps capability reduction visible during review.

The HIR package exposes canonical on-chain and mutation-capable sets plus a deterministic
read-only predicate. A contract profile may classify an exported entry as a view only when its
effect set contains none of `emit`, `external_call`, or `storage_write` and the target independently
rejects host-only authority such as filesystem, process, network, environment, clock, and random.

There is deliberately no source `@view` marker in this slice. A second marker would duplicate the
effect contract and permit drift such as `@view @effects(storage_write)`. Client generators and
contract metadata should derive view status from the canonical effect set instead.

## Call rule

For every resolved direct call, the enclosing function's declared set must contain the callee's
declared set. Otherwise the callee path is rejected with `E7102` and the diagnostic relates the
callee's effect declaration. The rule applies across modules, to forward declarations, and to raw
foreign functions. `main` and future contract entries receive no implicit authority.

The rule applies unchanged to on-chain calls. A function declaring only `storage_read` cannot call
one declaring `storage_write`, `emit`, or `external_call`. A dispatcher may declare a strict
superset and call narrower helpers. The compiler checks declarations rather than silently inferring
a larger public contract, so authority expansion remains a source-visible compatibility change.

A function may deliberately declare a strict superset so its body can evolve without changing its
callers. Removing an effect is compatible for callers; adding one can invalidate them.

Raw foreign calls independently require an `unsafe` boundary. `unsafe { raw_call() }` proves only
the foreign-memory boundary; the enclosing function must still declare every effect named by the
foreign declaration. A foreign declaration without `@effects` makes no external-authority claim to
the source checker; authors of a safe wrapper are responsible for assigning the correct contract.

## Explicit context boundary

Effect declarations expose authority but do not manufacture ambient values. There is no implicit
`msg.sender`, transaction value, block timestamp, global storage handle, or hidden contract object.
A future chain SDK must pass caller, value, block, storage, and call capabilities through explicit
compiler-known values such as an owned or borrowed `TxContext`. The type contract must prevent safe
code from constructing, persisting, duplicating, or returning target-owned authority when the
selected chain forbids those operations.

This separation lets effects answer “what authority may execute?” while ordinary types and borrows
answer “which exact capability value is being used, for how long, and by whom?”. Neither layer may
silently substitute for the other.

## Implicit cleanup rule

When Resource Flow proves that a live resource will be dropped, the enclosing function must declare
every effect on that resource. The rule applies to normal function and nested-scope exit, explicit
return, live-resource reassignment, and cleanup caused by `break` or `continue`. `Option<R>` and
`Result<R, E>` use the union of their resource-bearing payload effects.

A complete-root move makes the source dead before cleanup is considered. Passing or returning a
resource therefore does not require its cleanup effects in the sender; the new owner is checked when
and where it actually drops the value. Missing authority is `E7102`, related to the resource's
effect declaration.

A future contract profile must separately decide whether mutation-capable cleanup is admissible.
This generic effect contract exposes such cleanup accurately but does not authorize hidden storage
writes or event emission merely because a resource leaves scope.

## Target boundary

A deterministic contract target will normally reject the host authority set `console`,
`filesystem`, `environment`, `process`, `network`, `clock`, and `random`, unless it defines a
consensus-safe replacement with a separate exact effect meaning. The language-wide vocabulary does
not claim that every target implements every effect.

Effect metadata alone does not define storage layout, event encoding, external-call ABI, gas or fuel
accounting, transaction rollback, payable behavior, reentrancy guards, or deployment format. Each
of those requires a target-specific contract and executable evidence.

## Deferred forms

Contract-module declarations, exported `@entry` metadata, payable entry semantics, a compiler-owned
`TxContext`, storage and event SDK APIs, static external calls, value transfer, reentrancy policy,
chain-specific effect availability, gas accounting, effect variables, effect polymorphism,
user-defined effects, handlers, effect inference in public signatures, conditional effects,
function-type effect syntax, and effect-bearing first-class function values are outside this slice.
The implemented subset checks resolved direct calls and compiler-scheduled cleanup only.
