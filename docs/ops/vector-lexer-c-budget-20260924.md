# Three-image scanner C budget triplets

Date: 2026-09-24. Fairy C compiler source: `b0bc18531bbda730c3141a78c74d48f054a3ec7b` plus the measurement harness in this change. Windows AMD64, MSVC 14.51.36231, optimized profile, Go 1.26.4, `GOMAXPROCS=1`. The [raw report](evidence/vector-lexer-c-budget-20260924.json) records executable hashes, every QPC sample, image sizes, source sizes, specialization counts, the sample hash and run order.

The three standalone images run an independent C scanner, the existing explicit-index Fairy scanner, and the compiler-private vector-loop Fairy scanner. The input is `let n=12345; /*open`, with 250,000 scan and storage rounds per execution. Each execution checks its reporter, completed rounds and checksum against the independent Go lexer. QPC brackets the repeated work, excluding process startup and final output; token reporting is outside the timed loop. One warmup triplet preceded twelve measured triplets, with each image appearing four times in each execution position. There were no retries, exclusions or adaptive stopping.

| Series | Median | Relative full span |
| --- | ---: | ---: |
| Independent C | 449.520 ms | 77.99% |
| Explicit-index Fairy | 765.783 ms | 141.26% |
| Vector-loop Fairy | 776.265 ms | 73.32% |
| Paired index/C | 1.79191 | 131.80% |
| Paired vector/C | 1.76606 | 29.04% |
| Paired vector/index | 0.96728 | 68.82% |

The ratio of lane medians is 1.70356 for index/C, 1.72688 for vector/C, and 1.01369 for vector/index. These differ from the medians of paired ratios because the individual timings vary; this run does not isolate the cause. Every lane and paired-ratio series exceeds the predeclared 10% relative-full-span stability heuristic. No runtime 8% verdict follows from this run, and the ratio differences between the two Fairy spellings cannot be attributed to the loop form.

The C executable is 181,760 bytes. Both Fairy executables are 205,824 bytes, or 24,064 bytes (+13.24%) above this restricted C reference; they have different hashes but identical byte counts. Both Fairy lanes have 21 specializations. Generated C is 384,636 bytes for the indexed lane and 385,464 bytes for the vector lane. This establishes no EXE-size increase from the candidate spelling in this build. The independent C reference does not implement every lexer input and limit, so its smaller executable is a diagnostic observation rather than a representative whole-scanner 8% gate result.

The current data leave two separate questions: reproduce stable C-relative timings under controlled host conditions, and finish the independent C reference's missing lexical forms and limits before interpreting a full-scanner EXE ratio. The vector syntax remains private. No language, SDK, runtime ABI or target contract changes follow from this measurement.

The later C-reference expansion of the decimal-cooking cap is not part of these image hashes or timings. The raw report remains evidence for the exact earlier images, not a measurement of the revised reference.

The subsequent source audit also found that the Fairy-written scanner rejects decimal materialization above 16,384 bytes and limits all cooked bytes to 65,536, while the revised C reference accepts the 70,000-digit cases checked against Go. Those cases are outside Fairy/C parity today. The short timed sample is common to all three lanes, but its C token and diagnostic arrays are allocated for `length + 1` items up front, whereas Fairy uses geometrically growing `Vec<T>` storage. The timing gap cannot be assigned solely to the loop syntax or compiler overhead without controlling that algorithmic difference.

After this raw run, the C benchmark build gained a 16,384-byte per-number
decimal materialization limit matching the Fairy candidate at four
positive/negative exponent boundary cases. It now also caps total cooked
bytes at 65,536: four 16,384-byte decimals succeed and a fifth returns
candidate-size status 4 in both C and Fairy under low and optimized MSVC.
The C budget build now also charges numeric expansion work before allocation:
`1e3` at work limits 6/7 matches C, Go and Fairy exhaustion/success, and a
default-budget large exponent produces lossless `E0010` recovery. The broad
standalone C Go-oracle build still accepts the 70,000-digit cases. Each of
these changes produces a new C image; none of the earlier executable hashes,
sizes or QPC samples above describe the current image. Other recognition work
paths, remaining lexical forms and the C/Fairy storage difference are still
open, so the 8% gate remains undecided.

The next C budget image now grows token storage from 8 slots by doubling,
matching the private `Vec<Token>` capacity sequence; low and optimized C
probes check the 8/16/32-slot boundaries and exact Go-lexer output. The broad
standalone C reference retains its original up-front token allocation. A
further C budget image also grows diagnostic entries from 8 slots by doubling;
low and optimized probes cover 0/8/16/32-slot boundaries and compare malformed
source output with Go. The broad standalone reference still uses up-front
diagnostic storage. Fairy diagnostic entries own payloads differently from the
C reference, so equal capacity steps do not prove identical allocation cost.
Neither new image is represented by the earlier hashes or timing samples.
The full scanner 8% gate remains open.
