# ADR 0233: MSVC archive extraction and COMDAT selection proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Archive and link-map inspector: `internal/pecoff/archive.go`, `internal/pecoff/archive_link.go`
- Probe generator: `internal/windowsarchiveprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0078, ADR 0079, ADR 0231, and ADR 0232

## Context

An object-level COMDAT receipt does not prove which duplicate reaches the final image. Likewise, an
archive containing a required symbol does not prove that its preferred index points to the intended
member, that the linker extracts that member, or that an unrelated member stays out. Program output
alone can hide every one of those mistakes when candidate definitions happen to behave alike.

PE images do not retain complete static-library provenance. This proof therefore needs four
separate evidence layers: the raw archive and preferred index, the candidate objects and COMDAT
metadata, the MSVC map that attributes final public symbols to inputs, and a final-image receipt
whose addresses and selected value can be checked against that map.

## Decision

1. Generate four deterministic C11 translation units. `chosen.c` defines the demanded
   `fy_archive_target` and a read-only pick-any `fy_pick_any_value` candidate with value `17`.
   `unused.c` defines `fy_archive_unused`. `direct.c` defines the competing pick-any candidate with
   value `23`. `caller.c` references the demand and COMDAT and emits a read-only `.fysel` receipt
   containing their preferred-image addresses.
2. Compile every source separately with local x64 MSVC. Use `/Gy` for the caller, demanded member,
   and direct competitor, but `/Gy-` for `unused.c` so its function is an ordinary non-COMDAT
   sentinel that `/OPT:REF` cannot discard independently of its archive member. Build `choice.lib`
   from basename inputs `chosen.obj` and `unused.obj` so temporary host paths cannot become member identity.
   Link `caller.obj`, `direct.obj`, and the archive in both direct-first and archive-first orders at
   low and optimized profiles with `/OPT:REF`, `/OPT:NOICF`, non-incremental linking, and an exact
   `/MAP` output. Execute all four images and require zero with no output.
3. Parse the Microsoft COFF archive signature, even-byte member layout, both linker members, and
   every contained AMD64 object under explicit size, member, and symbol budgets. Prefer the second
   linker member as current Microsoft linkers do. Require its member offsets to name every closed
   probe object, its symbols to be lexically ordered, and its one-based indices to resolve to exact
   member headers.
4. Require the preferred index to map `fy_archive_target` to `chosen.obj` and
   `fy_archive_unused` to `unused.obj`. Require both symbols to be exact defined external functions
   in their owning objects, and require the unused sentinel's section to be non-COMDAT. The direct
   and archive `fy_pick_any_value` definitions must each be
   exact external data in readable, non-writable `IMAGE_COMDAT_SELECT_ANY` sections. Read each
   symbol-relative raw four-byte initializer and require the archive candidate to be `17` and the
   direct candidate to be `23` before consulting final-link evidence.
5. Parse only exact public-symbol rows from the bounded UTF-8 MSVC map. Require the demanded
   function's owner to be `choice:chosen.obj`, accept the observed extension-free archive spelling
   as well as explicit `.lib` forms, and require the non-COMDAT unused sentinel to be absent. The
   demanded owner proves `chosen.obj` contributed to the final link; the unused sentinel's absence
   proves `unused.obj` did not contribute, without claiming an unavailable internal extraction trace.
   Require the COMDAT
   owner to be either the declared archive member or `direct.obj`; no link-order winner is assumed.
6. Require one exact 16-byte magic in the file-backed logical span of the read-only `.fysel`
   section. Bind its target and COMDAT preferred addresses to the corresponding map rows. The target
   must map to executable code. The COMDAT must map to read-only data whose exact `i32` value is
   `17` for the archive owner or `23` for the direct owner.
7. Keep host-independent mutations for malformed archive headers and indexes, out-of-range indexed
   symbols, non-pick-any candidates, candidate object initializer drift, packaged unused sentinels,
   wrong map owners, duplicate rows, retained unused symbols, address drift, final
   candidate value drift, and writable receipt data.
8. Keep `pe-coff-object-fixtures` fail-closed until a fresh hosted Windows job succeeds on the exact
   commit. Local LIB/LINK evidence does not replace that same-commit gate.

## Evidence and limits

The configured Windows smoke now performs four archive input-order executions across low and
optimized profiles. Every candidate object crosses the raw COFF inspector, both archive members
cross the bounded archive inspector, and the demanded archive owner plus non-COMDAT unused sentinel
distinguish selected-member contribution from final non-contribution. All four linked images cross
PE and unwind inspection, the object initializers, map,
and final-image receipt agree, and runtime execution agrees with the selected value. Together with
the preceding probes, the smoke now inspects 53 linked target or probe PE images.

`IMAGE_COMDAT_SELECT_ANY` permits any candidate to win, so input order is an exercised dimension,
not a promised selection rule. This proof covers one ordinary MSVC archive, one demanded member,
one unrequested member, one pick-any read-only data COMDAT, and one direct competitor. ADR 0235
separately binds the selected function parent to its direct associative unwind children. This proof
does not prove cyclic archives, rescans, `/WHOLEARCHIVE`, import libraries, weak externals,
discarded-parent or non-unwind associative retention, `NODUPLICATES`, `SAME_SIZE`, `EXACT_MATCH`, or `LARGEST` final selection,
identical-code folding, incremental linking, LTCG, debug provenance, non-MSVC linkers, a stable map
syntax, a public Fairy library format, or hosted same-commit evidence.

## External basis

- Microsoft PE/COFF format and archive members: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC `/MAP` output: <https://learn.microsoft.com/en-us/cpp/build/reference/map-generate-mapfile?view=msvc-170>
- MSVC `selectany`: <https://learn.microsoft.com/en-us/cpp/cpp/selectany?view=msvc-170>
- MSVC `/OPT` behavior: <https://learn.microsoft.com/en-us/cpp/build/reference/opt-optimizations?view=msvc-170>
