# Candidate project C partitions v0

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
