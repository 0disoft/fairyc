# Scanner MSVC size-optimization probe

Date: 2026-09-25. Fairy C source HEAD before this note: `b9ef2e33aafd705ca6741104b86283e3c0290032`.

The optimized four-image scanner smoke was run with temporary MSVC flag changes
in the local Windows host driver. Each run checked the same numeric reporter
and 1,000-round checksum for independent C with up-front storage, geometric C,
explicit-index Fairy and private vector-loop Fairy. Both C and Fairy images
used the same compile flags within each row. The driver changes were reverted.

| Compile flags | Smaller C EXE | Both Fairy EXEs | Fairy/C size | Integer 8% ceiling | Fairy margin |
| --- | ---: | ---: | ---: | ---: | ---: |
| Current `/O2` | 184,320 | 201,728 | 1.09444 | 199,065 | 2,663 over |
| Experimental `/O1` | 181,248 | 194,048 | 1.07062 | 195,747 | 1,699 under |
| Experimental `/O2 /Os` | 181,248 | 193,536 | 1.06780 | 195,747 | 2,211 under |
| Experimental `/O2 /Gy` | 184,320 | 201,728 | 1.09444 | 199,065 | 2,663 over |
| Experimental `/O2 /Gy /OPT:REF /OPT:ICF` | 184,320 | 201,728 | 1.09444 | 199,065 | 2,663 over |

The geometric C images were 181,760 bytes in both experiments. The current
`/O2` geometric C image is 185,344 bytes. The ceiling is the largest integer
EXE byte count not exceeding `1.08 *` the smaller equivalent C image.

Each experimental flag combination also ran one warmup and twelve measured
rotations of the four standalone images at 250,000 rounds per image. The
diagnostic paired ratios below use QPC ticks and exclude the warmup. `span` is
`(maximum ratio - minimum ratio) / median ratio`; the predeclared stability
limit is 0.10.

| Flags | Fairy form / up-front C | Paired median | Span |
| --- | --- | ---: | ---: |
| `/O1` | explicit index | 1.19725 | 0.98799 |
| `/O1` | private vector loop | 1.26986 | 0.69331 |
| `/O2 /Os` | explicit index | 1.08726 | 0.40012 |
| `/O2 /Os` | private vector loop | 1.00958 | 0.34087 |

Both timing runs fail the stability rule. The `/O1` point estimates also
exceed 1.08. The `/O2 /Os` private-vector median is below 1.08, but its span
does not support a runtime admission claim. No check time, C compilation time,
general-workload runtime, or debug/cache artifact cost was measured. The raw
rotation samples were not retained by the diagnostic command, so these
summaries are not a release receipt. A separate size-optimized profile would
need retained samples, stable C-relative runtime, build-cost evidence and
other-workload validation before product adoption.

At source HEAD `2b52884e5bce31ddce08010fcc15fa49fadf56d0`, a later
four-image smoke repeated the current `/O2` build with `/Gy`, then with both
`/Gy` and explicit linker `/OPT:REF /OPT:ICF`. In each run, all four EXE sizes
matched the current `/O2` row exactly, and the reporter and 1,000-round
checksums passed. These runs did not measure time, so they establish only that
function packaging and explicit linker collection did not reduce this image's
size in this configuration. The temporary driver changes were reverted.

A separate source-only experiment hoisted the previous diagnostic reference
out of the inner selection loop. The existing lexical parity test passed, but
the `/O2` four-image smoke left both Fairy EXEs at 201,728 bytes. That source
edit was reverted. None of these experiments changes the accepted language,
SDK, runtime ABI, or stable optimized build profile.
