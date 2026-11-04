# Explicit remote package sources v1

- Status: Executable bounded Go Stage 0 experimental preparation contract, ADR 0498;
  Fairy-authored source-plan parsing, workspace binding, HTTPS retrieval and offline restoration
  are connected through the bounded Windows CLI `experimental fetch-packages [--offline]`.
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

Also declare the destination in the existing root manifest (retain its `[package]`):

```toml
[workspace]
members = ["vendor/math"]
[dependencies.math]
path = "vendor/math"
version = "1.0.0"
features = []
```

Prepare an archive with `fairy.toml` and `src/` at ZIP root; a repository-name wrapper directory
is not stripped. With PowerShell, an explicit `Compress-Archive -LiteralPath fairy.toml,src
-DestinationPath math.zip` invocation from the package directory supplies that layout. Obtain the
pin with `(Get-FileHash -LiteralPath math.zip -Algorithm SHA256).Hash.ToLowerInvariant()` and
review the package before hosting/using it. A hash calculated from arbitrary untrusted downloaded
bytes is not independent publisher authentication.

The placeholder digest is intentionally not accepted. All fields are required, exact canonical
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

Go existing-tree verification shares an 8,193-entry budget including the package root.
Directory reads admit at most 128 entries per batch plus one overflow probe before
retaining or sorting. A partial scan cannot certify reuse. Opened-directory identity
is checked, and each enumeration handle closes before descending into children.

Archives contain `fairy.toml` at ZIP root and the declared source directory with at least one
`.fy` file. Manifest UID, exact version and language must match the pin/root; nested workspace
declarations reject. Hooks/scripts are never executed. ZIP entry paths must use portable canonical
relative components; absolute, dot/traversal, backslash, drive, reserved-device, duplicate,
case-colliding, file/directory-conflicting, nested `.git`/`.hg`/`.svn` metadata and symlink/special entries reject. Explicit empty
directories are ignored. Files are created as ordinary non-executable data.

The cache is the contained workspace directory `build/package-sources`, with archive names
`<sha256>.zip`. Always rehash before use, including warm/offline reads. Offline mode makes zero
HTTP requests; missing/corrupt entries fail. Online corruption also fails; the caller may remove
the named cache blob and retry. Downloads are limited to 30 seconds per request, 16 MiB/archive;
one fetch is limited to 64 MiB compressed and 64 MiB expanded bytes, 4,096 files and 8 MiB/file. Each archive also limits explicit/implicit path entries to 8,192 and path depth to 32 components. Remaining operation budgets apply before reading/decompressing the next archive.
The operation has a two-minute context. Cache retention is manual, not an unbounded background job. Immutable cache publication requires filesystem hard-link support; unsupported filesystems fail rather than falling back to replacement.

The Go Stage 0 downloader shares one 30-second budget across GET attempts, backoff and body
reading. Only bodyless GETs can retry, at most three attempts with 200 ms then 400 ms backoff,
for temporary network failures or HTTP 502/503/504 without `Retry-After`. A `Retry-After` response
fails with its HTTP status rather than retrying ahead of the server's requested delay. Cancellation,
TLS trust errors, other HTTP statuses, partial-body errors, size/encoding rejection and hash or
archive validation failures do not retry. Discarded response bodies close before backoff; original
transport errors and cancellation causes remain inspectable. Cache and package publication are
outside this retry wrapper and are never automatically replayed. This policy describes the Go
downloader; it does not claim retry parity for the Fairy-authored Windows transport.

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
cache migration/GC and Linux Fairy-authored fetch parity remain outside v1.

The private Fairy `project_remote_plan` module parses the same source-plan JSON, validates
decoded fields and returns pins sorted by path. It reuses the existing manifest name/UID/version
rules and JSON duplicate-key checker, adds the 64 KiB and 1-32 pin limits, and shares the existing
16,777,216-unit work budget. `project_source_url` validates the HTTPS host, port, IPv6/zone and percent
escape forms against the Go source policy. It performs no network, archive or filesystem work.
The Windows low/optimized oracle compares acceptance and every returned field; see the
[source-plan receipt](../ops/windows-candidate-remote-plan-20261103.md). `project_remote_bind`
checks the already validated plan against the root manifest: explicit non-root membership,
exact declared dependency versions and case-insensitive directory-boundary conflicts with
source, cache and other members. Binding shares the work meter and reads no member files.

The private Windows `project_remote_offline::restore` API now restores verified cached archives.
It reads and binds the source plan/root manifest, acquires the exclusive fetch lock, reads only
the exact contained cache names and recomputes SHA-256 on every attempt, including reuse. Missing
or corrupt cache data fails without any transport fallback. `project_remote_archive` validates
single-volume ZIP32/ZIP64 stored/DEFLATE entries, payload and descriptor CRCs, portable paths,
case/type conflicts and package UID/version/language/source identity before publication.

`project_remote_fs` retains directory handles, rejects reparse/case aliases, writes/syncs staged
ordinary files and publishes the complete directory without replacement. Existing exact trees
are reused; differing, incomplete, additional-file and linked trees are preserved and rejected.
Ordinary failures clean only the owned current stage and fetch lock. Earlier complete packages
may remain when a later package fails. Source plans, manifests, graph locks and cache bytes remain
unchanged. No package hooks or subprocesses execute.

The byte/file/depth limits above apply. This private implementation also bounds each directory
scan to 8,192 rows and path-index work to 268,435,456 units; exceeding either fails closed. The
two-minute deadline is cooperative, checked around archive processing and publication rather
than interrupting a native I/O operation or a decoder mid-entry. Private staging names include
the archive digest; a leftover collision fails without deleting the old stage and requires
manual inspection. See the [Windows offline receipt](../ops/windows-candidate-remote-offline-20261103.md).
Windows candidate dispatch calls this API for `experimental fetch-packages --offline`;
see the [offline CLI receipt](../ops/windows-candidate-fetch-cli-20261104.md).
The invocation without `--offline` now calls `project_remote_online`, which reuses
the same plan/binding/archive/publication contracts and existing-tree verification.
Only an absent cache blob triggers `project_remote_https`; malformed, linked or
corrupt cache data fails without a network retry. HTTPS uses Windows system trust
and no ambient proxy, authentication, cookies or redirects. WinHTTP is loaded from
System32 only. Encoding is absent/identity, and declared/actual response bytes are
bounded independently. The 30-second deadline covers headers and every body read;
it is also capped by the remaining two-minute operation time. Native cancellation
keeps callback/read storage alive until WinHTTP reports its final handle callback.

After validating the archive digest, ZIP entries and manifest identity,
`project_remote_cache` writes and syncs a new private file, then creates a retained-
directory-relative hard link without replacing an existing cache object. Closing
removes the owned staging name. Unsupported hard-link filesystems fail; matching
concurrent cache bytes may be reused only after rehashing. The private stage name
is digest-derived; an existing stage fails without deletion. Earlier complete
cache/package publications survive a later failure. Stable commands gain no cache
consultation or networking, and neither fetch mode changes `fairy.lock`.

Both modes validate syntax/help before workspace reads, share the Go success and
invocation/help output and exit codes 0/2/7/4, and use bounded candidate preparation
reasons beneath `package fetch failed:`. The Windows source entry declares the
`network` effect for explicit online dispatch; offline execution makes no HTTP
request. Native Linux facade/transport selection, native replay and distribution
acceptance remain separate. See the [online receipt](../ops/windows-candidate-remote-online-20261104.md).


## Usage and command result

```sh
fairyc experimental fetch-packages
fairyc check
fairyc run --locked
# Restore an absent vendor directory from the verified local archive cache:
fairyc experimental fetch-packages --offline
```

No root/path/cache override, forced replacement, authentication, environment credential lookup or
implicit dependency recursion is admitted. Success prints one line with package/download/cache/
install/reuse counts. Codes are 0 on success, 2 on invocation errors, 7 on plan/network/cache/
identity/filesystem failures, and 4 on output failure. Output failure may follow completed work.
The exclusive root lock is `.fairyc-fetch.lock`; staged directory names are private and generated.
Interrupted-operation leftovers must be inspected before manually removing them; automatic fetch
never deletes an existing vendor directory, even when incomplete.

Primary API references: [Go ZIP reader](https://pkg.go.dev/archive/zip#Reader),
[HTTP client redirect/timeout policy](https://pkg.go.dev/net/http#Client), and
[opened directory roots](https://pkg.go.dev/os#Root). Explicit portable entry and identity checks
are Fairy policy in addition to the underlying APIs, not capabilities inferred from an archive.
