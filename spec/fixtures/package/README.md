# Resolved package graph fixtures

`manifest-v0.json` describes already-resolved package graphs for identity, direct-dependency visibility, module ownership, and explicit standard-library injection tests. It is not a user package manifest, lockfile, registry protocol, or version solver input.

Package, source, and version identities are opaque fixture tokens. Instance classes compare the tuple of package UID, resolved source identity, version identity, and sorted feature set without freezing a concrete `PackageInstanceId` hash or encoding.

Each package node owns one source root. Its source list is complete, `/`-separated, and relative to that root. Module declarations still obey the source-root-relative mapping from `spec/grammar/stage0.ebnf` and the Stage 0 parser contract.

No package resolver or compiler runner exists yet, so the graph and source cases are executable contract inputs rather than evidence that an implementation passes.
