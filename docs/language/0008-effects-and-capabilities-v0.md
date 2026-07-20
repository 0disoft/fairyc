# Effects and External Capabilities v0

- Status: Draft Stage 0 contract
- Product source: `docs/product/02-spec.md`
- Decision source: `docs/adr/0116-stage-0-source-effects.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Purpose

Effects make externally observable authority visible at the function boundary. They do not predict
every instruction a function executes and they are not an optimizer purity claim. A function that
can touch a file, console, environment, process, network, clock, or random source declares that
capability even when one particular execution path does not use it.

Stage 0 effects are checked before Flow and survive in typed HIR. They do not by themselves grant a
backend, interpreter, or standard-library implementation. Real host I/O remains unavailable until
the corresponding safe API, typed errors, runtime transport, and target evidence exist.

## Declaration syntax

An ordinary or raw foreign function may carry one `@effects(...)` attribute:

```fy
@effects(filesystem)
fn load_config() -> Result<Config, IoError> {
    // later safe standard-library call
}

@effects(console, filesystem)
fn main() -> i32 {
    discard load_config();
    0
}
```

Arguments are identifiers from the closed Stage 0 vocabulary. Source order is not semantic; typed
HIR stores a sorted duplicate-free set. An empty attribute, a duplicate attribute or argument, a
literal argument, and an unknown name are `E7101`. Omitting the attribute declares the empty set.
There is no separate `pure` spelling.

## Closed Stage 0 vocabulary

| Effect | Authority represented |
| --- | --- |
| `console` | Read or write an attached program console or standard stream |
| `filesystem` | Observe or mutate files, directories, or filesystem metadata |
| `environment` | Read or mutate process environment state |
| `process` | Start, signal, wait for, or inspect another process |
| `network` | Use sockets, name resolution, or other network transport |
| `clock` | Observe wall, monotonic, or CPU time |
| `random` | Observe an operating-system or external randomness source |

`unsafe` is a proof boundary, not an effect. It does not satisfy or erase an effect requirement.
Allocation is also not a Stage 0 effect: Fairy C has no hidden default allocator, and explicit
allocator or Region operations already expose their cost and ownership. Traps are defined language
control flow rather than external authority.

## Call rule

For every resolved direct call, the enclosing function's declared set must contain the callee's
declared set. Otherwise the callee path is rejected with `E7102` and the diagnostic relates the
callee's effect declaration. The rule applies across modules, to forward declarations, and to raw
foreign functions. `main` receives no implicit authority.

The compiler checks declarations rather than silently inferring a larger public contract. A
function may deliberately declare a strict superset so its body can evolve without changing its
callers. Removing an effect is compatible for callers; adding one can invalidate them.

Raw foreign calls independently require an `unsafe` boundary. `unsafe { raw_call() }` proves only
the foreign-memory boundary; the enclosing function must still declare every effect named by the
foreign declaration. A foreign declaration without `@effects` makes no external-authority claim to
the source checker; authors of a safe wrapper are responsible for assigning the correct contract.

## Deferred forms

Effect variables, effect polymorphism, user-defined effects, handlers, effect inference in public
signatures, conditional effects, function-type effect syntax, and effect-bearing first-class
function values are outside Stage 0. The implemented subset checks resolved direct calls only.
