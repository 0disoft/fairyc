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

The Rust adapter uses the released Zed extension API 0.7 and launches an existing
`fairyc-lsp` executable. Set `lsp.fairyc-lsp.binary.path` in Zed settings to its
absolute path, or make it available on the editor's PATH. No language-server
download or global installation occurs. The adapter has been checked with Cargo
on the Windows host; WASM compilation and in-editor installation remain separate.
