# Bounded canonical `.fymap` transport v1

- Status: Accepted
- Date: 2026-07-23
- Product source: `docs/product/02-spec.md`
- Transport source: `docs/atlas/fymap-transport-v1.md`

## Context

ADR 0185 fixes one immutable logical partition per package but deliberately leaves transport bytes
undecided. A map reader is an untrusted-input boundary: a checksum alone detects accidental byte
changes but does not reject unknown fields, noncanonical alternate encodings, contradictory counts,
malformed identities, or a declaration paired with somebody else's valid `SymbolId`.

Choosing filesystem publication before a strict in-memory codec would mix semantic validation,
serialization, path containment, replacement durability, and stale-source policy into one failure
surface.

## Decision

1. Define exact transport schema `fairyc.fymap/v1` and encoding
   `fairyc.fymap-json/v1`. Emit one compact UTF-8 JSON value with fixed struct field order and no
   trailing bytes.
2. Require a positive caller-provided byte limit for encoding and decoding. The project does not
   invent a universal map-size default.
3. Define `fairyc.fymap-checksum/v1-sha256`, rendered as `fymap1-sha256-` plus 64 lowercase
   hexadecimal digits. Hash a domain/version prefix and length-prefixed transport schema, encoding,
   and canonical logical document bytes. The checksum field itself is excluded.
4. Decode with unknown-field rejection, one-value enforcement, exact schema/encoding/checksum
   algorithm matching, checksum comparison, complete semantic validation, and byte-for-byte
   canonical re-encoding.
5. Encode fixed-width unsigned JSON integers for spans and counts. Reject host-integer overflow on
   decode and cap recursive type validation at 64 levels.
6. Revalidate every compiler, package, source, file, line-map, interface, and SymbolId text form;
   require nonzero digest identities; require canonical order and uniqueness; verify header counts,
   file/module bijection, declaration/symbol/function ownership, spans, relation counts, and explicit
   completeness omissions.
7. Add exact package namespace references for self and direct dependencies to every logical header.
   Recompute every `SymbolId` from decoded declarations and those UID/instance/coordinate mappings;
   reject any mismatch even when the attacker recomputed the map checksum.
8. Return an immutable `Partition` only after all checks pass. Transport failure remains downstream
   from compilation and cannot change diagnostics, MIR, emitted C, native artifacts, or execution.

## Consequences

- Equivalent logical partitions have one accepted byte representation and one checksum.
- Whitespace, duplicate or unknown fields, alternate array order, malformed IDs, stale counts,
  broken references, and checksummed semantic contradictions fail closed.
- JSON is selected for the first exact pre-release transport because the existing toolchain can
  implement and inspect it without a new dependency. Compression or a binary replacement requires
  a new encoding identifier and measured rewrite/read evidence.
- The checksum is corruption and canonical-integrity evidence, not authentication. A trusted-origin
  signature policy is not implied.
- This decision does not choose map paths, atomic publication, retention, current-source
  revalidation, dependency-map discovery, queries, or CLI behavior.

## Evidence

Real-workspace round trips cover every generated package partition and repeated byte identity.
Regression tests cover the fixed checksum vector, positive bounds, oversize input/output, empty
input, checksum corruption, wrong schema/encoding/checksum algorithm, malformed checksum text,
unknown fields, trailing values, noncanonical whitespace, contradictory counts and algorithms,
duplicate records, unknown spans, malformed SymbolId pairs, declaration/identity mismatch, package
reference mismatch, deep types, and checksummed invalid payloads.
