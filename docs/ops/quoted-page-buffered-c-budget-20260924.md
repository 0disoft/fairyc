# Buffered quoted-page C comparison (2026-09-24)

This repeats the [original quoted-page C comparison](quoted-page-c-budget-20260924.md)
after `pages::report` began collecting at most 4,096 decimal observations in an
explicit `std::bytes::Buffer` and writing them once. The C11 reference and
10,002-byte quoted ASCII input have identical SHA-256 hashes in both runs;
the Fairy page source hash changed. Each execution checked the exact 12,290
output bytes against the Go lexer oracle. This is one page-protocol workload,
not a full lexer comparison.

Both phases used Windows AMD64 MSVC `/O2`, three uncounted warmup pairs and
twelve alternating measured C/Fairy pairs. Each executable was built and
materialized once before timing. Process wall time includes startup, frame
parsing, scanning, output and collection; it excludes compilation and image
materialization. The two phases ran separately, so their absolute durations
are observations under different ambient scheduling, not controlled paired
before/after measurements. The independent C source and input are unchanged.

| Observation | Original page output | Buffered page output |
| --- | ---: | ---: |
| Median C process wall | 12.1715 ms | 12.3142 ms |
| Median Fairy process wall | 37.4546 ms | 20.5200 ms |
| Median paired Fairy/C ratio | 3.0454 | 1.7245 |
| Fairy/C ratio of medians | 3.0772 | 1.6664 |
| C executable | 146,944 B | 146,944 B |
| Fairy executable | 142,848 B | 142,336 B |
| Fairy/C executable ratio | 0.9721 | 0.9686 |
| Predeclared runtime stability verdict | `inconclusive` | `inconclusive` |

The buffered run has three Fairy samples above 44 ms while its other nine
are 19.3060–21.6019 ms. Two C samples exceed 36 ms while its other ten
are 11.1822–13.2792 ms. The C, Fairy and paired relative full spans are
206.9%, 126.6% and 186.0%; all exceed the predeclared 10% stability limit.
The raw samples and hashes are in the [buffered evidence](evidence/quoted-page-c-budget-buffered-20260924.json),
alongside the preserved [original evidence](evidence/quoted-page-c-budget-20260924.json).
Windows process CPU samples have coarse 15.625 ms steps and cannot isolate
the changed operation. Single-build durations cover different build scopes
and are not comparable.

The lower Fairy median and ratio are consistent with fewer Windows console
writes, but this experiment does not isolate that cause or certify an 8%
runtime result. The buffered Fairy image is 512 B smaller than the original
and 4,608 B smaller than the specialized C reference. The C reference does
not implement the full lexer or link the same support surface, so its size
only describes this restricted comparison. The full-application C-relative
release gate remains open. A useful next measurement would isolate scanning
from console output on equivalent workloads under controlled scheduling;
new language syntax does not follow from this result alone.
