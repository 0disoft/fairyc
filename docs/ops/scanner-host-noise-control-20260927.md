# Scanner timing host-noise control

Date: 2026-09-27. Child HEAD: `6c25781f`. Windows AMD64, MSVC 14.51.36231,
Go 1.26.4, optimized C11 host profile. This was a temporary, standalone
native timing probe; its Go test and mustflow intent were removed afterward.

One C function updated an unsigned 64-bit accumulator for a calibrated
134,217,728 iterations per call (`x = x * 2862933555777941757 + 3037000493 + i`)
and stored its result through a volatile sink. Calibration stopped after a
call exceeded 0.2 seconds; one fixed warmup followed. The same function was
then called twice per pair for 24 pairs, alternating which label ran first.
QPC measured only the function calls; output and process startup were outside
the intervals. Every call used the same iteration count. No sample was
removed, retried or selected adaptively. QPC frequency was 10,000,000 ticks/s,
and the final checksum was 5,873,319,649,285,767,169. The 141,312-byte
probe EXE is not a scanner size comparison. Its C unit SHA-256 was
`355764ea2e3c012bd388b83c4fb81dc8ae470fe9282d3c5608c70ac9a09c0acf`.

Raw QPC ticks, in execution-label order A/B, were:

```text
pair     A        B
 1   3116897  3147727
 2   3134965  3127073
 3   3131293  3135880
 4   3120758  3141143
 5   3123665  3113327
 6   3144808  3128960
 7   3138065  3118074
 8   3127055  3114705
 9   3124682  3128302
10   3128383  3120546
11   3123870  3131956
12   3158407  3175282
13   3120892  3124893
14   3152078  3117409
15   3129102  3136916
16   3131289  3122069
17   3121711  3179653
18   3145130  3131736
19   3122354  3122453
20   3127498  3121968
21   3134233  3132811
22   3136621  3134452
23   3123362  3118503
24   3125793  3134020
```

The paired B/A median was **0.9994274**, with a **2.96% relative full span**
`(max ratio - min ratio) / median`. This fixed arithmetic control meets the
predeclared 10% stability limit on this host. It does not model scanner
allocation, different generated code, Fairy/C interaction, or the scanner's
input-dependent paths. It therefore cannot rescue the unstable scanner
measurements or decide the cumulative 8% runtime gate. The next useful control
is two identical C scanner calls in one optimized image at the scanner's
250,000-round workload, before interpreting another Fairy/C ratio.
