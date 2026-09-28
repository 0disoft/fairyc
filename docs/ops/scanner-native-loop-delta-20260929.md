# Scanner linked-code delta for read-only Vec loops

Date: 2026-09-29. Source baseline: `2d81ecbc`. The existing optimized
Windows MSVC scanner debug-map test built the checked-in indexed form and the
private three-loop Vec form from the same checkout. It checked that each
debug build retained the release build's unwind function boundaries, that
PDB procedure records partitioned their modules' linked `.text` contributions,
and that both builds produced the same scanner output. The test now reports
the linked procedure sizes of the two functions containing the replaced loops.

| Linked Fairy procedure | Indexed `.text` | Vec-loop `.text` | Delta |
| --- | ---: | ---: | ---: |
| `scanner::report_work` | 2,278 B | 2,279 B | +1 B |
| `scanner::storage` | 275 B | 275 B | 0 B |

The complete `unit.obj` linked-code contribution is 22,022 B versus 22,023 B.
The `runtime.obj` contribution is 4,138 B in both forms. Both optimized
release executables are 197,120 B, and both debug executables are 197,120 B.
The test passed through the configured `fairyc_go_test_scanner_debug_layout`
intent after `fairyc_go_format`.

This is a one-build, one-workload code-size comparison, not a runtime timing
result or byte-identical executable proof. The Vec form does not meet a literal
zero-added-machine-code claim for this scanner: one linked function grows by
one byte. The unchanged whole-file size can mask a one-byte `.text` change
because executable layout and alignment affect the file length. The Linux
C-only timing null still fails; no Fairy/C 8% runtime verdict follows from
this Windows code-size evidence. The source form remains private.
