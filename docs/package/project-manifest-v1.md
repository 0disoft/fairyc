# Project manifest v1

- Status: Executable parser contract
- Decision: `docs/adr/0143-local-exact-project-manifest-v1.md`
- File name: `fairy.toml`
- Schema identity: `fairyc.manifest/v1`
- Implementation: `internal/projectmanifest/`
- Fixtures: `spec/fixtures/project-manifest/manifest-v1.json`

## Complete example

```toml
schema = "fairyc.manifest/v1"

[package]
uid = "example.tools.app"
name = "example-app"
version = "1.2.3"
language = "fairyc-language/3"
source = "src"

[workspace]
members = ["libs/net", "libs/util"]

[dependencies.net]
path = "libs/net"
version = "1.4.0"
features = []

[dependencies.util]
path = "libs/util"
version = "2.0.0"
features = ["base"]
```

The root key and every package/dependency field shown above are closed. Unknown fields reject; they
are not ignored for forward compatibility. A newer schema uses another exact `schema` identity.

## Package

- `uid` is long-lived identity input and has at least two lowercase dot-separated kebab segments.
- `name` is the lowercase kebab display/CLI name. It does not become a module prefix.
- `version` is one exact SemVer 2.0.0 value. A range is invalid.
- `language` is one exact `fairyc-language/<revision>` value.
- `source` is an explicit canonical portable relative directory. V1 path segments are ASCII
  alphanumeric/underscore followed by alphanumeric, dot, underscore, or hyphen; Windows device
  names and trailing dots reject.

The parser does not assume `src`, infer the version from Git, or choose the newest language.

## Workspace and dependencies

`workspace.members` is optional only because a single-package project needs no workspace table.
When the table exists, the array is required, sorted, unique, and explicit. No parent-directory or
recursive manifest search occurs.

Dependency table names are bindings. Each dependency contains exactly:

- `path`: canonical workspace-relative path spelling to be checked by the future loader;
- `version`: exact expected member package version;
- `features`: explicit sorted unique per-edge feature set.

`features = []` is required for no features. Manifest v1 does not silently apply default features
or unify two edges. The resolved graph may contain feature-distinct package instances.

## Parse and validation boundary

The parser accepts at most 1 MiB of valid UTF-8, applies the pinned TOML decoder, rejects syntax and
duplicate-key failures, rejects undecoded fields, proves every required field, and returns
dependencies sorted by binding. It emits internal reason/field categories without reserving final
public diagnostic codes.

It performs no filesystem access. ADR 0144 and `workspace-loader-v1.md` separately prove that member
and source directories exist, obey the exact-case/no-symlink boundary, contain the complete `.fy`
set, match declared versions and language, and form a valid root-reachable graph. Neither the byte
parser nor that loader proves or writes a lockfile.
