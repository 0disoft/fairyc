# ADR 0236: discarded associative COMDAT elimination proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Inspector: `internal/pecoff/discarded_associative_link.go`
- Probe generator: `internal/windowsarchiveprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0233, ADR 0234, and ADR 0235

## Context

ADR 0235 proves that the selected archive function keeps its direct associative `.pdata` and
`.xdata` children. That does not prove the inverse: a linker could select one parent correctly but
leave a losing parent's unwind child or raw metadata in the image.

A valid elimination proof needs both candidates to contribute to the link. Merely leaving one
archive member unextracted proves archive demand behavior, not section collection. It also needs a
candidate-specific child identity; absence of a duplicate parent symbol alone does not establish
that the associated child bytes disappeared.

## Decision

1. Keep the existing `choice.lib` extraction proof. Its uniquely demanded `fy_archive_target`
   forces `chosen.obj` into every link, while `direct.obj` is always an explicit input.
2. Compile `chosen.cpp` and `direct.cpp` as C++20 probe-only inputs. Each defines the same
   `extern "C"` inline `fy_associative_candidate` function, which MSVC emits as an
   `IMAGE_COMDAT_SELECT_ANY` parent with direct associative `.pdata` and `.xdata` children. The
   functions return the same value but use deliberately different volatile stack sizes.
3. Compile only those two probe candidates with `/GS-`. Large volatile frames would otherwise add
   `__GSHandlerCheck` relocations and turn the child payload into a handler-address oracle. This is
   test isolation, not a Fairy C product, runtime, or release security setting. C inputs and every
   ordinary product build keep their existing settings.
4. Require both raw objects to define the exact same external ANY-COMDAT function, direct
   associative `.pdata` and `.xdata` children, complete three-field `ADDR32NB` entries, and distinct
   self-contained version-1 or version-2 unwind payloads of at least eight bytes.
5. Use the exact MSVC map row to identify the selected archive or direct parent. Require the
   selected unwind target to have the same exact owner, then recompute begin, end, and unwind RVAs
   from the selected object's three raw addends and map addresses.
6. Require the final runtime-function entry and its unwind bytes to match the selected object.
   Enumerate every final runtime-function reference and every file-backed PE section; the losing
   object's distinct unwind payload must occur zero times in both views. Require the exception
   directory to own every nonzero file-backed byte in the final `.pdata` section, so an orphaned
   losing `.pdata` child cannot survive outside the directory.
7. Preserve host-independent mutations for identical candidate payloads, undeclared or mismatched
   owners, selected payload drift, a referenced losing payload, an unreferenced raw losing payload,
   orphan `.pdata` bytes, non-external or non-function parents, and a parent whose selection changes
   away from ANY.
8. Repeat the proof for direct-first and archive-first input orders at low and optimized profiles,
   then execute each image and retain the prior archive-selection assertions.

## Evidence and limits

The existing four archive images now prove that both competing parents reached the linker, exactly
one declared owner won, its direct associative unwind children produced the selected final entry,
the losing `.xdata` payload was absent from all runtime-function references and file-backed final
sections, and `.pdata` contained no nonzero bytes outside the exception directory. The probe still
contributes no additional image count.

This does not admit C++ as Fairy C source or make C++20 part of the language contract. It does not
prove nested association chains, non-unwind associative data, handler or chained unwind children,
`NODUPLICATES`, `SAME_SIZE`, `EXACT_MATCH`, or `LARGEST`, `/WHOLEARCHIVE`, incremental linking,
ICF, LTCG, non-MSVC linkers, or hosted same-commit evidence.

## External basis

- Microsoft PE/COFF COMDAT sections and x64 exception data: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC inline functions and function-level linking: <https://learn.microsoft.com/en-us/cpp/build/reference/gy-enable-function-level-linking?view=msvc-170>
- MSVC `/GS`: <https://learn.microsoft.com/en-us/cpp/build/reference/gs-buffer-security-check?view=msvc-170>
- MSVC `/OPT`: <https://learn.microsoft.com/en-us/cpp/build/reference/opt-optimizations?view=msvc-170>
