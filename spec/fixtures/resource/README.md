# Resource Flow Fixtures v0

This directory is the machine-readable companion to `docs/language/0001-values-and-resources.md`, `docs/adr/0003-stage-0-safety-boundary.md`, and `docs/adr/0011-stage-0-resource-flow-fixture-contract.md`.

`manifest-v0.json` is the case index. `schema-v0.json` defines its JSON Schema 2020-12 shape. Every case is a parser-valid, resolver-valid, and type-valid mini-package before resource Flow assertions are evaluated.

## State assertions

Each state assertion names:

- the resource binding declaration;
- the source span used as an observation point;
- whether the state is sampled immediately `before` or `after` that span;
- `Live`, `Dead`, or `MaybeLive`;
- zero or one complete-root cleanup obligation.

`Live` always has one obligation and `Dead` always has zero. `MaybeLive` is recorded only as a rejected source state and uses `cleanupObligations: null` because one unconditional count does not exist.

`internal/resourcefixture` executes all 12 v0 cases after requiring parser, resolver, and type setup
to succeed. It compares selected state observations and the complete independent `E11xx` set,
including mismatched outer-resource state on explicit break and continue edges.

## Diagnostic assertions

Fail cases list every independent `E11xx` diagnostic after cascade suppression. Primary and related spans use source-root-relative `/` paths and zero-based half-open UTF-8 byte offsets. Related spans sort by role, path, start byte, and end byte. Full wording, Flow block numbers, MIR locals, and backend drop flags are not assertions.

## Adding a case

Place sources below `cases/pass/<id>/src/` or `cases/fail/<id>/src/`, list every `.fy` file in unsigned UTF-8 path order, and keep module declarations equal to their source-root-relative paths. Prefer one independent rejection per fail case. Reassignment, close ownership, tagged resource payloads, partial construction, and executable drop traces need explicit new cases and cannot be inferred from these first whole-root fixtures.
