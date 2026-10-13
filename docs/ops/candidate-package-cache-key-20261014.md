# Candidate whole-package cache key v4

The Fairy key encoder retains all eight required environment strings, profile
flags, runtime/SDK identity, package/source/interface/state-body digests, sorted
dependency coordinate/interface pairs and features. Its canonical bytes and
`fycache4-sha256-` text match the existing Go packagecache v4 encoder. No missing
state-body identity is replaced by a placeholder. Inputs and results are owned;
failed validation or work exhaustion returns no encoded prefix.

Twenty Go-generated cases match complete bytes and hashes in Go-native execution
and both candidate C development/release images. They separately change each
environment member and digest, profile flags, features, dependency coordinates
and interfaces, and cover empty sets plus embedded NUL and Korean UTF-8. Four
authored tests also reject missing environment/digest facts, malformed UTF-8,
duplicate/reversed sets and dependency coordinates, preserve a detached result,
and verify exact-fit/exhausted/invalid 16MiB phase work. Private API images use
explicit 16MiB stacks. Root and locked checks pass 452 files / 5848 MIR functions.

The maintained Go oracle reads authored environment literals and regenerates the
binary fixture with `tools/run-go.ts run tools/verify-candidate-cache-key.go
--output <fixture>`. The private test verifier forwards that fixture to every
execution. Earlier missing resource constructors and one-byte-short key text
checks failed before correction; no failed run is counted as a pass.

Receipt: [evidence](evidence/candidate-package-cache-key-20261014.json).
Next: generic cache keys and exact complete-owner facts, followed by independently
verified reusable package artifacts and actual cache lookup/publication. This
encoder accepts caller-supplied facts and does not itself authorize a cache hit.
Current full fixed point, product defaults/performance, Linux/full suite and
same-head CI remain open. Public CLI/SDK ABI and product stack/timeouts are
unchanged; no push.
