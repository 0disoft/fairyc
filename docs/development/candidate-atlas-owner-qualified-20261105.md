# Owner-qualified Fairy Atlas declarations, 2026-11-05

Scope: bring the existing Go owner-qualified function source family into Fairy
Complete-bound Atlas production, current-source admission and impact/MCP/body-edit
commands. This is one slice of `implementation-queue-20261105.md`, not completion
of that objective. Native Linux remains batched; no push or dispatch.

## Required facts and sequence

1. Go snapshot/codec/query prerequisite: preserve the written declaration path
   `Cell::get` exactly once, the nominal owner with its defining module/package
   coordinate, the exact receiver form and ordered parameters including `this`.
   Local and imported call endpoints must resolve the same declaration.
2. Fairy Complete prerequisite: extend signature, body-call and public interface
   owners for the original eight and six added source cases below. The former
   signature check admitted borrowed receivers only; zero-parameter associated
   functions, consuming receivers and public owner-qualified interface extraction
   now have separate positive/negative evidence below.
3. Fairy Atlas production: retain those exact checked facts, qualified keys and
   canonical SymbolId v1 identities through production, reader/current validation
   and complete workspace joins. Preserve existing depth/work/admission bounds.
4. Actual impact/MCP/body editing and a fresh full authored-main replay. Keep all
   existing groups and independent current-source Go comparisons. Historical
   closure/pre-closure image results cannot attest a changed image.

The source matrix is associated/no parameter, associated/one parameter, shared
receiver, exclusive receiver, consuming record, consuming resource, imported
shared receiver and unused associated function, extended with the six cases below.
Generic owner methods and first-class closures are separate expansions.

## Go prerequisite evidence

`TestAtlasOwnerQualifiedSourceOracle` reproduces the source family against the
normal checked workspace pipeline, canonical v3 encode/decode, complete joins and
exact qualified-name catalog lookup. It independently checks owner path/package,
receiver, receiver-parameter retention and expected resolved call count; unused
associated functions retain declarations without inventing calls.

Two production problems were reproduced: snapshot construction doubled the owner
prefix (`Cell::Cell::get`), and the codec rejected contract-valid qualified keys.
The codec and query now reuse the existing nonempty Stage 0 path validator from
the SymbolId v1 contract, without changing its identity encoding or map version.
`not::an_identifier` is a valid path, so its old negative query control is replaced
with `not:::an_identifier`; a positive empty-result control preserves the valid
unknown-path behavior. Eleven malformed paths still reject at local/call keys.

Windows checks:

- Full `internal/semanticsnapshot` package: pass, 0.209s.
- Full `internal/atlasmap` package: pass, 2.022s. The initial run preserved the
  obsolete exact-name negative-control failure before that control was corrected.
- Eight source oracles: pass, 0.546s package time, including shared-method docs.
- Scoped Go vet: pass for snapshot, map and CLI packages.
- Formatting and `git diff --check`: pass.

Logs are local `.cache/atlas-owner-qualified-go-before-20261105.log`,
`.cache/atlas-owner-qualified-go-final-20261105.log` and
`.cache/atlas-owner-qualified-oracle-final-20261105.log`.

That prerequisite alone does not prove Fairy production or command support.

## Fairy implementation

Signature checking now retains a nominal struct/resource owner independently of
receiver presence. Associated functions accept zero parameters or ordinary
parameters; only a first parameter named `this` selects shared/exclusive/value
receiver semantics. A later or mismatched `this` still rejects. Public interface
extraction preserves the same qualified function name and stable owner spelling
as Go, including imported public methods.

Method lookup before the value phase can derive a nominal owner from explicit
types, checked constructor identities, earlier lexical aliases, grouping,
shared/exclusive borrow initializers and ordinary function result signatures.
The bounded helper follows only exact current source/name bindings. It confers
no value type, move, loan or mutability permission; all existing checks still run.
Associated functions without `this` cannot be invoked as dot methods.

Atlas sorting, declaration/function/call keys, owner TypeRefs, receiver modes and
the first parameter now agree on the written qualified path. Ordinary names keep
their borrowed-name path without extra name-buffer allocation. Owner lookup is
charged before source projection and public-interface extraction. The ordinary
reader admits the contract-valid qualified path; field/parameter identifiers
retain their stricter single-identifier checks.

The expanded matrix retains the original eight cases and adds an associated
borrow parameter, an explicitly typed receiver, alias/grouped receiver bindings
and record/resource consuming dot calls. The full-source Go oracle passes all
fourteen cases (0.857s before the later guard additions).

An independent Complete-admission probe also exposed a real implicit-receiver
move omission: `discard token.take(); return token.take();` was admitted. Dot
syntax omits its receiver from the explicit AST argument list. Move and origin
analysis now include it first, retaining consumed temporaries and holding its
loan while evaluating the remaining arguments. Reuse now rejects with 1101.
The generic-owner negative source already refuses at the earlier ordinary-source
gate with code 9; changing a test's incorrect 4501 expectation does not open it.

Windows evidence on 2026-11-06:

- Complete admission: thirteen positive sources and six negative controls, both
  native compiler profiles; pass in 60.503s. Controls cover mismatched/later
  receivers, the existing generic-owner refusal, immutable exclusive borrow,
  use after consuming a receiver and treating an associated function as a method.
- Authored source `check` and `check --locked`: pass, 643 source files and 8,110
  MIR functions; the lock is refreshed after the final code changes.
- Scoped Go vet, formatting and diff checks: pass.
- Final fourteen-case production/current/join comparison: pass at both profiles
  in 190.10s. Qualified declarations, owners, receivers, ordered parameters,
  original/public interfaces and resolved calls match independently rebuilt Go v3
  maps. The shared source also rejects exhausted work and missing coordinates;
  every workspace remains unchanged.
- Combined ordinary/generic/static-operation/copied-closure/owner-qualified
  production regression: pass in 629.968s, with no selected skip. Individual
  groups take 77.01s, 110.40s, 119.36s, 132.98s and 190.10s respectively.
- Final Go source oracle: all fourteen cases pass. The unsupported-source fixture
  is now a valid enum-owned associated function, beyond this bounded struct/resource
  owner family. Its independent Go source-validity test passes; the two Go roots
  take 0.876s together. Actual Fairy command refusal remains a command gate.

The initial native replay failed all original eight cases at both profiles
(73.99s). Two authored-code compile errors, the three inferred-receiver failures
and the receiver-move/control failures remain separate unsuccessful attempts in
local `.cache/atlas-owner-qualified-*` logs. The earlier 180.75s producer pass is
not final evidence after the subsequent move/origin and guard changes.

Final producer/regression log:
`.cache/atlas-owner-qualified-production-regression-20261106.log`.
Final Go source controls:
`.cache/atlas-owner-qualified-go-source-final-20261106.log`.

## Windows command acceptance

Actual command-leaf verifiers retain independent freshly checked Go v3 contexts
and run the Fairy executables with an empty PATH. They cannot delegate a query
or edit to a Go child or build companion.

- `TestCandidateAtlasOwnerQualifiedCommand`: all fourteen source families pass
  low/optimized in 368.12s. Each exact visible/full qualified subject agrees with
  Go, with explicit package/output cutoffs and unchanged workspace bytes. The
  shared-method body edit preserves exact SymbolIds/interface identity, changes
  source/instance identities, rejects the stale lock and succeeds after relocking.
  The valid enum-owner source remains a refused command input.
- `TestCandidateEditOwnerQualifiedCommand`: both profiles pass in 108.84s.
  An ordinary root body changes from shared method calls to associated, exclusive
  and consuming method calls; independent Go maps require three original and
  five edited calls. Exact preview/apply reports, published source bytes,
  unchanged lock and absence of staging/lock debris match Go. A replacement that
  consumes a Token twice rejects. Root-owned `Cell::get` is still noneditable;
  owner-qualified source admission does not widen the existing edit policy.
  Old lock/source hashes reject after publication and relocking respectively.
- `TestCandidateMCPOwnerQualifiedCommand`: six source families pass both profiles
  in 164.79s: associated, shared, exclusive, record/resource consuming dot calls
  and an imported public method. Real stdio sessions require protocol negotiation,
  the read-only tool catalog, exact Go impact contexts, stale-source refusal and
  recovery in the same session. Source/host bytes do not leak into responses and
  queries do not write the workspace.

The combined three-root command run passes in 641.864s without selected skips.

The three verifiers are appended to `TestCandidateAtlasCLI` as
`impact-owner-qualified`, `mcp-owner-qualified` and `edit-owner-qualified`, without
removing or reordering any existing group. Fresh full authored-main acceptance
has not run for these groups or this changed image. Historical closure/pre-closure
receipts cannot attest it.

Command log: `.cache/atlas-owner-qualified-commands-20261106.log`.
Independent edit oracle: `.cache/atlas-owner-qualified-edit-oracle-20261106.log`
(pass, 0.266s). Scoped vet, formatting and diff checks pass.

## Go distinct-owner namespace follow-up

A separate current-source regression reproduces E4501 in v3 preparation for
`Cell::get` and `Other::get`, with or without an ordinary `get`. Normal `check`
accepts both sources. Generic preparation's module-name collision table discarded
the owner and incorrectly merged these distinct functions. The table now uses
the written `Owner::name` key while preserving original AST owner/name fields.
Bare-name collisions and reserved private prefixes still reject; generic methods
remain closed.

`TestAtlasOwnerQualifiedDistinctNamespaces` now retains both exact method IDs,
the optional ordinary ID and two/three resolved calls through complete v3 joins.
Six isolated preparation controls cover accepted namespaces, duplicate ordinary
and same-owner declarations, generic owners and reserved generated names, with
unchanged caller ASTs. Full `genericsource` and `genericplan` packages pass
(0.412s/0.068s); the two namespace cases, fourteen original source oracles and
edit oracle pass together (1.392s). Scoped vet and diff checks pass.
Logs: `.cache/atlas-owner-qualified-namespace-before-20261106.log`,
`-after-20261106.log` and `-packages-20261106.log`.

This Go prerequisite does not attest the additional same-leaf Fairy source
family, same-leaf ordinary-body editing or CLI qualified-name filtering.
Those remain separate follow-ups. The existing native full-image replay uses
the unchanged fourteen-family Fairy sources; none of its evidence is broadened
to the two new namespace fixtures.

New-image full CLI, consecutive self-build, generated-consumer execution,
clean-machine distribution, same-commit CI and native Linux remain separate.
Enum owners, generic methods and
broader expression/closure combinations are not established by this matrix.
No stable CLI flag, SDK/runtime ABI, map wire version, dependency, DB, runner or
repository policy is changed. The full implementation objective remains active.
