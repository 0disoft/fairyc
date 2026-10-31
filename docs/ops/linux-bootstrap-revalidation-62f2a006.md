# Linux bootstrap acceptance after frontend memory reclamation

## Evidence and scope

This record transcribes an external Linux execution report and its two CSVs
provided by the maintainer. Their acceptance results, ELF identities and compiler
overlap intervals were checked for consistency. It is not a new local Linux run
or a hosted-CI receipt. The accompanying raw-results ZIP was not supplied for this
review; the original report describes logs, overlays and reproduction scripts in
that archive. The measurements are evidence, not changes to product contracts.

Validated source: `62f2a006436e3ef054955b1681d7041fa87892aa`.
Source ZIP SHA-256:
`c7bae70a1a9a23fd42d98db5963cf3b16e34582a6046a3c804f98816182a78cf`.

The host was Debian 13.3 amd64, Linux 6.18.57, glibc 2.41, Go 1.26.10,
GCC 14.2.0 and Clang 18.1.8. CPU quota was four cores, GOMAXPROCS four and
cgroup memory limit 4 GiB. The provided offline Go and Clang bundles were used.
All native inputs and images were regenerated. No product-source patch was needed;
the report records all 6,437 final source-manifest entries matching.

## Unchanged default acceptance

Each invocation used the original bootstrap, GNU time, and a single cgroup read
at start/end. No compiler wrapper, periodic sampler, input hashing/copying or Go
overlay was attached. Builds ran serially and used fresh native objects/runtime.

| Compiler | Profile | Attempt | Whole bootstrap seconds | Exit | Shared 60-second C deadline |
| --- | --- | --- | ---: | ---: | --- |
| GCC | optimized | first | 70.88 | 0 | passed |
| GCC | optimized | repeat | 68.64 | 0 | passed |
| GCC | low | first | 41.81 | 0 | passed |
| Clang | low | first | 29.95 | 0 | passed |
| Clang | optimized | first | 54.79 | 0 | passed |

Whole-bootstrap time includes frontend, emission, memory preparation, C compilation
and publication. The shared C deadline starts after memory preparation. Exact C
wall time and worker count were not observed in these five attempts. Separate
instrumented values must not be substituted for those missing observations.
GCC optimized produced the same ELF SHA-256 on both attempts:
`4137091c678ea79f86520d58eaa8e45fd72d5b7dd978c9cb50600fa3a0676cca`.
All four default manual images also passed `--help`. No 600-second diagnostic ran.

## Separate memory and compiler observations

After default acceptance, a separate GCC optimized build used an observation-only
Go overlay and a small C compiler wrapper. Worker budget, GC/scavenge behavior,
target, strict flags, optimization and timeout remained unchanged.

| Metric | Before memory preparation | After memory preparation |
| --- | ---: | ---: |
| Go HeapAlloc, MiB | 551.50 | 41.08 |
| Go process VmRSS, MiB | 1,081.71 | 99.88 |
| cgroup memory.current, MiB | 3,332.36 | 2,349.77 |

Memory preparation took 0.096795 seconds. After writing the C files, available
memory was 1,749,262,336 bytes (1,668.23 MiB), exceeding the unchanged 1,280 MiB
two-worker threshold. Two workers were selected and independent compiler calls
overlapped. The first two calls overlapped for 4.141 seconds.

This manual build's observed C interval was 56.176 seconds, including C file
writing, runtime, parts, linking, post-link input hashing and ELF validation.
Whole-bootstrap time was 76.69 seconds. The observed margin was only 3.824 seconds;
this does not guarantee completion under other memory or CPU loads.

Go HeapReleased was 1,848,008,704 bytes after preparation. It describes released
heap at that point, not bytes reclaimed by this invocation or process RSS.
cgroup values include other processes and file cache. Sampled process-RSS sums
double-count shared pages; neither those sums nor individual child maximum RSS
are exact peak physical-memory consumption.

## Regression results

The four original default manual images passed 144 CLI calls, 15 Go reference
comparisons and three generated-C executions each: totals 576, 60 and 12.
The repeated identical GCC optimized image was not counted twice. Expected
nonzero exits for invalid inputs, stale locks, failed tests and traps were checked
against the existing oracle rather than counted as unexpected failures.

Coverage included `check/build/run/test/new`, IR-cache cold/hit/corruption/change
invalidation, stale locked checks, test replay on cache hits, stdin/EOF/timeouts,
trap locations/receipts, binary output, argument boundaries and C output execution.
Clang selection overlays explicitly routed the actual build/run/test companion
to Clang. The companion's sequential FYCPARTS1 route remains distinct from the
Go bootstrap's bounded parallel build.

Full `TestLinuxFileCLI` also passed GCC and Clang low/optimized. Observed test-input
C intervals were GCC 22.855/57.294 seconds and Clang 10.161/35.079 seconds. These
intervals start at the first compiler wrapper and end at linker-child completion;
they exclude initial C writes and post-link hashing, unlike the manual interval.
Each selected and actually ran two workers. All 20 generated C files, runtime C
and header matched the previous same-kind input hashes; test/manual inputs differ.

The report also records passing `go mod verify`, `go vet`, partition identity and
callback execution with both compilers, worker/memory/error/cancellation tests,
stable native runtime-cache/trap/failure-category checks, and original/prepared
workspace `check` and `check --locked`. Prepared's initial missing lock returned
the expected exit 7, followed by successful check and locked check.

## Reproduction and limits

Use the [matching bootstrap pair](../architecture/candidate-linux-build-v1.md#matching-bootstrap-pair)
to build fresh Go binaries and prepare the Linux workspace. From that prepared
workspace, run this unchanged command for each compiler/profile:

```sh
../fairyc-bootstrap experimental build-workspace \
  --target x86_64-linux-gnu --cc gcc --profile optimized \
  --output build/fairyc-gcc-optimized --locked
```

Repeat GCC optimized once with fresh native objects. Build preparation always
creates private objects for large partitions. Preserve the original acceptance
result before adding observers. The full external procedure and logs are described
in `fairyc-linux-revalidation-62f2a006-results.zip`; this record does not claim to
have executed that archive's integrated reproduction script.

No full `go test ./...`, complete fixture/sanitizer/ABI sweep, native Windows or
Ubuntu run, clean-machine distribution, self-host acceptance, repeated load
distribution or compiler execution-throughput comparison was performed in this
external validation. No dependency, API, DB, target, warning, timeout, worker-budget,
repository-hygiene or CI/runner policy change was made to obtain these results.

## Supplied evidence identities

These SHA-256 hashes identify the maintainer-supplied files reviewed for this
record. They are not included as raw logs or normative source documents.

| File | SHA-256 |
| --- | --- |
| fairyc-linux-revalidation-62f2a006-report.md | `eb87e95d17daec787147e8a45865b4b952323b665aa6bdee47a3bc0d135c86f3` |
| fairyc-linux-revalidation-62f2a006-comparison.csv | `94f16833b66da24d61919170d61b58bd5e8e5d76f172cd5d14fd9df8cfefbd68` |
| fairyc-linux-revalidation-62f2a006-parts.csv | `326d2db4b44f8d7e0e7b0003f5272213584af5c969864f6fd72ada45944929f7` |
