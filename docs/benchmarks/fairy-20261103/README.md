# Fairy C build footprint: 2026-11-03

At clean commit [`71d34364`](https://github.com/0disoft/fairyc/commit/71d34364cfa1c1cfe33b8bdc262484332ba44535),
one Fairy-only local sequence measured retained build outputs plus cache on Windows/amd64. After
the 20th source edit, the measured footprint was 143,837 bytes (140.5 KiB). It covers build
outputs and cache, not the whole project, allocated disk space, or the compiler installation.

## Result

| Category | After edit 20 (bytes) | Maximum sampled (bytes) | After clean (bytes) |
| --- | ---: | ---: | ---: |
| Binary | 132,608 | 133,120 | 0 |
| Shared runtime cache | 11,229 | 11,229 | 11,229 |
| Separate debug files | 0 | 0 | 0 |
| Incremental files | 0 | 0 | 0 |
| Other intermediate outputs | 0 | 0 | 0 |
| Local cache | 0 | 0 | 0 |
| Retained temporary files | 0 | 0 | 0 |
| **Total** | **143,837** | **144,349** | **11,229** |

The maximum sampled total is 144,349 bytes (141.0 KiB). It is the largest retained total after a
sampled command between the cold start and clean, not peak disk or memory use during a build.

There were 22 build/run pairs before clean: one cold, one no-op, and edits 1–20. All 22 program
executions passed output verification. The clean sample removes build outputs and runs no program,
so its `verified=false` status in the receipt is intentional; only the shared runtime cache remains.

## Conditions

| Field | Value |
| --- | --- |
| Snapshot date | 2026-11-03 KST; raw receipt interval 2026-11-02 15:41:07–15:41:29 UTC |
| Source | Clean `71d34364cfa1c1cfe33b8bdc262484332ba44535` |
| Host | Windows amd64 |
| Build tools | Go 1.27.1; MSVC 14.51.36231; Windows SDK 10.0.26100.0 |
| Languages and profile | `fairy` only; `dev` |
| Workload | `many-modules`; configured `modules=16`, producing 18 Fairy source files |
| Sequence | `edits=20`, `repetitions=1`, `timeout=3m`; one cold, one no-op, then 20 edits |

This is one local sequence on the listed host and toolchain. It does not establish run-to-run
variance or a universal footprint guarantee.

## Measurement boundary

The benchmark sums logical file lengths in its known project output and cache roots after each
command. The total excludes allocated filesystem blocks, compression or sparse-file effects,
hard-link sharing, deleted transient files, peak disk use during a build, peak memory,
operating-system caches, and runtime speed. It is not a peak build-size or physical-disk-use
measurement.

The standalone Fairy compiler is inventoried separately at 24,239,104 bytes. MSVC and the Windows
SDK must be installed separately, and their installation files and the Go build environment are
excluded from the project footprint. The raw receipt records those toolchain inventories separately.

## Reproduce and evidence

From the repository root at the matching clean checkout, use a new output directory:

```text
go run -mod=readonly ./tools/dev/fairyc-bench run --output .cache/footprint-NEW --languages fairy --profiles dev --workloads many-modules --modules 16 --edits 20 --repetitions 1 --timeout 3m
```

The [raw receipt](receipt.json) is the unmodified record of this run. See the [benchmark
methodology](../README.md) for the measurement contract.

This run did not include Rust, Zig, or C. The [cross-language `ec446d2d`
report](../latest/report.html) remains a separate historical snapshot and was not regenerated.
