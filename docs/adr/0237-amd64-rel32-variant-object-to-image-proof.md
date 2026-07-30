# ADR 0237: AMD64 REL32 variant object-to-image proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object builder: `internal/pecoff/relocation_variants_object.go`
- Inspector: `internal/pecoff/relocation_variants.go`
- Probe generator: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0231, ADR 0232, and ADR 0234

## Context

ADR 0232 proves the ordinary AMD64 `REL32` formula for one compiler-emitted cross-object call.
PE/COFF also defines `REL32_1` through `REL32_5`, whose reference position advances by one through
five additional bytes. Merely admitting their numeric relocation types in a structural object
parser does not prove that Fairy C's link inspection uses those distinct formulas or that the local
MSVC linker accepts the object form.

Normal C source does not provide a stable way to ask MSVC to emit all six variants in one bounded
function. The test therefore needs a generated, exact COFF object while retaining a real LINK and
final-image oracle.

## Decision

1. Generate one closed AMD64 COFF object named `variants.obj`. Its external function `fy_vsrc`
   jumps over a non-executed 39-byte table, then returns 42. The table contains exactly one
   relocation targeting undefined external function `fy_vdst` for each type from `REL32` through
   `REL32_5`.
2. Give every table slot a distinct signed object addend. Preserve one through five marker bytes
   after the corresponding suffixed relocation so a shifted or overlapping final-image read cannot
   satisfy the displacement check accidentally.
3. Pass the generated object after the MSVC driver's `/link` boundary. `/TC` remains scoped to the
   two C source inputs and cannot reinterpret the prebuilt object as C text. Validate every prebuilt
   object with the same bounded COFF inspector before writing or linking it.
4. Require exact object symbols, one exact source-to-target relocation of each type, and exact MSVC
   map ownership: `fy_vsrc` belongs to `variants.obj` and `fy_vdst` belongs to `target.obj`.
5. Map the source and target preferred addresses into the final PE. For trailing-byte count `n` in
   zero through five, read the raw object addend `A` and final displacement and independently require
   `S + A - (P + 4 + n)`, where `S` is the linked target RVA and `P` is the linked relocation-site
   RVA.
6. Require every final site to remain in executable code, compare all marker bytes with the object,
   execute the resulting image, and repeat the proof at low and optimized profiles.
7. Keep host-independent mutation tests for missing or duplicate types, wrong targets, addend and
   linked-byte drift, marker drift, and wrong source or target map owners.

## Evidence and limits

The existing two relocation-probe executions now consume the exact generated object with local
MSVC 14.51 LINK, prove all six formulas against the low and optimized final images, and execute the
linked `fy_vsrc` path. This adds no PE image to the smoke count.

This is compiler-private validation evidence, not a stable Fairy C object format or permission for
arbitrary user-supplied objects. ADR 0238 separately proves object-level CodeView `SECTION`/`SECREL`
pairs and final PE/PDB identity without claiming a resolved object-to-PDB record mapping. This ADR
does not prove token, base-relocation, incremental-link, LTCG, weak-external, overflow-relocation,
non-MSVC-linker, or remaining CodeView/PDB record semantics.

## External basis

- Microsoft PE/COFF AMD64 relocation types: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC `/TC`: <https://learn.microsoft.com/en-us/cpp/build/reference/tc-tp-specify-source-file-type?view=msvc-170>
- MSVC `/link`: <https://learn.microsoft.com/en-us/cpp/build/reference/link-pass-options-to-linker?view=msvc-170>
