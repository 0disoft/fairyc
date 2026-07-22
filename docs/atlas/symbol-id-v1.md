# SymbolId v1

- Status: Implemented provisionally
- Algorithm: `fairyc.symbol-id/v1-sha256-128+256`
- Identity encoding: `fairyc.symbol-identity/v1`
- Signature encoding: `fairyc.symbol-signature/v1`
- Decision: `docs/adr/0183-provisional-symbol-id-v1.md`
- Implementation: `internal/symbolidentity/`

## Purpose and scope

`SymbolId` identifies one semantic declaration across deterministic rebuilds without serializing
HIR symbols, build-local package numbers, source positions, or exact package instances. It is an
Atlas join key, not a native linker name, language ABI, debugger symbol, source anchor, or authority
token.

The algorithm is executable and versioned so `.fymap` work can use a tested identity boundary. It
remains a pre-release provisional contract: no released reader or cross-version migration promise
exists. Any incompatible input, normalization, digest, truncation, or spelling change requires a
new algorithm identifier rather than silent reinterpretation.

## Identity tuple

The canonical tuple contains, in order:

1. package UID;
2. module path;
3. declaration kind;
4. canonical declaration path;
5. generic arity, fixed to zero for the current non-generic language; and
6. the complete `fairyc.symbol-signature/v1` byte sequence.

Each variable component is tagged and encoded as an unsigned 64-bit big-endian byte length followed
by exact bytes. The stream starts with `fairyc.symbol-identity`, NUL, and version byte `1`. Package
UIDs are valid UTF-8 without control characters. Module and declaration paths are nonempty `::`-
separated Stage 0 identifiers. Declaration kind is exactly `struct`, `enum`, `resource`, `function`,
or `foreign-function`.

The tuple excludes:

- `PackageInstanceId`, package version, feature set, and source digest;
- HIR package and symbol numbers, MIR entity strings, and compiler-private names;
- source path, byte span, declaration order, whitespace, body, and comments;
- compiler build, target, language revision, timestamps, and absolute paths.

Those exclusions keep a body edit, deterministic rebuild, version bump, or package renumbering from
renaming an otherwise identical declaration. A future `.fymap` header carries language, compiler,
package-instance, source, target, and compatibility identities separately.

## Canonical signature

The signature stream starts with `fairyc.symbol-signature`, NUL, and version byte `1`. It includes
the declaration kind and the semantic shape that distinguishes an incompatible declaration:

- result or declared type, owner type, receiver form, ordered parameter types, unsafe state, and
  canonical effects;
- foreign ABI and link name;
- ordered aggregate fields and enum variants with names and structured types;
- `@repr(C)` state; and
- resource cleanup kind and canonical cleanup effects.

Visibility, parameter names and local-binding mutability, source spans, provenance, and function
bodies are excluded. Visibility is a fact about access to the same declaration; parameter binding
details and spans do not change a call's typed contract. A visibility change can therefore be
reported as a fact change without destroying the join key.

Types are encoded recursively. Builtins retain their canonical semantic name. Borrow, pointer,
fixed-array, `Option`, and `Result` types retain kind, mutability, length, and ordered arguments.
Nominal types replace the snapshot's `PackageCoordinateId` lookup key with the referenced package
UID plus logical type path. Exact package-instance and version compatibility remains a map/header
and graph-edge concern, so upgrading a package version alone does not renumber every declaration.

## Digest and text forms

SHA-256 hashes the complete identity encoding. Every symbol retains both:

- visible ID: the first 128 bits, written as `fysym1-sha256-128-` plus 32 lowercase hexadecimal
  digits; and
- full digest: all 256 bits, written as `fysym1-sha256-256-` plus 64 lowercase hexadecimal digits.

Readers accept only those exact lowercase, full-width spellings. Every stored pair must satisfy
`visible == full[0:16]`. Two different full digests under one visible ID are a hard collision and
reject the map or symbol set. Duplicate equal pairs are not a collision. Random salts and
collision-dependent renumbering are forbidden because they would destroy reproducibility.

## Current evidence and exclusions

Unit and compiler-pipeline regressions freeze the canonical byte vector, tuple boundaries, strict
text parsing, invalid path and metadata rejection, visible/full consistency, synthetic 128-bit
collision rejection, body/span/version/build-number stability, nominal package-UID normalization,
semantic-change sensitivity, and deterministic ordering.

The Atlas transport now stores and rederives this identity under
`docs/atlas/fymap-transport-v1.md`. This package itself does not own map partitioning, bytes,
checksums, dependency-map joins, a stale-map validator, Atlas queries, Context Packs, anchors,
rename heuristics, LSP/SCIP identity, or a stable post-1 compatibility promise.
