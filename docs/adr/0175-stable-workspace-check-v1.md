# ADR 0175: Stabilize the local-workspace check command

- Status: Accepted
- Date: 2026-07-22
- Product source: `docs/product/02-spec.md`
- CLI sources: `docs/cli/command-contract.md`, `docs/cli/output-and-exit-codes.md`
- Implementation: `internal/cli/workspace.go`, `internal/workspaceloader/loader.go`

## Context

The compiler can already load a strict local workspace, select the bundled SDK when imported,
maintain `fairy.lock`, run package-aware semantics, lower MIR, and verify it. Keeping all of that
behind `experimental check-workspace` prevents the first ordinary Fairy C development loop.
Aliasing stable `check` to the explicit-file experimental command would bypass package discovery,
the SDK, dependency identity, and lock consistency.

## Decision

Implement `fairyc check [--locked]` as the stable v1 current-workspace command.

- The process source root is the workspace root and must contain `fairy.toml`.
- No positional source, target, backend, toolchain, cache, network, or stdin input is accepted.
- Ordinary mode writes canonical `fairy.lock` only when missing or different. `--locked` requires an
  exact existing lock and performs no write.
- The command checks the complete manifest-owned package closure through verified MIR without
  requiring `main`.
- Success emits one deterministic ANSI-free count line. Invocation, source diagnostic, internal,
  and package/lock failures use codes 2, 3, 4, and 7 respectively.
- Syntax diagnostics discovered during workspace loading remain source diagnostics rather than
  being collapsed into a package failure.

Stable JSON, target-specific validation, feature selection, persistent-cache defaults, and
configuration-file precedence remain future versioned additions. The experimental workspace command
remains available for compatibility but does not own the stable spelling.

## Evidence

CLI tests cover lock creation and locked verification, rejected positional input, missing manifests,
syntax diagnostics with code 3, exact success output, help discovery, and existing experimental
behavior. The full Go suite, build, vet, and Windows host smoke are the release checks for this slice.
