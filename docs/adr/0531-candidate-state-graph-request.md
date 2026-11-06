# ADR 0531: Fairy native graph snapshot request encoding

- Status: Accepted for private graph request encoding
- Date: 2026-11-07

`module_state_graph_request` validates caller-owned Snapshot storage before
publishing an immutable structural Packet. Its bytes and SHA256 match Go's
152-byte big-endian graph request, including exact layout/body/program hashes,
version, four positive limits and sorted unique physical keys/value bytes.

The existing 65,536-entry and 16 MiB payload ceilings remain exact. Known root
cells require canonical native codecs; unknown keys remain opaque, and read
values may exceed the write-value policy. The three Go int-backed limits fit
signed host int; write-byte limits and version retain all unsigned 64 bits.
Byte/text present-empty values remain distinct from missing entries.

The structure ledger retains 16 MiB. A separate bounded 1 GiB byte ledger pays
for input scans, UTF8 validation, output copies and whole-request SHA256; this
allows valid exact-16-MiB payloads without pretending their multiple passes
cost one byte. This is a private work ceiling, not a larger wire/storage limit.
Known-cell lookup is binary search over copied sorted prefixes. The output
reserves capacity once. Both ledgers refuse already overdrawn inputs without
subtraction overflow or reset. Failed admission publishes no Packet.

Packet owns all returned bytes and grants no compiled/execution/storage
authority. The Complete-bound transport wrapper separately seals its own Input
using hashes and known cells from one Prepared owner. Response admission and
one-attempt CAS remain separate requirements. Windows evidence:
`docs/development/candidate-state-graph-request-20261107.md`.
