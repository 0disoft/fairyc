# ADR 0240: AMD64 unwind-handler relocation proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Inspector: `internal/pecoff/handler_link.go`
- Probe generator: `internal/windowsunwindprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0080, ADR 0081, ADR 0234

## Context

ADR 0234 binds all three `.pdata` `ADDR32NB` fields to one final runtime-function entry, while
ADR 0081 proves that the Windows dispatcher and `RtlVirtualUnwind` consume the resulting metadata.
Neither proves the optional handler address stored after an `UNWIND_INFO` code array. A structurally
valid record could target the wrong runtime handler even when the enclosing function and `.pdata`
entry are correct.

The closed probe also demonstrates why the expected symbol must be profile-specific. With the same
ordinary `/GS` policy, current MSVC emits `__GSHandlerCheck_SEH` for the low profile and
`__C_specific_handler` for the optimized profile. Disabling `/GS` would make the fixture simpler by
removing the behavior under test, so this proof preserves the normal build settings.

## Decision

1. Preserve exact probe identities for `fy_unwind_frame`, `frame.obj`, and the low and optimized
   handler symbols. An unknown optimization profile has no fallback handler identity.
2. In `frame.obj`, locate the exact external function and its complete three-relocation `.pdata`
   entry. Use the raw unwind relocation and addend to locate the owning `.xdata` record by exact
   section index.
3. Require version 1 or 2 `UNWIND_INFO` with an exception or unwind-handler flag and without
   `UNW_FLAG_CHAININFO`. Compute the aligned optional-tail offset from the raw unwind-code count and
   require exactly one relocation at that site.
4. Require that tail relocation to be `IMAGE_REL_AMD64_ADDR32NB` and to target the profile's exact
   undefined external function symbol. Preserve its raw unsigned addend.
5. Bind the source function and unwind symbol to `frame.obj` through exact MSVC map rows. Bind the
   handler to a nonempty MSVC map owner, convert all preferred addresses to RVAs, and compute
   the expected handler value as `S + A - ImageBase`.
6. Find the unique final runtime-function entry by the independently computed source RVA. Require
   its unwind RVA, header and aligned unwind-code prefix, final handler RVA, and executable target
   range to match the object and map evidence exactly.
7. Keep host-independent mutations for relocation type, cardinality, and target, target symbol
   definition and function type, addend, map address and owner, source owner, final handler value,
   unwind prefix, handler flags, and executable target range.
8. Run the same proof before executing the existing three-unit exception probe at low and optimized
   profiles. Preserve its zero exit, empty streams, exact exception payload, virtual frame order,
   and abnormal `__finally` execution oracles.

## Evidence and limits

The local Windows smoke now proves the actual profile-specific handler tail from raw `frame.obj`
through the MSVC map into the final PE, then asks the Windows dispatcher to execute the unchanged
exception path. The low profile binds `__GSHandlerCheck_SEH`; the optimized profile binds
`__C_specific_handler`. No extra executable is added.

This is one closed MSVC C fixture, not a Fairy C exception ABI or public object format. It does not
parse language-specific handler payload data, prove recursive chain equivalence or chained-tail
relocations, admit arbitrary handler symbols, stabilize runtime-library ownership, cover another
MSVC release or linker, exercise Fairy/foreign-frame crossings, or provide a fresh hosted
same-commit receipt. `windows-x64-unwind-fixtures` remains fail-closed.

## External basis

- Microsoft PE/COFF AMD64 relocations and exception data:
  <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
- Microsoft x64 exception handling:
  <https://learn.microsoft.com/en-us/cpp/build/exception-handling-x64?view=msvc-170>
- MSVC `/GS` buffer security checks:
  <https://learn.microsoft.com/en-us/cpp/build/reference/gs-buffer-security-check?view=msvc-170>
