# ADR 0535: Fairy local-state record compatibility

- Status: Accepted for the existing Fairy graph layout family; no filesystem authority
- Date: 2026-11-07

The Fairy persistence bridge must retain ADR 0500's actual Go
`fairyc.local-state/v1` bytes. `module_state_local_codec` generates the expected
canonical JSON layout prefix from an independently validated layout Plan, then
admits only that exact prefix, canonical full-width version, ordered unique
physical entries and the complete trailing SHA256. It does not use the general
8MiB JSON parser or reduce the existing 16MiB state-file bound. Limits remain
explicit: 1..16MiB complete file bytes, 1..65,536 entries and positive per-value
bytes bounded by that file policy. Transaction policies remain separate.

Canonical field spelling/order, integer arrays, decimals, base64 and lack of
whitespace are required. Unknown members, alternate spelling/encoding, trailing
bytes, checksum damage, layout drift, duplicated/reordered physical keys and
integer overflow refuse. JSON null and empty byte strings remain distinct for
exact disk re-encoding. A zero-field layout uses Go's null fields member; an
empty entry list remains []. State is sealed and owns its prefix, identity,
entry rows and value bytes. Snapshot copying returns independently owned
request data; only immutable Snapshot projections are added.

`module_state_local_base64` accepts canonical standard padded base64, including
zero discarded pad bits. It preserves the 16MiB raw-value ceiling and its
22,369,624-byte encoded ceiling. Malformed input, bounded/work refusal and
allocation failure retain separate codes 1001/1002/1003. Structure and byte
histories remain cumulative with 16MiB/1GiB ceilings. Before layout validation,
the codec reserves the existing validator's whole 1MiB bounded operation;
that validator uses a fresh local ledger inside the already prepaid allowance.
No caller history is reset or refunded. Decimal output appends directly to the
shared output buffer rather than allocating one buffer per key octet.

This closes canonical record and detached snapshot prerequisites only. It
does not open a store, lock a file, publish a record, grant execution/CAS
authority, migrate layouts or admit wider graph source families. Actual native
file operations and one-attempt publication remain separate required work.
Evidence: `docs/development/candidate-local-state-codec-20261107.md`.
