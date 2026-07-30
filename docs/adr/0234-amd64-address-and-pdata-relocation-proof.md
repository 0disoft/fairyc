# ADR 0234: AMD64 address and `.pdata` relocation proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Inspector: `internal/pecoff/address_link.go`
- Probe generator: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0078, ADR 0079, ADR 0080, ADR 0231, and ADR 0232

## Context

ADR 0232 proves one `IMAGE_REL_AMD64_REL32` call displacement, but the same object already carries
two other link-critical forms. Its read-only receipt contains absolute function pointers emitted as
`IMAGE_REL_AMD64_ADDR64`, and its `.pdata` runtime-function records begin with image-relative
`IMAGE_REL_AMD64_ADDR32NB` relocations. Merely accepting those numeric relocation types in the COFF
parser says nothing about their addends, final values, symbol ownership, or unwind-table meaning.

MSVC also does not necessarily target the public function symbol from `.pdata`. It commonly creates
an internal label such as `$LN3`, points both begin and end fields at that label, and distinguishes
the fields with addends. At optimized profiles, several same-named `.pdata` sections may coexist.
The inspector therefore needs the raw section index and symbol address identity; section-name or
symbol-name matching alone is ambiguous.

## Decision

1. Preserve one-based `SectionIndex` on every internal `ObjectRelocation` receipt. Relocation
   consumers must select the exact section index as well as its name. Strengthen the existing REL32
   proof to use that identity so duplicate `.text$...` names cannot mix relocation streams.
2. Reuse the two-translation-unit C11 relocation probe and its `.fyreloc` receipt. Capture a bounded
   MSVC `/MAP` file and disable identical COMDAT folding for this proof. Require exact public rows
   for `fy_rel32_source` from `caller.obj` and `fy_rel32_target` from `target.obj`.
3. Require `fy_rel32_receipt` to be exact external data in the declared read-only object section.
   Its object bytes must begin with the exact 16-byte magic. At the target-pointer field, require
   exactly one `IMAGE_REL_AMD64_ADDR64` relocation to the undefined external target function. Read
   the raw unsigned 64-bit addend and require the linked field to equal `S + A`, where `S` is the
   target's preferred-image map address.
4. Locate `.pdata` or `.pdata$...` sections by exact section index. The runtime-function begin
   relocation must be `IMAGE_REL_AMD64_ADDR32NB` at a 12-byte entry boundary. Its target may be the
   public source symbol or one exact primary alias with the same section number and value and no
   auxiliary records. This admits observed `$LN...` labels without accepting a merely similar name.
5. Read the raw unsigned 32-bit addend and require the final exception directory to contain exactly
   one validated runtime-function entry whose begin RVA equals `S + A - ImageBase`. The map-derived
   source RVA supplies `S - ImageBase`; the existing unwind inspector independently requires a
   nonempty executable range and a valid file-backed `UNWIND_INFO` reference.
6. Keep host-independent mutations for relocation type, exact site, duplicate relocation, receipt
   magic, ADDR64 addend and final value, map address and object owner, `.pdata` section identity,
   begin-alias value, ADDR32NB addend, and final runtime-function begin. Execute the unchanged probe
   at low and optimized profiles and require zero with no output.
7. Keep `pe-coff-object-fixtures` fail-closed until a fresh hosted Windows job succeeds on the exact
   commit. Local MSVC evidence is not a hosted same-commit receipt.

## Evidence and limits

The configured Windows smoke now proves REL32, ADDR64, and one unwind-bearing `.pdata` ADDR32NB
begin relocation in the same two low/optimized executions. This does not increase the executable
count: all 53 target or probe PE images still cross the structural and unwind inspectors. The
synthetic tests use the same internal-label alias shape observed from current MSVC objects.

This proof covers one absolute target pointer and one runtime-function begin field. It does not
prove the `.pdata` end-address relocation, the `.pdata` unwind-info relocation, handler or chained
tail relocations, `ADDR32`, `REL32_1` through `REL32_5`, `SECTION`, `SECREL`, import thunks, base
relocations, overflow records, incremental linking, LTCG, non-MSVC linkers, or a stable public
Fairy C object format.

## External basis

- Microsoft PE/COFF relocation types and x64 exception data: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- MSVC `/MAP`: <https://learn.microsoft.com/en-us/cpp/build/reference/map-generate-mapfile?view=msvc-170>
- MSVC `/OPT`: <https://learn.microsoft.com/en-us/cpp/build/reference/opt-optimizations?view=msvc-170>
