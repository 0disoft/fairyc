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

## Remaining connection

Connect source-root ownership to the catalog, read and retain exact source
bytes, recheck regular files, enforce cumulative byte limits and bind declared
modules. Resolve the Go junction observation discrepancy before any loader
equivalence claim. Package/lock/CLI work remains separate.
ReadDir is currently Windows AMD64 and keeps the existing SDK path bound.
Linux and Stage 2 were not run for this component slice. Public CLI, SDK ABI,
DB and CI workflows are unchanged.
