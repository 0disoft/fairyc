# ADR 0254: Defer Nominal Scalars Pending User-Domain Evidence

- Status: Accepted deferral
- Date: 2026-08-04
- Candidate source: ADR 0253
- Product source: `docs/product/02-spec.md`
- Roadmap source: `docs/product/01-roadmap.md`
- Evidence scope: compiler-bundled SDK and checked-in non-fixture examples at language revision `/8`
- Depends on: ADR 0136, ADR 0218, ADR 0253

## Context

ADR 0253 ranks nominal scalar types as the first new post-Stage 0 candidate because values such as
IDs, counts, offsets, statuses, and alignments can share one machine representation while carrying
different meanings. That is a plausible problem statement, but candidate rank is not evidence that
the current Fairy C corpus needs another declaration, conversion, operator, HIR, MIR, interface, and
FFI type path.

A bounded source audit inspected all four compiler-bundled SDK `.fy` files and all thirty checked-in
non-fixture example `.fy` files. The SDK contains 66 declaration or signature occurrences using the
two proposed first representations: 40 `u32` and 26 `usize`. Nineteen `u32` occurrences are named
`status`; the recurring `usize` roles include six `length`, five `capacity`, three `alignment`, two
`read`, two `written`, and two `new_length` occurrences. The combined SDK/example scan contains 135
such occurrences: 81 `u32` and 54 `usize`.

The counts alone overstate the language-level problem:

- public allocation, buffer, text, console, and file failures already cross safe boundaries through
  distinct error structs and `Result`, not as interchangeable bare `u32` statuses;
- public byte counts, lengths, capacities, reads, and writes are separated by named functions,
  fields, and parameters, with no observed same-position overload or implicit conversion;
- the clearest size-versus-alignment and length-versus-count collisions occur inside explicit
  `unsafe extern C` wrappers, where any future nominal value must be deliberately unwrapped to the
  fixed C ABI scalar;
- the current non-fixture corpus contains no application-level `UserId`, `FileId`, handle index,
  timestamp unit, currency, coordinate, or other same-representation domain pair whose accidental
  interchange is demonstrated.

## Decision

1. Keep nominal scalars in `candidate` state. Do not accept syntax or begin parser, semantic, HIR,
   MIR, backend, SDK, interface-hash, or migration implementation from the present SDK counts alone.
2. Do not manufacture evidence by replacing well-named SDK storage fields with wrappers. Existing
   error structs and resource fields are successful type and ownership boundaries, not proof that
   they should become nominal scalars.
3. Reopen acceptance when a representative Fairy C application contains at least one real boundary
   where two same-representation domain values can be plausibly swapped and ordinary structs,
   separate functions, or `Result` make the API materially worse. The evidence must include the
   caller, callee, erroneous interchange, desired diagnostic, FFI behavior if present, and generated
   C representation.
4. A later feature ADR must still decide alias versus nominal identity, construction and extraction,
   constants, comparison, arithmetic, pattern binding, `ref`, `Option`, `Result`, public-interface
   hashing, cross-package identity, FFI conversion, layout, diagnostics, and migration. This deferral
   reserves no spelling, including `distinct`, `newtype`, `typedef distinct`, or constructor names.
5. Constrained scalar ranges remain ordered after nominal scalars. Function contracts remain later
   still. Neither feature may use this audit as evidence for its own syntax.
6. This decision changes no grammar, diagnostic, semantic behavior, language revision, SDK,
   runtime ABI, target support, or generated C. The current language remains `fairyc-language/8`.

## Consequences

- The roadmap retains nominal scalars as the first design candidate but marks implementation as
  evidence-blocked rather than ready.
- The next useful input is a real Fairy C application domain, not another synthetic syntax sample.
- The SDK audit provides a repeatable baseline for a later comparison without treating raw token
  frequency as user harm.
- `docs/product/02-spec.md` remains unchanged because no language behavior was accepted.

## Follow-up evidence (2026-09-17)

The [real scanner boundary experiment](../ops/token-index-struct-experiment.md)
demonstrates TokenIndex/ByteOffset interchange and compares ordinary structs.
The [nominal scalar prototype proposal](../architecture/nominal-scalar-prototype-v0.md)
defines a bounded validation/erasure experiment and its remaining admission gates.
This follow-up does not accept syntax or lift the stable-language implementation
deferral; the original revision `/8` audit remains historical evidence.

## Follow-up evidence (2026-09-22)

[Nominal scalar source usability v0](../architecture/nominal-scalar-source-usability-v0.md)
keeps the deferral, including the experimental source frontend. The real
`body_state` caller cannot retain a nominal `Cursor.index` field under the current
proposal. Call-site wrapping alone does not establish the raw value's domain,
and a source usability advantage over ordinary structs remains unproven.
This is not a permanent rejection; the review records bounded reopening criteria.

## Follow-up evidence (2026-09-27)

The [real record-parser cursor experiment](../architecture/nominal-scalar-source-usability-v0.md#실제-레코드-파서-커서-실험-2026-09-27)
types `records::Cursor.index` with an ordinary struct in a copied workspace.
The wrong-domain assignment fails with `E1006`. A subsequent bounded MIR
extension makes valid replacement of this one-`usize`-field struct executable;
the real parser copy now passes `check`. Full native and size comparisons for
this parser variant remain open. This does not establish a source-usability
advantage for nominal scalars or admit syntax.
