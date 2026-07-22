# ADR 0156: Reuse process-local package MIR only after complete verified hits

- Status: Accepted
- Date: 2026-07-22
- Coordinator source: `internal/workspacecompiler/`
- Cache source: `internal/packageartifact/`, `internal/packagecache/`

## Context

ADR 0155 made verified package artifacts retrievable from a bounded process cache, but no compiler
coordinator consulted it. A usable integration must compute keys from current semantic facts,
distinguish partial storage hits from executable reuse, and preserve ordinary compilation as the
fallback. The v0 package artifact also duplicates exact workspace-wide type context, so individually
valid artifacts from different compilation generations may be incompatible with each other.

## Decision

1. Split the package compiler boundary into `compiler.AnalyzePackages` and
   `compiler.LowerAnalyzed`. Existing `compiler.CheckPackages` still composes both operations and
   preserves its behavior.
2. `workspaceloader.SemanticPackages` exposes the exact build-local package number, resolved
   instance, source digest, feature set, role, runtime identity, and sorted direct dependencies in
   the same order as `SemanticInput`.
3. `workspacecompiler.Session` owns one bounded `packageartifact.Cache`. It performs workspace
   conversion and semantic analysis on every request, extracts every public-interface hash from
   typed HIR, and computes package-cache v1 keys from the process identity, language and MIR
   revisions, target schema and ABI profile, runtime ABI, bundled standard-library identity,
   package source and instance identity, direct dependency interfaces, and features.
4. Because this cache stores verified MIR rather than native objects, its backend identity is
   `verified-mir` and its native toolchain identity is explicitly
   `not-applicable:process-local-mir-v0`. It does not pretend to fingerprint a compiler that never
   consumes the artifact.
5. MIR lowering is bypassed only when every function-owning package lookup returns a verified hit
   and `packageartifact.Reassemble` accepts the complete set. A miss, rejected payload, partial hit,
   or incompatible mixed-generation context runs `compiler.LowerAnalyzed`, extracts all current
   package artifacts, and attempts to refresh them.
6. A cache record or codec size limit skips that admission without failing a successful compile.
   Other fresh-artifact encoding or verification errors remain compiler errors rather than being
   hidden as cache policy.

## Consequences

- a raw LRU hit or individually valid partial set can never bypass MIR lowering;
- semantic diagnostics preserve the existing no-MIR result and are not cache eligible;
- source changes invalidate the affected whole-package keys, and exact dependency instances make
  dependent invalidation conservative;
- exact-context disagreement is a safe fallback and the following successful compile heals the
  current keys;
- the deterministic two-package test executes cold and warm results as `42` and proves the lowering
  hook is called once across those two requests;
- there is still no claim that elapsed builds are faster, because no calibrated timing comparison
  exists;
- ordinary one-shot CLI commands create no long-lived Session, so separate invocations do not share
  this cache; ADR 0159 later exposes one explicit bounded foreground session for repeated workspace
  checks;
- persistence, remote reuse, stable type-context encoding, default cache limits, and cross-process
  compatibility remain deferred.

## Rejected alternatives

- Reusing a partial hit set was rejected because the missing bodies still require whole-program
  lowering and current v0 contexts can belong to different generations.
- Treating reassembly failure as a hard build failure was rejected because ordinary compilation is
  the authoritative recovery path.
- Wiring a global cache into one-shot CLI commands was rejected because process exit would erase it
  before a second request and create fake integration without reuse.
- Dropping target, runtime, or standard-library compatibility fields was rejected even though the
  first MIR subset is conservative; complete keys are cheaper than silent future miscompilation.
