# Independent C scanner storage A/B

Status: measurement pending. This compares two builds of the same independent
C scanner, not Fairy against C. The baseline allocates token and diagnostic
arrays for `length + 1` entries. The alternate starts at eight entries and
doubles capacity, matching the private Fairy `Vec<T>` storage strategy.

Both optimized MSVC images scan `let n=12345; /*open` for 250,000 rounds per
execution. Each execution must match the Go lexer reporter, completed rounds,
and checksum. QPC covers scan, storage and cleanup inside the repeated loop;
process startup and final output are excluded. One warmup pair precedes twelve
measured pairs. Execution order alternates between the two C images. There are
no retries, exclusions, adaptive stops or input changes. The opt-in smoke uses
one pair of 1,000 rounds and makes no timing claim.

Record every raw QPC sample, frequency, order, source and executable digest,
EXE byte count, Go and MSVC versions. Before interpreting a timing difference,
require `(max - min) / median <= 0.10` for both lane times and the paired
geometric/preallocated ratio. Even a stable A/B only identifies storage cost
for this short scanner sample. It cannot establish the full-scanner 8% Fairy/C
runtime or EXE verdict because the C reference still has missing lexical and
budget behavior, and Fairy diagnostic payload ownership differs.
