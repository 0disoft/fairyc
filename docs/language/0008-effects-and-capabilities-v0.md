# Effects and External Capabilities v0

- Status: Released language 11 contract
- Product source: `docs/product/02-spec.md`
- Decision sources: `docs/adr/0116-stage-0-source-effects.md`, `docs/adr/0121-resource-drop-effects.md`, `docs/adr/0337-onchain-effect-capabilities.md`
- Semantic fixtures: `spec/fixtures/semantic/manifest-v0.json`

## Purpose

Effects make externally observable authority visible at the function boundary. They do not predict
every instruction a function executes and they are not an optimizer purity claim. A function that
can touch a file, console, environment, process, network, clock, or random source declares that
capability even when one particular execution path does not use it.

Stage 0 call effects are checked before Flow; compiler-scheduled resource cleanup effects are
checked by Resource Flow. Both survive in typed HIR. They do not by themselves grant a
backend, interpreter, or standard-library implementation. Real host I/O remains unavailable until
the corresponding safe API, typed errors, runtime transport, and target evidence exist.

## Availability boundary

ADR 0342 admits the target-neutral on-chain effects in `fairyc-language/11`. Direct analysis and
resolved-package check, build, run, test, cache, and native emission now share the same closed
vocabulary. This admission carries authority facts only; an effect has no operation unless an
explicit SDK and target contract implements one.

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

Arguments are identifiers from the closed Stage 0 vocabulary. Source order is not semantic; typed
HIR stores a sorted duplicate-free set. An empty attribute, a duplicate attribute or argument, a
literal argument, and an unknown name are `E7101`. Omitting the attribute declares the empty set.
There is no separate `pure` spelling. An empty resource cleanup body cannot claim effects because it
has no user cleanup operation to authorize.

## Closed language 11 vocabulary

| Effect | Authority represented |
| --- | --- |
| `console` | Read or write an attached program console or standard stream |
| `filesystem` | Observe or mutate files, directories, or filesystem metadata |
| `environment` | Read or mutate process environment state |
| `process` | Start, signal, wait for, or inspect another process |
| `network` | Use sockets, name resolution, or other network transport |
| `clock` | Observe wall, monotonic, or CPU time |
| `random` | Observe an operating-system or external randomness source |

Language 11 additionally recognizes:

| Effect | Authority represented |
| --- | --- |
| `block_context` | Observe target-supplied chain context through an admitted explicit API |
| `emit` | Append consensus-visible event or log output |
| `external_call` | Invoke another contract or chain program |
| `storage_read` | Observe persistent chain state |
| `storage_write` | Create, replace, or remove persistent chain state |

`emit`, `external_call`, and `storage_write` are conservatively mutation-capable. `block_context`
and `storage_read` are read-only with respect to consensus state but remain observable authority.
No effect implies another. View eligibility is derived from the absence of mutation-capable effects;
there is no separate source `@view` marker.

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

On-chain effects grant no ambient sender, value, block timestamp, storage handle, or contract
object. Future target APIs must provide explicit compiler-known capability values and separately
define ownership, borrowing, ABI transport, rollback, metering, and target availability.

## Implicit cleanup rule

When Resource Flow proves that a live resource will be dropped, the enclosing function must declare
every effect on that resource. The rule applies to normal function and nested-scope exit, explicit
return, live-resource reassignment, and cleanup caused by `break` or `continue`. `Option<R>` and
`Result<R, E>` use the union of their resource-bearing payload effects.

A complete-root move makes the source dead before cleanup is considered. Passing or returning a
resource therefore does not require its cleanup effects in the sender; the new owner is checked when
and where it actually drops the value. Missing authority is `E7102`, related to the resource's
effect declaration.

## Deferred forms

Effect variables, effect polymorphism, user-defined effects, handlers, effect inference in public
signatures, conditional effects, function-type effect syntax, and effect-bearing first-class
function values are outside Stage 0. The implemented subset checks resolved direct calls only.
