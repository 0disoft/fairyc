# Candidate source-independent cleanup MIR contract

This private fragment accompanies `module_unit_cleanup_mir_model.fy`,
`module_unit_cleanup_mir_types.fy`, `module_unit_cleanup_mir_lower.fy` and
`module_unit_cleanup_mir_verify.fy`, with `tests/module_unit_cleanup_mir.fy`. It is a
source-independent cleanup MIR fragment: glue, drop sites and the canonical tables a
later C step needs. It is not executable whole MIR.

Owned data. `Program` is a resource owning a canonical `Type` table and `Frame`,
`Glue`, `Field`, `Operand` and `Drop` vectors plus a `work` budget. Accessors are
one-based (`type_at`, `frame_at`, `operand_at`) or zero-based (`field_at`, `drop_at`)
with `*_count` readers, and `assemble` runs `proof::check` before returning `.ok`.
After lowering, no original `Parsed`, `Values` or provider `Interfaces` is retained.

- `Frame {kind,context,module_index,declaration,result,effects,receiver,receiver_type}`:
  kind 1 concrete body, 2 cleanup body (void result), 3 ordinary body; only kind 2 owns
  a `receiver` local and its type.
- `Glue {kind,frame,nonempty,effects,abilities,field_start,field_count}` per type:
  kind 0 none, 1 resource, 2 Option, 3 Result.
- `Field {owner,ordinal,type}` in declaration order.
- `Operand {frame,kind,id,type,start,end}`: kind 1 lexical binding, kind 2 materialized
  expression, each frame-qualified with its original span.
- `Drop {operand,exit,start,end}`: one cleanup site with a bounded exit reason.
- `Problem {code,module_index,start,end}`: 1001 invalid, 1002 budget, 1003 allocation.

Lowering. `lower` rejects outstanding move diagnostics, then runs the cleanup-policy
gate, seeds `work`, copies the canonical type table, and resolves every pending
expression ID (binary search on source index) or binding ID against its frame before
the typed context is released. A resource glue takes effects, abilities, nonempty and
ordered fields from provider interfaces, checks the provider module/declaration against
the canonical type, and binds the single cleanup frame whose receiver type matches.

Verification. `check` revalidates the owned descriptors independently: per-kind type
shape; canonical descriptor uniqueness through an 8192-bucket hash; non-resource glues
zeroed; a nonempty resource bound to exactly one matching cleanup frame with a void
result; the owned nested and tagged effect union with recursive drop-ability checks
(memoized, cycle-detecting, depth 64, run over every type even when unused); field
ordinal contiguity; operand uniqueness per `(frame,kind,id)` through a 32768-slot index;
and drop sites that must be owned, droppable, inside the frame's authority and never the
cleanup receiver.

Validation. `tests/module_unit_cleanup_mir.fy` passes 4/4 and rejects thirteen malformed
handoffs: out-of-range drop operand or operand type, removed caller authority, removed
nonempty cleanup frame, changed provider module, reordered field ordinal, removed nested
drop ability, duplicate qualified operand, recursive field cycle, invalid Result
alternative, exceeded work budget, non-void cleanup result and duplicate canonical
descriptor. Receipt: `docs/ops/evidence/candidate-module-cleanup-mir-20261010.json`.

Invariants and limits. types 1..4096, frames 1024, operands 16384, drops 16384, fields
1024, effects 4095, abilities 7, recursion depth 64, work 1048576.

Pending. Drop sites keep their semantic request order; attaching them to CFG blocks and
ordering reverse-lexically with defer and Region sequence happens before any C handoff.
The model holds no ordinary function bodies, CFG or local-execution definitions, so it
does not yet carry all source provenance. C emission and execution, CFG placement and
defer/resource/Region sequencing remain pending; the existing `module_program` C path
still joins and is unchanged, and this is not a new C pipeline or the full compiler.
