# Fairy C editor support

Editor adapters are presentation clients. The compiler lexer and
[Stage 0 grammar](../../spec/grammar/stage0.ebnf) own language syntax; highlighting
does not imply semantic admission. The shared language ID is `fairyc`, extension
is `.fy`, and TextMate scope is `source.fairyc`. Use standard token scopes so
the user's theme controls colors.

## VS Code

The extension source is in `editors/vscode`. Copy it into a VS Code development
extension directory or launch VS Code with `--extensionDevelopmentPath` pointing
to that absolute directory. Syntax highlighting, nested block comments, comment
toggling, bracket pairs, indentation and snippets work without a compiler.
The extension is not yet published to the Marketplace.

The shared `fairyc-lsp` server adds diagnostics, definition navigation and
completion. See [the LSP preview contract](lsp-preview.md) for build/setup,
supported capabilities and explicit limitations.

## Tree-sitter and Zed

`editors/tree-sitter-fairyc` supplies the grammar, nested-comment scanner,
generated ABI-14 parser, node types and shared queries. `editors/zed` supplies
the language adapter. See its README for local development installation before
these commits are published. The syntax preview includes `i64`, repeated arrays,
shared and mutable partial views, function calls, constants and documentation
comments. It uses theme token names instead of hardcoded colors. Native
parser/query tests do not establish a successful Zed/WASM installation.

## Maintaining syntax assets

Run `go run ./cmd/fairyc-editor-assets` from the compiler repository to regenerate
`editors/language.json` and the TextMate grammar. Reserved words are obtained
directly from the compiler lexer. Contextual words remain distinct; the EBNF hash
forces review whenever the grammar changes. Review Tree-sitter syntax separately
rather than treating a regex highlighter as the compiler parser.

The editor packages have their own preview version `0.1.1`. They do not
change the Fairy C language, SDK, runtime ABI, or compiler release version.
Atom is not a supported target. Extension publishing and automatic tool downloads
are separate release work.
