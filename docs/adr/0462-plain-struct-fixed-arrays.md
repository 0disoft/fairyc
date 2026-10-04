# ADR 0462: Plain struct fixed arrays

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-05
- Related: ADRs 0415, 0416, 0441, 0455, 0457, 0459, 0460, 0461

## Decision

Admit `Token[N]` when Token is an actual nonempty, acyclic plain struct with
bool/integer leaves, including nested plain structs. Exact nominal identities
remain distinct even with equal layouts. The bound is still 1..=4096. Resources,
raw pointers, borrowed values, tagged fields, nested arrays, array fields and
foreign array signatures remain excluded.

List construction evaluates each element once in order. Repeated construction
`[template; N]` evaluates the initializer once and creates independent element
values. Array assignment, by-value parameters/returns and Option/Result payloads
retain ordinary copy semantics. These whole-array copies cost proportionally
to the array contents. Checked `array[index]` loads a whole independent value;
a named mutable array permits whole-element replacement. `array.len` remains
constant metadata. Direct indexed field stores are not admitted in this unit;
use a mutable typed view or an explicit checked element reference to update
stored fields.

```fairy
var Point[4] polygon = [Point { .x = 0, .y = 0 }; 4];
let ref mut Point[] vertices = ref mut polygon;
let ref mut Point[] middle = vertices[1..3];
for (let ref mut Point point in middle) { point.x += 10; }
```

Named fixed arrays yield contextual shared/exclusive views and checked element
references. Immutable named view bindings support slicing and the existing
copied/shared/exclusive loops. Direct named-array loops use the same private
whole-array descriptor: copied/shared items retain a shared owner loan;
exclusive items require a mutable array and hold the owner through loop exit.
Subview and lifetime rules are unchanged. Local or by-value array references
cannot escape, and simultaneous mutable sibling loans remain unsupported.

## Implementation

HIR's array-shape predicate recognizes nominal shape without claiming a valid
struct definition. Semantic admission and the MIR verifier independently check
the actual plain field graph. MIR construction, loads and stores check nominal
array operands against actual local definitions, so forged operand annotations
cannot relabel a scalar or unrelated array. No MIR instruction or wire schema
is added. Existing bounds checks continue to dominate access.

The interpreter clones struct values when constructing, loading and storing
array elements; repeated elements do not share mutable nested struct state.
The C emitter retains early scalar-array declarations but places nominal array
wrappers after complete struct definitions. Existing pointer/length struct views
borrow the original contiguous backing storage, with no Vec allocation.
Tagged array transport uses the actual element-struct layout.

## Verification and compatibility

A stored Fairy matrix covers nested fields, i64 values, list/repeated creation,
independent element/array copies, parameters/returns, Option payloads, element
references, shared/exclusive/empty/offset views, all three loop item kinds,
break/continue and early return. Compiler checks include repeated analysis,
identical encoding, MIR round-trip, interpreter execution and both Windows and
Linux C emission profiles. Mutations remove struct definitions, add borrowed
fields, forge scalar operands/receiver types and remove bounds; verification
and encoding reject them. Semantic negatives cover nominal mismatch, forbidden
field graphs, array limits, shared writes, owner writes, escaping locals and
sibling loans.

Windows dev/release executes the matrix and four bounds traps. The cached source
session checks cold lowering and a warm all-hit result. A separate compiler
runs public check/build/run for the matrix and the two-module polygon example,
with exact exit/stdout/stderr checks and four source diagnostic classes. Source,
SDK/compiler and EXE hashes with raw receipts are recorded in
`docs/ops/evidence/struct-array-20261005.json` and companions. Related array/bool/
struct regressions and Go vet are checked. Linux execution remains deferred.

Language /20, SDK stage0.86/source/exports, runtime /0, public ABI, MIR/wire
schemas and target maturity are unchanged. Compiler executable identity isolates
cached generations. No CI, installation, benchmark, push or release is included.
