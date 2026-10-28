# Release

- Status: Experimental Windows preview published: [v0.1.0-preview.1](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1)

## Windows editor preview draft (2026-10-28)

Local-only Korean draft: [windows-editor-preview-release-draft.ko.md](windows-editor-preview-release-draft.ko.md).
Exact file identities: [windows-editor-preview-assets.json](windows-editor-preview-assets.json).
The draft does not change the published status or release gates below.

## Operational Contract

New Windows/AMD64 preview candidates are experimental compiler and
language-server ZIPs for Windows/AMD64 developers. They contain the Go Stage 0
compiler, `fairyc-lsp.exe`, and their notices, not the Fairy-written parser
candidate. Both executables share the same clean source revision, Go toolchain
and `-trimpath -ldflags="-s -w"` flags. The VS Code VSIX remains a separate
package and is not in the ZIP. Checking and semantic tests need no C toolchain;
native tests and execution require a local x64 MSVC toolchain and Windows SDK.
Linux target promotion, self-hosting and C-relative performance budgets are
separate work.

Source-built Linux GNU amd64 compilers also implement ordinary `build/run/test --native` with
GCC or Clang and the production SDK. This does not add a Linux binary release
or promote target conformance. Linux native results for this change are still pending;
see [ADR 0426](../adr/0426-linux-native-workspace-tests.md).

Build one candidate from a clean `main` commit with
`node tools/starter-smoke.mjs --package-local`. Record its commit, ZIP SHA-256,
both executable hashes and sizes, Go toolchain and smoke receipt. Choose a
SemVer prerelease tag only after the candidate is fixed; the SDK version and
language revision are separate identities and must not be used as the compiler
release number. Publish the exact verified ZIP and its `.zip.sha256` file
together. Do not rebuild after approving an artifact and then reuse the earlier
validation receipt.

The packaging command uses `-trimpath -ldflags="-s -w"` for both Go executables.
It preserves version metadata and Go stack traces but omits native compiler debugging data.
Ordinary development builds keep that data. The receipt records build flags and executable/ZIP
bytes. At `3679d2a3` with Go 1.27.1, the compiler shrank from 17,849,344 to 13,094,400 bytes
(26.6%); these are executable sizes, not ZIP sizes or native-program size changes.

### Recheck a retained candidate

Before transfer or publication, recheck the exact retained candidate without rebuilding:

```text
node tools/windows-preview/verify.mjs --receipt <.cache/starter-smoke-.../receipt.json> --revision <full-approved-source-commit>
```

The command requires all sixteen successful starter steps (the original twelve
plus `lsp-build`, `lsp-build-metadata`, `lsp-version` and `lsp-protocol`), clean
Windows/AMD64 build identity, release build flags, the receipt-owned ZIP and its
checksum sidecar, the exact extracted file set, and compiler, language-server
and license-file hashes. A v2 receipt records both executable hashes and sizes.
The verifier still accepts historical v1 receipts with twelve compiler-only
steps; only v2 receipts establish the LSP build identity. Packaging now pins
`SHA256SUMS.txt` in the smoke receipt; older receipts without that identity
remain historical and cannot pass this recheck. Use a freshly packaged candidate
rather than editing an old receipt to add the field.

The checker streams file hashes and changes no artifacts. Its JSON reports only a local
artifact recheck, with clean-machine evidence and publication approval both false. It does not
authenticate a publisher or replace the exact-commit CI, security-reporting, or separate-host
gates below. Regression command: `node --test tools/windows-preview/verify.test.mjs`.

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

### Reported GRAM13 validation (2026-10-27)

The user reported successful validation on a separate physical laptop, GRAM13.
The original receiving-host JSON files are not present on the build host, so this
entry records the user's report rather than an independently inspected remote receipt.

- Candidate source revision: `1acc298e99182f530e8f1ba480c701dd8873e033`.
- Compiler SHA-256: `65c9c80834d9d858f3c34a95e76fb90e3ed651e83b54390e497992755bbd5e9a`.
  The reported pre/post-run hashes match the retained build-host candidate receipt.
- Transfer-fixture ZIP SHA-256: `24ef3776ba42b7164e1f6d0a6769379f4db6e14ba094ea930c9148df0a00403b`
  (build-host preparation identity; distinct from the compiler preview ZIP).
- Host: Windows 11 Pro 24H2, build `26100.9711`, x64; MSVC tools `14.51.36231`
  (`cl.exe 19.51.36248.0`) and Windows SDK `10.0.26100.0` installed.
- Transfer execution: ZIP checksum and nine internal file hashes/sizes matched;
  all seven verifier steps passed, with exit code 0 and an empty child PATH.
  This covers starter creation/checks/semantic tests and both prebuilt native probes.
- Fresh native compilation: all eleven commands returned 0. Both development and
  release passed native tests (one test each), built and linked new executables,
  directly executed those executables, and passed `fairyc run`.
- Receiving-host evidence: `fairyc-portability-verify-20261027-175035-results/portability-receipt.json`
  and `fairyc-native-verify-20261027-181502/results.json` under the laptop's Downloads;
  each directory also retains `REPORT.ko.txt`, and the native directory retains executables.

These are separate-host execution and fresh-compilation results with development tools
installed. They do not establish execution on a machine without MSVC/SDK, change
`cleanMachineEvidence`, or approve publication. Exact-candidate CI and public download
and private security-report route checks remain required before the next publication.

### Public distribution check (2026-10-27)

The build host checked GitHub without changing repository settings or publishing artifacts:

- The repository is public; the authenticated private-vulnerability-reporting API returns
  `enabled: true`. The unauthenticated [security overview](https://github.com/0disoft/fairyc/security)
  exposes the private report link. Following it anonymously reaches GitHub's login page
  with HTTP 200; no report was submitted and the authenticated submission form was not tested.
- The published [v0.1.0-preview.1](https://github.com/0disoft/fairyc/releases/tag/v0.1.0-preview.1)
  targets `682d6591d8db136e11e52ec5c01bdba454d9600e`. Anonymous downloads of its ZIP
  and checksum sidecar both return HTTP 200. The 4,648,779-byte ZIP matches its sidecar
  and GitHub asset digest: `d1dd38621691a92f99091f8b98d044cdf46739f0842a3264f9b9ff1ebe3fa73c`.
  These checks concern that older published artifact, not the GRAM13-tested candidate.
- Remote `main` is `c7355d733adb609f894db16b268b69da9e11d31f`. The commit API
  reports no commit for candidate `1acc298e99182f530e8f1ba480c701dd8873e033` (HTTP 422),
  and its workflow-run query returns no runs. The candidate has not been pushed or published;
  its exact-commit Windows CI gate remains pending.

Run CI on the candidate revision itself before publishing its retained ZIP. A push whose
tip includes later documentation commits runs CI on that newer tip and does not by itself
supply an exact-candidate workflow result. No workflow was dispatched during this check.

## Exact-candidate dispatch

`frontend.yml` now has an optional `workflow_dispatch` input, `candidate_sha`. Use a full
lowercase hexadecimal 40-character commit SHA other than 40 zeroes. An empty value keeps the
existing full validation. A nonempty value runs only the existing 10-minute `Windows native target
smoke` job against that revision; changed-path classification, the Go matrix, the Windows parser
groups and the C differential job do not run. This is not a replacement for full CI or evidence
that all Go and Linux checks passed.

The workflow change and the candidate commit are still local and must both be pushed with separate
approval before dispatch. After that approval, the current candidate command is:

```text
gh workflow run frontend.yml --ref main -f candidate_sha=315ef56991d20fa0716927e4fcecb464231db6a8
```

The exact-candidate CI result remains unconfirmed until that run completes.

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
- Public access gate: the repository is currently public. Before announcing
  a download, choose and verify a publicly accessible release location.
- Security gate: enable GitHub private vulnerability reporting on a public
  repository and verify that an unauthenticated visitor can find the private
  report route. Public issues are for nonsensitive bugs only.
- Release blocker status: any failed user-path step, relevant exact-commit CI
  failure, artifact hash mismatch, or unavailable private security-report route.
- Remaining operational risk: separate-host execution and fresh native compilation
  on GRAM13 are user-reported above; execution without development tools remains
  unverified. The first preview requires the user's separate MSVC installation and
  offers no compatibility or response-time promise.

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
