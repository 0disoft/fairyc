# ADR 0508: Explicit Fairy state-graph layout artifacts

- Status: Accepted for the private Fairy layout/source-matching prerequisite
- Date: 2026-11-07
- Extends ADRs 0506-0507 without admitting a Complete graph or ordinary CLI mode.

The Fairy compiler needs the same seven storage codecs as the explicit Go
source-graph profile. `module_state_declarations::build_graph` and its metered
entry admit `bytes/v1` and `utf8/v1` alongside the five original codecs. The
ordinary declaration builder and layout validator retain their five-codec
boundary. Storage codec IDs do not become executable scalar type IDs.

`module_state_artifact::encode_graph` and `decode_graph` use `FYSTATE1` with
the existing bounded layout/field/name encoding. Ordinary `FYSTATE0` readers
and writers remain separate. Both directions independently recompute field
prefixes and layout hashes; neither a marker nor stored digest grants trust.
Existing canonical preimages, hashes and ordinary wire bytes do not change.

`module_project_artifact_state::build_graph` reconstructs declarations from
the current parsed source/catalog and compares every saved identity, codec,
field order and source coordinate. A trailing comment may preserve those
layout facts; shifted coordinates reject. This detached match grants no
Complete, transaction body, native execution or cache-hit authority.

Declaration/hash work retains its independent 1 MiB limit. Artifact framing
retains its existing 16 MiB phase ledger; current-source matching uses the
existing aggregate-program ledger, charging actual independent phase work.
Exact budgets succeed and one-short budgets refuse without exceeding the cap.
Checked byte append now preserves bound/work error 10 separately from allocation
error 11; the existing boolean append entry remains available. Hash helpers and
source declarations propagate the actual cause instead of guessing from the
remaining counter. Artifact errors retain 1001/1002/1003.

Windows low/optimized source, byte-for-byte independent Go canonical encoding,
current-source, ordinary compatibility, corruption and budget evidence is in
[the implementation record](../development/candidate-state-graph-20261107.md).
Fairy transaction graph IR, helpers/borrows/branches, current-source Complete,
artifact/cache integration, ordinary commands, native Linux and every final
acceptance gate remain open. No stable language/CLI/SDK/runtime ABI, dependency,
database/provider, default bound, CI/runner or repository policy changes.
