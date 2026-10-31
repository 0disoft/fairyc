# Bounded Windows unwind and handler-data receipt, 2026-11-01

- Sources: `4ae0abbe` (C-specific scope payloads), `a025545d` (remaining linked chain entries),
  `01d30ff6` (RBP/XMM6 chained fixture). Host: Windows amd64, MSVC 14.51.36231.
- Owners: `internal/pecoff/handler_scope.go`, `internal/pecoff/chain_frame_probe.go`,
  `internal/windowsunwindprobe/{scope,chain,chain_frame}.go`, and the corresponding tests in
  `tools/dev/fairyc-windows-host-smoke`.
- Scope: exact source/object/image observations and native execution for the named frames.
  No Fairy exception syntax, foreign unwinding through Fairy, public ABI or target promotion.

## C handler payloads from object bytes to live dispatch

Microsoft's [x64 exception-handling description](https://learn.microsoft.com/en-us/cpp/build/exception-handling-x64)
places language-specific data after the handler RVA and leaves its format to the handler.
This proof admits only `__C_specific_handler` and the simple `__GSHandlerCheck_SEH` scope layout.
The selected SDK `winnt.h` defines the scope count followed by begin/end/handler/jump DWORDs;
the local MSVC C handler source supplies the dispatch interpretation. This is a separate
inspection proof, not a change making the general PE inspector interpret arbitrary payloads.

`InspectAMD64ScopeTable` first reuses the exact handler-link proof for the function, owned
xdata, unwind prefix, handler target and final runtime-function entry. It then resolves each
scope address from its actual ADDR32NB relocation and independently owned section/map base.
Literal handler value 1 and finally jump value 0 are distinguished from address relocations.
Final scope bytes must equal the relocated object bytes, ranges must be executable and valid,
and handler/jump targets must fit the admitted forms. The GS case additionally compares its
four-byte trailer and checks its EH/UH flags. GS layouts with extra alignment metadata, C++ EH tables and
other handler names fail this bounded admission rather than being inferred. Counts are bounded
to 1..128 records; missing relocations and ambiguous owned section bases also fail.

The new Fairy entry calls a C companion and receives 42 after the following cases complete:

| Actual C path | Event trace | Finally abnormal flag |
| --- | --- | ---: |
| Normal return | 3: finally | 0 |
| Inner filter selects handler | 1, 2, 3: filter, catch, finally | 0 |
| Inner filter continues search; outer filter selects handler | 1, 4, 3, 5: inner filter, outer filter, abnormal finally, outer catch | 1 |

Filters verify both application payload parameters. A separate constant `EXCEPTION_EXECUTE_HANDLER`
case executes too. Low compiles that case into a filter funclet; optimized uses the literal
handler sentinel 1. The actual body frame is captured while its PC lies inside both nested
scopes. `RtlVirtualUnwind` returns the exact handler and data pointer independently predicted
from the file, and the live count agrees with the object/image scope table. All exceptions
finish inside C before control returns through the generated Fairy frame.

| Profile | Scope tables / records | Relocated scope fields | Body handler | GS trailer | Rejected mutations |
| --- | ---: | ---: | --- | --- | ---: |
| low | 3 / 4 | 15 | `__GSHandlerCheck_SEH` | `0x7b` | 29 |
| optimized | 3 / 4 | 14 | `__C_specific_handler` | absent | 28 |

The two new native executions exit 42. Fifty-seven negative checks change every payload DWORD
independently in the final image, change source addends, or supply counts 0/129. Payload-only
image mutations still pass the earlier header/handler-link proof and must fail the new scope
comparison. Mutated images are never loaded or handed to the OS unwinder. Source, generated C,
companion, object and image hashes are printed with the receipt. Cookie-corruption fail-fast,
aligned GS handling and arbitrary language payloads are not claimed.

## Remaining CRT chained-fragment entries

The previous linked-CRT chain probe checked prologs and nine decoded add/pop/ret epilogs but
left CALL/INT3 tails and other instruction sequences outside its runtime partition. It now also
inverts each otherwise admitted fragment's complete parent metadata onto a bounded stack and
checks that fragment's entry context with the real OS unwinder. This is **entry-only** evidence
for those extra fragments, not instruction decoding or body-path execution.

| Profile | Nonzero-prolog fragments / contexts | Decoded zero-prolog epilogs / contexts | Other zero-prolog entries / contexts | Unaccounted fragments |
| --- | ---: | ---: | ---: | ---: |
| low | 16 / 32 | 9 / 36 | 12 / 12 | 0 |
| optimized | 18 / 36 | 9 / 36 | 14 / 14 | 0 |

The 37/41 linked fragments are independently counted in Go from the inspected image. Their
native partition must agree and have zero unaccounted fragments; new frame/operation forms
cannot silently become skips. The maximum observed chain depth is two. Both profiles still
check integer restoration, caller PC/SP and vector preservation; the CRT cases here have no
actual XMM save operation. A receipt retains each zero-prolog fragment's RVA, original size and
at most its first 128 code bytes. Those bounded previews do not imply full instruction coverage.
Existing cycle, frame mismatch, stack-change, instruction and wrong-context controls still pass.

## RBP and actual XMM save/restore in a closed chained fixture

The new raw COFF fixture preserves RBP, allocates 64 bytes, establishes RBP at RSP+32 and saves
XMM6 at RSP+32. A primary and two secondary function-table entries share that fixed frame.
The old exact chain-link inspector binds both secondary tails' six ADDR32NB fields, including
nonzero addends, to owned object/map anchors and final bytes. The function really executes
and returns 42 before controlled contexts are tested.

Independent stacks/register sentinels cover instruction offsets 0, 1, 5, 10, 15, 16, 21, 26,
30 and 31: partial prolog, established frame, both secondary bodies and already-restored epilog
states. Each profile verifies ten contexts, seven pre-epilog establisher-frame results, three
XMM6 restorations and preservation of the other nonvolatile integer/vector registers. No
general establisher-frame result is claimed after the epilog starts.

Two wrong controlled stacks per profile must disagree with the OS result: misplaced XMM storage
and an incorrect frame pointer. Two other mutations per profile change the linked primary's
XMM register/offset while retaining generic unwind structure; exact object/image comparison
rejects them before loading. Fixture SHA-256 is
`95a43a145182ad836ea34d4182f7587007fc765546b7ab89847d330057be619c`.
This is a named platform fixture, not proof of Fairy compiler emission or archive selection of
frame-register chains. `/OPT:NOICF` remains fixture-only.

## Version 2 and existing mixed boundaries

The unchanged version-2 native probe was rerun: each profile observes three linked v2 records,
three pop/ret epilogs, six instruction-boundary contexts, one explicit-offset descriptor and
six rejected size mutations. No unrecognized instruction is silently accepted by that decoder.
The original closed chain fixture also passes three contexts and six tail fields per profile,
with three structurally valid wrong-parent tuple mutations rejected per profile.

The mixed-stack suite passes ten native executions: two live generated-Fairy/C callback walks
with C-contained catch/finally and eight expected assertion traps, including changed handler,
handler-data and caller-stack expectations. It preserves the separate trap channel and the
prohibition on foreign exception transfer through Fairy. These local results are not hosted
same-commit CI evidence; ordered item 9 remains queued.

Across the v2, linked-CRT chain, old closed chain and new framed-chain suites there are **204
controlled contexts**: 12 + 166 + 6 + 20. Native execution totals **20**: ten executions across
those suites plus the scope suite, and ten mixed-boundary executions. Eight of the twenty
intentionally terminate with the expected assertion trap; they are not exit-42 runs. Repeated
development attempts are not double-counted.

## Commands and limits

Passed on Windows:

```powershell
go test -mod=readonly ./internal/pecoff ./internal/windowsunwindprobe -count=1
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindows(HandlerScopeData|FrameChainUnwind|Version2Epilogs|ChainedUnwind|ChainedLink|MixedStackBoundaries)$' -count=1 -v
go vet -mod=readonly ./internal/pecoff ./internal/windowsunwindprobe ./tools/dev/fairyc-windows-host-smoke
git diff --check
```

Transient final logs are `.cache/unwind-final-20261101.log`; optional local JSON receipts are
`.cache/unwind-{chain,v2,chain-link,mixed}-final-20261101.json`. Source counts/hashes and declared
omissions are retained here; temporary binaries are not distribution artifacts.

The remaining broad limits are live CRT capture at arbitrary PCs, arbitrary frame registers,
far/unobserved XMM and other opcode families, all v2 epilog instruction forms, complete chain
equivalence, C++/aligned-GS/other handler grammars, production chain-tail emission/selection,
hosted CI and clean-machine execution. `windows-x64-unwind-fixtures` remains missing as a
complete conformance gate and target maturity is unchanged.

Linux/WSL remains deferred at the maintainer's request. Full `go test ./...` and hosted workflows
were outside this focused run. No Linux command ran. Public API/SDK ABI, DB, dependency, production
compiler flags, repository hygiene rules, CI runners and workflows are unchanged. No push, workflow
dispatch, publication or deployment was performed.
