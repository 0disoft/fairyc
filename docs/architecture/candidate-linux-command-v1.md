# Private Fairy Linux command transport v1

`examples/lexer_candidate/src/linux_command.fy` prepares owned UTF-8 strings for
the private Linux process adapter ([process transport](candidate-linux-process-v1.md)).
The encoder itself adds no process execution, shell invocation, environment
selection, PATH search, native ABI symbol or candidate CLI dispatch. The Windows
command encoder and default Go compiler remain unchanged.

## Representation and admission

`build(program, arguments)` returns an owned `Arguments` resource. Its byte view
contains `program + NUL`, followed by each argument and its NUL, in input order.
The separate count includes `argv[0]`, which is exactly `program`. An empty list
still has count one; empty arguments, including a trailing empty argument, each
contribute one NUL. The private native adapter consumes both the explicit count
and byte span, validates their agreement, and appends the final null argv pointer.
Adjacent NULs alone cannot distinguish an empty argument from an end marker.

No quoting or escaping is applied: spaces, quotes, backslashes, newlines and shell
metacharacters remain literal bytes. Non-UTF-8 and embedded NUL reject. Program
paths must be nonempty and at most 4095 UTF-8 bytes, consistent with the private
Linux host-path slice. `path` applies the same checks and returns an owned,
NUL-terminated path for executable or working-directory transport. Relative paths
are preserved; existence, permissions and cwd resolution belong to execution.

The private logical admission budget is 1,048,576 bytes:

`encoded string bytes including NULs + 8 * (argc + 1)`.

The pointer charge is for the declared Linux amd64 target and includes the final
null pointer. Allocation metadata, the resource and caller-owned input vectors
are outside that logical budget. The encoder allocates no pointer table; the
private process adapter allocates its native table separately.
The budget does not guarantee OS execution admission: environment bytes, native
per-string/aggregate limits and system configuration can still cause exec failure.
The private launcher preserves that error instead of truncating arguments.

All lengths, NUL/UTF-8 validation and aggregate charges are checked before output
allocation. Remaining-capacity subtraction prevents arithmetic overflow. Argument
count is bounded by its minimum byte/pointer cost before traversing the vector.
For program `x`, 116,506 empty additional arguments fit; 116,507 reject. One
additional argument of 1,048,549 bytes exactly fills the logical budget.

Output uses one `bytes::with_capacity` initialization for the exact encoded size;
appends fit without growth. No partial result escapes a failed validation or
allocation. Inputs remain borrowed and unchanged; the returned resource owns its
copy and automatically releases it. Time is O(B + A), where B is input bytes and
A is the argument count; output space is O(B + A).

Problems distinguish invalid input (1), the private bound (2), and allocation (3).
Argument zero denotes the program, later entries are numbered from one. NUL
errors identify their byte offset; malformed UTF-8 reports offset zero. An early
argument-count bound or output-initialization failure reports argument zero.

## Validation boundary

Three authored tests in `examples/lexer_candidate/tests/linux_command.fy` cover
byte-exact encoding and ownership, malformed input/path boundaries, and exact
aggregate byte/count limits. `TestLinuxCommandTransport` uses those same source
tests as one entry point, emits Linux and Windows C, and executes low/optimized
native images with the existing private Vec runtime on the current host. Linux
hosts use GCC; Windows hosts use MSVC.

Windows native execution validates the portable encoder, not Linux process
execution or kernel argument limits. Native Linux execution and injected allocator
failure remain unverified from this Windows host. No CI job or workflow is added.
The private adapter now covers start, cwd and exit/signal collection as documented
in [Private Fairy Linux process transport v1](candidate-linux-process-v1.md).
Bounded I/O, deadlines, process-tree cleanup and the candidate CLI bridge remain
future work.
