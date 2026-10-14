# Private candidate project disk cache v0

`examples/lexer_candidate/src/project_cache_disk.fy` owns the store and drives
`windows_rooted_file.fy` for handle-relative I/O, reusing `sha256_block`
compression for its own digest. It transports raw `FYCACHE0` candidate bytes
between processes and generations inside one caller-supplied root. It admits no
checked compiler owner and returns no reuse decision: every buffer it hands back
is still unverified, and the public SDK and CLI contracts are unchanged.

A returned candidate is never authority. The store exists so raw `FYCACHE0`
envelopes can survive a process boundary, and it never answers a reuse question
on its own. `project_cache_artifact.restore` still re-runs current complete
source admission and exact key preimages before any cached bytes become a
compiler owner, so a corrupt, stale or hostile generation only costs a miss. The
caller supplies an existing dedicated root Directory that is not a reparse point,
plus mandatory `Limits`.

`Limits` requires 1..4096 retained entries, a total of at least 8 bytes and at most
1 GiB, a per-record size of 8..256 MiB that is not larger than the total, and a
nonzero temporary age in Windows 100 ns FILETIME units. Recognized generations and
stage files are collected against those bounds, including oversize recognized
records that the sweep must evict, and invalid limits fail the open with kind 1.

Routes use the same 81-byte `fylookup0-sha256-<64 lower hex>` shape as the lookup
keys. A published generation leaf is 144 bytes,
`fyproject0-<64 hex route>-<64 hex whole-payload SHA-256>.fyc`, so the name binds
both the pre-compilation route and the complete stored bytes. Staging leaves are
`.stage<generation>-<0..15>`, created exclusively. The digest reuses the
compiler's SHA-256 block compression under its own budgets: the source SHA stream
keeps its 96 MiB cap unchanged, while a stored record is bounded to 256 MiB
(268435456) and an aggregate call to 1 GiB of work (1073741824). Enumeration keeps
a separate 16 MiB history.

`load` scans the retained handle once, orders regular generations and stage files
newest first by modification time with a name tiebreak, and tries matching routes
in that order. Each candidate opens through the same root handle, its observed
size must be at least 8 bytes and at most the per-record limit, and bytes are read
in chunks of at most 64 KiB. A premature EOF (a zero-byte read before the recorded
size), a trailing byte past the recorded size, a missing `FYCACHE0` marker, or a
whole-content digest that disagrees with the leaf name marks that generation
corrupt and the lookup moves to the next one.
Allocation failure or exhausted work fails the whole lookup instead of returning a
prefix.

`put` validates the policy and route, rejects an oversize or non-envelope buffer,
derives the generation name and creates an exclusive stage. It writes the payload,
flushes, publishes the stage to the final leaf without replacement, flushes again
and closes by consumption. The published generation is then passed to the bounded
sweep, so a successful put never evicts its own result.

Because publication never replaces, a repeated identical put collides with the
existing leaf. When the colliding bytes compare equal the store keeps the original
identity and modification time and then runs the same bounded sweep; retaining that
entry without eviction is a tested case, not a universal guarantee. When the
colliding generation is corrupt the store retains the metadata from the failed
read and reopens the leaf with delete sharing denied to compare file ID, time, size
and attributes before removing it; only then does it retry the no-replace publish.
A changed, writer-held or reparse object survives untouched and the put fails
instead.

`sweep` enumerates the same root under its separate 16 MiB budget, recognizes
regular generations and stage files, and removes expired staging leaves oldest
first while skipping a stage whose writer still holds it and any stage younger than
the temporary age. It then evicts oldest retained entries until the entry count and
byte total fit the limits, never touching the generation just published. Success
reports the retained entries and bytes. There is no instantaneous global bound, no
lock and no atomic snapshot, so a concurrent store can change the directory after
the observation.

A failure before publication leaves existing valid generations untouched, because stages
are exclusive and consuming close deletes them. A failure after publication, or
during the sweep, may leave a complete candidate without establishing retention
bounds; callers treat any cache error as a miss. The store makes no physical
power-loss claim: flush and consuming close bound ordering only.

The Windows-only private ABI behind `windows_rooted_file.fy` adds a no-replace
publish adapter and an absolute FILETIME clock, and enumeration restarts the same
retained handle instead of mutating Directory state in memory. C marshals native
calls only; Fairy owns sequencing, budgets and cleanup.

Five transport tests cover the store: independent SHA vectors with exact padding
and work accounting, a 131073-byte binary round trip with reopen and idempotent
collision metadata, corruption fallback with metadata-guarded repair, entry and byte
retention with oversize and exhausted early failures, and an aged orphan stage
beside a live stage with writer-held publication obstruction. The seven rooted-file
regressions run with them on the same three paths, Go-native and candidate C
development/release, and the root and locked source checks pass 512 files / 6347 MIR
functions. Evidence:
[candidate-project-disk-cache-20261014.json](../ops/evidence/candidate-project-disk-cache-20261014.json).

## Checked disk owner admission

`project_cache_artifact.from_disk` and `to_disk` connect the private store to
checked compiler owners without widening the store contract. `from_disk` derives
the exact loaded-source, SDK and compiler route, inventories the retained root
once, then tries matching generations newest to oldest. Every candidate re-enters
the existing `restore` path with no shortcut: full current source, body, Flow,
state, SDK and table coverage plus exact instance-ordered v4/v0 key preimages.
Only an opaque `core::Complete` escapes, so a stale or hostile generation is never
lowered into replacement IR.

A permanent admission failure rejects only the leaf observed during that
successful read, matched by file ID, modification time, size and attributes, so a
concurrent replacement, active writer or reparse leaf survives. Bound and
allocation codes 10/11/1002/1003 stop the attempt and leave valid generations
intact. Every attempt shares one monotonic `DiskWork`: lookup, scan, projection,
source and key histories are 16 MiB each, I/O and artifact are 1 GiB each, and link
is 8 MiB. `Candidates` and `Candidate` are detached private owners usable only with
the same live `Store`; the numeric cursor index stays monotonic and every visited
row is charged.

`to_disk` checks the caller's original `Complete`, encodes and routes it, then
delegates immutable disk publication. No default CLI cache is enabled yet. The
three checked disk restore tests pass on all three profiles, Go-native, C
development and C release, and their stdout matches byte-for-byte at 61,968 bytes
per run (SHA-256 8acc48cf048cbee1998f7efd3dffcb6a3f39155e2f5b0bcded824b97819e7947). The five disk
transport regressions also pass on all three profiles and write no stdout. In the
development and release profiles the restored Vec executions build at exit 0, run
at exit 42 and leave empty streams.
Receipt: [candidate-checked-disk-owner-20261014.json](../ops/evidence/candidate-checked-disk-owner-20261014.json).

## Shared project compile coordinator

`examples/lexer_candidate/src/project_compiler_cached.fy` adds one private
`build` in front of the existing cold path and the checked disk store. It takes
the caller's already-verified `Store` and `Environment`, the caller's same live
`visibility::Resolved` loaded owner with its derived `input::Input` sources and
`coordinates::Coordinates` table, plus the reporting sink and observation outputs.
It attempts `artifacts::from_disk` first: a returned `Complete` is a checked disk
hit and is passed through opaquely, while an error records its phase and code and
falls back to the cold path.

The cold path is the existing `core::build_complete_reporting`, run under its own
independent source and link budgets, and the coordinator returns its exact error
or report unchanged. After a successful cold build it attempts `artifacts::to_disk`;
a store failure never discards the result, which is returned with the store phase
and code recorded. The fallback is limited to `from_disk` lookup, admission,
storage and budget errors, including a plain miss, a rejected leaf, the bind and
allocation codes and an exhausted I/O budget. A non-empty reporting sink at entry
is not a cache case: the coordinator delegates straight to
`core::build_complete_reporting`, so the existing core contract owns the rejection
and the cache path does not route around it.

`Observation` is private and never changes command output or compiler diagnostics.
`path` is 0 unattempted, 1 cold or 2 checked hit; `stored` reports a successful
publication; `lookup_phase`/`lookup_code` and `store_phase`/`store_code` carry the
failed lookup and store phases.

Two authored tests in `examples/lexer_candidate/tests/project_compiler_cached.fy`
pass on all three profiles, Go-native and candidate C development and release, with
the functional sources, tools and images re-hashed against their actual values.
`cold_then_disk_hit_skips_lowering_and_budget_failure_falls_back` observes a
storing cold build, a checked hit that leaves the cold compiler source and link
histories at zero while current-source admission is charged to `DiskWork` (the
fresh IR work check stays zero), and an exhausted cache budget that falls back to
a full cold build with empty reports.
`cache_miss_preserves_exact_cold_compiler_error` reproduces the cold error phase,
stage, code, module index, span, expected and actual values exactly on a miss. The
coordinator's source scope is 513 files / 6363 MIR functions: the unlocked root
check updates the lock and the locked rerun verifies it, both at exit 0 across two
instances. The coordinator is not connected to a CLI (cliConnected=false), and CLI
and default policy, per-package mixed reuse, the latest complete fixed point and
platform promotion remain open. The previous disk-owner checkpoint keeps its 512
files / 6361 MIR functions. Receipt:
[candidate-cached-compiler-20261014.json](../ops/evidence/candidate-cached-compiler-20261014.json).
