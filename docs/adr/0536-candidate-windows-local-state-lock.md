# ADR 0536: Fairy Windows local-state lock authority

- Status: Accepted as a prerequisite of the existing ADR 0500 local provider
- Date: 2026-11-07

`windows_rooted_file::state_lock_open` creates the fixed flat `state-v1.lock`
exclusively or reopens its existing regular file. Neither mode initializes,
truncates or removes authority bytes. The sealed StateLock retains that file
handle independently of operation leases. Legacy rooted-file modes, staging
cleanup and publication semantics remain unchanged.

`state_lock_acquire` reopens the same fixed name relative to the retained root.
Fairy checks regular-file metadata and file identity against StateLock; the
native check also compares volume serial and the complete 64-bit file ID.
The operation handle denies delete sharing, closing namespace replacement
between identity validation and lease release. Different roots, replaced or
missing authority names and reparse/nonregular files cannot acquire authority.

A private native wrapper calls LockFileEx with exclusive and fail-immediately,
exactly byte zero of that independent handle. Contention is stage 15 / Windows
code 33; identity mismatch is stage 16 / code 1006. Other open and metadata
failures retain the existing stages. There is no lock retry, callback or
transaction body inside this primitive. StateLease owns its reopened File;
nested resource cleanup closes it on every scope/return path and releases
the OS lock. Process termination also closes it. Authority survives lease
release and process restart. No borrowed alias handle is owned twice.

The three new compiler-private host symbols are admitted consistently by the
Go selector and authored Fairy runtime contract. The deterministic Windows
runtime asset pin is regenerated from current C sources, and complete emitted
runtime bytes are compared against live Go selection. This extends the private
host boundary only; it adds no public SDK/runtime ABI or CLI command.

This closes lock authority only. No store operation reads/writes state through
this primitive yet. Canonical record codec, sync-before-rename publication,
layout/version CAS, no-op and version-exhaustion behavior and one-attempt graph
orchestration still require their actual provider integration. Linux stays
batched. Evidence: `docs/development/candidate-local-state-lock-20261107.md`.
