# Pinned-thread C/Fairy scanner cycle comparison

Date: 2026-09-26. Child source HEAD: `beb6b15f7979ca8fa4f42d6e33d1584fd0cda126`.
The [raw receipt](evidence/scanner-same-image-cycles-20260926.json) records
two optimized Windows AMD64 MSVC combined images, one for the explicit-index
scanner and one for the compiler-private vector-loop scanner. Each image
contains the independent C workload and the corresponding Fairy workload.
The calling thread was pinned to its current logical processor before timing;
the two images selected processor-group indices 3 and 4 respectively. C and
Fairy alternated first position within each image. One warmup pair preceded
twelve measured pairs at 250,000 rounds per call, with no retry or exclusion.
Every call returned the Go-oracle work checksum of 9,000,000. The test also
recorded QPC ticks and `QueryThreadCycleTime` cycles for the calling thread.

| Fairy form / independent C | QPC paired median | QPC relative full span | Cycle paired median | Cycle relative full span |
| --- | ---: | ---: | ---: | ---: |
| Explicit index | 0.95432 | 50.77% | 0.97887 | 48.12% |
| Private vector loop | 0.95269 | 109.40% | 1.04561 | 39.35% |

The predeclared stability limit is a relative full span of at most 10%.
Neither clock meets it. Pinning removed thread migration within each image,
but did not control ambient contention or processor frequency. [Microsoft's
API contract](https://learn.microsoft.com/en-us/windows/win32/api/realtimeapiset/nf-realtimeapiset-querythreadcycletime)
also warns that cycle counts must not be converted into elapsed time because
the underlying CPU timer behavior varies. The medians therefore do not prove
a speed gain, regression or the cumulative 8% runtime gate. Repeating the same
host experiment without a new control is unlikely to resolve this gate.

Both instrumented combined images were 152,576 bytes. They cannot decide the
standalone executable-size gate, and this experiment did not use `/Os` or
measure clean check/build cost. The private vector source form stays closed.
