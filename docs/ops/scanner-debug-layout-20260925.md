# Scanner optimized PDB layout comparison

Date: 2026-09-25. Source HEAD before this comparison: `c7c7e2d5ddfb2263c8fe7ce4cb229ef9d5481fa2`.
`TestPrivateVectorSDKCandidateScannerDebugMapLayout` builds the same optimized
scanner sources once with the ordinary MSVC map and once with embedded CodeView,
a full PDB, and explicit optimized linker selection. It validates each PDB's
PE identity, runs the same 1,000-round input, and compares the reporter,
completed-round count, checksum and every validated `.pdata` begin/end RVA.
QPC ticks are excluded from output equality.

| Image | Release EXE bytes | PDB build EXE bytes | Matching unwind intervals |
| --- | ---: | ---: | ---: |
| Up-front C | 184,320 | 184,832 | 526 / 526 |
| Explicit-index Fairy | 206,336 | 206,336 | 589 / 589 |
| Private vector-loop Fairy | 206,336 | 206,336 | 589 / 589 |

The separate small-probe check also matched all 340 unwind interval boundaries,
although its release and PDB `.text` bytes differed. Matching interval layout
permits a bounded attempt to use PDB-private function starts to classify the
previously unmatched release intervals. It does not itself assign a function
owner, prove equal instructions or leaf-code layout, or satisfy the 8% runtime
and EXE gate. The C reference still needs complete scanner parity.

The next probe resolves each PDB-build unwind start through system DbgHelp.
Its small test finds a deliberately non-leaf private C function, so the
lookup can expose private symbols when they exist. On the scanner images,
none of the intervals without a public map start has an exact PDB symbol:
12,823 C bytes and 25,193 / 25,196 Fairy bytes remain unattributed. Those
totals agree with the earlier map-only accounting. This does not prove all
unmatched code belongs to the CRT; library or object contribution evidence
is needed before selecting a size optimization.
