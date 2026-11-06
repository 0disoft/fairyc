# ADR 0518: Derive state body identity from current source and package coordinates

- Status: Accepted for the private state-graph source profile
- Date: 2026-11-07

`module_state_graph_source_identity::derive` accepts a sealed authored source
owner, a coordinate table and a selected package owner. The retained immutable
Parsed supplies the normalized module name and actual private, parameterless
void root. Its verified graph supplies the canonical storage effects. Callers
cannot supply a replacement descriptor. The selected coordinate must be unique,
canonical and nonzero; unrelated coordinate rows retain existing selection rules.

The resulting opaque Bound owns its context, exact Go-compatible body preimage
and digest. Mutable getters and caller construction are absent. Work and errors
remain the existing 16 MiB / 1001 / 1002 / 1003 profile. This is source/package
identity association; ordinary Complete, cache reuse and execution still require
an owning integration factory. No ordinary/default source admission is widened.

Windows low/optimized independently check 46 cases each and seven ownership
refusals. Commands, exclusions and the companion interface prerequisite are in
`docs/development/candidate-state-graph-package-20261107.md`.
