# ADR0551 source-backed unit closure values, 2026-11-08

The private unit call plan now distinguishes declaration calls from named value
calls. `Call.callable` is the original source expression ID, zero for declaration
calls. A value call carries no fabricated provider, signature, generic instance
or result ID. Its single i32 argument and i32 result are checked against the
owned value graph. Immutable local/parameter identity and canonical graph kind23
are required before invocation kind4 becomes checked.

Construction kind36 retains a real private helper as a reachability edge. The
value checker matches the original literal, typed binding, helper signature and
body, and ordered i32 snapshots before admitting the canonical closure leaf.
Unknown captures stay pending until ordinary local typing resolves them; a cached
expression cannot bypass construction validation. Safe, pure, nongeneric helper
requirements remain closed. Lexical value callees no longer acquire unsafe,
effects or must-use policy from an unrelated same-named declaration. The existing
detached declaration-only `check` entry point remains available.

## Windows execution evidence

`TestCandidateClosureUnitValuesControls`, `Source` and `Native` passed with
`GOMAXPROCS=1` (package 54.591s, source 6.85s, native 47.48s).
The native harness compiled and executed the authored Fairy source-to-unit-values
checker under both `run` and `run --release --locked`:

- 29 original-source controls: 11 admitted and 18 refused, agreeing with the Go
  frontend; 58 ordered native receipts across the two profiles.
- Captures 0..4, returned factories, local and ordinary struct copies, different
  helpers through one parameter, signature-only consumption, mutable-i32 capture
  snapshots, and unsafe/effectful provider shadowing.
- Refusals for scalar callees, zero/two/labeled arguments, bool/i64 arguments,
  mutable closure bindings/copies, invalid captures/signatures, wrappers, nested
  carriers, repr(C) and FFI transport. Ordinary effectful calls without authority
  and unused must-use results remain refused. Allocation/budget/stale-record
  errors are explicitly excluded as evidence for a language refusal.
- Checked receipts inspect canonical type23, invocation kind4, original IDs,
  ordered argument/snapshot records, helper counts and no unresolved calls.

The fixture executes the compiler's value checker, not the input closure program.
This does not prove unit body/cleanup execution, publication or restored cache
behavior. Existing MIR source execution is a separate owner and receipt.

Full candidate `check` and `check --locked` also passed: 773 source files,
9,697 MIR functions. `go vet -mod=readonly ./internal/cli ./internal/sdkstdlib`,
Go formatting and `git diff --check` passed. Logs are
`.cache/closure-unit-values-integration-20261108.log`,
`.cache/closure-unit-values-main-check-20261108.log`,
`.cache/closure-unit-values-main-locked-20261108.log` and
`.cache/closure-unit-values-vet-20261108.log`.

The normal CLI regenerated the shared source lock after the source/optional-SDK
changes; its locked verification used the final sources. No normative document,
public API, SDK ABI, DB, dependency policy, hygiene config or CI runner changed.
Whole-compiler native/self-host, source-bound unit/project artifacts, persistent
cache acceptance, Linux and remote CI/release remain unverified in this slice.
