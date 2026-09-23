# Direct-address mutable vector C comparison

Date: 2026-09-24. Exact Fairy C source: `1edc1f4342204334e7c70459b00392299739550e` (SDK `0.0.0-stage0.64`). Windows AMD64, Go 1.26.4, MSVC 14.51.36231, `GOMAXPROCS=1`, optimized C11 `/W4 /WX /MT /O2 /INCREMENTAL:NO`.

The [original repeated checked `Vec<u32>.at_mut` comparison](c-budget-vector-slot-20260924.md) exceeded the cumulative 1.08 runtime budget because generated C called the private mutable slot lookup on every iteration. The SDK now checks `index < length`, computes a typed element address inside its private source, guards null and offset overflow, and returns the same owner-bound mutable borrow. The safe API, out-of-range trap, element ownership and public runtime ABI are unchanged. A malformed vector descriptor can be constructed only through unsafe or private code; the old private C lookup additionally checked descriptor fields on each access. The independent C fixture retains its own descriptor checks. Thus the lanes preserve valid-input and public bounds behavior but are not instruction-for-instruction equivalent.

The `vector_slot.fy`, `vector_slot.c` and `support.c` fixture hashes match both earlier vector reports. Each lane passed five boundary inputs. After one build/run warmup per lane, five alternating pairs ran the same 50-million-update checksum. No samples were discarded. The [ordinary-layout raw evidence](evidence/c-budget-vector-address-ordinary-20260924.json) and [one-translation-unit raw evidence](evidence/c-budget-vector-address-one-tu-20260924.json) contain all ten samples each, source hashes, build times, artifact sizes and exact compiler commit.

| Layout and metric | C median | Fairy median | Fairy/C |
| --- | ---: | ---: | ---: |
| Ordinary kernel | 326.043 ms | 307.779 ms | 0.94398 |
| Ordinary EXE | 167,936 B | 171,008 B | 1.01829 |
| Ordinary build | 537.952 ms | 755.058 ms | 1.40358 |
| One-TU kernel | 329.045 ms | 307.452 ms | 0.93438 |
| One-TU EXE | 167,936 B | 170,496 B | 1.01524 |
| One-TU build | 554.904 ms | 748.752 ms | 1.34934 |

The ordinary-layout paired kernel ratios range from 0.92594 to 0.95263, with median 0.94861. It is the primary comparison because both lanes keep their normal compilation layout. Runtime and EXE size are below 1.08 in every ordinary pair. The one-TU paired median is 0.94816, but one pair reached 1.22881, so it is a noisy diagnostic, not a stronger stability claim.

Build times are observations, not a compiler-speed pass: the ordinary median is about 40% above C, and the one-TU layouts differ between lanes. The measured build includes Fairy source loading, analysis, verified MIR and C emission, then MSVC compilation and linking; C goes through its independent source and MSVC. Toolchain discovery was warmed and OS caches were uncontrolled. The next performance task is to split Fairy frontend time from C compilation/link and reduce the repeatable part without expanding the language or weakening checks. This fixed-slot microbenchmark does not establish performance for variable indices, realistic parser workloads, Linux, cold CLI startup or all source programs.

A [follow-up phase diagnostic](c-budget-vector-build-phases-20260924.md) separates workspace load, verified check, C emission/support assembly, MSVC build and EXE write on the same fixture.
