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

Compiler-private aggregate admission (2026-11-05). `Program` retains a separate
aggregate-admission bit. `assemble_aggregates` verifies concrete struct/resource
kinds 20/21 and ordered scalar-argument tuple kind 22. Instance descriptors retain
module/declaration coordinates and a backwards reference to a nonempty tuple;
tuple links are backwards, scalar-only and bounded to eight arguments. Exact
source declaration arity is checked before lowering by the retained aggregate
source/type owner, not reconstructed from an absent source catalog here.

Kind 21 uses ordinary resource glue, exact cleanup-frame receiver identity,
abilities/effects and ordered field types. Tuples are metadata: frame results,
cleanup receivers, resource fields, operands, arrays and tagged/borrow payloads
cannot contain a tuple value. Canonical uniqueness and the existing work and
table bounds remain mandatory. The lowerer selects this private admission while
copying the already-checked graph; ordinary source types retain their old route.

`assemble`, `assemble_project` and `assemble_artifact` keep aggregate admission
disabled. This addition changes no artifact encoding and does not authorize
aggregate restoration, project linking, whole-body MIR, C output or an executable
cleanup trace. Those owners must preserve the mode and current-source proof
through explicit follow-up contracts. Authored Windows evidence and the current
remaining work are recorded in
`docs/development/candidate-generic-aggregates-20261105.md`.

Validation. `tests/module_unit_cleanup_mir.fy` passes 4/4 and rejects thirteen malformed
handoffs: out-of-range drop operand or operand type, removed caller authority, removed
nonempty cleanup frame, changed provider module, reordered field ordinal, removed nested
drop ability, duplicate qualified operand, recursive field cycle, invalid Result
alternative, exceeded work budget, non-void cleanup result and duplicate canonical
descriptor. Receipt: `docs/ops/evidence/candidate-module-cleanup-mir-20261010.json`.

Invariants and limits. types 1..4096, frames 1024, operands 16384, drops 16384, fields
1024, effects 4095, abilities 7, recursion depth 64, work 1048576.

Sequencing ownership. Drop sites retain semantic request order in this cleanup-only
fragment. The separate owned-body CFG consumer attaches them to blocks and orders
cleanup with defer/Region operations before its private C handoff. This fragment
holds no ordinary bodies or CFGs and does not by itself establish their provenance.
The bounded aggregate single-buffer C path has its own native Windows values/drop
trace evidence below. Project linking, restoration, source-bound identities, cache
and actual command acceptance remain separate; this is not the full compiler.


Owned body consumer (2026-11-05). The ordinary detached body lowerer can consume
an aggregate-enabled cleanup owner without erasing instance kinds or scalar tuple
identity. Its verifier rejects tuple IDs in executable expression, local, call,
field and callable type lanes while preserving instance layout coordinates.
Private aggregate cleanup admission is still required; no artifact/project gate
or cache encoding changes. This body acceptance does not establish CFG sequencing,
C emission, executed cleanup, source-authenticated identities or restoration.
The aggregate implementation plan owns the Windows acceptance evidence.


Single-buffer C consumer (2026-11-05). Kind-20/21 concrete layouts and constructors
use the existing checked body/CFG C owner, including kind-21 structural cleanup
and inert record arrays. Kind-22 tuples never receive a C value spelling.
The aggregate plan records native Windows execution of values, nested cleanup,
defer, partial construction propagation and loop exits under the existing private
consumer policy. Partitioned emission, source-authenticated project linking and
restoration/cache/Atlas/actual CLI are still required. This does not open ordinary
or artifact cleanup admission, promote a C ABI or change compiler warning flags.
