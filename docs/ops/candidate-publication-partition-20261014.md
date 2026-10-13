# Complete publication adapters and generic cleanup units

Complete-project C adapters now live in `module_project_complete_c`, separately
from the large per-body C emitter. They retain the same owner and naming gates;
the complete owner and its interfaces remain accessible through the bundle.
Generic cleanup request collection moves to `module_unit_generic_cleanup` while
the owning specialization Plan and its shared budget remain unchanged.

Four authored cases pass Go-native and candidate C development/release: two
complete-project ownership/partition cases, and cleanup shadowing/recursion plus
actual SDK Vec cleanup specialization with its required source fixture. An older
resource-receiver test also fails on the unchanged baseline; it is excluded from
the successful selection and not counted as a regression pass. The first Vec
test invocation omitted the SDK argument and is retained as failed harness
history. Receipt: [evidence](evidence/candidate-publication-partition-20261014.json).

The complete dispatcher attempt progresses past the former C-emitter stage-7
borrow failure and generic-planner stage-4 value failure. All checked units then
reach project linking, which rejects with code 1002 at `project_toml_values.fy`.
No successful C packet or native dispatcher image is claimed. Exact-capacity
classification and link work/representation measurements are the next task.

`project_emit` build/run integration is prepared but not admitted yet. Its patch
and the real-dispatcher build/native-verification tools are retained in ignored
scratch files for the next unit. The current source still uses its existing
build/run route. Root check and locked check pass 450 source files / 5822 MIR
functions. Current full-image/fixed-point, CLI execution, cache, product
stack/deadlines, Linux/full suite and hosted CI remain open. Public CLI/SDK ABI,
Go, DB, CI and runner configuration are unchanged; no push.
