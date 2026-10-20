# Developer Tools

- Status: Draft

Maintainer commands that are not part of the product live here. Product executables stay in
`cmd/`.

## Layout

```text
cmd/fairyc/          product compiler executable
cmd/fairyc-lsp/      product language-server executable
tools/candidate/     private candidate bootstrap and Go reference commands
tools/dev/           fixture, native-proof, migration, and evidence commands
tools/               maintainer scripts, probes, and verification drivers
```

Each command keeps its existing binary name and moves under `tools/candidate/` or `tools/dev/`;
the directory name is the command name. Maintainer scripts that are not Go commands stay directly
under `tools/`.

## Packages, module, and builds

The repository is a single Go module rooted at the repository root (`github.com/0disoft/fairyc`).
Tool packages are therefore addressed by their full path, for example
`github.com/0disoft/fairyc/tools/dev/fairyc-windows-host-smoke` and
`github.com/0disoft/fairyc/tools/candidate/fairyc-candidate-build`. Keeping one module avoids
additional module requirements, local replacements, and version coordination for shared compiler
packages.

Because the module root is unchanged, moving a command changes its package path and nothing else:

- `go build -mod=readonly ./cmd/...` builds the product commands only.
- `go vet ./...` and `go test ./...` still cover every package, including the tool commands.
- Relocation does not reduce the cost of a full `./...` vet or test run, and no such saving is
  claimed.
- A focused build of one tool names its new path, for example
  `go build -mod=readonly ./tools/dev/fairyc-windows-host-smoke`.

Typical invocations:

```text
go build -mod=readonly ./cmd/fairyc
go run -mod=readonly ./cmd/fairyc check
go run -mod=readonly ./tools/dev/fairyc-windows-host-smoke
go run -mod=readonly ./tools/dev/fairyc-native-fixtures spec/fixtures/c-emission/manifest-v0.json
go test -mod=readonly ./internal/cli
```

## Native and candidate commands

Run tools from the repository root. Native commands require their supported host C toolchain;
the native fixture runner requires Linux/amd64. Candidate generators support read-only checks:

```text
go run -mod=readonly ./tools/candidate/fairyc-candidate-sdk-embed --check examples/lexer_candidate/src
go run -mod=readonly ./tools/candidate/fairyc-candidate-cli-embed --check examples/lexer_candidate/src
```

Candidate bootstrap and Go reference commands live in `tools/candidate/`. CI keeps its ordinary
`go vet ./...` and `go test ./...` validation, and the Windows candidate test shards target
`./internal/cli`, which is the only package containing the `TestWindowsFairyLexer` and
`TestWindowsFairyParser` cases. Historical ADRs, receipts, and generated producer labels retain
their original paths; replace `cmd/<candidate-name>` with `tools/candidate/<candidate-name>` and
other developer command paths with `tools/dev/<name>` when following old instructions.

## Related documents

- [Build footprint benchmark](../docs/benchmarks/README.md) for the manual retained-output and
  cache-growth comparison tool.
- [Development guide](../DEVELOPMENT.md) for building the Windows compiler binary.
- [Contributing](../CONTRIBUTING.md) for building, focused tests, native backend selection, and
  historical tools and artifacts.
- [Workspace boundaries](../docs/monorepo/workspace-boundaries.md) and
  [package ownership](../docs/monorepo/package-ownership.md) for the boundary rationale.
- [Source of truth](../docs/development/source-of-truth.md) for the files that own each contract.
