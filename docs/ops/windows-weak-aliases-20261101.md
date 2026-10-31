# Windows weak linking receipt, 2026-11-01

- Code: `cbd3b26a`; native Windows amd64, MSVC 14.51.36231, C11 low/optimized.
- Owners: `internal/pecoff/weak_alias_graph.go`,
  `internal/cdriver/weak_alias_windows_amd64_test.go`, and
  `internal/cdriver/generated_relocations_windows_amd64_test.go`.
- Boundary: explicit-object weak chains and actual CRT alternatives. No Fairy weak-declaration
  syntax, public export rule, source ABI, production compiler option or target promotion is added.

## Actual use inventory

The five workspaces from the [generated relocation receipt](windows-coff-relocations-20261101.md)
were checked, lowered, emitted, separately compiled and executed again at both profiles. Their
twenty `unit.obj`/`runtime.obj` objects contain **zero class-105 weak externals**. Four workspace
pairs also contain zero `/alternatename` directives. `process_args_quickstart/runtime.obj` offers
`_Avx2WmemEnabled=_Avx2WmemEnabledWeakValue` once per profile through the SDK `wchar.h` header.
That runtime has zero relocations referring to the root; neither name appears in its final map.
An offered directive in this corpus is therefore not evidence that its fallback was selected.
The inventory rejects newly observed weak records, unknown directives or new live references;
an older SDK may omit this one known directive entirely.

An actual `libcmt.lib(initializers.obj)` member was separately extracted using the resolved
MSVC `lib.exe`. Its metadata SHA-256 is
`2f9820b576e7f8793f46aa38b0f402b0f695820b4e435745447526b8e569bb4d`.
It contains eleven alternate-name directives. The metadata-only inspection path keeps normal
executable-object admission unchanged. Both low/optimized startup probes prove that this member
is loaded and that the final map selects `__acrt_initialize` from `libucrt:initialization.obj`
and `__vcrt_initialize` from `libvcruntime:initialization.obj`, ahead of the offered stub.
Both programs exit 42 with empty streams. This observation does not enumerate all archive inputs
or establish behavior when a default library is absent; those belong to ordered item 6.

## Chain, replacement and cycle selection

The controlled matrix uses two distinct forms: real class-105 records in a synthetic AMD64
COFF object, and `/alternatename` directives in independently compiled C. Strong replacement
definitions are compiled into a separate C provider. The class-105 chain uses ALIAS search;
this record does not recount historical LIBRARY/NOLIBRARY native probes as new executions.

| Case | Root A selection | Root B selection | Native observation |
| --- | --- | --- | --- |
| `A -> B -> fallback` | fallback, 17 | fallback, 17 | exit 42 |
| Strong A | provider A, 42 | fallback, 17 | exit 42 |
| Strong B | provider B, 42 | provider B, 42 | exit 42 |
| `A -> B -> A`, strong A | provider A, 42 | provider A, 42 | exit 42 |
| `A -> B -> A`, strong B | provider B, 42 | provider B, 42 | exit 42 |
| Cycle without strong definition | unresolved | unresolved | rejected |

Five resolved cases, two forms, two profiles and forward/reversed object order give **40 native
executions**. Both roots are bound to their exact object relocation, addend, entry range, selected
map owner and final PE REL32 bytes. The C observer checks both return values; every executable
also passes PE/unwind inspection. Mutating the linked displacement or map owner independently
causes inspection failure. Choosing a replacement means the call site selects that target; it
does not claim that every unreferenced fallback byte is physically discarded from the image.
General archive/COMDAT retention remains ordered item 7.

For each of the four unresolved-cycle form/profile combinations, the bounded explicit-object
planner rejects the cycle. An independent real LINK invocation, with a five-second deadline,
also fails with `LNK2019` for A/B and `LNK1120` for two unresolved externals. No deadline was hit.
The planner is an inspection/preflight utility; it is not a new production driver rejection path.
Unit checks additionally reject unresolved terminals, contradictory alternatives, ambiguous
strong definitions, nonlocal object identities and LIBRARY-mode guesses without archive evidence.
Explicit-object plans are limited to sixteen objects and 4,096 visited symbols; archive search
and competing COMDAT definitions are not inferred by this planner.

## Actual SDK data alternative

A separate C probe includes the real SDK `wchar.h`, with no handwritten alias pragma. It reads
`_Avx2WmemEnabled` and verifies the default fallback value 0, then independently links a strong
C provider containing value 7. Both cases pass at both profiles: **four native executions**.
Inspection binds the selected map owner and four-byte data value to the exact final-image
REL32_1 field, including its trailing-byte adjustment. This is live selection evidence for the
otherwise unreferenced directive observed in the production runtime corpus. The test explicitly
skips this SDK-specific probe if the selected header does not offer the measured directive;
there was no such skip on the recorded host.

## Commands and outcomes

Executed successfully on Windows:

```powershell
go test -mod=readonly ./internal/pecoff -count=1 -v
go test -mod=readonly ./internal/cdriver -run '^(TestWindowsDirectWeakAliasSelection|TestWindowsActualCRTWeakAlternatives|TestWindowsActualCRTWeakDataAlternative|TestGeneratedRelocationCoverage)$' -count=1 -v
go test -mod=readonly ./internal/cdriver -run '^(TestWindowsActualCRTWeakDataAlternative|TestGeneratedRelocationCoverage)$' -count=1 -v
go vet -mod=readonly ./internal/pecoff ./internal/cdriver
git diff --check
```

The last focused native rerun covers the SDK-presence guard refinement; unchanged chain/CRT
startup and PE/COFF results are reused from the full focused run. No execution is double-counted.
The named suites establish **56 positive native executions**: 40 controlled selections, four
SDK data cases, two actual CRT startup cases and ten production corpus images. Four intentional
LINK failures are separate negative results. Local transient logs are under
`.cache/weak-alias-*-20261101.log` and `.cache/weak-header-final-20261101.log`; this document retains
the bounded observations, not the temporary binaries or timestamped image identities.

## Omitted work and remaining limits

No Linux or WSL command ran. The existing combined Linux queue remains deferred at the
maintainer's request. Full `go test ./...`, hosted CI, other Windows toolchains, clean-machine
distribution, complete archive search and arbitrary weak/COMDAT graphs were not verified here.
No dependency, public API/SDK ABI, DB, repository hygiene rule, CI runner or workflow changed.
No push, workflow dispatch, publication or deployment was performed.
