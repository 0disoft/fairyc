# Explicit remote package sources v1

- Status: Bounded Go Stage 0 experimental preparation contract, ADR 0498.
- Source owner: `internal/packagesource/`; CLI: `fairyc experimental fetch-packages [--offline]`.
- Does not change manifest v1, generated graph lock v1 or stable build networking.

## Source pins and resolution

Commit a `fairy.sources.json` beside `fairy.toml`:

```json
{
  "schema": "fairyc.sources/v1",
  "packages": [
    {
      "path": "vendor/math",
      "packageUid": "example.math",
      "version": "1.0.0",
      "url": "https://packages.example/math-1.0.0.zip",
      "sha256": "<64 lowercase hexadecimal digits from reviewed archive bytes>"
    }
  ]
}
```

The placeholder digest is intentionally not accepted. All fields are required, exact lowercase
spellings; duplicate/unknown fields and trailing JSON reject. The bounded plan admits 1-32 unique
paths and UIDs, sorts by path, and requires exact manifest versions. Every path must be an explicit
workspace member, root dependency versions must agree, and destinations/cache cannot overlap the
root source or other members. No parent lookup, implicit member, version range or registry query.
Remote dependency paths retain the existing workspace-root-relative interpretation. List every
needed remote package explicitly; the ordinary loader later validates graph edges/cycles/imports.

The URL must be absolute HTTPS without credentials, query or fragment. Redirects, authentication
and ambient credential configuration are not admitted. Hash pins are a trust input reviewed by
the caller, not learned from the downloaded package or an unsigned server-side checksum.

## Cache, extraction and publication

Archives contain `fairy.toml` at ZIP root and the declared source directory with at least one
`.fy` file. Manifest UID, exact version and language must match the pin/root; nested workspace
declarations reject. Hooks/scripts are never executed. ZIP entry paths must use portable canonical
relative components; absolute, dot/traversal, backslash, drive, reserved-device, duplicate,
case-colliding, file/directory-conflicting and symlink/special entries reject. Explicit empty
directories are ignored. Files are created as ordinary non-executable data.

The cache is the contained workspace directory `build/package-sources`, with archive names
`<sha256>.zip`. Always rehash before use, including warm/offline reads. Offline mode makes zero
HTTP requests; missing/corrupt entries fail. Online corruption also fails; the caller may remove
the named cache blob and retry. Downloads are limited to 30 seconds per request, 16 MiB/archive;
one fetch is limited to 64 MiB compressed and 64 MiB expanded bytes, 4,096 files and 8 MiB/file.
The operation has a two-minute context. Cache retention is manual, not an unbounded background job.

Validate all entries and identity before caching an archive or publishing a directory. Write
complete staged files beneath an opened contained root, then publish a new directory by rename
under an exclusive fetch lock. Existing exact trees are reused; differing files or any extra
regular file/symlink fail without modification. Empty directories do not affect equality.
The lock serializes cooperating fetches; stale locks after a killed process require manual
inspection/removal. A hostile process rewriting the same workspace during fetch is not supported.
Ordinary errors clean the current stage/lock. Prior complete packages and verified cache blobs
may remain if another package fails. Physical-media durability and global rollback are not promised.

## Compiler and lock behavior

After fetch, run `fairyc check` once to create/update ordinary `fairy.lock`; subsequent
`check/build/run/test --locked` use its existing exact graph/source checks. Fetch never changes
`fairy.toml`, source pins or `fairy.lock`. Stable commands do not fetch, consult the archive cache
or authenticate transport provenance. To audit or restore the pinned vendor snapshot, explicitly
run fetch again. Restore an absent package from the cache with `--offline`.

This experimental step admits a CLI surface, not a stable registry ecosystem. Registry selection,
signing/trust delegation, range solving, automatic updates, Git/submodules/LFS, publisher services,
cache migration/GC and Fairy-authored candidate parity remain outside v1.
