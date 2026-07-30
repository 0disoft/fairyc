# ADR 0232: AMD64 REL32 object-to-image proof

- Status: Implemented local synthetic and native-smoke gate
- Date: 2026-07-31
- Object and image inspector: `internal/pecoff/relocation_link.go`
- Probe generator: `internal/windowsrelocationprobe/`
- MSVC driver: `internal/cdriver/`
- Native runner: `cmd/fairyc-windows-host-smoke/`
- Depends on: ADR 0078, ADR 0079, and ADR 0231

## Context

ADR 0231 freezes primary symbols and the numeric `IMAGE_REL_AMD64_REL32` identity, but that does not
prove the linker consumed the object addend or wrote the correct signed displacement. An inspector
could preserve the relocation record while reading the wrong operand bytes, mapping the call site
to the wrong function inside a shared `.text` section, or accepting a final image whose branch lands
at another address.

A linker map is not the right local oracle. Its text syntax and symbol presentation vary across
toolchain revisions. The proof instead needs bounded binary inputs: the original caller object, the
linked PE image, exact source and target symbols, and a project-owned image receipt containing their
preferred-image addresses.

## Decision

1. Generate two deterministic C11 translation units. `caller.c` contains one non-inlined direct call
   from `fy_rel32_source` to the separately compiled `fy_rel32_target`, consumes the result before
   adding one, and retains a read-only `.fyreloc` address receipt. `target.c` defines the non-inlined
   target. Runtime execution must return zero after observing the value `42`.
2. Build and execute the pair with the configured local x64 MSVC toolchain at low and optimized
   profiles. The existing path compiles each source separately, does not request `/GL`, and links
   non-incrementally. Every object still passes the general raw-COFF inspector and the final image
   still passes PE and unwind inspection.
3. In `caller.obj`, require one exact defined source symbol and one exact undefined cross-object
   target symbol. Select one `IMAGE_REL_AMD64_REL32` targeting that raw primary-symbol index inside
   the source symbol's same-section range. The range ends at the next higher-valued primary symbol
   in a shared section or at the section end. This excludes unrelated references from `main`.
4. Require both the object and linked instruction to use opcode `E8` followed by a four-byte
   displacement. Read the signed object addend directly from the relocation operand.
5. Require the named linked-image receipt section to be readable, non-code, non-executable, and
   non-writable, then limit file-backed search to `min(VirtualSize, RawSize)` so file-alignment
   padding cannot impersonate loaded receipt data. Find exactly one 16-byte magic in that logical
   span. The following two eight-byte fields are the preferred-image virtual addresses of source and
   target. Both addresses must map to executable image ranges.
6. Map the object relocation's source-relative position onto the linked source RVA. Recompute the
   Microsoft AMD64 REL32 result as `S + A - (P + 4)`, require signed-32-bit range, and compare it to
   the exact four bytes in the linked PE image.
7. Keep host-independent synthetic tests. They independently mutate relocation type and target,
   object and linked opcodes, object addend, linked displacement, source and target receipt values,
   duplicate receipt magic, and source-range selection.
8. Keep `pe-coff-object-fixtures` fail-closed until a fresh hosted Windows job succeeds on the exact
   commit containing ADR 0231 and this proof. Local MSVC evidence does not replace that same-commit
   gate.

## Evidence and limits

The host-independent fixture proves the binary mapping and arithmetic without invoking a compiler.
The configured Windows smoke then repeats the proof over actual MSVC objects and linked executables
at both profiles before executing them. The object addend, preferred-image source and target RVAs,
linked site RVA, actual displacement, and independently computed displacement remain separate report
fields.

This is one direct `CALL rel32` crossing between two MSVC C translation units. Archive extraction
and pick-any COMDAT selection are separately proved by ADR 0233 and do not widen this arithmetic
proof. ADR 0234 separately reuses the probe to prove one ADDR64 pointer and one unwind `.pdata`
ADDR32NB begin field. This ADR does not prove
`REL32_1` through `REL32_5`, RIP-relative data addressing, remaining absolute, image-relative,
section-relative, token, or unwind relocations, overflow records, incremental linking, LTCG, weak externals,
import thunks, ASLR rebasing at load time,
independent producer compatibility, linker-map provenance, CodeView/PDB, or a public Fairy ABI.

## External basis

- Microsoft PE/COFF format: <https://learn.microsoft.com/en-us/windows/win32/debug/pe-format>
