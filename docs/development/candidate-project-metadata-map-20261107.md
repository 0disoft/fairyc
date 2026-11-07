# Project metadata field map initialization

Implementation: `8c2d5acb` (worker `2ca0cb9b`). The metadata builder previously
zero-filled every unit's complete field map before visiting layouts, then
overwrote mapped cells. Ordered field visits now append their final target
directly. Gaps and unused tails still receive separately charged zero writes;
out-of-order visits and repeated targets retain their existing meanings.
The completed map remains dense and has the same cells and ranges. This does
not reduce the retained map size or establish a measured RSS improvement.

The field visit already charges for writing a target. Avoiding its redundant
initial zero removes one charged write per directly appended cell. Work stays
linear; this is a constant-factor reduction, not a new complexity class.
The 1048576 metadata work bound and all layout, field, callable, parameter and
name limits are unchanged. A conflicting nonzero target still returns 1001;
it is never replaced by a different target. Bounds checks subtract lengths
before adding an index, and input IR remains borrowed and unchanged.

## Windows results

`TestCandidateProjectMetadataMap` passes both low and optimized native profiles
in 113.530s at package level. It compares complete metadata and map results
against the original builder, including layout/name/field/callable/parameter
data, map choices and ranges. It also checks holes, unused tails, reversed
layouts, same-target repeats, conflicting targets, zero fields, input stamps,
invalid indexes and the exact work bound. Production and fixture hashes are
held fixed throughout both native executions.

| Case | Original work | New work | Saved |
| --- | ---: | ---: | ---: |
| Ordered map, 8192 cells | 16384 | 8192 | 8192 |
| Reversed map, 8192 cells | 16384 | 16383 | 1 |
| Real IR, 8 units of 256 fields, ordered | 14312 | 12264 | 2048 |
| Real IR, 8 units of 128 fields, reversed layouts | 11240 | 10984 | 256 |
| Real IR, zero fields | 8248 | 8248 | 0 |

Evidence: `.cache/parallel-project-metadata-map-20261107/.cache/project-metadata-map/windows-native.log`.
Tested production SHA-256:
`e15d9836533553bb043d6c56f863f3aefe9d0e65e4c24009dcf756e0d445a151`;
fixture SHA-256:
`f86ed65f605983a73e29402080f73f64fad48d06267eed2b2a132fbaf908581a`.
The worker also passes go vet, gofmt and diff-check for this scope.

Integrated source `check` and `check --locked` pass with 755 source files and
9530 MIR functions in 13.413s / 13.338s. Both use the same source snapshot;
the resulting lock is copied back only after successful checks and hash
comparison. Evidence: `.cache/current-project-metadata-map-source-check-20261107.log`.
A fresh optimized full CLI seed builds from 205 C parts and answers `--help`
with 7726 output bytes; the build-only test passes in 131.862s. Evidence:
`.cache/current-project-metadata-map-seed-20261107/build.log`. This is not
an authored frontend self-build result.

## Acceptance boundary

The preceding normal full-source retry still refuses in phase 3/stage 0/code
1002 with zero output. A frozen-input diagnostic identifies metadata preparation
(33002), without establishing which internal bound failed. That is not a pass
or proof that this optimization resolves the normal failure. A new full-source
normal retry is running in `.cache/current-selfbuild-metadata-map-20261107`.
Normal self-build, fixed points and
current-source full parity remain pending. Linux native checks stay batched;
CI, clean-machine validation and push are unexecuted. API, DB, SDK ABI, runner
and source-of-truth contracts are unchanged.
