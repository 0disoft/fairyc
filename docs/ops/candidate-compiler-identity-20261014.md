# Exact running compiler image identity

`project_compiler_image_hash::read_path` streams 32KiB chunks and computes the
production Go `fairyc.compiler-executable-sha256/v1` identity, without a label
or version fallback. It retains the SHA stream's 96MiB image bound and charges
blocks, buffer/output and read overhead to an independent 16MiB ledger. Empty,
non-disk, oversize or changed-length observations and path/read/close/allocation/
work failures publish no identity. `project_compiler_identity::current` uses the
existing Windows module-path adapter, never argv[0] or a companion executable.

Go `buildidentity.Identity.String`/`Parse` and SHA-256 supply eleven file vectors
covering padding, 32KiB boundaries, binary bytes and a Korean copy path. Three
file tests pass Go-native and candidate C development/release at default stacks.
One current-image test passes both C profiles with normal and false argv[0]
launches: four executions match independent Python hashes of complete PE bytes.
Go-native current-image execution is skipped for private Windows adapter linkage;
Linux remains unverified. Root and locked checks pass 460 files / 5921 MIR functions.

The verifier's opt-in `--stdout-executable-identity` leaves its default output
comparison unchanged, exercised by the file tests. SDK/target/cache environment,
complete-owner key composition and verified artifact reuse remain next. Go's
verified-MIR cache explicitly uses `not-applicable:verified-mir` for toolchains;
native object dependencies are separate. Full fixed point, product defaults/
performance and platform/CI gates stay open. Public CLI/SDK ABI, stack and
timeouts are unchanged; no push.

Receipt: [evidence](evidence/candidate-compiler-identity-20261014.json).
