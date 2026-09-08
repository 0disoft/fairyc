# Aggregate-aware `.fymap` store v3

- Status: Implemented internal opt-in store; catalog CLI uses in-memory v3 maps, not this disk store
- Layout: `fymap-v3/<PackageInstanceId>/<checksum-hex>.fymap`
- Constructors: `NewMapStoreV3`, `NewMapStoreV3InRoot`
- Transport: `docs/atlas/fymap-transport-v3.md`
- Filesystem contract: `docs/atlas/fymap-store-v1.md`
- Decision: `docs/adr/0387-generic-aggregate-source-preparation.md`

## Version and ownership

Each opened store selects exactly one version. V1 and v2 constructors retain their
original directories and codecs; v3 opens only `fymap-v3`. Unknown internal version
values fail before directory creation. No format discovery, migration or fallback
is performed. A partition of another logical version cannot be published.

V3 reuses opened-root publication, checksum-derived generation names, temporary
file handling, sync, retention and corruption cleanup. It introduces no new
path-based I/O helper. The selected version directory must be a real directory
with the same identity before and after opening. Caller ownership and portable
durability limits remain those of the shared filesystem contract.

Generation count, total bytes, per-map bytes and temporary age are mandatory,
independent limits per opened version store. V3 eviction and corruption cleanup
remain within its retained directory, without removing v1/v2 generations.

## Current-source reads

`CurrentWorkspaceExpectationsV3` derives source identities and both ordinary and
generic source-interface hashes from a verified compiler result and loaded
workspace. `LoadCurrent` uses the exact v3 decoder and checks compiler identity,
interfaces, package/file/line-map identities, directory and checksum filename
before returning a hit. Original aggregate declarations, concrete field types,
cleanup calls and borrow owners survive close-and-reopen.

Stale source, incompatible interfaces, corrupt bytes and conflicting current
generations remain distinct outcomes. A legacy map copied into the v3 directory
is rejected and may be removed there; its original legacy file is untouched.
Checksums do not authenticate the publisher or prove compiler authorship.

## Evidence and remaining integration

Host tests cover used/unused aggregate templates, repeated publication, reopened
root handles, exact payload recovery, an on-disk template edit followed by reload,
interface mismatch, corruption and repair, generation eviction, preservation of
legacy bytes, copied-version rejection, per-map limits and parent traversal.
Directory-alias checks are conditional on host symlink support.

No CLI automatically opens the store, and no default root or quota is introduced.
V3 local queries use `NewQueryIndexV3` as documented in
`docs/atlas/fymap-query-v3.md`. The v3 workspace join and experimental catalog
commands consume in-memory maps without opening this store.
Cross-platform execution and power-loss guarantees are not established by these
host tests.
