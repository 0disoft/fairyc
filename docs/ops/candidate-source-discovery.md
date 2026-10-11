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

## Remaining connection

Connect package inputs and their exact source identity to workspace catalog/package
resolution and the project CLI. Broader parser-cap compatibility remains separate.
Resolve the Go junction observation discrepancy before any loader
equivalence claim. Package/lock/CLI work remains separate.
ReadDir is currently Windows AMD64 and keeps the existing SDK path bound.
Linux and Stage 2 were not run for this component slice. Public CLI, SDK ABI,
DB and CI workflows are unchanged.
