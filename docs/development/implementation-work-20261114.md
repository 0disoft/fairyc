# Ordered implementation work — 2026-11-14

User request: implement items 1 → 2 → 3 → 4 → 5 → 6 from the current remaining
work review. Work on main, preserve unrelated work, use small tested local
commits and no desktop/keyboard/clipboard automation. No push/publication,
credential invention, external registry deployment or hosted CI is authorized.

## Execution order

1. Reproduce the valid flat-Result generic helper-call invariant on current
   source, isolate the first failing compiler layer, fix it and replay the
   affected cache/invalidation controls. Go-reference success alone does not
   establish Fairy candidate success.
2. Reconcile candidate CLI/diagnostics with the Go additions: Win32 generation,
   resource preparation/local packaging and precise generic/closure diagnostics.
   Keep native transport separate from Fairy policy and preserve failure output.
3. Extend generic/inline transport one concrete type/control-flow family at a
   time, with explicit contracts, source/restore/cache/native checks and unchanged
   legacy formats outside each newly admitted family.
4. Extend closure signatures/captures in similarly bounded steps. Borrowed or
   owned captures require explicit lifetime/drop contracts before implementation;
   a wider signature does not imply arbitrary capture or FFI admission.
5. Implement missing portable filesystem/library operations individually;
   distinguish source/emission/model evidence from actual Linux execution.
6. Extend package version/Git/update/registry/trust capabilities through concrete
   local contracts and validators first. Do not invent a registry endpoint or
   key/provider. External service and live credentials remain owner inputs.

Each stage will record its implemented families, exact checks and still-open
scope. The task is not complete merely because one admitted slice passes.
Whole compiler/self-build, target maturity and release evidence remain separate
from a successful focused implementation. Existing immutable closure execution
and the shared Win32 modules must not be reimplemented from stale checklists.

## Progress

- 1: current-source reproduction in progress.
- 2–6: pending, in the order above.
