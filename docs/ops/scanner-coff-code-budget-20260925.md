# Scanner COFF object code accounting

Date: 2026-09-25. Current Fairy C source: `83cdcda4844b90ee663af7eb1f994e6e78755496`. The same four-image optimized Windows AMD64 MSVC scanner smoke that produced the [PE section receipt](scanner-pe-section-budget-20260925.md) now inspects its retained COFF objects. The smoke checks reporter, 1,000 completed rounds and checksum. The [raw object report](evidence/scanner-object-code-20260925.json) records the owning image hash, object names and lengths, code and other raw section bytes, section counts and relocations.

| Image | Object | Object bytes | Code-section raw bytes | Other-section raw bytes |
| --- | --- | ---: | ---: | ---: |
| Up-front C | `probe.obj` | 61,311 | 13,653 | 5,887 |
| Geometric C | `probe.obj` | 63,987 | 14,803 | 6,103 |
| Explicit-index Fairy | `unit.obj` | 88,595 | 32,116 | 6,620 |
| Explicit-index Fairy | `runtime.obj` | 42,132 | 8,039 | 3,703 |
| Private vector-loop Fairy | `unit.obj` | 88,595 | 32,116 | 6,620 |
| Private vector-loop Fairy | `runtime.obj` | 42,128 | 8,039 | 3,699 |

The two Fairy forms have identical `unit.obj` size and code-section totals in this build. Their runtime objects have the same code-section total and differ by four bytes outside code. Against the up-front C object, the Fairy `unit.obj` alone contains 18,463 more raw code bytes; the separate Fairy `runtime.obj` contributes another 8,039. These are object inputs, not an additive explanation of the final image: the linker selects and lays out code and common library support, so the 26,502-byte input-code difference must not be equated with the 20,992-byte final `.text` difference in the PE receipt.

The result points the next investigation toward reachable generated Fairy functions and the linked support functions. It does not identify which functions account for the size excess, prove that the vector-loop syntax adds no machine-code bytes, or establish the cumulative 8% C-relative gate. A linker map or equivalent function-level attribution is needed before removing code. C scanner parity, check/build time and whole-application performance remain open.
