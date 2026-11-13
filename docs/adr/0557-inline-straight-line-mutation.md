# ADR 0557: Inline straight-line local mutation

- Status: Accepted for bounded prototype
- Date: 2026-11-14
- Extends: ADR 0509 and ADR 0548

Admit mutable i32 locals and plain `=` assignments to preceding mutable local
paths in the existing public safe pure i32 inline family. Parameters remain
immutable. At most four parameters, sixteen declared locals, sixty-four
straight-line initializer/assignment steps and sixty-four expression nodes are
admitted. Existing same-module resolved inline call limits remain unchanged.
Compound assignment, parameter/field/indirect assignment, branches, loops,
other types, resources and cross-module calls remain separate families.

Each declaration or assignment defines a new canonical value version. A path
reads the latest preceding version of its local; an immutable snapshot retains
the version read at initialization. Initializers and assignment expressions
execute once in source order even if their resulting value is unused. Existing
checked arithmetic and source-attributed traps remain authoritative.

Such bodies use `fairyc.inline-function/v4`. Its root uses the existing ordered
`bindings` node, with one initializer child per straight-line step followed by
the return expression. Local path indices identify value versions, not physical
storage slots. Literal, parameter and same-module call node encodings remain
unchanged. A mutable declaration alone selects v4; immutable bodies retain
their exact v1/v2/v3 bytes. The existing digest carrier and package inline export
class remain unchanged, so assignment changes invalidate dependent cache keys.

Source checking validates writable local declarations. Detached verification
independently checks declaration kind/mutability, target ownership, preceding
initialization and value versions before reconstructing the identity. Matching
a digest does not authorize an otherwise-invalid assignment or local record.
