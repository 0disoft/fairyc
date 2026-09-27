# Pinned identical-source scanner build control

Date: 2026-09-27. This distinct protocol is fixed before its first run. The
[unpinned identical-source control](scanner-build-null-result-20260927.md)
failed its predeclared 10% full-span rule. Its five measured same-source pairs
also had full spans of 10.7% for Fairy check, 11.1% for C emission and 18.5%
for the native phase. Splitting those wall-time phases alone cannot validate a
candidate build-cost claim on that run.

The new run uses the same explicit-index scanner in both lanes and retains the
same one excluded warmup pair, five measured pairs, alternating order, optimized
MSVC profile, raw phase times, source hashes, specialization counts, C-source
lengths, EXE lengths and toolchain identity. Before source preparation, the Go
test process sets its affinity to the lowest logical processor in its allowed
mask. [Windows process affinity is inherited by child processes](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-setprocessaffinitymask),
so the MSVC compiler and linker are constrained by that mask as well. The
receipt records the selected mask. The test restores its previous mask on exit
and does not change process priority.

The paired ratio is still clone/index total build time, summing `check_ns`,
`emit_ns`, `support_ns` and `native_ns`. This pinned null passes only when its
five-ratio median lies in `[0.98, 1.02]` and its relative full span is at most
10%. Every pair is retained. One run is permitted; a failure is not retried to
select a favorable result. A pass would permit a separately pinned
indexed/vector comparison under an independently recorded source revision;
it would not itself establish candidate cost, runtime speed or portable build
performance. The earlier unpinned failure remains valid evidence for its own
environment.
