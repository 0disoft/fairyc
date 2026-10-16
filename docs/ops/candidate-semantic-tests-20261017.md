# Candidate default semantic tests

The candidate's default `fairyc test` command executes checked owned body IR
through the existing verified CFG lowering. It never emits C or calls the
native build driver. Workspace loading, production lock handling, test-source
validation and canonical test selection use the same preparation as `--list`
and `--native`.

The interpreter admits `i32`, `i64`, `u8`, `u32`, `usize`, `bool`
and `void`: exact-width checked arithmetic, comparisons, bitwise operations,
value-preserving casts, scalar bindings and assignments, function and generic
calls, if/block expressions, short-circuit evaluation, while/classic-for loops,
break/continue, assertions, returns and frozen scalar `defer` arguments. Copy-only structures and fixed arrays
also support declaration-order fields, list/repeated construction, indexing and
length, nested field/index stores, scalar compound stores, copying, function
parameters/results and aggregate block/conditional values. Immutable per-test
cells retain value semantics; stores rebuild the projected path. Allocation
and copies consume the same step allowance and have a one-million-cell ceiling.
Runtime arithmetic reuses the constant evaluator while retaining distinct
language trap categories and unsigned host-width arithmetic.

Each test gets fresh values, local slots, captures, compound-assignment snapshots
and a one-million-step runtime work allowance. CFG operations, per-cell
allocation/copying and descriptor traversal count toward that allowance. The
call-depth limit is 256. Its numerical exhaustion point need not equal Go
MIR operation/block accounting. Traps fail individual tests and execution continues.
Capability or execution-limit rejection aborts with code 2 and no partial report.
Internal invariant/allocation failures return code 4.

Enum/Option/Result construction, exhaustive tagged switches, active payloads,
`Result.is_ok`, `?` and resource cleanup now execute in the interpreter. Cleanup
runs before reverse-order resource fields; traps stop that cleanup and fail its
test. Explicit burn follows the checked CFG without running cleanup.

Safe scalar/record references, mutable calls, scalar/record array views, slicing,
static UTF-8/byte literals, and copy/shared/mutable element iteration use frozen
place descriptors into per-test root cells. They never retain host pointers or
addresses into vectors. `source_location()` uses exact retained source paths
and UTF-8 geometry, including lexical generic-function locations.

Region operations, general raw-pointer execution and host/foreign calls remain
outside the implemented candidate interpreter slice. Native execution retains
its existing separate admission. Matching the measured stable CLI cases does
not establish universal Go interpreter or cross-platform equivalence.

Text and JSON execution reports share the existing writer. Default reports use
`engine: "interpreter"`, null target/profile fields and no captured-output fields. Existing
exact/file/directory/package/workspace selection, source locations, lock state,
exit codes and JSON schema versions are retained. Native reports keep their
existing target, profile and optional captured output.

The full dispatcher remains within its 512-source limit. Execution lives in
`module_unit_interpreter.fy`; aggregate cells and place descriptors live in
`module_unit_interpreter_values.fy`. Runtime arithmetic adapters sit beside the
exact-bit constant evaluator in `module_constant_arithmetic.fy`. Production
callers share the cast-admission rule in `c_abi_types.fy`; `scalar_casts.fy`
retains its compatibility wrapper for authored tests. This makes room for the
interpreter without increasing the source limit. Keeping it separate from the
test dispatcher also preserves each module's existing analysis work allowance.
The complete join exceeds 8,192 rebound frames at `windows_rooted_file`.
The private aggregate frame ceiling is now 16,384; original callables and
generic specializations retain their separate 8,192-entry limits. Project work
remains 16 MiB and independent body/unit work remains 1 MiB. Source, byte, type,
field, process stack, interpreter limits and deadlines are unchanged.
The enum verifier now checkpoints its prefix and each function into the existing
project ledger. Previously it retained every function's scan in the single
1 MiB unit meter and rejected a valid complete dispatcher. Every prefix/body
still has its original unit allowance; no work is discarded or counted twice.
The project ceiling remains 16 MiB. Exact aggregate admission, ceiling-plus-one,
unit exhaustion and unit-mode ledger preservation pass Go-native and both
candidate C profiles in `tests/module_unit_enum_work_budgets.fy`.
The Go packet decoder and Python C-part verifier share the 16,384-frame
ceiling. The emitted complete dispatcher has 65 C units, 7,566 selected
functions and largest original frame ID 8,236, explaining why the previous
8,192 decoder ceiling rejected it. Its C packet is 41,406,363 bytes.
No language, SDK ABI, stable schema or CI workflow changes are required.

Build with `tools/build-candidate-native-probe.py`, then verify with
`tools/verify-candidate-semantic-tests-cli.py --build-receipt <receipt>
--reference <Go compiler> --output <new .cache directory>`.
The verifier authenticates source/image hashes, copies only each candidate
compiler image without its native build companion, compares real reports with
Go, and checks locked workspace file bytes and modification times after every
invocation. All current verifier cases require exact stdout, stderr and status equality.
The existing native and test-list verifiers cover the shared report writer and
discovery regressions.

When the available seed still enforces the old frame ceiling, first build
with `--compiler-only --profile release`. This 450-source transport implements
only `--c-project-file`, fits the older compiler, and carries the current frame
ceiling into its generated image. Use that image as `--candidate` for the complete
dispatcher build. Both generations use the Fairy frontend; no Go frontend
fallback is involved. The seed receipt is marked `compilerOnly` and is rejected
by the full CLI verifiers.

Validation notes:

- Initial seed creation explicitly uses the Go workspace C emitter and the
  existing private host builder. Compilation of the final 512-source dispatcher
  then runs the authored Fairy frontend's `--c-project-file` transport. This is
  separate from a current-source consecutive-generation bootstrap fixed point.
- Earlier Go-authored seed diagnostics pass 33 CLI comparisons: 30 exact Go
  output/exit matches and three aggregate capability rejections. This seed
  diagnostic is separate from final candidate C-build validation.
- Runtime arithmetic/cast tests pass all three authored cases with Go-native
  and both candidate C profiles. The enum meter tests pass
  two authored cases with Go-native and both candidate C profiles; the packet
  decoder also passes exact 16,384-frame and ceiling-plus-one admission tests.
- Linux, the full compiler suite and consecutive Stage1/Stage2 equality are not
  rerun for this change. No CI workflow, public Go API, SDK ABI or database is
  changed, and no push or release is performed.

`tools/build-candidate-native-probe.py --reuse-frontend <previous output>`
rebuilds native images from a successful C emission. Reuse validates the full
current source graph, generated entry bytes, seed executable, production lock,
original invocation and C packet digest. The reused original entry path stays
in the source record, and the receipt explicitly records frontend reuse.
Changed sources and changed packet bytes pass rejection probes. This permits a
native-driver admission fix without repeating the expensive Fairy frontend.

Initial scalar implementation results (superseded by later additions below):

- The full 512-source Fairy frontend emits the complete 41,406,363-byte packet
  in 515.997 seconds. Development and release native builds pass in 34.413 and
  341.246 seconds using the authenticated packet and updated native driver.
- Default semantic execution passes 66 comparisons across both compiler profiles:
  60 exact Go stdout/stderr/status matches and six expected aggregate capability
  rejections. Tests execute with no adjacent native driver; locked fixture bytes
  and modification times remain unchanged.
- Test listing passes 178 exact comparisons. Shared native reports pass six exact
  comparisons (mixed text, mixed captured/located JSON, and empty JSON in each
  compiler profile). The full native CLI matrix is not repeated.

The complete source/image hashes, profile results and limited validation scope
are recorded in [candidate-semantic-tests-cli-20261017.json](evidence/candidate-semantic-tests-cli-20261017.json).
The initial scalar release CLI was
`.cache/semantic-test-20261017/cli-final-verified/release/fairyc.exe`, with its
matching `fairyc-candidate-build.exe` companion in the same directory.

## Copy aggregate extension

The three earlier record/array capability differences now require exact Go
reports in the CLI verifier. Added coverage exercises reordered fields, nested
record/array stores, repeated record elements, copy isolation across calls,
aggregate branch/block values, and bounds traps that continue to later tests.
The initial Go-authored diagnostic image passes 38 comparisons: 37 exact and
one explicitly retained resource-cleanup capability rejection. This diagnostic
is not proof of current-source candidate C emission; that full build is separate.

## Expanded value and cleanup execution

The source-authenticated Go-authored diagnostic image passes 98 strict CLI
comparisons, with no tolerated differences. The suite includes all former
record/array/resource boundaries, tagged values and propagation, cleanup traps
and order, scalar/record references and views, slices, long literals, element
iteration and lexical source-location geometry. It also checks matching
rejection messages for scalar/bool switch subjects. The updated Go MIR oracle
now drops discarded owned temporaries instead of skipping their cleanup.
These diagnostic results precede the separate complete candidate C rebuild.

## Preserve the complete source and work budgets

The complete C frontend rejected the expanded single interpreter unit with
phase 2/stage 3/code 10 at `rx_array_view`. Splitting the value/place helpers
keeps each unit within its existing 1 MiB analysis allowance. Canonical body
node/statement/local lookup now lives in the body model; its earlier module
remains a thin compatibility wrapper for authored tests. Production consumers
use the model directly, freeing one source slot for the interpreter value unit.
The complete dispatcher remains at 512 sources and all work limits stay intact.

## Final expanded candidate C validation

The authored candidate frontend emits the complete 512-source C packet in
447.574 seconds: 41,750,782 bytes. Development and release native builds pass
in 32.600 and 332.790 seconds. Source, stack, frame and work ceilings are kept.
The Go executable is used only for source-version metadata; Fairy frontend
C generation uses the candidate with no Go frontend fallback.

The final current-source images pass **440 exact comparisons**: 196 default
semantic reports, 178 test-list comparisons, 60 shared CLI invocations and six
native report comparisons. Every measured case requires identical status,
stdout and stderr; no capability rejection is tolerated as a match. Default
semantic cases run without an adjacent native driver and preserve locked
workspace bytes and modification times. `--version` matches the fresh Go
source-build oracle in both profiles.

Go `internal/mir` tests and focused temporary/linear-resource semantic tests
pass. The discarded constructor/returned-resource regression also matches
through real native execution: both test entries trap as `assertion-failed`.
Linux, the full native/compiler suite and consecutive Stage1/Stage2 fixed-point
equality are not rerun. Region/raw-pointer/foreign execution and independent
runtime work accounting remain the limits described above. No push or release
is performed.

The current release image and matching native driver are under
`.cache/cli-parity-rest-20261017/final-candidate-split-verified/release/`.
Full hashes, build invocations and comparison receipt hashes are recorded in
[candidate-cli-parity-20261017.json](evidence/candidate-cli-parity-20261017.json).
