# Scanner thread CPU-time comparison

Date: 2026-09-25. Windows AMD64, MSVC 14.51.36231, Go 1.26.4,
`GOMAXPROCS=1`, optimized `/O2` profile. The
[raw report](evidence/vector-lexer-four-image-cpu-time-20260925.json) retains
the four image hashes, fixed execution order, one warmup rotation and twelve
measured rotations. Each execution performs 250,000 scanner rounds and checks
the reporter, completion count and work checksum. There are no retries,
excluded samples or adaptive stopping.

The scanner-only timing option records both QPC and current-thread
`GetThreadTimes` user-plus-kernel time. Its 100 ns units are quantized: the
1,000-round smoke reported zero CPU ticks for three of four lanes. The longer
rotation has nonzero CPU observations in every measured lane, but the spread
still exceeds the predeclared 10% stability limit.

| Paired ratio to up-front C | QPC median | QPC span | CPU median | CPU span |
| --- | ---: | ---: | ---: | ---: |
| Explicit-index Fairy | 0.96730 | 61.22% | 0.95235 | 56.32% |
| Private vector-loop Fairy | 1.01224 | 66.01% | 1.03923 | 40.88% |

Here `span = (maximum ratio - minimum ratio) / median ratio`, with the warmup
excluded. The up-front C lane itself spans 98.16% by QPC and 70.37% by CPU
time. CPU-time observation therefore does not rescue this run as an 8% runtime
admission result. The point estimates alone cannot establish that either Fairy
form is faster than, equal to, or within 8% of independent C.

The CPU-time option adds instrumentation to the benchmark support source.
The instrumented C executables are 184,832 and 185,856 bytes; both Fairy
executables are 201,728 bytes. These sizes are **not** the uninstrumented
product-size comparison. The unchanged QPC-only four-image smoke still builds
the smaller C image at 184,320 bytes and both Fairy images at 201,728 bytes,
or 2,663 bytes above the integer 8% ceiling. Neither this run nor the size
profile experiment admits read-only `Vec<T>` iteration as stable source.
