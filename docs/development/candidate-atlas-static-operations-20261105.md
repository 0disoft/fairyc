# Fairy Atlas static-operation source family, 2026-11-05

Scope: carry the already admitted single-operation interface family through the
actual Fairy Complete/Atlas producer, current-source admission, workspace joins
and commands. This does not add multi-operation interfaces, dynamic dispatch or
new language syntax. The full implementation queue remains active. Native Linux
execution stays batched; no push, dispatch, release or deployment is authorized.

## Sequential implementation

1. **Independent Go prerequisite implemented.** Five two-package v3 fixtures
   retain seven original member declarations: the public generic operation and
   algorithm, three scalar providers, an ordinary export and an ordinary private
   function. Used sources have exactly four concrete implementations; an unused
   source has none. Root calls bind exact cross-package algorithm implementations;
   each algorithm binds its matching operation, and each operation calls the exact
   ordinary i32/i64 provider. Canonical codec roundtrips retain all these facts.
   Selected and unused provider body edits change the generic interface and
   specialization identities. An unrelated same-width private body edit does not.
   All five preserve original declaration/spans/symbol facts after normalizing
   only the exact-snapshot package instance, which legitimately changes with source
   bytes. Type identity uses binder index `0`, not the display name `T`.
2. **Actual Fairy production verified.** The existing generic producer accepts
   all five independent sources in Windows low/optimized (165.744s), preserving
   exact original headers, interfaces, symbol identities, concrete tuples and
   provider call endpoints. Strict reader/current-source admission and workspace
   joining run inside the actual Fairy probe. All twenty returned package maps
   also pass independent Go v3 decoding/comparisons. Exhausted work and missing
   coordinates reject in each profile. No product change was necessary; the
   existing source/identity/call owners already retain this family. See the
   [production receipt](../ops/windows-candidate-atlas-static-operations-20261105.md).
3. **Impact, MCP and ordinary body editing verified.** The actual impact command passes
   Windows low/optimized in 175.514s: eight visible/full original selectors,
   exact six-call admission, one-short call/package/output refusals and stale-lock
   rejection followed by independently rechecked provider-edit recovery.
   Actual read-only MCP passes both profiles in 100.776s, including negotiation,
   read-only tool policy, exact Go query contexts, stale-source error and recovery
   in the same real stdio process. Full authored-main registration is updated but
   its new groups are not replayed. Actual ordinary root-body preview/apply passes
   both profiles in 109.395s, including an independently checked overlay that
   removes i64 implementations and introduces two bool implementations. Exact Go
   reports/published bytes, invalid type/dependency-write refusals, unchanged lock,
   staging cleanup and stale lock/hash checks pass. Editing interface/provider/public
   generic declarations remains governed by the existing explicit editor scope.
   All three verifiers are registered; run the complete authored-main groups.
4. Keep broader Atlas source forms, complete compiler comparison, consecutive
   self-build, current Windows distribution and clean-environment acceptance,
   batched native Linux and same-commit hosted CI as separate final gates.

## Validation accounting

Owner: `internal/cli/atlas_static_operation_oracle_test.go`,
`TestAtlasStaticOperationSpecializationOracle`.

The initial new test did not compile because it used a nonexistent constant name;
after correction it exposed incorrect test expectations for generic binder display
names. The model owns numeric binder identity; these were harness errors, not
product defects. Exact-snapshot instance differences are normalized explicitly,
without dropping original declaration/span/symbol comparisons.

```powershell
go test -mod=readonly -count=1 -timeout 2m -v ./internal/cli -run '^TestAtlasStaticOperationSpecializationOracle$'
go vet -mod=readonly ./internal/cli
gofmt -l internal/cli/atlas_static_operation_oracle_test.go
git diff --check
```

The final Windows Go replay passes all five fixtures without skips in 0.563s.
Scoped vet, Go formatting and diff checks pass. Logs:
`.cache/atlas-static-operation-oracle-final-20261105.log` and
`.cache/atlas-static-operation-oracle-vet-20261105.log`.

No product source, source grammar, public CLI, SDK/runtime ABI, map wire, DB,
dependency, repository hygiene, timeout, target or runner change is introduced by
the oracle prerequisite. Go evidence does not attest Fairy/native production.
