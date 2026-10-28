# Fairy C for VS Code

Fairy C editor support for `.fy` files. The extension
registers the `fairyc` language ID and `source.fairyc` TextMate scope, with
syntax highlighting, snippets, comment toggling, bracket pairs and indentation.

The VSIX contains the syntax assets and JavaScript language client. The Windows
preview ZIP includes the prebuilt `fairyc-lsp.exe` beside `fairyc.exe` and the
existing notices. The VSIX is distributed separately and is not in that ZIP.

## Install a local VSIX

From the repository root, build the extension package:

```powershell
cd editors/vscode
npm ci
npm run package:vsix
```

Use Node.js 22 or newer. `npm ci` is required; the packaging script uses the
repository-local `@vscode/vsce` 4.x dependency.

The script writes the artifact and checksum under the repository cache:

- `.cache/editor-release/fairyc-0.1.2.vsix`
- `.cache/editor-release/fairyc-0.1.2.vsix.sha256`

The filename version comes from `editors/vscode/package.json`. Install the
absolute artifact path and verify the installed version:

```powershell
code --install-extension <absolute path to .cache/editor-release/fairyc-0.1.2.vsix>
code --list-extensions --show-versions
```

The installed identifier and current version are `0disoft.fairyc@0.1.2`.
Reload VS Code after installation.

## Language server

Syntax features work without a compiler or language server. Diagnostics,
definition navigation and completion use `fairyc-lsp`. After installing the
local VSIX, extract the Windows preview ZIP and set `fairyc.server.path` to the
absolute path of the extracted executable:

```json
{
  "fairyc.server.path": "C:\\path\\to\\fairyc-preview\\fairyc-lsp.exe"
}
```

Using the prebuilt compiler and language server requires no Go, Node.js or Bun.
Native Fairy C builds still require the local x64 MSVC toolchain and Windows SDK.
The language client starts in trusted workspaces. Grant workspace trust to
enable the server; syntax support continues to work without it.

Building the language server from source remains a developer alternative. From
the repository root:

```powershell
go build -o .cache/fairyc-lsp.exe ./cmd/fairyc-lsp
```

Set `fairyc.server.path` to the absolute path of that source-built executable.

## Maintainer install validation

`tools/test-vscode-install.ps1` validates the existing local VSIX and can reuse a
retained Windows preview candidate without building or repackaging either artifact.
Run it from the repository root with the absolute path to a VS Code installation:
Use the receipt and full source revision of your retained candidate; the values
below identify the locally tested pair.

```powershell
./tools/test-vscode-install.ps1 `
  -CodeRoot 'C:/Users/cherr/AppData/Local/Programs/Microsoft VS Code' `
  -PreviewReceipt '.cache/starter-smoke-hcotc1/receipt.json' `
  -PreviewRevision '315ef56991d20fa0716927e4fcecb464231db6a8'
```

`-CodeRoot` is mandatory. `-PreviewReceipt` and `-PreviewRevision` are optional
and must be supplied together. The Node.js verifier rechecks the v2 starter-smoke
receipt, full source revision, candidate archive, and recorded compiler and LSP
hashes. The harness then extracts the retained ZIP into a fresh isolated test
directory, sets the absolute server path, and installs
`.cache/editor-release/fairyc-0.1.2.vsix` into an isolated VS Code profile. It
checks `.fy` association, activation, definition, completion, unsaved
diagnostics, and recovery, then writes schema `fairyc.vscode-install/v2` to
`.cache/editor-release/install-*/receipt.json` with the VSIX, ZIP, and LSP hashes.

Without the paired preview arguments, the harness uses
`.cache/editor-release/fairyc-lsp.exe`. It does not modify the user's VS Code
profile. The disposable test workspace is trusted (the workspace-trust prompt is
disabled), so this check does not prove untrusted-workspace behavior.

The maintainer harness needs Windows PowerShell and VS Code; its preview verifier
also needs Node.js 22 or newer. End users need no standalone Go, Node.js, or Bun
installation for the prebuilt VSIX and ZIP. Record separate-host verification
independently; this harness establishes local integration only.

## Transfer bundle for another Windows host

To prepare a self-contained receiver bundle, run this from the repository root on
Windows AMD64 after packaging the local VSIX:

```powershell
node tools/windows-editor/prepare.mjs `
  --receipt .cache/starter-smoke-hcotc1/receipt.json `
  --revision 315ef56991d20fa0716927e4fcecb464231db6a8 `
  --code-root 'C:/Users/cherr/AppData/Local/Programs/Microsoft VS Code'
```

The maintainer step needs Node.js 22 or newer. It uses the installed VS Code for
the local isolation check.

The command rechecks the retained v2 starter-smoke receipt, full source revision,
preview ZIP, compiler and LSP hashes, and the checksum of the existing
`.cache/editor-release/fairyc-0.1.2.vsix`. It does not rebuild the VSIX or compiler.
It writes the new bundle and checksum as:

```text
.cache/windows-editor-<random>/fairyc-windows-editor-verify.zip
.cache/windows-editor-<random>/fairyc-windows-editor-verify.zip.sha256
```

The exact repository-relative archive path is printed in the `archive` field of
`preparation.json`, which is written beside the ZIP. That file also records the
archive size and digest, source revision, manifest digest, and local test receipt
digest. Preparation extracts the finished ZIP, runs its verifier against the
requested VS Code installation with an external-tool-free child PATH, and requires
a passing isolated installation receipt before writing the final checksum.

Transfer only the ZIP and its `.sha256` sidecar. `README.txt` inside the bundle is
the receiver guide; it covers checksum verification, extraction, VS Code discovery,
results locations, retained evidence, and scope limits. The receiver needs Windows
AMD64, Windows PowerShell 5.1 or newer, and locally installed VS Code, with no
checkout, Go, standalone Node.js, Bun, or MSVC for the editor test.

The harness disables the workspace-trust prompt for a trusted disposable workspace
and does not verify untrusted-workspace behavior. A preparation receipt is local
evidence only: it is not clean-PC evidence, does not alter the user's normal VS Code
profile, does not publish remotely, and does not claim that GRAM13 has passed. The
preview compiler ZIP remains experimental and unsigned; SHA-256 detects modification
but does not establish publisher identity. The bundle retains the repository license
and third-party notices.

## License

Project-owned extension files are Apache-2.0; the package includes `LICENSE`.
Third-party dependencies retain their own license files. This README documents
a local VSIX package and does not claim Marketplace publication.
