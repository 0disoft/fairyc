# Windows default-library and CRT archive receipt, 2026-11-01

- Code: `53d7ed5f`; native Windows amd64, MSVC 14.51.36231, SDK 10.0.26100.0.
- Owners: `internal/pecoff/default_libraries.go`,
  `internal/cdriver/default_libraries_windows_amd64_test.go`, and
  `internal/cdriver/default_library_inventory_windows_amd64_test.go`.
- Profiles: ordinary C11 `/MT /W4 /WX` with `/Od` (low) or `/O2` (optimized).
  Low does not select the debug CRT. No new compiler or product-linker policy is introduced.

## Ordinary search and selected inputs

The actual Go Windows builder's existing map/library-log capture is used, with no extra default
libraries, debug switches or archive replacement in the ordinary build. The generated objects
request `LIBCMT` and `OLDNAMES` through `.drectve`. The real verbose link records six absolute
searched inputs, including the CRT's additional library searches. Every searched input is checked
against the resolved toolchain's recorded `LIB` directories and hashed, including libraries
that contribute no loaded member. A searched input is not automatically an extracted member.

The recorded roots are:

- MSVC: `C:\Program Files\Microsoft Visual Studio\18\Community\VC\Tools\MSVC\14.51.36231`
- SDK libraries: `C:\Program Files (x86)\Windows Kits\10\Lib\10.0.26100.0`

| Input relative to its root | Loaded members in the C control, each profile | SHA-256 |
| --- | ---: | --- |
| MSVC `lib/x64/LIBCMT.lib` | 38 | `74fe34fdba1d3c2d9ba052642dc995253ca918185b6036b1fb29cdb9beee5f80` |
| MSVC `lib/x64/OLDNAMES.lib` | 0 | `67653ba64d59a33964a811d950c9d424f7afacba8e613af60138f64fe10c2679` |
| MSVC `lib/x64/libvcruntime.lib` | 26 | `39fc636ac1c993517e8cdb86d030ff0fd9bbf4c497b4f741cde138797acfe93e` |
| SDK `ucrt/x64/libucrt.lib` | 159 | `ea2ffa5d9b1ebc3fa76f34c364471750db8951b341dbccbc1bef628ef1a20e39` |
| SDK `um/x64/kernel32.lib` | 90 | `341c7d56125a03b458e4d5093e4c79b33123ccfdfd610fe236937b8e6f3134bb` |
| SDK `um/x64/uuid.lib` | 0 | `29558f81492e99d7de1b921b6eb8f21d74c96474e18653465aff10fcb01c2626` |

The inspector binds `mainCRTStartup` to `LIBCMT:exe_main.obj`, `__acrt_initialize` to
`libucrt:initialization.obj`, and `__vcrt_initialize` to `libvcruntime:initialization.obj`.
Each owner must have an actual loaded-member trace associated with a searched absolute archive.
Mapped addresses must point to executable, file-backed final PE bytes; the startup address must
equal the PE entry point. This is symbol/address attribution, not byte-for-byte comparison of
every CRT function or verification of every archive index.

The native C control allocates, fills, observes and frees storage, then exits 42 with empty
streams. Its source SHA-256 is
`ce29fc12089665fe6a83b3afe3a3f2cf1e89c5cc1b10b8f5fa0baa8bb0b930cb`.
The same objects are linked again using `/NODEFAULTLIB`, all six absolute input files, and empty
`LIB`/`LIBPATH`. This isolated explicit-input build passes the same startup-owner checks,
PE/unwind inspection and native result. It does not rely on a fallback developer library path.

## Production generated-object corpus

The existing five-workspace relocation suite now records this library receipt for every image:
`std_bytes_quickstart`, `std_alloc_quickstart`, `std_text_from_buffer_quickstart`,
`std_vec_quickstart`, and `process_args_quickstart`. Low/optimized gives **ten native executions**
and twenty actual generated `unit.obj`/selected `runtime.obj` objects. No handwritten object
replaces Fairy output. Source, object and image hashes remain in the same test log alongside
the absolute library paths, archive hashes, loaded-member counts and selected CRT symbol RVAs.
All existing relocation, exact-output, argv and negative-control assertions still pass.
The table above gives C-control member counts; it is not a universal production member count.

## Missing, suppressed and reduced archives

Only disposable files and process-local environment copies are changed. Installed archive
hashes are checked again after the failure matrix. No installed file is removed or modified.
Each negative link has a ten-second deadline; no deadline was reached.

| Controlled change, each profile | Actual result |
| --- | --- |
| Replace absolute `LIBCMT`, `libucrt`, or `libvcruntime` input with a nonexistent path, with all default search disabled | Three `LNK1181` failures naming the missing input |
| Suppress each of those libraries with `/NODEFAULTLIB:name` during ordinary default linking | Three `LNK1120` failures; demanded `mainCRTStartup`, `malloc`, or `memset` remains unresolved |
| Remove the demanded member from a temporary copy of each actual archive and use that copy in the explicit-input link | Three `LNK1120` failures for the same demanded symbols |
| Search only a curated temporary library directory containing all inputs except `OLDNAMES` | `LNK1104` for `OLDNAMES.lib`, despite zero extracted members in the baseline |
| Also suppress that unused `OLDNAMES` default in the curated environment | Link succeeds; native exit 42, empty streams |

The member-removal cases are `LIBCMT(exe_main.obj)` defining `mainCRTStartup`,
`libucrt(malloc.obj)` defining `malloc`, and `libvcruntime(memset.obj)` defining `memset`.
The original map must attribute each demanded symbol to that exact owner. `lib.exe /LIST`
identifies a unique actual member, `/EXTRACT` yields a structurally inspected object defining
the symbol, and `/REMOVE /OUT` produces a separate reduced archive. A second listing confirms
the member is absent. Extracted-object and reduced-archive hashes are logged. Embedded vendor
build paths in member names are archive identities, not directories on this host.

There are **twenty expected link failures**, ten per profile. Positive runs total **sixteen**:
ten generated-corpus images, two ordinary C controls, two explicit-input relinks and two
unused-default suppression controls. Repeated development attempts are not added to these
counts. Wrong CRT map ownership and missing loaded-member evidence fail inspection. Portable
checks reject missing/relative/ambiguous library traces, loaded members without searched inputs,
invalid names and oversized text. Inspection is bounded to sixteen objects, sixteen searched
libraries, 4,096 loaded entries per library, 8 MiB logs and 64 KiB lines.

## Validation and limits

Passed on Windows:

```powershell
go test -mod=readonly ./internal/pecoff -count=1 -v
go test -mod=readonly ./internal/cdriver -run '^(TestWindowsDefaultLibrarySelection|TestGeneratedRelocationCoverage)$' -count=1 -v
go vet -mod=readonly ./internal/pecoff ./internal/cdriver
git diff --check
```

The focused native command was rerun on committed code `53d7ed5f`. Logs are transient local
`.cache/crt-final-53d7ed5f-20261101.log`, `.cache/crt-selection-20261101.log`,
`.cache/crt-generated-20261101.log`, and `.cache/crt-pecoff-20261101.log`. Temporary binaries
are not distributed or retained as a target-conformance artifact.

This evidence covers the Go Windows builder and the named generated corpus, not every Fairy
program, the complete self-hosted compiler or a clean Windows machine. `/MD`, `/MTd`, `/MDd`,
arbitrary default-library overrides, complete archive/COMDAT competition and runtime DLL
deployment remain outside this record. Ordered item 7 owns broader archive/COMDAT forms;
item 11 owns DLL dependency and transfer evidence. Target maturity is unchanged.

Linux/WSL work remains deferred at the maintainer's request; no Linux command ran. Full
`go test ./...` and hosted CI were outside this focused local validation. No public API/SDK ABI,
DB, dependency, repository hygiene rule, CI runner, workflow or production compiler flag changed.
No push, publication, workflow dispatch or deployment was performed.
