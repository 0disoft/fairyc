# Local package preparation receipt — 2026-11-14

Implemented stable exact/caret/tilde resolution, strict detached Ed25519 catalog
verification with explicit key/sequence/expiry inputs, exact source-plan output,
read-only update proposals, and bounded local immutable Git snapshots. Connected
three Go Stage 0 experimental CLI commands. No Fairy candidate CLI promotion,
network service, provider, credential, SDK version or lock schema was invented.

Checks actually run:

- Full `internal/packagesource`: 4.834 s; full `internal/cstdlibcontract`: 0.029 s.
  `.cache/package-and-fs-final-20261114.log`.
- Range/trust/resolve/update checks include signature mismatch, strict duplicate
  JSON, unknown key, expiry, rollback floor, canonical version boundaries,
  same-version repin/downgrade refusal and unchanged input plans.
- Real temporary local Git repositories verify committed bytes over dirty files,
  ignored export attributes, deterministic ZIPs, ambient Git override isolation,
  symlink/gitlink refusal, immutable IDs and cancellation. The pipe-copy bound
  also checks that `io.ReaderFrom` cannot bypass the checked writer.
- New Go CLI built successfully. Seven actual CLI processes cover resolve,
  update, signature refusal, snapshot creation, existing output refusal, mutable
  Git reference refusal, and generated ZIP consumption by `fetch-packages
  --offline`. No network request, desktop or keyboard interaction.
- Final focused CLI batch: 1.615 s including old offline-fetch checks and output
  publication conflict/temporary cleanup. Process test: 1.20 s.
  `.cache/package-cli-final-20261114.log`.

Built executable used for replay:
`.cache/fairyc-package-prep-20261114.exe`, SHA-256
`b389779389a2a5a6ce673c2daf0915afaacb40a553bbc9ea4da7bf9cd91d451f`.
Its product sources match commit `3b3647d0`; the executable was built before
that local commit, so its embedded Git build metadata is not a clean HEAD
rebuild claim. The later offline-replay test addition changes no product bytes.
Installed Git: `2.56.1.windows.1`. The disabled lazy-fetch/replacement settings
were checked against the [official Git environment reference](https://git-scm.com/docs/git).

Inputs remain read-only for registry operations; the test confirms byte-identical
manifest/source-plan/lock/trust files after both successful commands. Git ZIP
output uses atomic hard-link publication without replacement, independently
checked against the JSON digest and existing offline source preparation.

See `../package/local-registry-preparation-v1.md` for the local contracts and
explicit remaining remote-service/candidate/automatic-update scope.
