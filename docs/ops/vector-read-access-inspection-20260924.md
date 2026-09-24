# Generated vector-read access inspection

Date: 2026-09-24. Fairy C source: `6bb77052f16e3db3a1c403a92e5af355f8702dbe`.
Windows AMD64, MSVC 14.51.36231, optimized `/O2`. The configured
`fairyc_inspect_vector_read_access` intent rebuilt the indexed and private
candidate workloads separately, retained generated C and `dumpbin /DISASM`
output under `.cache/constant-divisors/before/`, and checked their disassembly
after removing only the path-specific `Dump of file` header.

| Observation | Indexed Fairy | Candidate Fairy |
| --- | ---: | ---: |
| Generated C | 29,161 B | 29,187 B |
| Optimized `unit.obj` | 8,182 B | 8,182 B |
| Normalized disassembly | 441 lines | 441 lines |

The two normalized disassemblies are byte-identical with SHA-256
`7a60bf542575dd7d560b7a0adcb720c0e9ff4d661f8eae556e8a1522c3b38b7d`.
Their object-file hashes differ, so this is an instruction-stream comparison,
not a claim that all object metadata matches. Both generated C files call
`fy_private_vector_element_slot_v0` in `std::vec::at<u32>`. The optimized
disassembly retains a call to that symbol inside the timed inner loop. The
support-source SHA-256 is the same in both measured Fairy lanes:
`032b8a8f7b2077fa3fff042687157862dc89ab5ba2c0b316c92072b995adb1bc`.

Thus the separate-run timing difference in the [indexed](c-budget-vector-read-index-20260924.md)
and [candidate](c-budget-vector-read-candidate-20260924.md) reports cannot be
credited to different optimized loop instructions. The observed C-relative
runtime gap includes an out-of-line descriptor-validation call on every
element read. Removing that call is a plausible next experiment, not yet a
measured attribution of the entire gap: safe index checks, arithmetic, call
layout and C compiler choices also remain. An immutable typed-address path
must preserve initialized-prefix and bounds checks and owner-bound shared
borrows. The existing mutable `at_mut<T>` path provides a related private
operation, but its behavior does not by itself prove a shared-borrow change.
