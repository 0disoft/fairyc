# Generated lockfile v1

- Status: Executable prototype contract
- Decision: `docs/adr/0145-generated-lockfile-v1.md`
- File: `fairy.lock`
- Schema: `fairyc.lock/v1`
- Implementation: `internal/lockfile/`

`fairy.lock` is generated canonical JSON, not a second human manifest. It contains only the
root-reachable resolved graph. Package records are sorted by full PackageInstanceId; dependency
records are sorted by owner ID, binding, and target ID. Empty feature and dependency collections
are JSON arrays, never `null`.

Each package record retains the complete tuple needed to recompute PackageInstanceId v1 and a full
package-source digest. A local record uses `sourceKind: "local"`, uses that digest as source
identity, and stores `localPath` as `.` or the canonical workspace-relative member path. An SDK
record uses `sourceKind: "sdk"`, has an empty `localPath`, and records exact `sdkIdentity` and
`runtimeAbi`; its source identity composes those values with the digest. Absolute paths, timestamps,
environment values, internal node handles, compiler cache paths, and credentials are forbidden.

Strict reading recomputes identities and rejects unknown/duplicate fields, malformed or
contradictory IDs, unknown edge endpoints, duplicate bindings, noncanonical ordering, and cycles.
`MatchesWorkspace` compares a parsed document to a newly validated workspace and reports any drift
as stale. `WriteAtomic` writes only canonical validated bytes through a flushed same-directory
temporary file and rename.

The library does not decide whether a CLI invocation may refresh a missing/stale lock. ADR 0147
selects one compiler-bundled Stage 0 SDK package without filesystem discovery; public stable build
policy remains separate.

The unstable `experimental check-workspace` supplies the first narrow policy: ordinary mode
refreshes after a valid graph exists; `--locked` requires an exact match and never writes. This does
not settle the future stable build/check policy.
