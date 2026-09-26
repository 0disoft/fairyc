# Current same-image C/Fairy scanner comparison

Date: 2026-09-27. Fairy C source HEAD `3045f69c`. Windows AMD64,
MSVC 14.51.36231, Go 1.26.4, `GOMAXPROCS=1`, optimized profile. The
[complete raw report](evidence/scanner-same-image-c-current-20260927.json)
retains both generated-image identities and all timing pairs. The configured
`fairyc_measure_current_same_image_c_scanner` intent ran once; no sample was
retried or removed.

Two instrumented images each linked the independent C scanner with one
Fairy-written scanner form. Within each image, one warmup pair preceded twelve
measured adjacent C/Fairy pairs with alternating first position. Every call
scanned `let n=12345; /*open` for 250,000 rounds and returned the checked
9,000,000 work checksum. Token reporting was outside the timed section.

| Fairy form / C | QPC median | QPC relative full span | Thread CPU median | Thread CPU span |
| --- | ---: | ---: | ---: | ---: |
| Explicit index | 0.97932 | 5.43% | 1.00000 | 20.23% |
| Private `Vec<T>` loop | 1.01439 | 32.66% | 0.98276 | 28.83% |

Both combined images are 148,992 bytes and retain 21 specializations. They
cannot decide the standalone C-relative executable-size gate. The
[immediately preceding identical-C control](scanner-identical-c-retained-control-20260927.md)
passed its two 24-pair QPC sessions with 5.80% and 8.56% spans. That success
did **not** guarantee a stable C/Fairy comparison: the candidate image's QPC
span and both images' thread-CPU spans exceed the predeclared 10% stability
rule. The two Fairy forms occupy different combined images, so their medians
are not a direct vector-versus-indexed pair. These point estimates cannot
establish the cumulative 8% runtime gate or admit the private source form.

The same-host [earlier C control](scanner-identical-c-control-20260927.md)
also retained a failing 14.22% session. The next runtime experiment needs a
new predeclared comparison protocol or a genuinely controlled environment;
repeating these measurements until a favorable sample appears would bias the
decision.
