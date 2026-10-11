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

## Remaining connection

Connect the manifest source value, recursively collect every regular .fy,
reject source-case collisions, enforce cumulative file/byte limits, retain exact
bytes and bind declared modules. Package/lock/CLI work remains separate.
ReadDir is currently Windows AMD64 and keeps the existing SDK path bound.
Linux and Stage 2 were not run for this component slice. Public CLI, SDK ABI,
DB and CI workflows are unchanged.
