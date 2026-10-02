# Zed preview adapter

The manifest references the grammar subdirectory of the Fairy C repository.
Until the editor commits are pushed, change the grammar repository to the
`file://` URL of your local Fairy C checkout and set `rev` to the local grammar
commit. Install this directory with Zed's **Install Dev Extension** command.
Building the grammar for Zed requires the editor's WASI toolchain.

Language metadata and four query sets are supplied. The queries are copied from
the Tree-sitter grammar; edit that single owner and regenerate adapters.
The adapter is not registered in the Zed extension registry. Native grammar and
query validation is distinct from a Zed/WASM installation smoke test.

## Local syntax preview

From the compiler repository root, run `node editors/zed/prepare-dev.mjs` to
prepare `.cache/zed-syntax-dev`. Under the workspace command contract this helper
is called by `fairyc_editors_generate`. It uses a local `file://` grammar URL and
the current commit, with no edits to the tracked extension manifest. Commit
grammar/parser edits first: Zed clones the pinned grammar, not uncommitted files.

Select the generated directory with **Install Dev Extension** in Zed. The
preview deliberately contains only language configuration and queries: no Rust
adapter or language server is loaded, so it needs no Rust WASM target or LSP
configuration. Zed may still download its WASI SDK to build the grammar. The
helper itself does not download tools, install an extension, or change settings.

Open `editors/zed/examples/highlighting.fy` and check that Zed selects **Fairy C**,
then check `@effects`, qualified types such as `bytes::Buffer`, nested generic
types, builtins including `i64`, function declarations/calls, constants, enum
variants, strings/escapes, nested comments, `///` documentation and `..` views.
Check matching/autoclosing braces, brackets, parentheses and quotes, comment
toggling, indentation and the outline. Exact colors come from the active theme.
The file is a parser/highlighting showcase, not a runnable SDK program.

Version `0.1.2` captures each type identifier and the attribute name/`@` token
directly. Capturing only their enclosing nodes lets the generic identifier
capture win on the inner text. Expression paths and attribute arguments retain
their own captures; a call such as `bytes::empty()` is not a type.

The regular `editors/zed` adapter retains the LSP integration below. Both packages
use extension ID `fairyc`; installing one replaces the other rather than providing
two independent extensions. Native query/capture and Rust checks passed locally;
fresh installation, WASM compilation and LSP interaction are separate checks.
For an existing syntax-only dev extension using the same grammar revision,
refresh the language queries and use Zed's **Reload Extensions** command; a
query-only correction does not require a new grammar or Rust adapter build.

See [Zed's extension development guide](https://zed.dev/docs/extensions/developing-extensions)
for toolchain and local installation requirements, and
[language extensions](https://zed.dev/docs/extensions/languages) for token captures.

The Rust adapter uses the released Zed extension API 0.7 and launches an existing
`fairyc-lsp` executable. Set `lsp.fairyc-lsp.binary.path` in Zed settings to its
absolute path, or make it available on the editor's PATH. No language-server
download or global installation occurs. The adapter has been checked with Cargo
on the Windows host; WASM compilation and in-editor installation remain separate.
