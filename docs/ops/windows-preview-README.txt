Fairy C - EXPERIMENTAL WINDOWS/AMD64 PREVIEW CANDIDATE

This is an unsigned, experimental compiler package, not an installer or a
production-supported release. Obtain a published package only from the
project's official GitHub Releases page and verify its published SHA-256.
Project-owned components use Apache-2.0; third-party components retain their
licenses. See LICENSING.md for redistribution terms.

Contents: fairyc.exe, this README, SHA256SUMS.txt, LICENSE, LICENSING.md,
THIRD_PARTY_NOTICES.md and GO-LICENSE.txt. Keep applicable licenses/notices
with redistributions. See LICENSING.md for runtime/SDK and generated-code scope.
The adjacent .zip.sha256 file checks the archive. SHA-256 detects changes; it
does not authenticate a publisher.

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
Report nonsensitive bugs through the project's GitHub Issues page. Do not put
credentials, private code or vulnerability details in a public issue. Before
public distribution, the maintainer must enable GitHub private vulnerability
reporting and confirm the reporting link in the release announcement.
