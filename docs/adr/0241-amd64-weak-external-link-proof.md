# ADR 0241: AMD64 weak-external link proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object generator: `internal/pecoff/weak_external_object.go`
- Inspector: `internal/pecoff/weak_external_link.go`
- Probe: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0079, ADR 0232, ADR 0237

## Context

The COFF inspector previously skipped auxiliary-record meaning except section definitions. An
undefined symbol with one opaque auxiliary record could therefore look structurally acceptable even
though LINK did not recognize it as a weak external. Weak resolution also has two distinct outcomes:
an absent symbol selects its declared fallback, while a present strong definition overrides that
fallback. Proving only one outcome leaves half of the selection contract unchecked.

Microsoft's PE/COFF weak-external description names a `TagIndex`, a search characteristic, and ten
reserved bytes. The selected local LINK additionally requires storage class 105 for the weak symbol;
using ordinary external class 2 with the same auxiliary bytes produced `LNK2019`. The native
consumer result therefore owns that producer detail for this closed probe.

## Decision

1. Extend private `ObjectSymbol` receipts with weak fallback index/name and search characteristic.
   Parse weak records only for storage class 105. Require undefined section zero, value zero,
   exactly one auxiliary record, a primary-symbol fallback index, search characteristic 1 through 3,
   and ten zero reserved bytes. Reject ordinary undefined externals that carry opaque auxiliaries.
2. Generate one exact AMD64 object with one readable executable `.text` section. It contains two
   leaf `JMP rel32` entries, two weak `SEARCH_ALIAS` symbols, and two defined local fallback
   functions. The first weak name is absent from the link; the second is strongly defined by
   `target.obj`.
3. Require each entry relocation to target its exact weak primary-symbol index with
   `IMAGE_REL_AMD64_REL32`. Bind each entry to `weak.obj` through the MSVC map.
4. For the missing name, bind the selected map target to the declared local fallback in `weak.obj`.
   For the present name, bind the selected target to the strong definition in `target.obj`, not the
   alternate local fallback.
5. Recompute both final displacements from the raw object addends and final preferred-image RVAs
   using `S + A - (P + 4)`. Require source and selected target addresses to be executable and both
   linked opcodes to remain direct tail jumps.
6. Reuse the existing low and optimized relocation images. The C caller executes both entries and
   requires the missing-name path to return 17 and the strong-override path to return 29. No new PE
   image or process execution is added.
7. Preserve mutations for invalid search, ordinary-external storage class, auxiliary fallback
   index, reserved bytes, relocation target, selected map owner, and each linked displacement.

## Evidence and limits

The configured local Windows smoke now proves both `SEARCH_ALIAS` outcomes through one exact raw
object, real MSVC LINK, final-image formula inspection, and native execution at low and optimized
profiles. The first attempted class-2 producer failed with `LNK2019`; class 105 is the actual
accepted producer contract for the selected MSVC 14.51 toolchain.

This is a private validation object, not a Fairy C object writer, public linker API, or source-level
weak declaration. ADR 0245 separately proves `SEARCH_NOLIBRARY` and `SEARCH_LIBRARY` against two
archive members. Missing fallbacks, weak chains or cycles, COMDAT interaction, imports, incremental link,
LTCG, non-MSVC linkers, another MSVC release, a versioned raw-object fixture, or hosted same-commit
evidence. `pe-coff-object-fixtures` remains fail-closed.

## External basis

- Microsoft PE/COFF weak externals and AMD64 relocations:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
