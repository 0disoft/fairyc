# Windows production archive and COMDAT receipt, 2026-11-01

- Inspected sources committed as `e9ca8758` (selected archive indexing) and `4f300ea1` (native
  matrix). Host: Windows amd64, MSVC 14.51.36231, Windows SDK 10.0.26100.0.
- Scope: the Go Windows builder, the five existing generated/runtime workspaces, five selected
  CRT members and the existing controlled archive/COMDAT probe. No new source or public ABI.
- Ordinary compilation remains C11 `/MT /W4 /WX` with `/Od` or `/O2`. Vendor CRT objects are
  precompiled inputs, not recompiled under those profiles. Their archive hashes match the
  [default-library receipt](windows-default-libraries-20261101.md).

## Generated-object forms and complete accounting

`TestGeneratedRelocationCoverage` now also invokes the existing generated COMDAT parent and
unwind-child inspectors for every image. Actual sources are checked, lowered, emitted and
compiled separately as `unit.obj` and `runtime.obj`; no control object replaces Fairy output.
The observed nonzero COMDAT selections are NODUPLICATES (1), ANY (2) and ASSOCIATIVE (5).
New selection modes require review rather than being silently added to this receipt.

| Workspace/profile | Parents kept/discarded | Associative unwind children kept/discarded | Checked child RVA fields | Matched pdata entries | Matched xdata bytes |
| --- | ---: | ---: | ---: | ---: | ---: |
| bytes/optimized | 20 / 32 | 46 / 40 | 96 | 23 | 416 |
| alloc/optimized | 8 / 21 | 12 / 42 | 18 | 6 | 64 |
| text-from-buffer/optimized | 18 / 52 | 58 / 98 | 126 | 29 | 508 |
| vec/optimized | 19 / 37 | 52 / 68 | 109 | 26 | 400 |
| process-args/optimized | 11 / 7 | 26 / 20 | 53 | 13 | 184 |
| process-args/low | 0 / 1 | 0 / 0 | 0 | 0 | 0 |
| Other four low images combined | 0 / 0 | 0 / 0 | 0 | 0 | 0 |
| Total | **76 / 150** | **194 / 268** | **402** | **97** | **1,572** |

Full workspace names are `std_bytes_quickstart`, `std_alloc_quickstart`,
`std_text_from_buffer_quickstart`, `std_vec_quickstart` and `process_args_quickstart`.
The twenty compilation objects contain 226 non-associative COMDAT parents (216 NODUPLICATES,
ten ANY) and 462 associative unwind sections. The low process-runtime ANY datum is the SDK
alternative already distinguished from live selection in the [weak receipt](windows-weak-aliases-20261101.md).

Each retained parent requires an owned map entry and matching final non-relocation bytes;
each discarded parent requires absence from its owned map plus the exact discard record.
Every direct associative `.pdata`/`.xdata` child must follow its parent's decision. Retained
children have independently relocated final byte/entry checks, with parent-bounded function
intervals and no orphan pdata bytes. General relocation fields retain the separate complete
accounting from the relocation suite. This is not execution of the Windows unwinder through
every represented function.

Ten production images pass exact native outcomes, including argv/Unicode and vector output.
The existing bytes-only parent and unwind suites also run at both profiles, adding four
executions and independent negative controls for missing map/discard rows, contradictory
retention/discard evidence, altered machine code, structurally valid changed unwind bytes and
changed source pdata ranges. Their observations are not added to the corpus counts above.

## Actual CRT archive routes and shapes

`InspectIndexedArchiveObject` validates the Microsoft archive container and both linker-member
routes for one demanded symbol. The first and preferred second indexes must uniquely select
the same body, and that inspected AMD64 object must uniquely define the symbol. The selected
member's long name, header/data offsets, size, bytes/hash, loaded-member log and exact final
map owner are tied together. Other object bodies remain opaque; the existing full
`InspectArchive` admission is not relaxed. Import-library and metadata-only bodies are not
silently labeled as inspected executable objects.

| Actual member and demanded symbol | Observed selected-function shape | Direct associative pdata/xdata proof |
| --- | --- | --- |
| `LIBCMT(exe_main.obj)`, `mainCRTStartup` | NODUPLICATES | Both profiles; final entry and all three pdata fields |
| `libucrt(initialization.obj)`, `__acrt_initialize` | NODUPLICATES, 19-byte code section | No direct associated pdata for this root; no triple claimed |
| `libvcruntime(initialization.obj)`, `__vcrt_initialize` | NODUPLICATES | Both profiles; exact function interval/unwind RVA |
| `libucrt(malloc.obj)`, `malloc` | Ordinary five-byte code section | No direct associated pdata; no triple claimed |
| `libvcruntime(memset.obj)`, `memset` | NODUPLICATES | Both profiles; exact function interval/unwind RVA |

The five unique objects contain ordinary sections and COMDAT modes 1/2/5, including vendor
debug/metadata associations. Only the three named direct unwind associations are bound
semantically here: six triples across two linked images. Inventory of other CRT associations
does not claim their retention, runtime handling or full CodeView meaning. The C control
allocates, fills, observes and frees storage and exits 42 with empty streams at both profiles.
Archive, selected-object, control-object and final-image hashes are logged.

Supplying two copies of the actually selected NODUPLICATES startup object as explicit inputs
causes real LINK to reject `mainCRTStartup` with `LNK2005` at both profiles. These are **two
expected link failures**, not native executions. No installed archive was changed.

## Controlled archive extraction and competing ANY candidates

`TestWindowsArchiveCOMDATForms` runs the existing `runWindowsArchiveProbe` without launching
the complete smoke command. Low/optimized and direct-first/archive-first give **four native
executions**, each exit 0 with empty streams. The existing probe proves:

- Both archive indexes identify the demanded member and the unused member. The unrequested
  sentinel is in ordinary indivisible code, so its map absence cannot be explained by per-function
  garbage collection after extraction.
- Competing read-only ANY values are 17 and 23. The exact selected owner and final value agree;
  input order is varied without promising a particular ANY winner.
- The demanded function retains its associative pdata/xdata with three exact final RVA fields.
- Competing ANY function parents have distinct unwind payloads. The selected parent and children
  agree; the discarded payload is absent from final runtime-function references and raw
  file-backed image bytes, with no orphan pdata.

These are controlled C11/C++20 fixtures. Their existing `/Gy` packaging and C++ `/GS-` setting
remain fixture-only and are not introduced into production compilation. The generated corpus
and real CRT-member observations above establish which forms the ordinary measured path uses.

## Validation and remaining boundaries

Passed locally:

```powershell
go test -mod=readonly ./internal/pecoff -count=1
go test -mod=readonly ./internal/cdriver -run '^(TestWindowsCRTArchiveCOMDAT|TestGeneratedRelocationCoverage|TestGeneratedCOMDATSelection|TestGeneratedUnwindSelection)$' -count=1 -v
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindowsArchiveCOMDATForms$' -count=1 -v
go vet -mod=readonly ./internal/pecoff ./internal/cdriver ./tools/dev/fairyc-windows-host-smoke
git diff --check
```

There are **twenty positive native executions**: ten production images, four existing bytes
mutation-suite images, two CRT controls and four controlled archive images. Repeated development
attempts are not double-counted. Unit checks reject mismatched/corrupt archive routes and a
selected object missing the demanded definition, and distinguish selection from validation of
unselected bodies. The new helper retains the existing 64 MiB archive, 4,096-member and bounded
symbol/name limits. Temporary logs are `.cache/archive-final-20261101.log`,
`.cache/archive-forms-20261101.log` and `.cache/archive-crt-20261101.log`.

SAME_SIZE, EXACT_MATCH and LARGEST are not observed in this corpus and receive no new native
claim. This record does not enumerate every CRT/archive member, arbitrary C++ or LTO inputs,
ICF equivalence, chained/non-unwind associations, full compiler self-hosting, clean-machine
transfer or every Fairy program. Runtime unwind v2/handler/chained-frame semantics remain
ordered item 8; broader debug and DLL deployment evidence remain items 10/11. Target maturity
and the missing-fixture status are unchanged.

Linux/WSL remains deferred at the maintainer's request, with no new Linux command or job.
Full `go test ./...` and hosted CI were outside this focused local run. No public API/SDK ABI,
DB, dependency, production flag, repository hygiene rule, CI runner or workflow changed.
No push, workflow dispatch, publication or deployment was performed.
