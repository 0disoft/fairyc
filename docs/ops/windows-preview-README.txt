Fairy C - LOCAL VERIFICATION PACKAGE ONLY

NOT FOR PUBLIC DISTRIBUTION. This is not an approved release or installer.
Distribution terms, third-party notices and release ownership are not finalized.
Do not upload this archive or describe it as a supported public release.

Contents: fairyc.exe, this README, SHA256SUMS.txt.
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
