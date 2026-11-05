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


Partitioned C consumer (2026-11-05). The explicit private `emit_aggregates` entry
requires aggregate-enabled, non-project cleanup IR and revalidates body/CFG proof
before sharing the ordinary bounded header/literal/body-unit preparation owners.
All-frame and entry-reachable modes preserve structural drop dependencies and
contiguous frame ranges. Only a complete Emission escapes; the checked input
remains borrowed and retains its admission mode. Ordinary partition/scoped/test
routes keep their project gate, and kind-22 tuples remain metadata-only.
The aggregate plan records 120 Windows low/optimized native executions with fixed
values and cleanup traces. This does not admit aggregate project linking, source
identity restoration, artifact/cache/Atlas/SDK or actual command execution; those
remain required follow-ups. No stable C ABI or compiler-warning policy changes.


Source-bound unit identity consumer (2026-11-05). Aggregate-enabled typed inputs
must preserve the exact source-prepared descriptor prefix. Concrete kind-20/21
rows select a completed original-source node from ordered scalar tuple arguments;
kind-22 metadata has no specialization row. The existing identified-unit factory
checks every generic cleanup frame against the source identity of its exact
receiver type, preserves specialized call endpoints and rejects late unselected
Vec/aggregate types. The aggregate plan owns the actual Windows source/Flow/field
oracle and current/work-boundary results. This does not authorize linked project
identity, artifact restoration, cache/Atlas/SDK production or actual CLI admission.


Linked aggregate project consumer (2026-11-05). The explicit
`assemble_project_aggregates` factory re-enters the same verifier with both project
work accounting and aggregate descriptors enabled. Ordinary project and artifact
constructors retain their old non-aggregate mode. The linker remaps kind-22 first
and nonzero second edges, selects kind-21 resource glue and coalesces cleanup
receivers by the canonical global type. Source units may omit a globally known
kind-21 storage recipe only when the existing liveness traversal proves no body
carries it, matching the existing ordinary/Vec rule. Selected recipes still match
rebound fields, frames, abilities and effects before complete publication.
Original-source aggregate identity rows survive project type/frame merging and
normal named partitioned C emission. The aggregate implementation plan owns exact
Windows values/drop traces and ordinary graph/link regressions. This is not
Complete interface publication, artifact reconstruction or cache/Atlas/SDK/CLI
admission, and does not change any artifact encoding or stable ABI.


Detached aggregate artifact reconstruction (2026-11-05). The explicit
`assemble_aggregate_artifact` constructor resets producer work and runs the same
cleanup verifier in aggregate mode. Ordinary `assemble_artifact` remains closed.
Kind-20/21 identity snapshots retain a source specialization; kind-22 tuples do
not. Receiver identities must still match the exact resource instance, and all
coverage, uniqueness, work and provenance module bounds remain enforced.

`FYCLEAN1`, `FYPROG01` and `FYPRJ001` distinguish aggregate cleanup, program and
project payloads from ordinary `FYCLEAN0` / `FYPROG00` / `FYPRJ000`. Other row and
body encodings retain their existing order and limits. Encoders select the new
markers only from an aggregate-enabled verified context. Named
`decode_aggregates` routes require those exact markers; ordinary decoders never
fall back or infer aggregate admission from node kinds. Framing preflight precedes
nested allocation, then cleanup, body, metadata, SDK/state and identity/provenance
verification run before a detached owner escapes. Ordinary bytes keep their old
markers and layout; old readers reject aggregate payloads rather than treating
them as compatible ordinary artifacts.

A decoded project is still `Restored`, not `Complete`, a cache hit or a current
source identity. The ordinary current-source planner remains aggregate-closed;
new payloads cannot bypass that gate. Original-source matching, rebasing, fresh
body/Flow and interface admission, persistent cache and actual command integration
remain separate required work. This changes private artifact encoding only, not
stable public source syntax, SDK ABI, MIR opcode, cache default or target maturity.
The aggregate plan records current Windows roundtrip/refusal evidence; Linux,
executed reconstructed aggregate C and same-commit hosted acceptance remain open.


Current-source aggregate restoration (2026-11-05). The explicit
`project_compiler_core::restore_complete_aggregate_sources` route authenticates
the complete retained provenance against current bytes, package coordinates and
import providers before admitting any executable owner. It rebuilds the aggregate
source plan and independently prepares current signature/layout metadata. Both
planning histories enter the same source-work ledger; producer planner ordinals
are never accepted as current identities. Ordinary restoration remains closed to
aggregate payloads, and the aggregate route requires aggregate mode explicitly.

Kind-20/21 type identities resolve to their original declaration kinds 4/6 and
exact ordered scalar argument tuples. Kind-22 tuple tails carry a second type
edge through checked-type matching; their handles are metadata, not executable
or public value authority. Module/owner rebasing preserves the aggregate cleanup
mode and keeps all type/frame IDs, operand IDs and source spans intact.

The existing fresh body/Flow, call, cleanup, metadata, whole-project coverage and
SDK/state admission gates run before regenerated public interfaces and the same
checked owner escape together as `Complete`. Kind-21 resource recipes now receive
the same current-source field/ability/effect/receiver comparison as ordinary and
Vec resources, including the existing unused-recipe liveness rule. The source and
link ledgers each retain their existing 16 MiB work bounds; artifact verification
retains its independent aggregate transport-work bound. A valid detached packet
with changed resource abilities or exchanged instance identities is still not
current-source authority.

The aggregate implementation plan records the Windows same-source C/interface
comparison, rebased execution and refusal evidence. This private factory is not a
unit/disk cache hit or actual CLI/Atlas/SDK integration. No ordinary artifact
format, default cache policy, public source syntax, SDK ABI, runner or target
maturity changes here. Linux and same-commit hosted acceptance remain separate.
