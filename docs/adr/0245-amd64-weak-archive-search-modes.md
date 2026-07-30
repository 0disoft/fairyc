# ADR 0245: AMD64 weak archive search modes

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object generator: `internal/pecoff/weak_external_object.go`
- Link inspector: `internal/pecoff/weak_search_link.go`
- Probe: `internal/windowsweaksearchprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0233, ADR 0241, ADR 0244

## Context

ADR 0241 admitted all three Microsoft weak-external search characteristic values but linked only
`SEARCH_ALIAS`. Parser acceptance is not linker-semantics evidence. `SEARCH_NOLIBRARY` and
`SEARCH_LIBRARY` differ specifically in whether the linker may search an archive for the weak
symbol before using its fallback, so a direct strong object cannot distinguish them.

The weak symbol name itself is also not a valid extraction oracle. MSVC map output can retain that
name when the final relocation resolves to the local fallback. Archive extraction needs an
independent member-owned witness.

## Decision

1. Generate one exact ordinary AMD64 object with two direct tail jumps. The first weak auxiliary
   uses `SEARCH_NOLIBRARY`; the second uses `SEARCH_LIBRARY`. Both name distinct object-local
   fallbacks and distinct archive strong symbols.
2. Compile each strong symbol into a separate C11 archive member. The NOLIBRARY member also defines
   a unique sentinel in the same ordinary non-COMDAT `.text` section, compiled without function
   packaging. If that member is extracted for its strong symbol, its sentinel cannot be explained
   away as an independently discarded COMDAT.
3. Require the archive's preferred index to map both strong symbols and the sentinel to their exact
   members. Require the final map to omit the NOLIBRARY sentinel and attribute the LIBRARY-selected
   strong definition to its exact archive member.
4. Recompute both final `IMAGE_REL_AMD64_REL32` displacements from the object addend and final map
   addresses. The NOLIBRARY jump must target its local fallback; the LIBRARY jump must target the
   archive definition.
5. Execute the same closed probe under MSVC low and optimized profiles. Require values 17 and 31,
   empty output, and status zero.
6. Add the generated search-mode object to `fairyc.coff-generated-object-fixtures/v0`, freezing
   generator inputs, complete-byte length and SHA-256, all primary symbols, both weak search values,
   and both relocations.

## Evidence and limits

Unit tests freeze both auxiliary values and reject independent changes to either search mode,
archive extraction sentinel, selected archive owner, and linked displacement. The configured local
Windows smoke builds one two-member archive with MSVC 14.51, links and inspects two PE images, and
executes both successfully. The lane now inspects 55 target or probe PE images.

This proves one compiler-private ordinary-object/archive arrangement. It does not define a Fairy C
weak declaration, prove missing or invalid fallback behavior, weak chains or cycles, COMDAT weak
targets, default libraries, `/NODEFAULTLIB`, incremental linking, LTCG, another linker or MSVC
release, raw stored object copies, or hosted same-commit evidence. `pe-coff-object-fixtures` remains
fail-closed.

## External basis

- Microsoft PE/COFF weak external search characteristics:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
