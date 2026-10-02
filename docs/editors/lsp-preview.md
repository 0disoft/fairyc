# Language server preview 0.1.0

Build `go build -o .cache/fairyc-lsp.exe ./cmd/fairyc-lsp` on Windows (omit
`.exe` on other hosts). The separate `fairyc-lsp` executable speaks LSP 3.17
JSON-RPC over stdio. Protocol output is isolated from stderr.
No project builds, execution, downloads, lockfile writes or source writes occur.

## Capabilities

- Full document synchronization with monotonically increasing versions.
- Compiler syntax, type, ownership and unsupported-feature diagnostics.
- Definition locations from current compiler resolution facts.
- Prefix completion for keywords, built-in types, module declarations,
  parameters, locals and exact imported-module public declarations.
- Unsaved new .fy sources under explicit project source roots.
- UTF-16 positions, including non-ASCII source and supplementary-plane characters.

Initialize with one local workspace folder. For each document the nearest
`fairy.toml` inside that folder selects a project. Declared members, local
dependencies and the bundled SDK are loaded using the compiler's ordinary
workspace rules. Source contents are overlaid in memory and receive fresh graph,
digest and compilation identities. A file without a manifest receives
single-file compiler analysis.

When a workspace cannot load, publish precise available syntax diagnostics and
an explicit unavailable diagnostic for affected open files; never reuse stale
definition results. Completion can use the last clean declaration snapshot while
the user is typing invalid syntax. That fallback is a suggestion, not a current
compiler fact.

## Editor setup

For VS Code install the development extension from `editors/vscode`, with its
production npm dependencies, and set `fairyc.server.path` to the absolute
language-server executable path. Language-server activation requires workspace
trust. Syntax support works without the server.

The VS Code development extension 0.1.2 assigns attribute scopes to
`@effects`, `@must_use` and other attribute names. Nominal types such as
`bytes::Buffer` are scoped as namespace/type in declarations, return types,
parameters, fields and single-line generic/array types. Ordinary calls such as
`bytes::empty()` remain functions. Comments and strings retain their own scopes.
These TextMate rules use syntax positions, not capitalization or resolved SDK
symbols; they are not semantic tokens or a complete multiline type parser.
Theme settings choose the actual colors. Reload the editor after updating the
development extension files to load the regenerated grammar.

For Zed use `editors/zed`, with the local grammar-repository instructions in its
README. Set `lsp.fairyc-lsp.binary.path` to the executable path. Its small Rust
adapter launches the same server; it does not download tools.

## Explicit preview limits

One workspace folder per session, 32 open documents, 8 MiB per source document,
32 MiB total open text, 9 MiB per framed message, and the compiler's existing
workspace/source guards. Work is synchronous; cancellation and incremental
analysis are deferred. Saved external changes are refreshed on watched-file
notifications; no watcher runs inside the server.

Rename, references, semantic tokens, hover, member completion, embedded-SDK
virtual source navigation, formatting and code actions are not advertised.
The grammar is for editor recovery, not a substitute for semantic checking.
Multi-root support and complete malformed-input JSON-RPC recovery are deferred.

This is a local preview, not a Marketplace/Zed-registry release. Windows native
server and Tree-sitter tests do not imply a Zed/WASM or macOS/Linux GUI smoke test.
