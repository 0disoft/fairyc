# Fairy copied closure value interpreter, 2026-11-08

Scope: ADR 0551 `fn i32(i32)` values in the authored Fairy semantic
interpreter. No cache, publisher, C emitter, lockfile, CI, or remote changes.

## Implementation

- `module_unit_interpreter_values::admitted` admits only the canonical closure
  leaf recognized by `module_unit_body_ir_closures::leaf`.
- `module_unit_interpreter_closures::construct` consumes kind 36 expression
  results already evaluated by the checked CFG. Reverse IR argument links are
  copied into written-order inline capture slots, with zero-filled unused slots.
- The immutable per-test heap cell has nine payload words: tag, creator frame,
  construction node, resolved helper frame, capture count, and four i32 slots.
  Frame/node numbers refer to the real checked IR; no native pointer or creator
  local-root cell is retained. Struct/local/parameter/result copies may share the
  frozen cell, just as other interpreter aggregate handles do.
- `module_unit_interpreter_closures::invocation` reads the frozen callable and
  argument expression results for kind 4 calls. It reauthenticates the original
  construction, capture count and lifted helper through
  `module_unit_body_ir_closures::helper`, validates cell shape, helper identity,
  i32 capture width and unused zero slots, and prepares captures followed by the
  ordinary argument. `module_unit_interpreter::invoke` then calls the existing
  checked-frame `execute` entry at `depth + 1`.
- Existing CFG operand order, helper-local execution and trap propagation are
  retained. Construction/copy never executes the closure body. Captures are
  snapshots of evaluated values, not fresh reads of mutable binding roots.
- Existing depth 256, one-million-word per-test heap and caller step budgets
  remain in force. Closure link traversal, helper authentication and heap
  allocation charge the same step budget. Invariant/allocation failures use
  error 4, budget exhaustion error 2, depth exhaustion error 3.

## Acceptance fixture

`tests/closure_value_interpreter.fy` accepts original module text, resolves real
imports, builds `project_compiler_core::Complete`, selects the source-authored
main entry, prepares checked CFGs, and executes the Fairy semantic interpreter.
It reports parse/Complete/prepare/execute failures with case, phase, stage, code,
module and span; outcome mismatches include actual and expected error/trap/bits.
Successful cases return the actual interpreter outcome to the Go harness.

`internal/cli/candidate_closure_value_interpreter_test.go` independently checks
and interprets the original sources with Go. Its 19 cases cover returned
factories with 0-4 captures, different helpers through one parameter, returned
struct copies, mutable-source snapshots, separate environments, argument
evaluation once, unused trapping bodies, argument-before-body trap order,
helper-local trap order, signed remainder overflow and repeated invocation.
Seven original multi-module cases include equal helper declaration numbers in
different source modules and a three-module forwarding path. The authored
fixture also checks zero steps, the exact call-depth boundary, and a full heap.

## Validation and handoff

Source validation passed on the final implementation with:

```powershell
$env:GOMAXPROCS='1'
go test ./internal/cli -run '^TestCandidateClosureValueInterpreterSource$' -count=1 -v -timeout 10m
```

```text
19 independent Go interpreter outcomes and authored Fairy Complete/semantic-interpreter fixture source checked; no native execution
--- PASS: TestCandidateClosureValueInterpreterSource (26.87s)
PASS
ok  github.com/0disoft/fairyc/internal/cli  27.150s
```

An initial control used the wrong trap-category spelling; it was corrected to
the actual Go category `signed-division-overflow`. The first Fairy source check
then caught private resource-field access across modules; the final code uses
explicit `frame` and `arguments` accessors. Both corrections are included in
the passing run above. Product and harness sources are now frozen for parent
native scheduling; no native profile has run in this worker.

The source test runs Go interpreter controls and checks the imported Fairy
implementation and generated fixture main. It does **not** execute the Fairy
fixture assertions or establish a native pass.

Parent-only command after concurrent cache/publisher sources freeze:

```powershell
$env:GOMAXPROCS='1'
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
go test ./internal/cli -run '^TestCandidateClosureValueInterpreterNative$' -count=1 -v -timeout 15m
```

The native test builds the authored Fairy interpreter fixture in Windows MSVC
low and optimized profiles and compares every semantic result/trap. It stops
after the first profile failure. There is no generated C consumer standing in
for Fairy semantic execution. This worker does not run native builds.

The parent Native acceptance below completes actual Fairy semantic execution
for this fixture. Batched Linux and whole CLI/distribution checks remain separate.
Runtime cell-tamper
and exact helper-authentication step-cost boundary tests are not part of this
bounded fixture. No Stage 0 API, DB, repository hygiene or runner policy changed;
no normative source-of-truth document changed. No staging or commit performed.

## Parent Windows execution

After all worker source writes froze, the parent ran the exact Native test with
`-mod=readonly`, `GOMAXPROCS=1` and `FAIRYC_RUN_PARITY_INTEGRATION=1`:

```text
--- PASS: TestCandidateClosureValueInterpreterNative (166.82s)
    --- PASS: TestCandidateClosureValueInterpreterNative/low (31.41s)
    --- PASS: TestCandidateClosureValueInterpreterNative/optimized (113.56s)
ok github.com/0disoft/fairyc/internal/cli 167.002s
```

Each compiled Fairy fixture executed all 19 original-source semantic outcomes,
compared against independently checked Go results: 38 native Fairy interpreter
outcomes across the two profiles. Zero steps, depth 255/256 and full-heap
rejection ran in each profile as four additional boundary assertions. Stdout
was compared exactly and stderr had to remain empty. No generated C consumer
substituted for semantic interpretation. Log:
`.cache/closure-value-interpreter-native-20261108.log`.

This does not execute the full Fairy CLI closure test or the distribution/self-host
acceptance. The parent commits this completed interpreter slice separately;
push, CI, release and deployment remain unexecuted.
