Fairy C - LOCAL VERIFICATION PACKAGE ONLY

NOT AN APPROVED PUBLIC RELEASE. This is not an installer or supported release.
Project-owned components use Apache-2.0; third-party components retain their
licenses. LOCAL-ONLY describes release status, not additional license restrictions.
Release ownership and complete release validation remain outstanding.

Contents: fairyc.exe, this README, SHA256SUMS.txt, LICENSE, LICENSING.md,
THIRD_PARTY_NOTICES.md and GO-LICENSE.txt. Keep applicable licenses/notices
with redistributions. See LICENSING.md for runtime/SDK and generated-code scope.
The adjacent .zip.sha256 file checks the archive. SHA-256 detects changes; it
does not authenticate a publisher. This build is unsigned and experimental.

Requirements: Windows AMD64. Native testing and running require the locally
installed x64 MSVC toolchain. Go/Node/Bun are build tools, not included here.

In PowerShell, retain the absolute executable path, then use an empty scratch
parent folder (do not create hello-fairy yourself):

$fairyc = (Resolve-Path .\fairyc.exe).Path
# Change to your chosen empty scratch parent folder before the next command.
& $fairyc new hello-fairy --uid example.hello-fairy
Set-Location hello-fairy
& $fairyc check
& $fairyc check --locked
& $fairyc test --locked
& $fairyc test --native --locked
& $fairyc test --native --release --locked
& $fairyc run --locked
& $fairyc run --release --locked

The generated starter has one passing test and returns zero without output.
Do not assume syntax recognition means all backend/target capabilities exist.
The Fairy-written native parser remains an internal experiment, not this compiler.
No compatibility, clean-PC installation or production-readiness claim is made.
Public issue/security-reporting routes must be confirmed before distribution.
