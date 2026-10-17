# Candidate project C partitions v0

## Reuse authenticated SDK plans across native tests (2026-10-17)

The test session now retains its authenticated `sdk::Plan` alongside the same
immutable IR and proven CFGs. Construction lowers and proves the IR, charges the
same graph counts, then authenticates SDK declarations against that retained
source snapshot. There is no separately replaceable plan or mutable owner API.
A new invocation or changed program creates a fresh session.

Every image starts its preparation meter with the recorded graph-plus-SDK work
before building callable/frame indexes. The original combined preparation cap
and whole-project ledger therefore account for the same work on cache hits and
misses. Entry reachability, declarations, bodies and output remain per-test.
`Emission` retains neither the SDK plan nor a reference to the session.

Source check and the cached-versus-uncached native regression pass, including
exact C bytes, partition ranges, selected counts, work/retries and output use
after session destruction. The new real-CLI fixture has three tests spanning
scalar and owned Buffer vectors and passes Go native execution.
See [implementation evidence](../ops/evidence/candidate-sdk-plan-session-20261017.json).

The current-source build covers 512 source files. The native Fairy C frontend
compiled them into one 41,766,237-byte C packet, and both full `file_cli`
compiler profiles were built from that packet without invoking the Go frontend.
Go still supplied the version metadata oracle and remains the bridge that turns a
C packet into a native Windows build.
Development and release `--version` match the authenticated text, and the
candidate project lock, `examples/lexer_candidate/fairy.lock`, is unchanged.

The release profile regenerated the whole packet byte for byte, and the optimized
Windows PE rebuilt from that regenerated packet matches the first generation
exactly: 5,907,968 bytes, identical SHA-256, no excluded bytes. `--version`
matches again.

Both profiles then ran all seven Go comparison suites: 544 exact comparisons
each, 1,088 together, with no differences. Per profile those are check 169,
run 50, cli 30, semantic 98, list 89, native 30 and examples 78. Recorded skips
are unchanged: one check fixture that needs a packageGraph workspace adapter and
two run fixtures with no passing native `main::main` entry. The native suite has
no skips, and the `lexer_candidate` example fixture is checked separately by the
full `file_cli` bootstrap fixed point above. See
[final verification](../ops/evidence/candidate-sdk-plan-reuse-final-verification-20261017.json).

This checkpoint covers the Windows full `file_cli` dispatcher. Non-Windows targets
and the separate stdin lexer protocol are not covered. Stage times were single
observations under uncontrolled load, not a performance benchmark. No product API,
SDK ABI, work limit, product timeout, database or CI change.

## Reuse validated CFGs across native tests (2026-10-17)

`module_project_complete_c::prepare_tests` consumes one `Complete` into a
`TestSession`. Its private `module_project_c_owner::Lowered` retains both the
immutable program and the graphs built by `flow::lower_all`. That constructor
performs the full IR proof and graph checks before publishing the session.
There is no mutable owner access, independently replaceable graph set, disk
cache or reuse across CLI invocations.

Each test borrows that session while generating its own names, SDK plan,
indexes, entry reachability and C output. The returned `Emission` owns its
buffers and metadata and retains neither graphs nor a session reference.
Per-test SDK planning is an observation at this checkpoint; the session above
retains one authenticated plan and charges its work to every image.
Cached graph counts still enter every image's preparation/project work ledger;
all work, frame, output and child-process bounds remain unchanged. With no
selected tests, the CLI reports directly without creating a session.

The scoped diagnostic measured 52.266 seconds in C generation across two tests:
36.328 seconds in four IR proofs and 15.797 seconds in SDK planning. CFG lowering
(17.781 seconds) includes two of those proofs; inclusive stages must not be
summed. Reuse therefore targets the validated graph preparation, while SDK
planning remained per-test at this checkpoint. These are single
development-profile observations under uncontrolled load, not a general
benchmark. See
[stage profile](../ops/evidence/candidate-native-c-emission-stage-profile-20261017.json).

Legacy single-buffer rendering lives in the existing `module_unit_c_function`
source unit, preserving its function bodies and the existing emitter API while
leaving room under the per-source borrow-analysis limit. This adds no source
module and does not increase the 512-file manifest bound.

The focused native regression compares cached and uncached output, rejects invalid
entries/frame bounds, destroys the session before reading its output again, and
checks a fresh session for changed source. Source check and this regression pass.
See [implementation evidence](../ops/evidence/candidate-native-test-cfg-session-20261017.json).

The current-source development compiler was rebuilt from a 512-source,
41,758,634-byte packet produced by the C candidate, with no Go frontend fallback.
Two end-to-end native-test comparisons exactly match the authenticated prior Go
stdout, stderr and exit status for identical inputs: six trap cases (including
continuation after traps), and SDK vector resources. The original lock is
unchanged. The trap fixture took 137.479 seconds versus 255.977 seconds before
CFG-session reuse, a 46.3% reduction in this single observation under uncontrolled
load. This checkpoint does not rebuild the release compiler or repeat full CLI,
fixed-point, platform and native-test matrices. See
[scoped verification](../ops/evidence/candidate-native-test-cfg-reuse-verification-20261017.json).

## Reuse checked IR across native tests (2026-10-17)

`module_project_complete_c::emit_test` borrows the caller's checked `Complete`
and returns an owned `Emission`. Preparation and C body rendering read that IR
only during the call. The result owns its name table, indexes, selected
frame metadata and C buffers; it retains no IR reference or resource owner.
Invalid entries and exhausted budgets still return no output.

The native test loop retains one `Complete` for all selected tests. Each test
gets fresh reachability selection, C output, a packet and an isolated child
process. It no longer rebuilds the checked program between tests. Packet bounds,
framing, write/sync/close handling, child deadlines and reports are unchanged.

Existing `partition*` APIs still consume and retain their IR in a `Bundle`,
which now contains the owner alongside its independent `Emission`. Existing
bundle accessors expose that emission's bytes and metadata; the single-buffer `render` APIs
are unchanged. The regression fixture emits two different entries, repeats the
first entry, rejects an invalid entry, then consumes the original `Complete`.
It compares the independent output against the owning test partition byte for
byte, including frame intervals, selected counts and work totals.

Scoped verification rebuilt both compiler profiles from one 512-source,
41,744,870-byte packet emitted by the C candidate. Five native-test CLI
comparisons match Go's complete stdout, stderr and exit status, covering trap
recovery, optimized children, SDK resources and per-test stdin replay.

For the same six-test trap fixture, development compiler time changed from
456.4 to 256.0 seconds and release compiler time from 44.3 to 26.4 seconds.
These are single before/after observations under uncontrolled machine load.
At this earlier checkpoint, C preparation and native compilation still ran
separately for each test.
See [verification evidence](../ops/evidence/candidate-native-test-reuse-verification-20261017.json).

## Current whole-project work and bootstrap (2026-10-16)

The shared `candidate_limits::project_work()` ledger is now 16,777,216. Each
independent preparation/body meter retains its 1,048,576 bound; output-part byte,
frame, packet and native object validation limits are unchanged. The complete
512-source compiler emits byte-identical 39,546,346-byte C packets across two
consecutive generations, and the optimized native PE images are exactly equal.
Both images pass the current Go lexer differential (249 cases each) and seven
malformed framing cases each. See [fixed-point evidence](../ops/evidence/candidate-bootstrap-fixed-20261016.json)
and [whole-project admission](../ops/evidence/candidate-bootstrap-project-work-20261016.json).
Earlier dated 8Mi work observations below describe their original checkpoints.

## Source-identified project owners

`partition_project` and `partition_project_entry` consume the checked
`module_project_identified_ir::Program`. Preparation derives one owned name table
from that same linked owner. Specialized prototypes, definitions, calls and
cleanup glue use `fy_fn_g0_<digest>` and `fy_cleanup_g0_<digest>` consistently
across the common header and every body partition. Ordinary frames retain
numeric names. The bundle retains the linked IR and its source identities through
a single owner; it does not copy IR or borrow from two competing inputs.

Name collection and frame-contract matching retain their bounded work histories
and enter the existing project ledger. Invalid entry, partition size, identity
or exhausted work returns no bundle. Both selection modes retain the original
frame intervals and the same conservative call/cleanup closure as ordinary
partitioning. The legacy `partition` and `partition_entry` APIs remain available.

The packet verifier's explicit `identified=True` mode checks canonical names,
unique ordered definitions, exact prototype coverage and resolved references.
Its default numeric mode retains contiguous-frame checks. Large compiler-output
tests may opt into the private test verifier's `--native-program` with
`--isolated-native`: the Go compiler builds the same authored entry, whose full
streams are compared directly against both candidate profiles. This leaves the
product native-test report's 64KiB stream bound unchanged.

Receipt: `docs/ops/evidence/candidate-project-c-names-20261013.json`.

`module_unit_c_emit::partition` consumes a linked, owned project IR and returns
an owned `Bundle` only after complete frame coverage succeeds. This is private
candidate transport. The existing single-buffer `render` path retains its C
spelling and transaction behavior.

Preparation verifies the project IR, lowers checked CFGs, authenticates SDK
shapes and prepares the existing callable/body indexes. A guarded `fy_project.h`
contains aggregate definitions, runtime bridges, foreign declarations, external
frame prototypes and private cleanup helpers. Each body file includes this
header and defines a contiguous interval of original frame IDs with external
linkage. Exactly one interval defines the selected i32 entry wrapper and main.

Literal collection runs once using the existing exact-byte collision checks
and full SHA-256 ordering. The header declares `extern const uint8_t` storage;
`fy_literals.c` defines each unique payload once. Equal text and byte payloads
therefore keep their shared pointer identity across translation units, including
empty payload sentinels. CStr terminators remain part of the cooked payload.

Checked analysis, common declarations and each body attempt each retain the
1048576 work cap. Each output buffer retains the 2097152-byte cap. Both
preparation histories, successful body attempts and
failed attempts charge the existing 4194304 project ledger. A code-1002 body
failure may bisect its interval, to depth 12; a single frame, other error codes
or an exhausted project ledger reject. Frame intervals must cover every original
frame exactly once without gaps or overlaps. The bundle retains the validated
IR and exposes borrowed byte views, so failure cannot publish a successful
prefix. Filesystem publication and the native compiler driver remain separate.

`tools/verify-candidate-project-c-parts.py` parses exact, length-delimited test
packets, checks ordered frame coverage and unique literal definitions, and links
the actual separate C files with MSVC /Od and /O2. Fixtures cover shared nominal
cleanup/foreign calls, generic instances and literal pointer identity. A legacy
800-literal stress fixture retains its exact static-storage digest.

Receipt: `docs/ops/evidence/candidate-project-c-parts-20261011.json`.
The original project still includes unreachable SDK functions. Its first
partition attempt passes the earlier shared render-work failure and reaches
unsupported `std::fs::ReadDir` cleanup (code 1004, module 209, bytes
30613..30660). This observation does not establish original executable creation
or entry reachability. No stable CLI, SDK ABI, language version or CI changes.

## Entry-scoped function and cleanup selection

`partition_entry` retains the same checked project IR and original interval
coordinates, but emits only the conservative call/cleanup closure of the entry.
`module_unit_c_reachability` starts with the entry frame and records each visited
frame once. It resolves original ordinary/specialized call coordinates through
the existing body index. Foreign calls retain the original callable identity;
selection does not grant SDK authority or relax the emitter's ABI checks.

Checked CFG drop sites seed canonical cleanup type IDs. Resource cleanup frames,
resource fields with glue, and Option/Result success/error payloads join the same
fixed point. Cleanup bodies may discover further ordinary or specialized calls.
Each frame/type is queued once; slot preparation, call lookup, CFG visits and
cleanup edges charge the existing preparation work cap. Missing identities,
allocation failure and exhausted work retain 1001, 1003 and 1002.

Only selected frames have body definitions/prototypes, and only selected cleanup
types have drop helpers. Type/literal definitions and foreign declarations remain
conservative over the retained project. Original intervals still cover every
frame coordinate; excluded bodies can leave an include-only file. The new
`project_selected_frames` accessor distinguishes emitted frames from the total.
The all-frame `partition` entry point remains available.

Two Release native cases and four /Od-/O2 separate-file executions pass. A six
frame fixture emits four frames and three nested cleanup helpers, omits an unused
resource/helper pair, and observes outer-before-inner cleanup (`99,7,`). The
all-frame literal fixture keeps its earlier packet digest and pointer identity.
The original 236-file project still rejects with code 1002, module 0, span 0..0;
the failing preparation/project-budget phase has not yet been isolated. No
original bundle or executable is published. Receipt:
`docs/ops/evidence/candidate-c-entry-reachability-20261011.json`.

## Bounded preparation phases

Analysis/SDK authentication/indexes/reachability and common C declarations now
have separate 1048576 counters. Each completed history is folded into the same
4194304 project ledger before its local counter resets. Resetting a counter
never discards work from the project total. Output limits, body splitting,
original identities and transactional bundle publication retain their contracts.

A cache-only observation isolates the earlier failure: analysis used 787768
work and selected 3222 frames; aggregate declarations brought the combined
counter to 1016330. Literal collection then exhausted 1048576 work with only
107055 header bytes retained. The 2097152 output limit was not reached.

After phase separation, the unchanged 236-source original project emits the
complete entry bundle: 3558 original frames, 3222 selected definitions, 28 body
files, no retries, 3233248 total work, a 655466-byte header and a 149489-byte
literal source containing 775 unique definitions. Body sources total 19288554
bytes; the largest is 1432228 bytes. Exact packet parsing verifies unique ordered
frame definitions, one entry wrapper and shared numeric literal storage.

Two Release native cases retain their previous packet bytes exactly; their four
unchanged /Od-/O2 executions are reused. Original executable linkage/execution is
a separate next gate. Receipt:
`docs/ops/evidence/candidate-c-preparation-scopes-20261011.json`.

## Independent reachability history

Index/SDK preparation and reachability now use independent 1048576 work meters.
Both completed histories enter the existing 8388608 whole-project ledger exactly
once before common declarations begin. Both all-frame and entry-only partitioning
use these boundaries. The previous 4194304 ledger mentioned above is historical.

The current 355-file dispatcher uses 545600 work for index/SDK preparation and
625962 for reachability: each fits its own bound, their sum exceeds the former
shared bound. Aggregate declarations then use 343438 work; literal collection
exhausts the shared common-declaration meter at 1048576. No bundle is published.
Literal collection is the next bounded phase to isolate.

Two Release native cases pass with a current-source verification bootstrap whose
MSVC build allowance is 240 seconds. The product's 60-second allowance timed out
and remains unchanged. Exact packet bytes match the previous cases, allowing
their four /Od-/O2 C executions to be reused. Source check passes 362 files /
4679 MIR functions. Receipt:
`docs/ops/evidence/candidate-c-reachability-scopes-20261012.json`.

## Independent literal-pool history

Aggregate declarations, literal-pool construction/storage and remaining SDK,
foreign, prototype and cleanup declarations each retain a 1048576 work meter.
Each completed history is folded once into the unchanged 8388608 project ledger.
The 2097152-byte output-buffer cap and transactional publication remain unchanged.

The complete current 355-file dispatcher now emits its full entry bundle in
67.27 seconds: 29372117 packet bytes, 5200 selected frames, 45 body files,
1030902 header bytes, 507881 literal-source bytes and 1226 shared literals.
Body sources total 27832988 bytes; the largest is 1302387 bytes. Packet parsing
checks frame uniqueness, one entry wrapper and the shared literal definitions.
The Go-produced optimized seed emitted this packet; linkage and execution of
this complete C-part compiler and consecutive-generation comparison are next.

The literal-pointer-identity Release native case passes with the verification
build allowance; its exact packet retains two prior /Od-/O2 C executions.
Source check passes 362 files / 4679 MIR functions. No product timeout, API,
SDK ABI, DB or CI change; no push. Receipt:
`docs/ops/evidence/candidate-c-literal-scopes-20261012.json`.
