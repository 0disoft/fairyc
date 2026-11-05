# Private candidate project disk cache v0

`examples/lexer_candidate/src/project_cache_disk.fy` owns the store and drives
`windows_rooted_file.fy` for handle-relative I/O, reusing `sha256_block`
compression for its own digest. It transports raw `FYCACHE0` and explicit
aggregate-family `FYCACHE1` candidate bytes
between processes and generations inside one caller-supplied root. It admits no
checked compiler owner and returns no reuse decision: every buffer it hands back
is still unverified, and the public SDK and CLI contracts are unchanged.

A returned candidate is never authority. The store exists so raw private cache
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
size), a trailing byte past the recorded size, an unrecognized envelope marker, or a
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

The explicit aggregate unit family (2026-11-05) adds opaque `FYCACHE1` envelope
transport alongside `FYCACHE0`. The raw store accepts exactly those two markers;
it does not parse a key, grant aggregate admission or fall back between semantic
readers. All content hashes, retained-directory I/O, bounds and publication/
repair behavior above apply to both. Unit and whole-project readers retain their
own exact marker/member and fresh-source contracts. See
[the unit cache contract](candidate-package-unit-cache-v0.md) and the aggregate
implementation plan for Windows execution evidence; Linux remains deferred.

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

## Private check cache CLI

`project_check_cli` and `project_check_options` expose the coordinator through a
private opt-in `check --cache-root <dir> [--cache-status]` route. Without
`--cache-root` the command runs the existing uncached `evaluate` path and still
accepts only the optional `--locked` flag, so the default check and the Go product
CLI keep their current behavior and no default cache location or enablement exists.

The root is parsed as a nonempty, NUL-free directory that must not look like
another option. A missing, empty, option-like or NUL-containing path, a duplicate
root, a repeated `--cache-status` or `--cache-status` without a root are usage
problems that exit 2; an internal allocation or argv decoding failure exits 4. The
root is opened with a private opt-in `disk::Limits` of 64 entries, 1 GiB total,
256 MiB per record and a 10 minute temporary age, which are the private command
policy rather than the measured product defaults.

`evaluate_cached` keeps the loader/lock and coordinate/input projection gates in
front of the cache: a load, lock or projection failure returns the original error
owner exactly as the uncached path does, and only the post-projection `Complete`
selection changes. A missing root, an environment initialization failure and any
lookup, admission, restoration or storage failure fall back to the cold
`build_complete_reporting` path, so a cache error is a disposable miss rather than a
source diagnostic. `--cache-status` adds exactly one stderr line derived from the
private `Observation` (`cache: hit`, `cache: cold stored`, `cache: cold uncached`
or `cache: skipped`) and leaves the ordinary stdout and stderr contracts untouched.
`module_signatures` accepts the complete 8192-declaration catalog, rejects the
8193rd with code 10 and keeps the 128 parameter and 32768 aggregate parameter unit
caps, and the C callable index reuses the same `project_callables()` bound as the
project metadata. Two option tests pass Go-native and candidate C development/release. The
isolated HEAD plus four command paths passes ordinary and locked checks with
518 files / 6377 MIR functions. A Go-frontend-built current authored dispatcher
passes 21 fresh-process observations: cold storage, verified hits, silent hits,
corruption repair, missing-root fallback, body invalidation, unchanged error
diagnostics and imported generic/Vec reuse. The first generic fixture omitted
its explicit workspace member and was corrected before the passing run.

The candidate-generated full command image remains unverified. Diagnostic-only
C tracing passes all preparation histories (3084149 total work), then exhausts
the unchanged 8388608 project ledger during body output at work 8229686 with a
189380 next charge. C body-work reduction, build/run cache connection, current
complete fixed point, per-package mixed reuse, product defaults, Linux/full suite
and same-head CI remain open. No default cache or public Go CLI change. Receipt:
[candidate-check-cache-command-20261014.json](../ops/evidence/candidate-check-cache-command-20261014.json).


## Explicit aggregate whole-project restoration, 2026-11-05

`project_cache_artifact.encode_aggregates`, `restore_aggregates`,
`to_disk_aggregates` and `from_disk_aggregates` admit only the aggregate Complete
family. Existing ordinary entry points remain ordinary-only. The whole-project
aggregate envelope is `FYCACHE1` / `FYCKEYS1` / `FYPRJ001`; unit envelopes also
start with `FYCACHE1` but require their different `FYPUNIT1` first member.
The semantic readers require exact member types and never fall back between
families. Ordinary `FYCACHE0` / `FYCKEYS0` / `FYPRJ000` bytes are unchanged.

`project_cache_lookup.collect_aggregates` changes only the lookup domain's final
version byte from zero to one. It retains the same authenticated compiler,
profile, SDK, exact package instances, original module paths and resolved imports.
This creates a different route for the same sources under the aggregate mode;
ordinary lookup cannot accidentally reject/delete a valid aggregate generation.
Neither route nor a valid content hash establishes reuse authority.

`project_cache_keys.collect_aggregates` retains the checked Complete mode and
fresh complete source/workspace facts. Its specialization projection validates
kind-20/21 aggregate instances as well as SDK instances and concrete frames
against the original source-derived specialization set. Tuple kind 22 is metadata
only. The existing canonical ordinary/generic package-key algorithms and public
interface hashes are unchanged; the private ledger marker binds their mode.

Restoration frames the complete envelope before reconstruction, uses the named
aggregate project decoder and current-source Complete factory, then derives all
current keys and compares every ledger preimage. Source drift, wrong families,
contradictory keys or any failed verification returns no Complete. Source,
projection, artifact, key, link, disk-I/O and scan meters retain separate bounded
histories across candidates; errors remain disposable cache misses. Publication
still requires a fully checked Complete and uses the unchanged retained-directory
no-replace/repair/retention owner. Actual CLI cache selection remains a separate
implementation step; this API adds no implicit aggregate command admission.

Windows evidence is recorded in the generic aggregate implementation plan.
Linux, full CLI/self-build, distribution and same-commit hosted acceptance remain
separate requirements; stable API, SDK ABI, DB, dependencies and runners do not change.
