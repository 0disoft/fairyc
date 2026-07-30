# ADR 0235: selected associative COMDAT unwind proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Inspector: `internal/pecoff/associative_link.go`
- Probe generator: `internal/windowsarchiveprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0079, ADR 0080, ADR 0233, and ADR 0234

## Context

ADR 0233 proves that the archive's demanded function comes from `chosen.obj`, but selecting one
parent COMDAT is not by itself proof that its associated metadata belongs to the same group or
survives into the final image. Microsoft x64 unwind data splits one function across an executable
parent section plus `.pdata` and `.xdata` children. Their section-definition auxiliary records own
the association; names or final-image presence alone cannot establish it.

The selected function must also require real unwind metadata at both optimization profiles. A leaf
function could link and execute while producing no runtime-function entry, which would make an
associative-unwind assertion vacuous.

## Decision

1. Keep the existing four archive links and input-order variants. Make the demanded function use a
   bounded volatile stack array so low and optimized MSVC both emit unwind metadata without changing
   its result or the archive-selection contract.
2. In the exact `chosen.obj` bytes already inspected and archived by ADR 0233, require the demanded
   external function to reside in an executable, non-associative COMDAT root.
3. Locate the exact section-indexed 12-byte `.pdata` entry through its three `ADDR32NB`
   relocations. Require that `.pdata` section to use `IMAGE_COMDAT_SELECT_ASSOCIATIVE` and name the
   function section as its direct parent.
4. Follow the entry's exact unwind relocation target into `.xdata`. Require that section to use
   `IMAGE_COMDAT_SELECT_ASSOCIATIVE` with the same direct function parent. Parent, `.pdata`, and
   `.xdata` must be three distinct sections.
5. Bind the demanded function and unwind target's exact MSVC map owners to `chosen.obj` inside the
   declared archive. Read all three raw `.pdata` addends, recompute the selected function's begin
   and end RVAs plus the selected `.xdata` unwind RVA, and require exactly one final
   exception-directory entry whose three fields equal those independent object-and-map results.
6. Keep host-independent relationship mutations that independently remove the parent COMDAT,
   change either child's selection, or redirect either child's parent. Keep exact archive-member
   owner spelling separate from direct-object owner spelling and reject suffix spoofs.
7. Execute the unchanged four archive images across low and optimized profiles and require their
   existing address, selected-value, process-result, PE, and unwind checks to remain green.

## Evidence and limits

The four existing direct-first/archive-first images now prove more without increasing the image
count. For each link, the preferred archive index and final map select `chosen.obj`; that exact
object contains a demanded function root plus direct associative `.pdata` and `.xdata` children;
and all three raw `.pdata` addends plus the exact parent and unwind map symbols reproduce the unique
final runtime-function entry. All 53 PE images still cross the bounded structural and unwind inspectors.

This proves retention of the selected parent's unwind children. ADR 0236 separately proves that one
discarded ANY-COMDAT function loses its distinct direct unwind payload. Neither decision proves
nested association chains, non-unwind associative data, handler or chained children,
`NODUPLICATES`, `SAME_SIZE`, `EXACT_MATCH`, or `LARGEST` selection, cyclic archives,
`/WHOLEARCHIVE`, incremental linking, LTCG, non-MSVC linkers, or hosted same-commit evidence.

## External basis

- Microsoft PE/COFF COMDAT sections and x64 exception data: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC `/Gy`: <https://learn.microsoft.com/en-us/cpp/build/reference/gy-enable-function-level-linking?view=msvc-170>
- MSVC `/MAP`: <https://learn.microsoft.com/en-us/cpp/build/reference/map-generate-mapfile?view=msvc-170>
- MSVC `/OPT`: <https://learn.microsoft.com/en-us/cpp/build/reference/opt-optimizations?view=msvc-170>
