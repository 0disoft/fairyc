# Identical-source scanner build control

Date: 2026-09-27. This protocol is fixed before collecting its first receipt.
It calibrates the host and the existing fresh-package, optimized MSVC build
measurement after the current indexed/vector five-pair comparison exceeded its
10% paired-ratio stability heuristic. It does not measure a language-feature
cost.

The two lanes use the same checked-in explicit-index scanner and benchmark
main, selected through the same private SDK package graph. The harness builds
one retained warmup pair excluded from the decision, then five measured pairs
with alternating lane order. Each build includes the existing load, metadata,
source analysis, verified MIR, C emission, runtime support assembly and
optimized MSVC native phases. Each pair must retain equal scanner source
hashes, specialization counts, generated-C lengths and linked EXE lengths;
the MSVC version must remain fixed. Raw phase times and every pair, including
outliers, are retained.

For each measured pair, divide clone-lane total time by index-lane total time,
where total is `check_ns + emit_ns + support_ns + native_ns`. The null control
passes only if the median of the five paired ratios lies in `[0.98, 1.02]`
and `(maximum ratio - minimum ratio) / median ratio <= 0.10`. A failed
control means this host/run cannot validate an indexed/vector build-cost
comparison. A passing control permits a separately recorded candidate run;
it does not prove the candidate's check or build cost, the 8% C-relative
runtime gate, or cold stable CLI startup. Do not discard a pair or rerun to
select a passing control.
