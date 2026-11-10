# Distribution build size, 2026-11-11

Owner: the [Win32 example](../../examples/win32_minimal/README.md) and its private
native bridge. Native build inputs continue to follow
[ADR 0552](../adr/0552-minimal-windows-gui-build.md).

## Baseline before separation

The authored release image verified at code revision `b15fe506` is 144,384
bytes (141 KiB), SHA-256
`ed05eee2b0e38cf67dbe794961ce57f344f85b038e13ed527453e25fe2d6c945`.
`python -B size_report.py <image>` reads PE sections and named DLL imports using
only Python's standard library. It does not execute the image.

| File component | Bytes |
|---|---:|
| Headers | 1,024 |
| `.text` | 86,016 |
| `.rdata` | 45,056 |
| `.data` | 3,072 |
| `.pdata` | 5,632 |
| `.fptable` | 512 |
| `.rsrc` | 1,024 |
| `.reloc` | 2,048 |
| **Total** | **144,384** |

There are no unassigned/overlay bytes. Raw section sizes include file-alignment
padding; virtual sizes describe mapped sections, not disk size or process RSS.
The image imports USER32 (50 names), GDI32 (4), IMM32 (6) and KERNEL32 (80).
Those DLLs are Windows dependencies, not bundled files. Static CRT code/data
is included in the image's sections; no CRT DLL is imported. The executable
has no COFF symbol table, so this measurement does not invent exact per-function
or static-CRT byte attribution from section totals.

The baseline includes automation JSONL/pipe handling, smoke scenarios, injected
DPI requests and mouse/IME observation. These are runtime opt-in features in
the existing image, not compile-time exclusions. The distribution work will
exclude these diagnostics while retaining the same Fairy application source,
native event processing, CRT initialization, UTF conversion, IME priority,
resize/DPI layout and resource text. The ordinary build must retain its CLI
verification features. CRT replacement, compression and disabling safety checks
are outside this change.

Local baseline files are retained under `.cache/windows-gui-slim-20261111/`;
`baseline-authored-release.json` records the section/import inventory and image
hash, and `baseline-authored-release.exe` preserves the measured image.
