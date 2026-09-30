# Tree-sitter Fairy C

This preview grammar is an editor parser, not a type checker. The compiler EBNF
owns syntax. Generated `src/parser.c` and `src/node-types.json` are committed so
consumers do not need a Node generator at runtime. `src/scanner.c` handles nested
block comments and unfinished comments during editing.

Generate with Tree-sitter CLI 0.27 (`tree-sitter generate --abi 14`), then run
`tree-sitter test`. The shared queries also feed the Zed adapter.
The grammar is shipped in this repository under `editors/tree-sitter-fairyc`;
no new repository or registry package has been published.
