# Candidate source discovery

The owning boundary remains `docs/package/workspace-loader-v1.md` and
`internal/workspaceloader/loader.go`. The candidate APIs are private and do not
replace the public project loader or its package/module checks.

## Directory resolution

`project_directory.resolve` walks each decoded portable path component through
actual `std::fs::ReadDir` observations. It requires exactly one Unicode-folded
match with the exact on-disk byte spelling, rejects observed reparse entries and
non-directories, and retains the OS error code and relative-component span.
Directory handles close on success and are owned by cleanup on early error.
The caller's root spelling anchors the returned owned path; no absolute-path
canonicalization or filesystem snapshot is claimed by this leaf.

`project_path` implements UTF-8 decoding/encoding and the separate Go lower-case
and EqualFold rules in Fairy. The metadata comes from the installed Go reference
(Unicode 17.0.0): 191 lower ranges and 30 fold exceptions, including singleton
SimpleFold cycles such as dotted I. `tools/generate-candidate-path-unicode.py`
regenerates the balanced lower lookup from
`spec/fixtures/workspace-loader/candidate-path-unicode-v1.json`. The reference
helper uses Go only for data/oracle generation; runtime comparison is Fairy.

The C backend now admits the four already source-authenticated ReadDir raw
operations, verifies its usize/u8 descriptor and six-field entry result, and
uses the directory adapter's real C result tags. The existing source, lexical
unsafe, signature and layout authority checks remain. A fresh candidate compiler
was built from current Fairy sources before these C execution checks.

Native Unicode 2 and directory 1 tests and both candidate C profiles pass.
The Unicode checks cover full-range output checksums plus UTF-8, Kelvin sign,
long s, sigma, supplementary-plane case, dotted I and malformed input. The real
filesystem tests cover exact/mismatched case, absent/non-directory entries,
Unicode-equivalent non-exact spelling, a junction, a reparse prefix and exhausted
work. Seven source-root acceptance observations from the Go workspace loader
are recorded; its outer error category is invalid-source-root, while this leaf
retains more specific private component errors. Evidence:
`evidence/candidate-path-unicode-20261011.json` and
`evidence/candidate-directory-20261011.json`.

## Manifest source binding

`project_source_root.load` validates supplied exact manifest bytes before any
directory observation, resolves the decoded `package.source`, and returns one
owner containing the typed manifest and actual source-root path. These values
survive release or mutation of the caller's input buffer. It retains manifest
errors as phase 1 with document spans and directory errors as phase 2 with
decoded source-component spans, field 7 and the original OS code.

Two native tests and both candidate C profiles pass, including escaped TOML
source spelling, input-buffer release, mismatched/nested case, absent/file
entries, Unicode folding, junction rejection, invalid manifests and exhausted
directory work. Evidence: `evidence/candidate-source-root-20261011.json`.
The compiler image from the preceding directory unit was reused: this unit
changes the private loaded program and uses the same compiler capabilities.
Full production check passes: 306 files and 3909 MIR functions.

## Recursive source paths

`project_source_catalog.collect` walks actual directory entries with one open
stream at a time and an explicit directory queue. It owns exact relative UTF-8
paths, returns them in byte order, includes only regular files with an exact
`.fy` extension and rejects every observed reparse entry, including otherwise
ignored names. Directories are traversed regardless of their extension.

Separate merge sorts detect collisions under Unicode lower-case rules and
produce deterministic exact-path order. An explicit work meter bounds path
storage, traversal and comparisons; there is no all-pairs collision scan.
The caller's cumulative 4096-file counter changes only on success. Errors own
their observed path and OS code. Entry sizes are observations only; subsequent
reads must enforce exact-byte budgets and recheck file identity/type.

One native test and both candidate C profiles pass. Real filesystem cases
cover nested sorted paths, ignored `.FY` and text files, input-path release,
dotted-I/i lower-case collision, empty roots, junctions, non-directory roots,
cumulative file bounds, exhausted work and malformed UTF-8. Full production
check passes: 307 files and 3936 MIR functions. The test-only fixture generator
and Go reference helper are included in the input identities. Evidence:
`evidence/candidate-source-catalog-20261011.json`.

The Go reference agrees on the ordinary sorted source paths, empty-root
rejection and Unicode collision. It accepts the two trees containing Windows
junctions, which its WalkDir callback ignores; the candidate rejects them by
the SDK's reparse flag. This discrepancy remains a compatibility item and these
observations do not establish full Go-loader equivalence. No Go loader behavior
was changed by this candidate unit.

## Exact source-file bytes

`project_source_file.read` resolves directory prefixes and observes the exact
leaf spelling, reparse flag and regular-file kind. It opens the file, checks
disk kind and its native identifier, repeats the path observation, and compares
the identifier from a second open. It then retains the exact bytes from the
first handle, with explicit close on success and owned cleanup on failure.
These bounded observations do not create an atomic tree snapshot.

Reads stream through a 4096-byte buffer with a one-byte overflow probe. The
per-file read bound is the Go frontend's 8 MiB and the caller supplies its
remaining workspace byte budget. Empty files are retained, as are NUL and
malformed UTF-8 bytes; syntax validation is a later stage. This does not raise
the candidate parser's distinct token, cooked-value, AST or joined-owner caps.

The private C bridge now validates and copies the authenticated file-kind
result and all six fields of the mutable native file-ID output. One native test
and both candidate C profiles pass, including distinct-file versus repeated
same-file IDs, exact bytes, 65 KiB and exact 8 MiB reads, oversize/remaining-byte
rejection, case/nonregular/reparse/prefix errors and exhausted work. The candidate
compiler was rebuilt from current Fairy sources. Evidence:
`evidence/candidate-source-file-20261011.json`.

The first release compile hit the old 60-second deadline. The final execution
uses the committed real private deadline configuration from
`candidate-compile-timeout.md`, without a timeout overlay.

## Retained source sets

`project_source_set.load` composes manifest validation, source-root resolution,
recursive path discovery and exact file reads into one owned source set. It
retains the typed manifest, source-root path, sorted relative file paths and
exact bytes. File and byte counters are cumulative across calls and commit only
after all files have loaded: 4096 files and 64 MiB of source bytes. The manifest
is outside the source byte counter, as in the Go loader.

`load_manifest` also reads the real root `fairy.toml` through the exact spelling,
reparse and file-ID observation boundary, with a 1 MiB manifest input cap.
Errors retain their leaf phase/reason, OS code, relevant span and owned path.
Manifest-validation spans are document-relative; source-root spans refer to
decoded `package.source`; file-read spans refer to the retained relative path.
Caller root spelling remains an explicit anchor, without whole-workspace root
canonicalization or an atomic filesystem snapshot.

One native test and both candidate C profiles pass, including exact-limit
aggregate success, a later-file byte failure with unchanged caller counters,
file/collision/reparse/empty-tree failures, input manifest release and actual
manifest-file loading. Empty fallback path/byte borrows stay rooted in their
source owner. The ordinary sorted paths and all retained bytes agree with the
Go loader reference observation. Full production check passes: 309 files and
3958 MIR functions. Evidence: `evidence/candidate-source-set-20261011.json`.

The candidate image from the source-file unit was reused; its compiler/backend
implementation is unchanged and this unit's program was emitted from fresh
hashed source inputs. No module binding or package-graph success is claimed by
the byte-retention layer.

## Source-module binding

`project_source_modules.bind` consumes the retained source set and parses every
file through the actual Fairy scanner/parser. Each file must declare exactly
one module and that declaration must match its exact source-root-relative path.
The ASCII identifier rule comes from `internal/source/source.go` and the language
source-file mapping; physical path discovery itself keeps UTF-8 names.
`project_module_path.expected` operates only on canonical catalog paths and does
not implement the Go helper's general path-cleaning API.

Success owns the source set, independent parser results and a sorted module
index. Binary lookup returns the original file index, preserving file ordering
when module-name ordering differs. A defensive sorted duplicate check remains;
two ordinary files declaring the same module are already rejected by their
distinct required module paths. Failure owns the original source set together
with its file index, byte span and lexical/syntax/bound classification, allowing
diagnostics to read the exact input after partial parser owners are released.

The byte reader previously reused ASCII manifest-path validation and rejected
Unicode source filenames prematurely. `project_path.source_relative` and
`project_directory.resolve_source` now observe UTF-8 physical source paths,
while manifest paths retain their portable manifest-v1 rules. Unicode files
and directories are retained and then rejected by module binding with E0002.
Traversal, wildcard, device syntax and noncanonical path components still reject.

Two native tests and both candidate C profiles pass. Fourteen source-tree
acceptance observations agree with the Go loader, including normal/nested and
65 KiB files, module/path sort differences, mismatched names and case, second or
missing modules, malformed syntax, invalid UTF-8, Unicode/hyphen source paths
and duplicate declared names, plus a module name containing a 70000-byte block
comment. Original source bytes/spans survive failures. Full production check
passes: 311 files and 3987 MIR functions. Current evidence:
`evidence/candidate-source-modules-token-20261011.json`; the preceding binding
unit is recorded in `evidence/candidate-source-modules-20261011.json`.

This layer retains the parser's separate declaration, AST, cooked-value and
storage limits. Module declarations now compare their already parsed name tokens
directly, skipping trivia and locating the name span with binary search. No
second scanner or temporary canonical-name buffer is used; the former 64 KiB
name-fragment guard no longer constrains binding. Wider parser compatibility
remains explicit work.
It does not establish complete Go syntax or workspace-loader equivalence.

## Package input transaction

`project_package_input.load` stages the cumulative source budget and commits it
only after source loading and module binding both succeed. `load_manifest`
performs one exact file read of the real `fairy.toml`, then follows the same
transaction. Failed manifest parsing, filesystem collection and module binding
leave the caller's file and byte counters unchanged; the work meter remains
consumed. Typed manifest access is available without copying or reparsing it.

Uniform failures retain the source-loader phases 0..5 and module phase 6,
including OS/field/diagnostic codes, file index and original span. They own the
relevant manifest or source-file bytes and path, so callers can release their
input document and render diagnostics after temporary source/parser owners
close. The wrapper retains the selected diagnostic file rather than the entire
failed package. Phase 7 reports failure to allocate this diagnostic snapshot.

One native test and both candidate C profiles pass, covering exact cumulative
limits, module/syntax/UTF-8/path failures after reads, manifest error snapshots,
input-buffer release and real manifest-file loading. Module mismatch retains
the exact original 7..12 span and bytes. Full production check passes: 312 files
and 3999 MIR functions. Evidence: `evidence/candidate-package-input-20261011.json`.

## Streaming source hashing

`sha256_stream` retains one 64-byte partial block and eight SHA-256 words,
with checked modular compression shared by the existing one-shot implementation.
It accepts up to 96 MiB without materializing a padded input buffer; the old
one-shot API keeps its 1 MiB guard. Finalization copies the small state so the
original can continue receiving input. Invalid state or an update exceeding the
bound leaves the original unchanged.

Two native tests and both candidate C profiles pass: empty/abc vectors, lengths
55/56/63/64/65/127/128/129, seven-byte chunks, continuation after finalization,
and 1,052,672 repeated bytes beyond the old cap. Explicit expected hashes were
generated independently with Python hashlib. Full check passes 314 source files
and 4010 MIR functions. Evidence: `evidence/candidate-sha256-stream-20261011.json`.

## Exact package source identity

`project_source_identity.compute` hashes the retained `Sources` owner using
`docs/package/package-source-digest-v1.md`: domain/version bytes, network-order
64-bit file count and path/body lengths, exact path order and exact file bytes.
It returns `fysrc1-sha256-` plus 64 lowercase hexadecimal digits. The source-set
constructor establishes canonical paths; the digest also verifies file/byte
bounds and strict ordering before returning a result. The 4096 paths of at most
4095 bytes and 64 MiB of bodies plus framing fit the streaming 96 MiB bound.
Hashing borrows existing bodies and retains only a small hash state; it does not
copy the encoded package or reopen files.

`project_package_input.source_digest` exposes this identity over the owned,
validated package input. Module lookup order, physical root, manifest UID and
filesystem metadata do not enter the source digest. Package-instance identity
and dependency metadata remain separate work.

The Go reference generator writes eight deterministic source trees and computes
their expected text with `internal/packagecontent.Compute`, checking reversed
input order too. The authored native test and candidate C development/release
program compare every tree against that oracle. Cases cover nested source paths,
different physical roots, CRLF, NUL/invalid UTF-8, renamed paths, module-name
versus file-path ordering, and a 1,052,672-byte body beyond the old hash cap.
Full check passes 315 files and 4014 MIR functions. Evidence:
`evidence/candidate-source-identity-20261011.json`.

Reproduce the reference inputs with
`bun tools/run-go.ts run ./cmd/fairyc-candidate-source-digest-reference <output>`;
pass its `root.bin` to the `project_source_identity` private-test verifier.

## Explicit local workspace catalog

`project_workspace_catalog.load` retains the root and exactly the root manifest's
explicit members. Each entry owns the validated package input and its exact
source digest. Canonical member paths and UIDs have independent owned byte
indexes; stable merge sorting and binary lookup return original package
positions. Mutable byte/vector columns are sealed into their immutable index
owners after construction.

The catalog rejects more than 1024 local packages, case-folded member collisions,
ancestor/descendant overlap, nested workspace tables, duplicate package UIDs,
root language other than `fairyc-language/20`, and member language mismatches.
Overlap detection searches every segment prefix in the sorted folded index, so
an intervening name such as `a-b` cannot hide `a` versus `a/b`. Comparisons in
sorting and overlap search consume the shared work budget. Existing directory
resolution supplies exact physical spelling, directory type and reparse checks.

Source file/byte counters remain staged across the whole catalog, including
digest and metadata checks, and commit only when all entries succeed. Catalog
failures own their package and related paths. Nested package-input errors move
into the error owner, retaining the selected original source bytes and span
after temporary catalogs close. Undeclared manifests are ignored.

Fourteen acceptance/reason cases and the complete successful path/UID/digest
records agree with the Go workspace loader. One authored native test and both
candidate C profiles pass. Cases include root-only and unused members, nearby
non-overlapping names, intervening-name overlap, case collision, nested
workspace, duplicate UID, root/member language, missing manifest, original
module-error bytes/span, noncanonical directory case, and an over-limit member
list. Successful cumulative counters reach exactly 4096 files and 64 MiB;
failed loads preserve caller counters. Full check passes 317 files and 4063 MIR
functions. Evidence: `evidence/candidate-workspace-catalog-20261011.json`.

The matrix does not establish complete loader equivalence. Catalog metadata
checks follow package-input reads, so multiple simultaneous failures can have
different priority from Go. Existing parser/work bounds and the junction
discrepancy remain. Dependency declarations, feature instances, module-import
visibility, SDK injection and lock projection are not validated by this catalog.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-workspace-catalog-reference <output>`.

## Validated local dependency declarations

`project_workspace_input.load` stages the catalog's source counters, validates
every catalog package's dependency declarations, and retains one ordered edge
per declaration. Each edge preserves owner/target catalog positions, binding
and its independent sorted feature list; the declaration order is already
canonical in the manifest owner. Repeated targets with different features stay
separate. Unreachable catalog members are checked too.

Targets must name an exact explicit workspace member, and required version
bytes must equal its declared version, including prerelease/build metadata.
Physically present but undeclared packages do not become dependency targets.
The compiler-owned `fairyc.std` UID, `std`/`std::*` local module namespace and
`stdlib` dependency binding reject. Similar user names such as `fairyc.stdtools`,
`stdtools` and `stdlib-extra` remain ordinary names.

The source-root owner now retains the original manifest alongside its decoded
tree. Declaration failures own the selected original manifest/source bytes,
file and package paths, decoded context/target and original byte span. Nested
catalog failures move their existing error owner intact. Allocation and work
failures have explicit private categories. Caller counters commit only after
catalog and declaration checks both succeed; the work meter remains consumed.

Fifteen acceptance/reason cases and all successful owner/binding/target/feature
records agree with the Go loader. One native test and both candidate C profiles
pass, including independent feature edges, exact build-version mismatch,
unreachable-member validation, root/member reserved identities and namespaces,
similar allowed names, original error bytes/spans after catalog cleanup, and
nested missing-manifest errors. Successful counters reach the exact cumulative
source limits, while all failures preserve caller counters. Full check passes
318 files and 4105 MIR functions. Evidence:
`evidence/candidate-workspace-input-20261011.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-workspace-input-reference <output>`.

The retained edges are catalog declarations, not resolved feature instances.
Instance hashing/expansion, graph cycles, import visibility, SDK injection and
lock projection remain separate work. This unit does not claim whole-loader
equivalence or change the earlier junction/parser/work-bound limitations.

## Canonical package-instance identity

`project_instance_identity` implements `package-instance-id-v1.md`: tagged
UID/source/version byte strings, network-order 64-bit lengths and a sorted
unique feature set under the versioned domain. Components are nonempty opaque
UTF-8, without manifest-specific name/version restrictions or normalization.
Feature ordering is checked before empty/UTF-8 feature checks, matching Go's
rejection priority. The private shared-work and 96 MiB encoding/hash bounds
remain explicit implementation guards.

The result owns both canonical bytes and its full lowercase
`fypi1-sha256-` text. Record comparison requires exact canonical-byte equality
after a digest match, and rejects malformed IDs or a claimed digest collision.
The collision test injects a contradictory record; it does not find or claim a
real SHA-256 collision. Local root/dependency adapters bind actual catalog UID,
source digest, version and the independent feature set for that edge.

All seven normative identity fixtures and thirteen additional UTF-8, Unicode
composition, NUL, empty-value, feature-order/rejection-priority and over-1-MiB
vectors match Go's canonical bytes, IDs or rejection category. Four identities
from a real local workspace match Go; equal feature aliases compare equal and
different feature sets remain distinct. Two native tests and both candidate C
profiles pass. Full check passes 319 files and 4119 MIR functions. Evidence:
`evidence/candidate-instance-identity-20261011.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-instance-identity-reference <output>`.
Feature graph expansion, cycle rejection and import visibility are still separate.

## Root-reachable feature-instance graph

`project_instance_graph.load` expands only root-reachable local instances using
the exact PackageInstanceId tuple. Equal feature aliases share one node;
different feature sets remain separate. Each instance follows its catalog
package's fixed declaration features, without feature unification or propagation.
Outgoing ranges avoid rescanning the whole catalog, and per-declaration target
memoization avoids recomputing an ID for every owner feature instance.

The bounded hash index compares retained canonical bytes after digest equality
and fails closed on contradictory tuples. Its bucket growth rebuilds links from
current node headers; edge-span publication rereads those headers so a copy made
before growth cannot overwrite rebuilt links. Mutable construction columns seal
into the graph owner after all checks succeed.

Iterative DFS rejects self, multi-package and feature-induced cycles after exact
instance deduplication. Successful postorder is a dependency-first build order.
Cycle errors retain the original workspace, parser/manifest owners, declaration
coordinates and a closed path of instance IDs. Source counters commit only after
declaration loading, expansion and cycle checks all pass. Unreachable catalog
cycles do not enter the resolved graph, while their declarations remain retained
and validated by the preceding layer.

Eleven Go acceptance/reason cases and complete successful node/path/feature/edge
ID records match. One native test and both candidate C profiles pass, covering
aliases, feature split/diamond, self/two-package/feature cycles, unreachable
cycles, nested version failures and a 71-node/70-edge chain exercising bucket
growth and traversal beyond a small recursive depth. Every successful edge has
its dependency before its owner in the build order. Cycle instance IDs are
computed with the Go identity implementation. Failed graph loads preserve caller
source counters. Full check passes 320 files and 4199 MIR functions. Evidence:
`evidence/candidate-instance-graph-20261011.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-instance-graph-reference <output>`.
With multiple available cycles, witness selection follows declaration order and
can differ from Go's sorted-class traversal. Import visibility, SDK injection,
lock projection and compiler/project-CLI consumption remain separate; earlier
filesystem/parser/work-bound limitations still apply.

## Local module import visibility

`project_module_visibility.load` resolves each root-reachable instance's imports
through its own modules and direct dependency instances only. Equal-instance
dependency aliases enter the visible set once; distinct feature instances remain
separate providers. Current-instance modules have no shadowing precedence: two
different visible instances exposing the same imported path reject as ambiguous.
Dependency bindings do not rewrite source module paths. Unreachable catalog
imports and transitive-only modules do not become visible.

The source-module owner exposes its existing sorted name index. Instances and
modules traverse in the same class/name order as Go, retaining source-order
imports. Lookups preserve both source and provider instance/module coordinates,
original declaration indices, target byte spans and canonical paths. Contiguous
owner ranges prepare later compiler consumption. Direct providers are deduplicated
with node stamps and heap-sorted, and module binary searches consume the shared
work budget.

`project_import_path` reads the already verified import tokens, skipping trivia
and stopping before the alias. No second lexer or fragment copy introduces the
older 64 KiB normalization limit. Successful and failed imports containing a
70000-byte comment preserve their full original path spans. Missing/ambiguous
errors own the graph, selected path, prior successful lookup prefix and actual
visible/hidden provider coordinates, so original source remains available after
temporary construction owners close. Input/cycle errors move intact. Caller
source counters commit only after graph and visibility checks both succeed.

Nineteen Go acceptance/reason cases, successful lookup records, partial lookup
counts and rejection subjects match. One native test and both candidate C
profiles pass, covering own/self and direct lookup, nested paths, long trivia,
duplicate aliases, direct versus transitive visibility, own/direct and feature
ambiguity, current-instance provider ordering, binding/case mismatch, unreachable
imports, preserved lookup prefix, module-name versus file-path ordering, and
cycle rejection before lookup. Original error bytes/spans and exact source-budget
commit/rollback are checked. Full check passes 322 files and 4269 MIR functions.
Evidence: `evidence/candidate-module-visibility-20261012.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-module-visibility-reference <output>`.
The full Go loader confirms rejected cases but does not return their partial
graph. For lookup-error observations, the helper obtains the loader's feature
graph with import-free headers, restores every original file, replaces modules
with Go-parsed original imports and exact Go source digests, and calls the owning
Go graph resolver to retain partial lookup/subject evidence. It does not implement
the visibility decision itself.

SDK injection and compiler/project-CLI consumption remain separate. Existing
filesystem/parser/work bounds and multiple-cycle witness differences still apply;
this local matrix does not establish complete loader equivalence.

## Remaining connection

Connect the resolved local import graph to SDK injection, lock
projection and the compiler/project CLI. Broader parser-cap
compatibility remains separate.
Resolve the Go junction observation discrepancy before any loader
equivalence claim. Package/lock/CLI work remains separate.
ReadDir is currently Windows AMD64 and keeps the existing SDK path bound.
Linux and Stage 2 were not run for this component slice. Public CLI, SDK ABI,
DB and CI workflows are unchanged.

## Authenticated SDK source bundle

`project_sdk_bundle` assembles a complete core (seven sources), public vector
(eight sources) or separately identified vector-candidate bundle from owned
compiler SDK bytes. Caller labels never grant authority: the existing
`sdk_authority` source pins authenticate every file, and missing, duplicate,
out-of-range or changed inputs reject. There is no filesystem lookup or source
fallback in assembly. Production embedding and graph injection remain pending.

The bundle keeps freshly parsed original sources in exact canonical path order,
module lookup and the SDK package metadata. `project_content_hash` is the shared
streaming package-source/v1 encoder for local sources and SDK files. The composed
source identity uses SDK identity, runtime ABI and the complete exact-byte source
digest. The vector-candidate lane uses the vector bytes but retains its distinct
SDK identity. SDK package metadata mirrors the current `internal/sdkstdlib`
constants; it does not change the public SDK version or ABI.

Source file/byte counters commit only after authentication, parsing, digest and
identity construction all succeed. The existing 4096-file, 64-MiB aggregate,
8-MiB individual source and 16,777,216 work bounds apply. Bundle authenticity is
separate from source selection, graph injection, direct module visibility, lock
projection and compiler/project CLI consumption, which remain later work.


Three complete SDK profile/source/module/import records match the owning Go SDK
loader, including canonical order from reversed inputs, exact retained bytes,
metadata, digest and composed identity. Twelve invalid label, duplicate, changed,
incomplete and source/work-bound cases reject without changing caller counters.
Two native tests and development/release candidate C executions pass. The shared
encoder also passes the existing eight local package-digest fixtures in native
and both candidate C profiles. Check passes 324 source files and 4297 MIR
functions. Evidence: `evidence/candidate-sdk-bundle-20261012.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-sdk-bundle-reference <output>`.
Linux, the full suite and Stage1/Stage2 are deferred; these results establish only
the private bundle/encoding slice. Stable API, SDK ABI/version, DB and CI are
unchanged, and no push occurred.


## Root-reachable SDK selection

`project_instance_graph::select_sdk` reads the already verified import tokens
of root-reachable local instances and returns a private owner list with profile
0 (none), 1 (core), 2 (vector) or 3 (explicit vector candidate). `std` and exact
`std::` prefixes select the SDK; `stdx` and `foo::std` do not. Unknown reserved
paths still select a bundle and remain for module visibility to reject later.
Candidate selection overrides core/vector only when there is a reachable SDK
owner. A reachable `std::vec` import selects vector for the shared bundle even
when other owners need only core. Unreachable catalog modules do not participate.

Owner indices are unique graph-order instance indices. Equal-instance aliases
are already deduplicated; different feature instances of the same package each
receive their own owner row. Shared import/path work is metered, including long
trivia, and selection leaves source file/byte counters unchanged.

Fourteen successful full Go-loader observations match exact profile and owner
PackageInstanceIds: no SDK, unused candidate, core, duplicate import aliases,
vector, candidate with core import, child-only and joint ownership, child vector,
unreachable vector, feature split, equal-instance aliases, 70000-byte import
trivia and similar ordinary namespaces. Each also rejects exhausted selection
work. One native test and development/release candidate C execution pass; check
passes 324 sources and 4301 MIR functions.
Evidence: `evidence/candidate-sdk-selection-20261012.json`.

Reference fixture command:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-sdk-selection-reference <output>`.
The reference calls the full owning loader and observes its injected SDK package
and dependencies; it does not implement a second selection algorithm. SDK graph
injection, visibility, embedding, lock projection and compiler/CLI consumers
remain. Stable API/ABI, SDK version, DB and CI are unchanged. No push, Linux,
full suite or Stage1/Stage2 runs.


## Selected SDK graph injection and module visibility

`project_workspace_graph` owns the complete local feature-instance graph and
optionally one authenticated SDK bundle and its PackageInstanceId. It retains
local IDs and ownership; it does not move resource fields or mutate nested
resource fields. The new SDK node is last, with `standard-library` role and
catalog-count as a non-local package-index sentinel. Local package indices stay
stable. Direct `stdlib` edges attach to exactly the selected local instance
owners; the SDK has no outgoing package edges. Public core/vector bundle mismatch
rejects before attachment, while the separately identified vector-candidate lane
is explicit through its authenticated bundle metadata.

Node and edge ranges are built off to the side. Dependency-first local order
inserts the SDK immediately before the first direct SDK owner. All retained
edges remain dependency-first, every node occurs once and there is no new cycle:
the original graph is acyclic and the SDK has no outgoing edges. This is a valid
order, not a promise to duplicate Go's choice of topological order. SDK digest
collisions with an existing local instance reject. Existing local IDs, source
caps, work caps and multiple-cycle witness limitations remain unchanged.

`project_module_visibility` consumes the unified graph's module accessors,
resolving both ordinary imports and SDK internal imports through own plus direct
instances. SDK file paths and module names borrow from the bundle's retained
owned metadata and original parsed source. `workspace_graph` and
`error_workspace_graph` expose the full graph. Legacy `instance_graph` and
`error_graph` expose the retained local prefix only; SDK consumers must use the
full workspace accessors. `load_sdk` takes an already authenticated compiler
bundle and commits local source counters only after load, injection and all
imports succeed. SDK construction has its own bundle budget. Injection performs
no source reads and adds no SDK files/bytes to local counts, matching Go's loader.
Production embedding of those bytes is still separate work.

Fourteen full Go-loader records match node IDs/roles, owner-grouped dependency
IDs/bindings and every source/import/provider/module lookup. Tests exercise core,
vector, candidate, child-only SDK ownership, distinct features, aliases,
unreachable SDK imports, long trivia and ordinary similar namespaces, with local
source counts at their exact ceilings. Wrong selected public bundles, missing
`std`/`std::missing`, exhausted injection work and source counter rollback reject
correctly. Missing SDK import rejection is confirmed by the full Go loader;
partial error graph/source checks are candidate invariants, not a returned Go
partial graph. Two native tests and development/release candidate C executions
pass. The existing 19 local module-visibility cases also pass in native and both
candidate C profiles after the unified accessors change. Source check passes 325
files and 4349 MIR functions.
Evidence: `evidence/candidate-sdk-graph-20261012.json`.

The existing reference fixture command now also records full SDK graph and
lookup observations:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-sdk-selection-reference <output>`.
Production SDK embedding, lock projection and compiler/project CLI consumption
remain. Stable CLI/API, SDK ABI/version, DB and CI are unchanged. Linux, full
suite and Stage1/Stage2 are deferred; no push occurred.


## Compiler-owned embedded SDK bytes

`project_sdk_embedded` supplies a pure, metered SDK factory. Ten generated Fairy
byte-literal parts contain the exact eight current core/optional-vector source
files from `internal/sdkstdlib`. The owning Go sources remain authoritative;
`cmd/fairyc-candidate-sdk-embed` only quotes/chunks those bytes and verifies the
source set. It implements no runtime loading, hashing, selection or lookup logic.
Literal chunks are at most 2048 raw bytes and generated source parts stay below
40 kB (current maximum 18204 bytes). Each append charges the existing work meter.
No filesystem, environment, installer or registry lookup supplies SDK bytes.

Factory profiles 1/2/3 construct core, public vector and explicit vector-candidate
bundles respectively. Every decoded source still passes the existing Fairy source
pins and owned bundle parser/digest/identity construction; literal placement or
labels grant no authority. Profile 0 and unknown profiles reject. The default
`project_module_visibility::load` now selects the SDK from actual reachable
imports, creates it only when needed, injects it and resolves all imports.
`load_embedded` admits the explicit candidate lane. Caller local source counters
commit only after the complete path succeeds; SDK sources retain their separate
budget. The supplied-bundle `load_sdk` remains available for private invariant
checks. SDK-unneeded projects never invoke the factory.

Generation and freshness commands:
`bun tools/run-go.ts run ./cmd/fairyc-candidate-sdk-embed --write examples/lexer_candidate/src`
and the same command with `--check`. The check compares exact source-derived file
contents and rejects stale extra generated parts. Data regeneration and any pin
or SDK metadata update require deliberate source review; there is no runtime
fallback. These checks do not establish standalone SDK API or ABI changes.

Fourteen full Go graph/node/edge/lookup cases match through automatic loading,
including exact retained SDK bytes, owned paths/names and all three profiles.
Missing reserved imports retain rollback, and invalid factory profiles plus
exhausted pure literal-copy work reject. Three native tests and development/release
candidate C executions pass. Check passes 336 files and 4367 MIR functions; the
source-derived literal check passes all ten parts.
Evidence: `evidence/candidate-sdk-embedded-20261012.json`.

Selection-work failure retains the original graph problem through its own
adapter; embedded construction failure retains its separate SDK reason and file
index. Lock projection and compiler/project CLI consumers remain. Stable CLI/API, SDK ABI and
version, DB and CI are unchanged. Linux, full suite and Stage1/Stage2 remain
scheduled later; no push occurred.


## SDK failure category and lazy work boundary

The automatic SDK loader preserves the original graph selection problem instead
of translating it to a bundle problem with the same numeric leaf code. Exhausted
selection work remains graph code 4; exhausted literal copying is graph code 8
with SDK reason 5 and original SDK file index. Both errors retain the untouched
local graph and preserve caller source counters. A local source-limit failure
still wins before SDK selection and keeps its original loader phase.

Two native tests and both candidate C profiles pass four measured phase cases:
selection exhaustion, embedded literal-copy exhaustion, earlier local source
limit, and an SDK-unused project ending exactly at the final resolver work unit.
The tests measure existing local load/selection work, then place the meter at the
owning phase without hard-coded fixture work or timing. The unused SDK case
succeeds at meter 16777216 without creating any SDK node, proving factory laziness
at the work boundary. Source check passes 336 files and 4368 MIR functions.
Evidence: `evidence/candidate-sdk-failures-20261012.json`.

The prior embedded SDK graph/source corpus remains the evidence for unchanged
successful SDK loading; this small fix covers error categorization and the exact
lazy boundary. Lock projection and compiler/CLI integration remain. Stable
CLI/API, SDK ABI/version, DB and CI are unchanged; Linux, full suite and Stage1/2
are scheduled later and no push occurred.


## Owned generated-lock document projection

`project_lock_document` projects the validated resolved local/SDK graph into an
owned document model with the exact five header values, ten scalar package
fields, feature arrays and dependency endpoints/bindings of lockfile v1.
Package records sort by full instance ID; edges sort by owner ID, binding and
target ID, using a metered heap sort. Dependencies point into the sorted package
array and equal-instance aliases retain their distinct binding records. Only
root-reachable instances enter the document; local paths remain workspace-relative
and SDK local paths/feature arrays remain empty. Schema/algorithm/language/root
and SDK identity/ABI/version/source digest/source identity are exact.

All strings copy into an owned bounded buffer; the original graph/source can
close before reading the document. Projection consumes the existing shared work
budget and an 8-MiB text ceiling. It performs no filesystem IO or counter update.
The private vector-candidate SDK rejects instead of being normalized into the
public SDK identity: current Go `lockfile.FromWorkspace`/`Validate` rejects that
separately identified source too. An unused candidate lane with no injected SDK
can project the ordinary local document.

Thirteen full Go documents match every retained header/package/feature/dependency
value and order. Exact-source graph, aliases, feature splits, core/vector SDK,
unreachable members and similar namespaces are covered. All thirteen reject
exhausted projection work; the private candidate rejection matches Go. Two native
tests and development/release candidate C execution pass. The document remains
readable after its source graph closes. Check passes 337 files and 4409 MIR
functions. Evidence: `evidence/candidate-lock-document-20261012.json`.

The SDK reference command now also calls owning `lockfile.FromWorkspace` and
`Marshal`, saving canonical JSON for the next serializer unit and a test-only
length-prefixed full-field observation for this projection. No second Go
projection algorithm is used. This unit does not implement JSON, strict reading,
stale matching or writing. Those and compiler/project CLI consumption remain.
Stable CLI/API, SDK ABI/version, DB and CI are unchanged. Linux, full suite and
Stage1/Stage2 remain scheduled later; no push occurred.


## Canonical lock JSON emission

`project_lock_json` emits the owning lockfile-v1 field order, two-space indentation,
explicit empty arrays and exactly one final LF from the owned projected document.
Sorted package/edge order and exact local/SDK identity values come from projection.
`project_json_string` matches Go `encoding/json` string spelling for valid UTF-8:
controls, quotes/backslashes, HTML `<>&`, U+2028/U+2029 and unchanged other Unicode.
Invalid UTF-8 rejects instead of using Go's replacement behavior; projected lock
values already pass the source/manifest UTF-8 validation boundary. Ordinary runs
append as whole slices, avoiding per-byte calls into the target buffer runtime.

The existing shared work meter and 8-MiB output ceiling bound emission. Failure
returns no partial buffer. Thirteen complete canonical Go lockfiles match every
byte, including empty/nonempty feature and dependency arrays. Eight direct Go
string cases cover all 32 control bytes, escapes, HTML, Korean/emoji/separators,
DEL/high Unicode, empty input and a 65537-byte string. Invalid UTF-8, exhausted
work and an oversized 8-MiB input reject. All thirteen documents reject exhausted
marshal work too. Three native tests and development/release candidate C execution
pass; check passes 339 files and 4418 MIR functions.
Evidence: `evidence/candidate-lock-json-20261012.json`.

The boundary fixture originally used both `file_source` readers, which share a
65536-byte cap; their native failures were read assertions. Fixture reads now
use the existing project file reader with explicit 1-MiB maximum, retaining the
large boundary case rather than shrinking it. Product caps are unchanged.

The reference uses the owning `lockfile.Marshal` and standard `encoding/json`,
not a second canonical serializer. Strict JSON reading, identity/graph validation,
stale comparison, atomic persistence and compiler/project CLI consumers remain.
Stable CLI/API, SDK ABI/version, DB and CI are unchanged. Linux, full suite and
Stage1/Stage2 remain scheduled later; no push occurred.


## Bounded JSON string decoding and token scanning

`project_json_decode` decodes JSON escapes and UTF-16 surrogate pairs, matching
Go replacement behavior for malformed raw UTF-8 and unpaired surrogates.
`project_json_tokens` owns source bytes, decoded strings and token spans and
preserves raw number lexemes. Input/decoded pools cap at 8 MiB, tokens at 1048576,
and both use the existing shared 16777216 work meter. Decoding restores the
caller's prior output on failure; codes are 1 syntax, 2 allocation and 3 bound.

The Go `encoding/json` oracle covers 24 string cases (14 accepted, 10 rejected).
Three exact native tests and candidate development/release execution pass,
covering the oracle, retained token values/spans, malformed lexemes, rollback,
work exhaustion and oversized input. Check passes 341 files and 4443 MIR
functions. Receipt: `evidence/candidate-json-tokens-20261012.json`.

This is the lexical layer only. JSON structure/duplicate keys, lock shape and
identity/graph validation, stale comparison, atomic persistence and compiler/CLI
integration remain. Stable CLI/API, SDK ABI/version, DB and CI are unchanged;
Linux, full suite and Stage1/Stage2 remain scheduled later and no push occurred.
