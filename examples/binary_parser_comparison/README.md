# Binary parser comparison

This example compares the same length-prefixed binary parser in
[Fairy C](src/main.fy) and [Zig](parser.zig). A record starts with a little-endian
16-bit payload length. Its payload may be empty. The parser sums the payload
bytes and returns status 2 for a truncated header or 3 for a truncated payload.

On Windows AMD64 with x64 MSVC and the Windows SDK installed, run from the
repository root with Python, Fairy C preview.3, and Zig 0.17.1:

```powershell
python -B examples/binary_parser_comparison/verify.py --fairyc C:\path\to\fairyc.exe --zig C:\path\to\zig.exe
```

The verifier builds both programs in a temporary directory, checks their results
against five shared fixtures, and prints timings and diagnostics as JSON.
It also checks three changes to the Fairy C example:

| Change | Result |
| --- | --- |
| Release the input owner while the decoder's view is still live | `fairyc check` rejects it with E2105 and reports the last use. |
| Open a file inside the decoder without declaring the effect | `fairyc check` rejects it with E7102. |
| Add the file access together with its effect declaration | `fairyc check` passes. `atlas-diff --symbol main::decode --deny-new-effect filesystem` rejects the change with status 2. |

The corresponding lifetime and file-access changes compiled to LLVM IR in Zig
ReleaseSafe. The modified Zig programs were never executed. The Atlas check
compares declared effects between two locked workspaces.

## Measured example

Windows AMD64, Intel Core i5-8265U, Fairy C preview.3 (`415e77c7`) and Zig
0.17.1. Both builds target the Windows MSVC ABI. Fairy C uses its release profile
and Zig uses ReleaseSafe with the baseline x86-64 CPU. Both parsed 16,777,042
bytes across 103,690 records with checksum 2,108,743,530.

| Compiler | Median parse time | Throughput |
| --- | ---: | ---: |
| Fairy C | 54.95 ms | 291.2 MiB/s |
| Zig | 37.75 ms | 423.8 MiB/s |

The common Windows performance counter surrounds only the parse. Each language
has one warm-up run and nine timed runs, with execution order alternating.
Both decoders use a borrowed slice and fixed-size accumulators, with zero heap
allocation calls and zero requested heap bytes in the parser by source
inspection. File loading, input-buffer allocation and output are outside the
timed interval. Process-wide allocation traffic was not measured. The numbers
describe this input and these compiler profiles.
