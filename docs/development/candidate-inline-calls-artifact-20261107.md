# Fairy inline call artifacts, 2026-11-07

Implements the Fairy detached-body and complete-program portion of
[ADR 0548](../adr/0548-inline-same-module-call-graphs.md).

`module_inline_ir.fy` reconstructs v3 call identities from the original callee
node text and ordered checked argument records. It cross-checks the reverse
syntax argument links, resolved invocation ownership, argument types and
ordinary-call state. Calls cost one expression; their callee paths cost none.
Argument expressions retain their normal costs. Existing v1/v2 no-call paths,
four parameters, sixteen locals and sixty-four expressions are unchanged.

`module_inline_graph.fy` validates resolved invocation coordinates after all
bodies pass ordinary verification. It uses the existing callable/frame indexes
and a per-root set of at most 64 distinct frames. Active visits reject cycles;
finished shared targets are reused. Targets must be marked, same-module,
ordinary, safe, nongeneric, nonforeign and effect-free, with matching argument
and result types. Project work checkpoints use the existing ledgers and limits.
The callable table contains actual invocation targets, so an unused exported
root need not have its own callable row; every edge still requires that row.
This check runs inside the normal Program assembler, including program codec
restoration. An isolated body is not a source-bound Complete owner.

Detached callable metadata does not contain declaration names or visibility.
Current-source authentication remains responsible for proving the spelling,
public visibility and exact declaration coordinate. A redirection to another
otherwise valid marked declaration must still fail Complete restoration.
Callee hashes are not embedded in callers; the existing all-export package
inline class closes the same-module body dependency.

## Authored replay

`TestCandidateInlineCallsArtifact` reuses the source fixture and private image
builders from the existing inline locals pipeline. The Go frontend builds only
the compiler image. Authored Fairy code parses/checks target source, constructs
Complete IR, round-trips body/program/project artifacts, restores source
authentication and emits partitioned C. The generated C is compiled/executed
with both low and optimized producer/consumer profiles, serially with
`GOMAXPROCS=1`. The existing timeouts and 16 MiB private compiler stack remain.

The target covers forward, nested, repeated and zero-argument calls, calls in
initializers and returns, v3 with/without locals, v1/v2 leaves, checked division,
an unused call-bearing exported root, and the existing caller argument
evaluation-order observer. All eleven inline bodies round-trip; only the ten
reachable from the target entry are emitted into its C header.

Nine artifact mutations cover a removed depended-on marker, an ordinary
unmarked target, a self-cycle, a different valid marked target, changed callee
text, a forged argument source ID, another-module target, generic instance
forgery and an unchecked invocation. The marked-target redirection is admitted
only as detached IR and rejected by current-source Complete authentication.
The other cases must fail at their appropriate body or Program boundary.

## Validation record

- A private full-source snapshot passed bootstrap `check` and `check --locked`;
  owned source hashes matched. The shared source lock was not changed by this
  worker. Receipt: `.cache/inline-calls-ir-source-final-check.json`.
- Native full replay passed in 162.25 s (Go package: 162.401 s), with all four
  producer/consumer combinations exiting 23 and no output. Receipt:
  `.cache/inline-calls-artifact-final.log`. Each Fairy producer also passed
  the nine artifact mutations and valid body/program/project/source restore.
- Strengthened per-declaration symbol replay passed in 170.95 s (Go package:
  171.111 s): both Fairy producers passed all artifact probes and all four C
  products exited 23 without output. Each source declaration's numeric C symbol
  was checked: all eleven body markers survived, each of ten reachable functions
  had exactly one header prototype and definition, `rival` was included, and
  the unused call-bearing export had no callable row or emitted C symbol.
  Receipts: `.cache/inline-calls-artifact-symbols-final.log` and
  `.cache/inline-calls-artifact-symbols-final.json`.
- Focused `git diff --check`, new-file trailing-whitespace checks and Go format
  checks passed. No unrelated validation was rerun after the symbol replay.
- Failed attempts are retained in `.cache/inline-calls-artifact-attempt*.log`.
  Attempts 1/2 caught test-source syntax/unsupported narrowing. Attempts 3-6
  diagnosed that an uncalled marked target has no callable row; the final probe
  registers `rival` through a real call and separately tests an unused exported
  call-bearing root. Attempt 7 passed all artifact probes but expected 22 C
  declarations instead of the correct 20 reachable declarations. The final
  symbol probe verifies individual membership, including `rival` and `unused`.
- Linux native execution stays in the coordinated batch. Full self-build,
  fixed point, hosted CI, release and distribution promotion are outside this
  worker's scope.

No public API/ABI, DB, dependency, wire schema, timeout or runner policy change.
This worker makes no commit, push or publication.
