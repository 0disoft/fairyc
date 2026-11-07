# Region source-selection bounds, 2026-11-07

The optimized Go-seeded full Fairy CLI reached project phase 2, stage 3,
code 9 while emitting the 646-source current compiler. The earlier source and
import admissions had passed. `module_unit_region_demands::collect` still
rejected more than 512 sources and indexed a fixed 512-element selection mask.

Source selection now uses `candidate_limits::source_files()` and an owned,
zeroed byte buffer sized to the admitted source count. The private `selection`
helper rejects empty, reversed, out-of-range and over-limit selections before
allocation; allocation failure returns code 11. Retained generic and cleanup
contexts still check their source indices before marking the same mask. The
body-checker's source stamps use the shared source limit as well.

`TestCandidateRegionSourceMask` compiles the actual private production helper.
It checks seven valid selections at 1, 512, 513, 646 and 1024 sources, including
the first and last selected bytes, and five invalid selections including 1025
sources. Every mask byte is checked. Windows low and optimized passed in 1.63
and 1.71 seconds respectively; the command completed in 3.808 seconds.

`TestCandidateRegionSourceConsumersCheck` checks both actual production modules
and their dependency closure through the Go frontend. It passed in 5.633
seconds. This supplements the small helper probe, which deliberately excludes
the complete checker dependency closure.

```text
go test -mod=readonly ./internal/cli -run '^TestCandidateRegionSourceMask$' -count=1 -v -timeout 3m
go test -mod=readonly ./internal/cli -run '^TestCandidateRegionSourceConsumersCheck$' -count=1 -v -timeout 3m
```

The logs are `.cache/current-region-source-mask-small-20261107.log` and
`.cache/current-region-source-consumers-20261107.log`. These checks do not prove
whole-compiler emission, native self-build or a fixed point. Those require a
fresh seed containing these source changes. Native Linux execution is deferred.

No public API, SDK ABI, package-owner range, dependency, DB or CI runner policy
changes. The mask adds one bounded owned allocation; its lifetime is tied to
the collection call. This replaces a source-count rejection, not a measured
performance bottleneck.
