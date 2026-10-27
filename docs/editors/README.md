# Fairy C editor support

Editor adapters are presentation clients. The compiler lexer and
[Stage 0 grammar](../../spec/grammar/stage0.ebnf) own language syntax; highlighting
does not imply semantic admission. The shared language ID is `fairyc`, extension
is `.fy`, and TextMate scope is `source.fairyc`. Use standard token scopes so
the user's theme controls colors.

## VS Code

Version 0.1.2 is packaged as a local VSIX from `editors/vscode`; see the
[VS Code README](../../editors/vscode/README.md) for packaging, installation and
verification commands. The VSIX contains the syntax assets and JavaScript
language client, but not the compiler or language server executable. Syntax and
editor configuration work without a server. Set `fairyc.server.path` to the
absolute `fairyc-lsp` executable path to enable diagnostics, definition
navigation and completion in a trusted workspace. No Marketplace publication is
claimed.

The shared `fairyc-lsp` server adds diagnostics, definition navigation and
completion. See [the LSP preview contract](lsp-preview.md) for build/setup,
supported capabilities and explicit limitations.

## Tree-sitter and Zed

`editors/tree-sitter-fairyc` supplies the grammar, nested-comment scanner,
generated ABI-14 parser, node types and shared queries. `editors/zed` supplies
the language adapter. These preview packages remain at version 0.1.1; see their
READMEs for local build and installation steps. The syntax preview includes
`i64`, repeated arrays, shared and mutable partial views, function calls,
constants and documentation comments. It uses theme token names instead of
hardcoded colors. Native parser/query tests do not establish a successful
Zed/WASM installation.

## Maintaining syntax assets

Run `go run ./tools/dev/fairyc-editor-assets` from the compiler repository to regenerate
`editors/language.json` and the TextMate grammar. Reserved words are obtained
directly from the compiler lexer. Contextual words remain distinct; the EBNF hash
forces review whenever the grammar changes. Review Tree-sitter syntax separately
rather than treating a regex highlighter as the compiler parser.

The VS Code package is version 0.1.2; the Tree-sitter and Zed preview packages
are version 0.1.1. These preview versions do not change the Fairy C language,
SDK, runtime ABI or compiler release version. Atom is not a supported target.
Extension publishing and automatic tool downloads are separate release work.
