Fairy C - EXPERIMENTAL WINDOWS/AMD64 PREVIEW CANDIDATE

This is an unsigned, experimental compiler and language-server package, not an
installer or a production-supported release. Obtain a published package only
from the project's official GitHub Releases page and verify its published
SHA-256.
Project-owned components use Apache-2.0; third-party components retain their
licenses. See LICENSING.md for redistribution terms.

Contents: fairyc.exe, fairyc-lsp.exe, this README, SHA256SUMS.txt, LICENSE,
LICENSING.md, THIRD_PARTY_NOTICES.md and GO-LICENSE.txt. Keep applicable
licenses/notices with redistributions. See LICENSING.md for runtime/SDK and
generated-code scope.
The adjacent .zip.sha256 file checks the archive. SHA-256 detects changes; it
does not authenticate a publisher.

Requirements: Windows AMD64. Using the prebuilt compiler and language server
requires no Go, Node.js or Bun. Native builds, tests and runs of Fairy C
programs require the locally installed x64 MSVC toolchain and Windows SDK.
Go/Node.js/Bun are needed only when building compiler or editor assets from
source.

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

VS Code editor support is a separate local VSIX and is not included in this
ZIP. After installing that extension, set fairyc.server.path to the absolute
path of the extracted fairyc-lsp.exe, for example:

{
  "fairyc.server.path": "C:\\Users\\you\\fairyc-preview\\fairyc-lsp.exe"
}

The language server starts only in a trusted workspace. Syntax support remains
available without workspace trust.

No compatibility, clean-PC installation or production-readiness claim is made.
Report nonsensitive bugs through the project's GitHub Issues page. Do not put
credentials, private code or vulnerability details in a public issue. Before
public distribution, the maintainer must enable GitHub private vulnerability
reporting and confirm the reporting link in the release announcement.
