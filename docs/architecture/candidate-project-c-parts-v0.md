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

Preparation and each body attempt retain the 1048576 work cap. Each output
buffer retains the 2097152-byte cap. Preparation, successful body attempts and
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
