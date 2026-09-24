# Scanner PE section size accounting

Date: 2026-09-25. Current Fairy C source: `8c87df39260447b3d22bbdd4e9b979381ea51031`. The existing four-image scanner smoke builds one up-front C image, one geometric C image, and the explicit-index and private vector-loop Fairy images with Windows AMD64 MSVC 14.51.36231 under the optimized profile. It checks the same `let n=12345; /*open` reporter, 1,000 completed rounds and checksum before this size receipt is used. The [raw PE report](evidence/scanner-sections-20260925.json) records image hashes, section raw and virtual sizes, offsets, file alignment and byte accounting.

| Image | EXE bytes | `.text` raw | `.rdata` raw | `.pdata` raw |
| --- | ---: | ---: | ---: | ---: |
| Up-front C | 184,320 | 124,416 | 46,592 | 6,656 |
| Geometric C | 185,344 | 125,440 | 46,592 | 6,656 |
| Explicit-index Fairy | 206,336 | 145,408 | 47,104 | 7,168 |
| Private vector-loop Fairy | 206,336 | 145,408 | 47,104 | 7,168 |

Every image has 1,024 header bytes, 512-byte file alignment and zero bytes outside its headers and section raw sizes. `.data`, `.fptable` and `.reloc` raw sizes match across the four images. Relative to up-front C, each Fairy image has 20,992 more `.text` bytes, 512 more `.rdata` bytes and 512 more `.pdata` bytes: together they account for the full 22,016-byte EXE gap. The two Fairy source forms have identical section raw sizes.

The current Fairy/C EXE ratio is 1.11944, above the cumulative 1.08 screen. With these 512-byte file steps and the C size fixed at 184,320 bytes, a passing Fairy image would be at most 198,656 bytes, requiring at least 7,680 bytes less than the current image. This section receipt identifies the code-section priority; it does not attribute bytes to individual functions, prove that linker options can remove them, or establish that the C reference has complete scanner parity. Object-level or linker-map evidence is needed before selecting a size change. Check/build time and general workload behavior remain unmeasured.
