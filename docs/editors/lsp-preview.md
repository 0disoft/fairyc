# Language server preview 0.1.0

Build `go build -o .cache/fairyc-lsp.exe ./cmd/fairyc-lsp` on Windows (omit
`.exe` on other hosts). The separate `fairyc-lsp` executable speaks LSP 3.17
JSON-RPC over stdio. Protocol output is isolated from stderr.
No project builds, execution, downloads, lockfile writes or source writes occur.

Clean EOF is accepted only between frames. EOF after any header byte or before
the declared body is complete is `io.ErrUnexpectedEOF` and terminates the session
with exit code 1. Headers are bounded to 8 KiB, bodies to 9 MiB; duplicate
`Content-Length` fields reject even when the first value is negative. Frame
reads share the session context and check cancellation around body reads of at
most 32 KiB. A blocked reader still needs to be closed by its owner to unblock.
After consuming a complete frame, malformed UTF-8 or JSON receives JSON-RPC
`-32700` and an invalid request envelope receives `-32600`. Both use a null
response ID and leave the session ready for the next frame. Invalid framing
terminates the session.

## Capabilities

- Incremental document synchronization with monotonically increasing versions.
  Ranged UTF-16 edits in a notification use the text produced by earlier edits
  in that notification. A full-text replacement also works. The server rejects
  a whole batch if any range is invalid, any intermediate text exceeds the
  document budget or the sum of input document and replacement bytes across
  ranged edits exceeds 128 MiB. After a rejected batch, send full text or
  close and reopen the document before sending more ranges.
- Compiler syntax, type, ownership and unsupported-feature diagnostics.
- Definition locations from current compiler resolution facts.
- Read-only embedded SDK navigation for clients that set
  `initializationOptions.fairycSdkSourceNavigation`. The server returns
  `fairyc-sdk://stdlib/<sha256>/std/<source>.fy` locations and serves their
  exact text through `fairyc/sdkSource` with a `uri` parameter. The VS Code
  extension registers the content provider. The source hash ties coordinates
  to the matching embedded text across workspace reloads.
- Hover for resolved names and function declarations in a clean current analysis.
  It shows a plain-text function signature or symbol kind and name, with the
  checked value type when available. Unavailable or ambiguous facts yield no hover.
- Find references to a resolved name across the current project analysis, including
  unopened local files and unsaved source overlays. Clients can include the local
  declaration, including unused nominal types, variants, fields, import aliases
  and parameters. Results are deduplicated, ordered by URI and position, and limited
  to 4,096 locations and the protocol response budget. Oversized results fail
  explicitly. The search covers recorded path uses and returns no references for
  unavailable or ambiguous analysis.
- Prefix completion for keywords, built-in types, module declarations,
  parameters, locals and exact imported-module public declarations. After
  `.` the checked receiver type selects visible nominal fields and methods.
  For an incomplete member expression the server can check a temporary
  in-memory overlay. Further syntax errors can prevent suggestions.
- Full-document semantic tokens for authored declarations and compiler-resolved
  names in the current clean analysis. Tokens use UTF-16 coordinates and
  distinguish declarations and read-only constants. An unavailable analysis
  returns an empty token stream. Results are limited to 32,768 tokens.
- Quick fixes for isolated missing punctuation reported by the parser. The
  server checks the resulting syntax before offering an insertion at the
  current document version. Clients need code action literals and versioned
  workspace edits.
- Whole-document and range formatting for the current open text. The compiler's
  whitespace formatter normalizes CRLF, trailing horizontal whitespace and the
  final newline. It verifies the parser shape and significant tokens before
  returning an edit in the original document's UTF-16 coordinates. Unchanged
  or malformed text produces no edits. Range formatting limits whitespace
  changes to a nonempty selection and adds a missing final newline only when
  the selection reaches the end of the file.
- Unsaved new .fy sources under explicit project source roots.
- UTF-16 positions, including non-ASCII source and supplementary-plane characters.

Initialize with local workspace folders and add or remove them during the
session. The most specific containing folder bounds each document's project
lookup. Removing a folder keeps its open text and rechecks it as a single file
when no other folder contains it. The nearest `fairy.toml` inside the selected
folder identifies its package. If that exact package
directory is a declared member of a valid ancestor `[workspace]`, the nearest
such ancestor selects the project, so sibling members and the root's reachable
dependency closure remain available. A nested package that its ancestors do not
declare stays in its own project scope. Selection never goes above the
selected folder or through a symlinked package directory. Declared members,
local dependencies and the bundled SDK are loaded using the compiler's ordinary
workspace rules. Source contents are overlaid in memory and receive fresh graph,
digest and compilation identities. A file without a manifest receives
single-file compiler analysis.

When a workspace cannot load, publish the precise syntax diagnostics that are
available and an explicit unavailable diagnostic for affected open files; never
reuse stale definition results. Completion can use the last clean declaration
snapshot while the user is typing invalid syntax. That fallback is a suggestion,
not a current compiler fact.

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
These TextMate rules use syntax positions. The language server provides
semantic tokens when the current compiler analysis is available.
Theme settings choose the actual colors. Reload the editor after updating the
development extension files to load the regenerated grammar.

For Zed use `editors/zed`, with the local grammar-repository instructions in its
README. Set `lsp.fairyc-lsp.binary.path` to the executable path. Its small Rust
adapter launches the same server; it does not download tools.

## Explicit preview limits

Up to 64 workspace folders per session, 32 open documents, 8 MiB per source document,
32 MiB total open text, 9 MiB per framed message, and the compiler's existing
workspace/source guards. A session retains at most 128 MiB of analyzed source in
unique `View` and last-clean analyses. The charge covers each loaded workspace
catalog, including declared members outside the root-reachable closure, and
closed source files retained by diagnostic positions after failed loads, and
counts shared analysis pointers once. This is a source-byte retention proxy,
not an AST/HIR heap measurement. When it is exceeded, the least recently used
analysis is evicted; affected open files receive an unavailable diagnostic and
reload their project on the next definition or completion request. One incoming
workspace is still loaded and parsed under the existing 64 MiB workspace guard
before retention admission; the 128 MiB figure limits retained source, not
transient project-load working memory. Work is synchronous with cooperative
cancellation of superseded requests and edits; incremental reparsing is
deferred. Saved external changes are refreshed on watched-file notifications;
no watcher runs inside the server.

Rename is not advertised.
The grammar is for editor recovery, not a substitute for semantic checking.

This is a local preview, not a Marketplace/Zed-registry release. Windows native
server and Tree-sitter tests do not imply a Zed/WASM or macOS/Linux GUI smoke test.
