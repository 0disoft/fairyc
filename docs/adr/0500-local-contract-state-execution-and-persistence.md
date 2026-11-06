# ADR 0500: Bounded local contract-state execution and persistence

## Status

Accepted for an experimental Go developer path on Windows and Linux. Windows execution
is verified; native Linux execution is deferred. Chain/VM selection remains **UNDECIDED**.
This does not select a production database, a chain SDK, RPC, consensus or deployment.

## Decision

Verified contract-state MIR may execute one attempt through the existing
[persistence port](0376-contract-state-persistence-port.md). Verification precedes even
snapshot access. The existing transaction overlay retains limits, read-your-writes,
borrow invalidation and managed lifecycle. Only a commit exit submits the complete
sorted overlay. Rejection, trap, verifier failure and execution errors submit no writes.
Trap panics propagate after rollback. Conflict/layout/backend errors remain explicit;
the runner performs no retry. This is Go execution of verified state MIR, not generated
C using a disk store or on-chain execution.

`LocalPersistence` is an internal implementation of the port for one existing, trusted,
local directory. Creation and reopening are separate explicit operations. Creation
requires an exclusive new lock file. Reopening requires both authority files and never
initializes missing/corrupt state. Interrupted creation requires explicit operator
inspection; no automatic reset or repair is permitted.

Every operation acquires a nonblocking exclusive OS lock on a stable opened lock-file
identity, then rereads the complete current state. Windows uses `LockFileEx`; Linux uses
`flock`. Contention returns `ErrPersistenceBusy` without retry. The layout/version guard
and publication share the lock. A nonempty mutation or append-only upgrade increments
the version once; no-op publication preserves it. Version exhaustion rejects.

The private `fairyc.local-state/v1` record retains the exact ordered layout identity,
version and sorted unique physical entries. Canonical JSON bytes followed by their
SHA-256 digest reject damage, inconsistent layouts, duplicate/unordered entries,
unknown fields and alternate encodings. The digest detects corruption, not malicious
rewriting by someone who owns the directory. Full-state limits are mandatory, separate
from execution limits: at most 16 MiB encoded state and 65,536 entries, plus an explicit
per-value bound. Updates rewrite the complete bounded state; this is not an unbounded
or high-throughput database design.

All store files and temporary publication use an opened directory root and fixed flat
authority names. Final-root and authority-file symlinks/nonregular files reject;
opened file identities are checked. The caller must trust ancestors and directory
ownership. Hostile directory replacement, hard-link attacks and network filesystems
are outside this prototype's authority model.

## Publication and failures

A complete staging file is written, synced and closed before same-directory rooted
rename. The rename is the publication decision. There is no fallible post-publication
flush that could turn known success into a generic failure. Normal pre-publication
errors preserve the old bytes and remove staging. A process killed before rename can
leave an ignored `.tmp-*` file; a killed process after successful publication leaves
the new complete state observable on reopen. These are process-restart guarantees.
Directory-entry persistence after power loss, hardware retention, rollback attacks,
backup/recovery and ambiguous remote commit handling are not certified. This prototype
does not discharge the production durability requirements in ADR 0376.

The local lock is released when each operation ends or the process exits. There is no
transaction body or callback while holding that lock, no RPC and no user reentrancy
inside publication. A separately invoked writer during execution may win; the original
attempt then fails its version/layout guard. Caller-visible retries remain governed by
[ADR 0372](0372-explicit-caller-visible-contract-state-retry.md).

## Validation boundary

Windows tests cover exact byte preservation on pre-publication fault, data corruption,
missing-state failure, separate-process lock exclusion, forced exit immediately before
and after rename, reopen, two-writer comparison, append-only layout changes, deletion,
limits, version exhaustion and symlink rejection. MIR lifecycle tests cover commit,
reject, trap, verifier failure, backend errors and conflict without retries. Linux
replays belong to the combined queue, not to Windows evidence. No target flag, default
CLI, Fairy source grammar, generated-C persistence or production chain is admitted here.

## Developer command

`go run -mod=readonly ./tools/dev/fairyc-state <init|execute|snapshot>
<existing-store-directory> <source.fy>` accepts one bounded explicit source file with
exactly one verified experimental state transaction. It uses the existing
`CheckExperimentalState` admission, not a new parser or unverified MIR constructor.
The command basename/module relationship follows the ordinary explicit-source rules.
Creation never replaces a store; snapshot does not execute the transaction body.

The named developer policy is 16 MiB full-state bytes, 65,536 entries and 64 KiB per
value; the execution overlay allows 1,024 writes, 1 MiB write bytes, 64 KiB per value
and 1,024 encoded map-key bytes. These fixed prototype limits are printed by `--help`.
JSON reports identify `fairyc.local-state-run/v1` and the exact layout hash. Successful
execute reports the receipt and reads; init/snapshot report committed state. An
execution report is its attempt's observation, not a later snapshot of concurrent state.

Exit codes are 0 success, 2 invalid invocation, 3 source/admission rejection, 4 local
I/O/backend/output failure, 6 transaction rejection, 7 conflict/layout drift/busy and
8 language trap. The command converts only the named language trap panic to a trap
report; unexpected implementation panics remain panics. A report-output failure after
publication is explicitly described as possibly committed and must not trigger an
automatic retry. There is no default compiler CLI integration or target flag.

## Explicit native graph developer mode

The developer command additionally accepts exactly
`<init|execute|snapshot> <directory> <source.fy> --engine native-graph --profile low|optimized`.
This selects [ADR 0503](0503-experimental-source-state-graphs.md) source admission
and the existing sealed Windows graph/CAS path. The original three-argument
Go-MIR form, report v1 and single-cell admission remain unchanged. There is no
interpreter fallback. Invocation/profile validation precedes source/store I/O.

The native mode requires Windows amd64. Every call reloads the bounded regular
source file through the existing opened-root/identity checks and re-enters the
compiler and native handoff. Execute builds one C image before opening the
store; init/snapshot admit the source and graph without building or executing
its body. Storage and transaction policies are identical to the original mode.
Creation remains explicit and exclusive; reopening never initializes a store.

Native reports use `fairyc.local-state-run/v2`, retaining the action/layout and
outcome/snapshot fields and adding engine, selected profile, exact verified
body hash and framing-program SHA-256. These identities report fresh admission;
they are not a cache, trusted Complete provenance or artifact restoration.
Execute takes one snapshot, runs actual generated C once and performs at most
one CAS. Existing exit codes retain their meaning; trap conversion catches
only the named state trap. An inconsistent publication acknowledgement retains
both its outcome and actual receipt, with failure text and exit 4. Report-output
failure explicitly warns of possible publication. Neither path retries.

Interrupt cancellation reaches native process execution and is checked before
store access and CAS by the existing one-attempt owner. Native construction
retains the existing bounded compiler timeout; cancellation is checked before
and after construction. Successful CAS acknowledgement remains success when
cancellation arrives during publication. The synchronous storage port is not
made cancellable. Source helper and root borrow/view/copy admission are added by
ADRs 0504/0505. Native Linux, dynamic byte/text admission, ordinary CLI selection
and Complete/artifact/cache integration stay
in the remaining queue. No SDK, ABI, dependency, default limit or runner changes.
