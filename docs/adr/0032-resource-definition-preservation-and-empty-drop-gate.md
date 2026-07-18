# ADR 0032: Preserve resource definitions and gate executable drop bodies

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- Product source: `docs/product/02-spec.md`
- Language source: `docs/language/0001-values-and-resources.md`
- MIR source: `docs/mir/mir-v0.md`
- Fixture source: `spec/fixtures/mir-verifier/manifest-v0.json`
- Implementation source: `internal/ast/`, `internal/semantic/`, `internal/hir/`, `internal/mir/`, and `internal/interpreter/`

## Context

The parser already accepted a resource's `drop fn(self: &mut Self) { ... }`, but AST lowering kept
only the resource name and fields. Typed HIR then contained functions only, and MIR carried resource
shape only on individual construction operations. The interpreter built a field map from each
operation and synthesized drop traces without knowing whether the source drop body was empty.

That was tolerable only while every repository fixture used an empty body. Emitting a native resource
struct from that state would silently erase user cleanup as soon as a nonempty body appeared. It
would also let construction, field-read, interpreter, and backend code disagree about field order or
type because no verified program-level definition owned those facts.

## Decision

1. AST `Resource` preserves a `ResourceDrop` declaration and its complete lowered block. Syntax may
   no longer disappear merely because its execution is unfinished.
2. Typed HIR owns one resource definition per source declaration. It contains the canonical type,
   source span, declaration-ordered resolved fields, and a closed drop classification: `missing`,
   `empty`, or `non-empty`.
3. The current semantic subset accepts only `empty`. Missing or nonempty user drop bodies produce an
   explicit unsupported-subset result before Flow. They are not treated as no-op cleanup.
4. Executable MIR owns one program-level resource definition per canonical type. It contains ordered
   field names and types plus the drop classification. MIR lowering derives `NeedsDrop` from these
   definitions, removing the former caller-built AST resource-type side channel.
5. Resource construction must name one definition and list exactly its fields in declaration order.
   Field reads record both owner and result type and must match the definition. Duplicate or invalid
   definitions, missing definitions, field mismatch, and any executable drop kind other than `empty`
   are `V1001` compiler defects.
6. The interpreter builds its resource table from verified MIR definitions. Construction, field
   reads, drop traces, and reverse-field recursive cleanup use that table. It still does not execute
   user drop-body effects.
7. The resource-return verifier seed carries a valid definition. One new isolated mutation removes
   only that definition and requires one `V1001`, bringing the v0 manifest to 10 baselines and 21
   mutation cases.
8. This decision is a representation and fail-closed prerequisite. It does not implement resource C
   structs, native drop glue, partial initialization, nonempty user drop execution, or resource ABI.

## Consequences

- Source cleanup meaning can no longer vanish between parsing and executable lowering.
- Every interpreter and future backend sees the same field order and resolved field types.
- MIR test builders and verifier seeds that claim a resource obligation must carry its definition.
- Empty fixture drop bodies remain executable with the existing synthetic trace semantics; a real
  body must wait for calls, unsafe effects, failure policy, and native drop-glue contracts.
- Resource C emission can now start from verified definitions instead of guessing from operations,
  but it must remain disabled until its own representation and execution oracles exist.

## Rejected alternatives

- Treat every parsed drop body as empty until execution exists: silently changes valid source meaning
  and turns future cleanup into a wrong-code bug.
- Reconstruct resource fields from the first constructor: fails for unused types, permits constructor
  disagreement, and loses source declaration order as an authority.
- Keep passing a separate `[]resourceType` from AST into MIR lowering: duplicates semantic authority
  and can drift from typed HIR.
- Implement native resource C emission in the same step: representation without preserved drop
  semantics would create executable evidence for the wrong contract.
