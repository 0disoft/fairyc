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

## License

Project-owned extension files are Apache-2.0; the package includes `LICENSE`.
Third-party dependencies retain their own license files. This README documents
a local VSIX package and does not claim Marketplace publication.
