# Generated Windows COFF relocation receipt, 2026-11-01

- Code: `07fd28a9`; native Windows amd64, MSVC 14.51.36231.
- Scope: generated Fairy C plus symbol-selected production native support, compiled as separate
  `unit.obj` and `runtime.obj`. No synthetic object, source rewrite or substitute runtime.
- Owners: `internal/pecoff/generated_relocations.go` and
  `internal/cdriver/generated_relocations_windows_amd64_test.go`.

## Observed range and final-link result

Five source workspaces pass verified MIR and Windows emission, then ordinary C11 low/optimized
compilation and native execution. The driver retains map and verbose-link evidence only; it keeps
the existing `/MT`, `/Od` or `/O2`, `/W4 /WX` and `/INCREMENTAL:NO` behavior. It adds no `/Z7`,
PDB capture, no-ICF switch, LTO or custom relocation-generation switch.

| Actual object family | Observed fields | Linked values checked | Explicitly discarded |
| --- | ---: | ---: | ---: |
| AMD64 REL32 (`0x0004`) | 1,510 | 1,322 | 188 |
| AMD64 ADDR32NB (`0x0003`) | 1,623 | 1,082 | 541 |
| Total | 3,133 | 2,404 | 729 |

| Source workspace | Profile | Observed | Linked | Discarded | Native result |
| --- | --- | ---: | ---: | ---: | --- |
| `std_bytes_quickstart` | low | 499 | 499 | 0 | exit 0, empty streams |
| `std_bytes_quickstart` | optimized | 343 | 238 | 105 | exit 0, empty streams |
| `std_alloc_quickstart` | low | 129 | 129 | 0 | exit 42, empty streams |
| `std_alloc_quickstart` | optimized | 143 | 31 | 112 | exit 42, empty streams |
| `std_text_from_buffer_quickstart` | low | 431 | 431 | 0 | exit 42, empty streams |
| `std_text_from_buffer_quickstart` | optimized | 456 | 186 | 270 | exit 42, empty streams |
| `std_vec_quickstart` | low | 354 | 354 | 0 | exit 0, stdout exactly `42`, empty stderr |
| `std_vec_quickstart` | optimized | 355 | 178 | 177 | exit 0, stdout exactly `42`, empty stderr |
| `process_args_quickstart` | low | 268 | 268 | 0 | exit 42, empty streams |
| `process_args_quickstart` | optimized | 155 | 90 | 65 | exit 42, empty streams |

The argv example receives empty, ordinary, flag-like, quoted/backslash and Unicode arguments.
The corpus covers Buffer/heap, Arena, owned text, vectors, Windows console and process-argument
native support. Counts include emitted but discarded runtime/utility functions. These are
twenty actual compilation objects and ten native images, not twenty executions.

## Verification boundary

For every retained field the checker records object identity, exact section index/name, object
offset, target symbol, raw addend, linked site/target RVA, expected value and actual value. Owned
map symbols anchor local code/data/unwind section bases. Undefined targets require an exact map
symbol; CRT/import-target relocation addresses may resolve through that map, but archive member
selection itself is not proved by this slice. Pdata fields use a unique final exception-directory
begin/end pair, so linker sorting is not mistaken for contiguous input layout.

- REL32 uses `S + signed(A) - (P + 4)`, checked against the final signed displacement bits.
- ADDR32NB uses `target RVA + unsigned(A)`, with overflow rejection. This includes actual pdata
  and xdata references in the corpus.
- The existing generated-COMDAT proof supplies exact owned-map versus discard-log decisions and
  compares non-relocation parent bytes. Discarded fields have no claimed linked value.
- Every object passes structural COFF inspection; every final image passes PE/unwind inspection.
  Unsupported kinds, unknown ownership, duplicate/overlapping fields and incomplete accounting
  fail closed. This validation API is bounded to 65,536 fields per object.

Negative controls alter a live source addend and its final image field independently for every
object/family combination. Missing maps, wrong source ownership and removed discard evidence are
also rejected. Pure formula checks cover signed/backward REL32, REL32_1 through REL32_5, ADDR64,
ADDR32NB, overflow and unmeasured types. Formula coverage does not relabel those unobserved kinds
as new native evidence.

Native logs print SHA-256 for emitted unit C, selected runtime C, both objects and the final
image. The four SDK examples ran together; the process-argument example was then added and ran
separately. The section-index receipt refinement reran the bytes example at both profiles.
Unchanged source/runtime inputs reuse those passing observations. Local transient logs and
counts are under `.cache/coff-relocation-20261101*`; this document is the bounded retained receipt,
not a promise to retain the temporary binary objects or reproduce timestamped image bytes.

## Commands and outcomes

Executed successfully on Windows:

```powershell
go test -mod=readonly ./internal/cdriver -run '^TestGeneratedRelocationCoverage$' -count=1 -v
go test -mod=readonly ./internal/cdriver -run '^TestGeneratedRelocationCoverage/process_args_quickstart$' -count=1 -v
go test -mod=readonly ./internal/cdriver -run '^TestGeneratedRelocationCoverage/std_bytes_quickstart$' -count=1
go test -mod=readonly ./internal/pecoff -count=1
go vet -mod=readonly ./internal/pecoff ./internal/cdriver
git diff --check
```

The first command had the four original workloads at execution time. On the final source it
also includes the separately verified process-argument workload.

## Remaining limits

This corpus does not enumerate every possible Fairy program, the full compiler CLI or every
CRT/archive/DLL object. Debug-only SECTION/SECREL, ADDR64 and REL32 variants are not observed in
these ordinary inputs. Existing synthetic/native targeted probes retain their original evidence;
they are not counted above. Weak-external selection and archives remain the next separate tasks.
Handler payload semantics, actual unwinding through every frame and PDB/DbgHelp retention remain
separate work. Target maturity and public ABI support are unchanged.

No Linux command or WSL query ran; the existing batched Linux queue remains unchanged. Full
repository tests, hosted CI, distribution and clean-machine transfer were not run. No public API,
DB, dependency, production compile/link flags, CI/runner or hygiene configuration changed.
No push, publication or deployment occurred.
