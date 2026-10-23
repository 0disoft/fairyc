# Release

- Status: Experimental Windows preview preparation; not published

## Operational Contract

The first user-facing distribution is a Windows/AMD64 experimental compiler ZIP
for developers with local x64 MSVC and the Windows SDK. It contains the Go Stage 0
compiler and its notices, not the Fairy-written parser candidate. Checking and
semantic tests need no C toolchain; native tests and execution do. Linux target
promotion, self-hosting and C-relative performance budgets are separate work.

Source-built Linux GNU amd64 compilers also implement ordinary `build/run/test --native` with
GCC or Clang and the production SDK. This does not add a Linux binary release
or promote target conformance. Linux native results for this change are still pending;
see [ADR 0426](../adr/0426-linux-native-workspace-tests.md).

Build one candidate from a clean `main` commit with
`node tools/starter-smoke.mjs --package-local`. Record its commit, ZIP SHA-256, executable
SHA-256, Go toolchain and smoke receipt. Choose a SemVer prerelease tag only
after the candidate is fixed; the SDK version and language revision are separate
identities and must not be used as the compiler release number. Publish the
exact verified ZIP and its `.zip.sha256` file together. Do not rebuild after
approving an artifact and then reuse the earlier validation receipt.

The packaging command uses `-trimpath -ldflags="-s -w"` for the Go compiler executable.
It preserves version metadata and Go stack traces but omits native compiler debugging data.
Ordinary development builds keep that data. The receipt records build flags and executable/ZIP
bytes. At `3679d2a3` with Go 1.27.1, the compiler shrank from 17,849,344 to 13,094,400 bytes
(26.6%); these are executable sizes, not ZIP sizes or native-program size changes.

## Portability fixture

Use the extracted preview compiler to prepare a separate transfer fixture:

```text
node tools/windows-portability/prepare.mjs --compiler <extracted/fairyc.exe> --dumpbin <Hostx64/x64/dumpbin.exe>
```

Preparation needs Node, Go, MSVC and the Windows SDK on the build host. It records the exact
compiler identity, CGO setting, direct/delay-load DLL imports and file hashes, then checks the ZIP
round trip. The ZIP contains the compiler, two prebuilt native probes, a PowerShell verifier and
licenses; no source checkout, compiler cache or editor tooling is included.

On the receiving Windows/AMD64 host, `verify.ps1 -Receipt <new-receipt.json>` requires only Windows
PowerShell 5.1 or PowerShell 7. It verifies the bundle, creates/checks/tests a fresh starter, and
executes both native profiles with an empty child PATH. The probes exercise allocated bytes,
binary file reads/writes, sync/close and console output. Scratch data is removed; existing receipts
are never overwritten. The builder also runs this verifier locally, but that receipt explicitly
does not prove a clean host. Retain separate host/image and transfer evidence for that claim.

Current generated programs use `/MT`; the fixture requires only `kernel32.dll` in each image's
direct/delay-load import list. It does not enumerate dynamic DLL loads or establish compatibility
with every Windows version. A changed import stops preparation for dependency review. The separate
fixture does not replace the preview's native-build checks or promote target maturity.

Verifier regression check, using an already prepared bundle:

```text
node tools/windows-portability/test.mjs --bundle <extracted-bundle>
```

## Owners

- Primary owner: the `0disoft/fairyc` repository administrator performing the release
- Backup owner: none assigned for this experimental preview
- Escalation path: stop publication and return the decision to that administrator

## Validation

- Required validation names: `VALIDATION.md` (`smoke`, `check`, `docs`)
- Candidate gate: the extracted ZIP must pass `new`, `check`, locked check,
  semantic test, native test and run in both normal and release modes. The
  exact commit's `Windows native target smoke` CI job must pass. Triage other
  failed CI jobs and block on failures that affect the stable user path. Verify
  the ZIP and executable hashes against the retained candidate receipt.
- Public access gate: the repository is currently private. Before announcing
  a download, choose and verify a publicly accessible release location.
- Security gate: enable GitHub private vulnerability reporting on a public
  repository and verify that an unauthenticated visitor can find the private
  report route. Public issues are for nonsensitive bugs only.
- Release blocker status: any failed user-path step, relevant exact-commit CI
  failure, artifact hash mismatch, or unavailable private security-report route.
- Remaining operational risk: no clean user-machine installation proof yet;
  the first preview requires the user's separate MSVC installation and offers
  no compatibility or response-time promise.

Only publish after the gates above have been checked. After publication,
download the ZIP from the public page, compare its hash with the approved
candidate, and repeat the README starter path. If a blocking defect is found,
stop new downloads, mark the release withdrawn or superseded and publish a
correction. Already downloaded copies cannot be recalled; retain their exact
hash and the defect notice. Avoid extra measurement workflow runs for this
preview: the C-relative 8% budget does not decide whether people can try the
compiler.

## Parser preview assessment

See [parser public-readiness](parser-public-readiness.md) for the reviewed Go
user path, native candidate boundary, retained local evidence and outstanding
publication checks. This assessment is not release approval and does not fill
the unassigned ownership or undecided blocker fields above.
