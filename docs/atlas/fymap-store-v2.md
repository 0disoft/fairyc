# Generic-aware `.fymap` store v2

- Status: Implemented internal opt-in store; query and CLI integration pending
- Layout: `fymap-v2/<PackageInstanceId>/<checksum-hex>.fymap`
- Constructors: `NewMapStoreV2`, `NewMapStoreV2InRoot`
- Transport: `docs/atlas/fymap-transport-v2.md`
- Shared filesystem contract: `docs/atlas/fymap-store-v1.md`
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`

## Explicit version selection

The constructor selects one version for the lifetime of the opened store. V1 constructors still
open only `fymap-v1`; v2 constructors open only `fymap-v2`. Publish rejects a partition from another
logical version. Readers use the selected decoder and checksum schema/encoding, not the filename
or a guessed document version. Neither path migrates, searches, or falls back to the other version.

Both versions share root-relative temporary creation, immutable publication, sync handling,
generation naming, retention, corruption cleanup, and statistics. Before keeping the version-root
handle, the constructor rejects a linked/non-directory version entry and checks that the opened
directory has the inspected identity. The caller must still select application-owned storage;
this does not add a guarantee against hostile mounts or unrestricted mutations of the parent.

Limits are mandatory and apply independently to each opened version store. Opening both versions
under one selected root does not turn either version's byte quota into a combined quota. Callers
needing a total budget must allocate those two quotas accordingly. Cleanup remains within the
opened version directory; it never sweeps the other version's tree.

## Current-source loading

Publish accepts v2 partitions from PrepareWorkspaceV2. LoadCurrent takes expectations from
CurrentWorkspaceExpectationsV2, including the original generic source-interface identity. Every
candidate must pass canonical decoding, checksum/name agreement, package-directory agreement,
current package/file/line-map identity checks, exact compiler identity, and both ordinary and
generic interface compatibility. Generic declarations, concrete summaries, type arguments, calls,
and completeness survive close-and-reopen.

Stale source, incompatible interfaces, corruption, and conflicting compatible generations retain
their distinct fail-closed outcomes. A v1 file copied into v2 storage is corrupt for that reader
and may be removed there; the original v1 file is untouched. Checksums remain integrity checks,
not a proof that a publisher was authorized or that claimed compiler facts were genuine.

## Evidence and limits

The host tests cover used/unused generic map publication, repeated publication, close/reopen through
an opened root, exact payload recovery, incompatible generic interfaces, stale source, corruption
and repair, independent version quotas, cross-version rejection, and per-map byte limits. Existing
v1 store tests continue to exercise the shared path. Directory-link tests are conditional on host
link support; runtime and filesystem durability limitations remain those of the shared store.

No command automatically creates this store. V2 package queries, cross-package specialization
joins, and CLI activation are not enabled by the constructors. No network, watcher, background
cleanup, default path, or default retention policy is added.
