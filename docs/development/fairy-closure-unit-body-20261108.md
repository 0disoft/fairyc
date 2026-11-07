# ADR0551 unit body/CFG closure proof, 2026-11-08

Unit Body IR now retains construction kind36 and value-call kind4 only under
the explicit closure cleanup family. The construction target remains a source
helper declaration ID, distinct from type IDs. Ordinary body checks continue to
reject kinds34/35, unknown future operations and kind36 in old families.

`module_unit_body_ir_closures` checks the selected same-module helper frame,
unique safe/pure/nongeneric callable contract, direct i32 parameters/result,
bounded arithmetic-only helper body, and immutable initialized bindings. The
helper is limited to 0..4 captures, one value parameter, 16 immutable locals and
64 expression nodes. Construction matches its symbolic helper token and ordered
snapshot operands; value calls require a checked immutable named leaf and one
checked direct i32 argument. No provider/foreign declaration identity is invented
for a value call.

Direct ordinary struct fields may carry a leaf. Nested carriers, repr(C),
wrapper/pointer transport, foreign calls and resource/tag fields remain closed
independently of source annotation validation. Construction enters the CFG's
ordered argument walk, so captures materialize before the value is built. Value
calls reuse the callable-before-argument CFG order.

Old FYBODY00 and FYCLEAN0/1 encoders now explicitly refuse the closure family.
The old body decoder and old full-body artifact assembler also refuse it.
Separately versioned closure transport and current-source restore are still
required before artifact/cache publication. The structural helper proof does
not authenticate a detached body against an original source literal.

## Executed Windows validation

With `GOMAXPROCS=1`:

- `TestCandidateClosureUnitBodySource`: PASS, 15.72s. The authored fixture/import
  closure and 11 original-source Go controls were checked.
- `TestCandidateClosureUnitBodyNative`: PASS, 97.05s. Both `run` and
  `run --release --locked` executed the original-source -> checked bodies ->
  Body IR -> full proof -> CFG pipeline for the 11 controls.
- Applicable forged-body controls reject zero/different helper targets, stale
  construction state, removed captures, invalid snapshot tokens, unchecked
  invocations, invented provider identities and uninitialized closure locals.
  Eight mutation classes preserve
  the original immutable Program and test separately cloned body records.
- Old body/cleanup encoding is refused for every closure-family input; snapshot,
  constructor and invocation counts match their original-source controls.
- Earlier full candidate `check`: PASS, 775 sources and 9,721 MIR functions.
  This predates final initialization hardening and concurrent artifact work;
  the final integration check/lock update remains separate.

Final hardened Source/Native run: `.cache/closure-unit-body-final-20261108.log`
(113.020s package total). Earlier logs: `.cache/closure-unit-body-source-20261108.log`,
`.cache/closure-unit-body-native-20261108.log`, and
`.cache/closure-unit-body-main-check-20261108.log`.

These native executions run the authored compiler/proof fixture, not the input
closure programs. Generated-C execution belongs to the C backend acceptance.
Project linking, versioned source-bound artifacts, persistent cache, whole
compiler native/self-host, Linux and remote CI/release remain separate work.
No normative document, public API/ABI, DB, dependency policy, runner, workflow or
repository-hygiene configuration changed.
