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
now passes these groups and the refusal/help controls at both profiles below.
Historical closure/pre-closure receipts cannot attest the other groups on this
changed image.

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

This Go prerequisite does not itself attest the additional same-leaf Fairy
source family. Its subsequent production proof is recorded below; actual
command/editor and generated-consumer replay remain separate.
The CLI qualified-name filter is now verified separately below.
The existing native full-image replay uses
the unchanged fourteen-family Fairy sources; none of its evidence is broadened
to the two new namespace fixtures.

## Go exact qualified-name CLI filter

The internal exact-name query admitted `Cell::get`, but the CLI still rejected
every qualified name before loading a workspace. `atlas-symbols list --name`
now validates the same nonempty `::`-separated Stage 0 declaration path, retaining
the whole-value 512-byte bound. ADR 0201's qualified-name amendment and Symbol
Catalog v1/CLI contracts record this accepted input domain; canonical envelope,
checksum domains, bare-name behavior and versioned cursor policy are unchanged.

All fifteen combinations of the three catalog modes and exact owner/missing/
bare/case-mismatched names pass. Thirteen parser controls include the 512/513-byte
boundary, empty segments, malformed separators, digit starts, NUL and newline.
Existing symbol CLI pagination/filter/budget/output regressions also pass;
the combined command takes 1.631s. Scoped CLI vet, format and diff checks pass.
Log: `.cache/atlas-symbols-qualified-cli-final-20261106.log`.

This is the Go experimental CLI filter. The Fairy `atlas-symbols` dispatcher is
still a separate missing command; no Fairy CLI help/source or image changes here.

## Go ordinary-body selection beside same-leaf methods

With both owner namespaces admitted, editing the ordinary `get` beside
`Cell::get` and `Other::get` reproduced `ambiguous source function`. The AST
selector compared leaf names before applying its existing no-owner policy.
It now selects only ownerless functions before the uniqueness check. Method
editing remains refused; no editor capability or publication policy expands.

The new CLI regression previews and applies only the ordinary body, compares
the exact source/report, refuses both method subjects even with `--apply`, and
requires every other byte, graph lock and staging entry to remain unchanged.
All existing Atlas edit tests, including native execution, and the namespace/
owner-edit oracles pass together in 7.980s. The complete sourceedit package
passes in 1.638s; scoped vet, format and diff checks pass. Reproduction/final
logs: `.cache/atlas-owner-qualified-same-leaf-edit-before-20261106.log` and
`-after-20261106.log`. This is Go editor evidence; the separate Fairy production
proof below does not attest actual Fairy editing.

## Fairy distinct-owner namespace production

`TestCandidateAtlasOwnerQualifiedNamespaceProduction` reuses the original
Complete-bound producer verifier with two independent Go v3 workspaces:
`Cell::get`/`Other::get`, and both methods beside ordinary `get`. The original
fourteen-case test and all of its controls remain registered unchanged.

Both namespace sources pass Windows low/optimized in 84.432s. Eight returned
package maps agree with independently canonicalized Go declarations, owner and
receiver types, source spans/interfaces, distinct SymbolIds and two/three exact
calls. Current-source admission and complete joins pass; workspaces do not
change and the native processes have an empty PATH. The existing Fairy
implementation already handles this source family; no compiler or runtime
change is required by this proof. Scoped CLI vet, formatting and diff checks
pass. Log: `.cache/atlas-owner-qualified-namespace-production-20261106.log`.

Actual same-leaf impact/MCP/editor commands and their full authored-main replay
are recorded below. Generated-consumer execution remains separate. The original
fourteen-family full-CLI receipt is unchanged.

## Full authored-main acceptance

The actual authored dispatcher, without a companion, passes the three new
owner-qualified groups and `unsupported`/`help` under both profiles in 817.462s.
The five selected groups pass without skips; the other twenty-four of the
twenty-nine registered groups are not attested for this image. This is neither
full 29/29 acceptance nor new same-leaf namespace-family evidence.

| Group | Low | Optimized |
| --- | ---: | ---: |
| unsupported | 0.06s | 0.05s |
| help | 0.17s | 0.19s |
| impact-owner-qualified | 487.26s | 8.43s |
| mcp-owner-qualified | 139.18s | 2.69s |
| edit-owner-qualified | 67.38s | 1.36s |

Reproduction (the cache directory is test-only):

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path $PWD '.cache/atlas-owner-qualified-cli-20261106'
go test -mod=readonly -count=1 -timeout 16m -v ./internal/cli -run '^TestCandidateAtlasCLI$/(low|optimized)$/^(impact-owner-qualified|mcp-owner-qualified|edit-owner-qualified|unsupported|help)$'
```

Log: `.cache/atlas-owner-qualified-full-cli-20261106.log`.
The C input has a 1,982,488-byte header, 153,115-byte runtime and 24 body units;
the largest unit is 2,097,059 bytes, within the unchanged 2 MiB consumer bound.
Images retain the existing 16 MiB stack reserve and profile/toolchain/C-input
and executable digest verification. Later Go namespace/filter/editor fixes do
not change these Fairy/runtime sources; the accepted fourteen-family scope is
unchanged and does not absorb those separate Go-only namespace fixtures.

| Profile | Source/profile/toolchain digest | Image SHA-256 |
| --- | --- | --- |
| low | `82e2b663aee139e21ee4989260628bddd8f360e2189a1aa71b26d1c958078988` | `867662ac5ef14e697deafb5938b6b9f5744555b3f130ac3409be9686f2afc599` |
| optimized | `b6b4ee70d71a49d7762e792665e1567968107b32fc9b8139f7b9e3867ad03b7b` | `443842c2cdeab427eaeb5d0dc46bac9da4f89a5885ea9f20d17984d4f75c8808` |

The other full-CLI groups, consecutive self-build, generated-consumer execution,
clean-machine distribution, same-commit CI and native Linux remain separate.
Enum owners, generic methods and
broader expression/closure combinations are not established by this matrix.
No stable CLI flag, SDK/runtime ABI, map wire version, dependency, DB, runner or
repository policy is changed. The full implementation objective remains active.

## Distinct-owner namespace command acceptance

`candidate_atlas_owner_namespace_command_test.go` appends three groups to the
existing full authored-main verifier. The two source variants independently
retain `Cell::get` and `Other::get`, with an optional ordinary `get`. The
two-owner variant reverses method declaration order. Impact compares visible
and full selectors against the independent current Go v3 call index, admits the
exact two/three-call budget and rejects one-short budgets without output. Read
queries preserve every workspace byte. Real stdio MCP resolves each namespace;
ordinary `get` preview/apply succeeds while direct owner-method edits reject.

```powershell
$env:FAIRYC_RUN_PARITY_INTEGRATION='1'
$env:FAIRYC_ATLAS_CLI_ARTIFACTS=Join-Path $PWD '.cache/atlas-owner-qualified-cli-20261106'
go test -mod=readonly -count=1 -timeout 10m -v ./internal/cli -run '^TestCandidateAtlasCLI$/(low|optimized)$/^(impact-owner-namespaces|mcp-owner-namespaces|edit-owner-namespaces)$'
```

The command passes in 296.675s with no selected skips. Both profiles reuse the
exact verified C-input/profile/toolchain/image digests in the preceding table;
no compiler/runtime source changed and no fresh native compilation is claimed.

| Group | Low | Optimized |
| --- | ---: | ---: |
| impact-owner-namespaces | 141.99s | 2.11s |
| mcp-owner-namespaces | 57.04s | 0.89s |
| edit-owner-namespaces | 72.86s | 1.30s |

Log: `.cache/atlas-owner-qualified-namespace-full-cli-20261106.log`.
Together with the earlier five groups, this attests eight of the now thirty-two
registered groups on these exact images. The other twenty-four groups,
generated consumers, consecutive self-build, clean-machine distribution,
same-commit hosted CI and native Linux remain separate. Fairy `atlas-symbols`,
`atlas-calls` and `atlas-pack` dispatchers remain unimplemented.
