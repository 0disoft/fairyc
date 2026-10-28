Fairy C Windows editor verification bundle

This self-contained bundle checks the retained Fairy C compiler preview and the
original Fairy C VSIX against a locally installed VS Code. It needs no repository
checkout, Go, Node.js, Bun, or MSVC installation for the editor test. The VS Code
extension host runs the included JavaScript harness with VS Code's bundled runtime.

Requirements

- Windows AMD64.
- Windows PowerShell 5.1 or newer, started as powershell.exe.
- A local VS Code 1.90 or newer installation containing Code.exe and bin\code.cmd.

The bundle contains manifest.json, verify.ps1, the install harness, extension.json,
this guide, license and notice files, the retained Windows preview ZIP and its
SHA-256 sidecar, and the original VSIX and its SHA-256 sidecar. The manifest records
the source revision and the artifact and language-server hashes.

Receiver steps

1. Copy the delivered ZIP and its .sha256 sidecar to the Windows host. Keep both
   files together and retain their delivered names.
2. In Windows PowerShell, compare the ZIP digest with the sidecar:

   $zip = 'C:\path\to\fairyc-windows-editor-verify.zip'
   $expected = ((Get-Content -LiteralPath "$zip.sha256" -Raw).Trim() -split '\s+')[0]
   if ((Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash -ine $expected) {
       throw 'SHA-256 mismatch'
   }

3. Extract the ZIP into a new folder, then run the verifier from that folder. Set
   the CodeRoot value to the installed VS Code directory for this PC:

   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\verify.ps1 `
     -CodeRoot 'C:\Users\<user>\AppData\Local\Programs\Microsoft VS Code'

   The default results directory is a new, uniquely named sibling of the extracted
   bundle folder. To choose another location, pass an absolute path to a new folder:

   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\verify.ps1 `
     -CodeRoot 'C:\Users\<user>\AppData\Local\Programs\Microsoft VS Code' `
     -ResultsDirectory 'C:\Users\<user>\Documents\fairyc-editor-results-1'

   ResultsDirectory must be absolute and must not already exist. The verifier prints
   the receipt path. Exit code 0 with passed=true indicates success; on failure the
   receipt is retained with passed=false and an error message.

What the verifier checks

- The bundle manifest schema, every listed file size and SHA-256, the absence of
  unlisted bundle files, and both artifact checksum sidecars.
- The retained preview ZIP extraction and the extracted fairyc-lsp.exe hash.
- Installation of the original VSIX into a new isolated VS Code profile, including
  the expected 0disoft.fairyc@0.1.2 extension ID and version.
- A sample .fy file's association, activation, definition, completion, unsaved
  error diagnostics, and diagnostic recovery.

The results folder retains receipt.json, the extracted artifacts, and the isolated
installation with its VS Code logs and host payload. receipt.json records the PC
name, OS, PowerShell and VS Code versions, source revision, and the preview ZIP,
VSIX, and language-server hashes. It also records cleanMachineEvidence=false and
publicationApproved=false.

Scope and limits

- The harness uses an isolated user-data and extensions directory and does not
  change the user's normal VS Code profile.
- The disposable test workspace is trusted and the workspace-trust prompt is
  disabled. The run does not verify behavior in an untrusted workspace.
- A pass describes only the host that ran it. It is not clean-PC evidence.
- The preview compiler ZIP remains experimental and unsigned. SHA-256 detects
  modification; it does not establish publisher identity.
- The bundle is prepared for transfer. It does not claim that GRAM13 has passed,
  and preparation does not upload or publish anything. Keep a receiving-host
  receipt with the transferred artifacts.

The bundle retains LICENSE, LICENSING.md, THIRD_PARTY_NOTICES.md, and
GO-LICENSE.txt. The preview ZIP retains its packaged license and notice files.
