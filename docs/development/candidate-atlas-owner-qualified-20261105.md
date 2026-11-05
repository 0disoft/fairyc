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

Actual owner-qualified impact/MCP/body-edit commands, new-image full CLI,
consecutive self-build, generated-consumer execution, clean-machine distribution,
same-commit CI and native Linux remain separate. Enum owners, generic methods and
broader expression/closure combinations are not established by this matrix.
No stable CLI flag, SDK/runtime ABI, map wire version, dependency, DB, runner or
repository policy is changed. The full implementation objective remains active.
