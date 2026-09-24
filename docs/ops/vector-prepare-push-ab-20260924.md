# Vector prepare-push reserve-path A/B

Date: 2026-09-24. The [raw measurement](evidence/vector-prepare-push-ab-20260924.json)
comes from mustflow run `mf-run-076e204dea6c432a`, which passed on Windows
AMD64 with MSVC 14.51.36231, Go 1.26.4 and `GOMAXPROCS=1`. The opt-in harness
is `TestPrivateVectorPreparePushAB`.

Both images use one verified real-scanner MIR program, one generated C unit,
the same SDK and support, and the optimized MSVC profile. The only C source
change restores the prior unconditional `reserve` call inside
`fy_private_vector_prepare_push_v0` for the baseline image. The current image
skips that call when a validated spare slot exists. The harness requires one
exact source replacement, equal generated C and 21 specializations before
building. The two images produced the same reporter, completed all rounds and
returned the expected per-round checksum.

The input was `let n=12345; /*open`. Each execution scanned and stored it
250,000 times. QPC brackets those rounds, excluding process startup and final
reporting. One warmup pair preceded twelve measured pairs in alternating order;
there were no exclusions, retries or adaptive stopping.

| Image | Median QPC time | Relative full span | `unit.obj` | `runtime.obj` | EXE |
| --- | ---: | ---: | ---: | ---: | ---: |
| Prior unconditional reserve | 1,184.932 ms | 44.82% | 86,925 B | 41,858 B | 205,824 B |
| Current spare-slot path | 1,154.374 ms | 38.38% | 86,925 B | 41,844 B | 205,824 B |

The median of twelve paired current/prior ratios is 0.98287, with a 37.90%
relative full span. Every time series exceeds the predeclared 10% stability
heuristic, and some pairs favor the prior path. The size observation is exact
for these two built images: the generated unit object and EXE byte counts are
unchanged, while the current runtime object is 14 bytes smaller. Executable
hashes differ. This run does **not** establish a runtime speedup or a pass of
the cumulative 8% C-relative budget.

These benchmark images include the timing and scanner support code, so their
EXE sizes must not be compared directly with the smaller native parity images
in [the earlier size probe](evidence/vector-lexer-size-20260924.md). The
independent C scanner also remains incomplete and has a different storage
algorithm; this A/B isolates the private helper only. The token reporter is
outside the timed loop, and check/build time and unused-feature overhead are
not measured here.
