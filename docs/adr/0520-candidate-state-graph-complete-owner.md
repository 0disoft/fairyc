# ADR 0520: Publish explicit state graphs through the common Complete owner

- Status: Accepted for the existing private single-module graph source family
- Date: 2026-11-07

`project_compiler_core::Complete` owns exactly one ordinary or state-graph mode.
The ordinary factories retain their required signatures and publish their same
checked program and interface records through one private constructor. The new
`build_complete_state_graph` and `restore_complete_state_graph` factories select
the explicit graph mode; normal workspace factories do not select it implicitly.

`project_compiler_state_graph` retains either fresh authored source or an exact
current-source restored graph owner. It derives the actual root descriptor and
selected package coordinate, the exact Go state-body identity and both original
public/source interfaces before publishing Complete. All graph source bodies
are independently verified, including reachable helpers, effects, branches and
borrow lifetimes. This admits all seven codecs of the existing graph source
profile; it does not broaden that profile's declaration/signature restrictions.

`is_state_graph` selects the mode. `checked_state_graph` exposes its immutable
owner; `checked_program` requires ordinary mode. `public_interfaces` works in
both modes. The existing package body identity API copies the checked graph
identity for its actual published owner. Graph mode retains no ordinary program
or ordinary interfaces vector; the graph owner holds its checked records.

The graph factory keeps one bounded 16 MiB work history and private errors
1001/1002/1003. Core maps graph build/restoration errors to phase 8, stages 1/2;
no partial Complete or interface prefix escapes. Ordinary unit caches publish
only after common Complete construction succeeds. Body identity keeps its existing
9/10/11 statuses. Source identity now accounts for entry_at's bounded table scan.

Existing ordinary C ownership/partition/test entry points reject graph mode with
1001. Ordinary workspace/key/specialization cache paths reject it with 9; Atlas
preparation/publication rejects it with 7; the ordinary semantic-test interpreter
returns 2 with a diagnostic even for zero selected tests. They cannot read the
ordinary-only getter and accidentally process a graph as ordinary function IR.

This closes explicit source/coordinate/body/interface Complete publication and
current-source graph-envelope restoration. It does not close memory/disk graph
cache admission, native graph lowering or persistence, ordinary command
selection, wider workspace source families or final acceptance. Evidence and
remaining work: `docs/development/candidate-state-graph-complete-20261107.md`.
