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

## License

Project-owned extension files are Apache-2.0; the package includes `LICENSE`.
Third-party dependencies retain their own license files. This README documents
a local VSIX package and does not claim Marketplace publication.
