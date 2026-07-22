# `.fymap` contained store v1

- Status: Implemented internal opt-in store
- Layout: `fymap-v1/<PackageInstanceId>/<checksum>.fymap`
- Decision: `docs/adr/0187-contained-fymap-store-and-current-source-validation.md`
- Implementation: `internal/atlasmap/store.go`, `internal/atlasmap/current.go`
- Transport: `docs/atlas/fymap-transport-v1.md`

## Authority boundary

The caller selects either an explicit store root or an already-open `os.Root` plus a canonical
relative child. The store opens and retains the version directory and performs every create, open,
read, scan, remove, and rename relative to that handle. Lexical prefix checks are not used as
filesystem authority. Parent traversal and external symlink or junction escape fail.

The direct constructor treats its explicit path as the selected boundary. Code that needs
containment inside a workspace must use the opened-root constructor. `Close` releases the retained
directory handle and must not race another store operation.

## Layout and publication

```text
<selected-root>/
  fymap-v1/
    fypi1-sha256-<package-digest>/
      <map-checksum-hex>.fymap
      .tmp-<random-128-bit-nonce>
```

The generation filename is the 64 lowercase hexadecimal digest portion of the logical map checksum,
not an arbitrary label. The map's canonical envelope carries the full self-identifying
`fymap1-sha256-...` form.

Publication writes, file-syncs, and closes an exclusive `0600` temporary in the final directory.
Shared `internal/durablefile` publication then performs a handle-relative same-directory rename and
platform post-publication sync. Concurrent identical writers converge. A different payload under
the same checksum-derived filename is removed and repaired because no valid different payload can
own that name.

## Mandatory bounds

Every store requires:

- maximum retained generation count;
- maximum retained generation bytes;
- maximum bytes per encoded map; and
- maximum unpublished temporary age.

The per-map limit cannot exceed the total limit or the bounded reader range. Oversize maps fail
before decoding. Sweep orders valid generations by modification time then relative path, removes
expired temporaries, and evicts oldest generations until both limits hold. Time controls retention
only; it never decides compatibility or truth.

## Current-source acceptance

`CurrentWorkspacePackages` requests exact semantic package and file metadata from the validated
workspace. `SemanticFiles` recomputes the package digest and per-file content/line-map identities
from retained bytes; mutation after workspace validation rejects expectation construction.

`CurrentWorkspaceExpectations` additionally requires the matching verified compiler result and
exact executable identity, then recomputes the complete public-interface-hash set. This is the
preferred safe-reader input path; it prevents callers from copying a stored interface hash back into
the expectation that is supposed to verify it.

A map is current only when all of these equal the caller-authorized workspace:

- package UID, language, `PackageInstanceId`, `PackageCoordinateId`, package-source digest, source
  kind, and runtime ABI;
- canonical features and direct dependency instances;
- exact self/direct-dependency UID-instance-coordinate references; and
- every canonical file path, exact content digest, line-map identity, and source/test kind.

The caller also supplies the exact compiler executable identity and the current package
public-interface hash. These dimensions must match before the store returns the map. Stale source,
incompatible compiler/interface, corruption, and multiple compatible generations are distinct
fail-closed outcomes.

## Read behavior

Readers scan only the exact current package-instance directory. Every checksum-shaped candidate is
size-bounded, canonically decoded, semantically revalidated, matched to its filename and directory,
then checked against current compatibility. Corrupt generations are removed best-effort. Invalid
unrelated names are ignored and never interpreted as maps.

Exactly one compatible current generation is a hit. Zero generations is a miss. More than one valid
compatible generation is `ErrMapConflict`, because silently selecting one would hide compiler
nondeterminism or contradictory fact production.

## Interruption and limits

Interruption before rename can leave only an unpublished temporary. Interruption after publication
can leave one complete generation. New readers therefore observe a miss or a fully verified hit;
they never accept partial bytes. Physical media or filesystems that violate flush semantics remain
outside the portable guarantee.

## Not enabled

- no default store path, generation count, byte quota, or recovery age;
- no stable or experimental CLI command yet;
- no automatic write during `check`, `build`, `run`, `test`, or `emit`;
- no dependency-map discovery or transitive join;
- no dependency-map join, Context Pack, source-excerpt, LSP, SCIP, or MCP reader; package-local
  `overview`, `show`, and `around` are separately owned by `docs/atlas/fymap-query-v1.md`;
- no remote store, signing, authentication, watcher, daemon, or background cleanup; and
- no rewrite-speed, retained-size, or default-policy claim.
