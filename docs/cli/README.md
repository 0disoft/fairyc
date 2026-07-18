# `fairyc` CLI

- Status: Draft
- Repository type: cli-tool
- Command contract: `docs/cli/command-contract.md`

`fairyc` is the canonical compiler and toolchain executable. The CLI must expose fast development checks separately from expensive production optimization and must make emitted intermediate forms explicit.

## Planned command families

- project: `new`, `build`, `run`, `test`, `check`;
- source: `fmt`, `lint`, `doc`;
- compiler inspection: `emit`, `explain`, `targets`;
- packages and reproducibility: `add`, `vendor`, `cache`;
- interoperability: `bindgen`, `abi-check`;
- semantic navigation: `atlas`.

Only `check`, a minimal `build`, compiler inspection, and semantic test execution are Stage 0 candidates. The remaining commands are product commitments only after their contracts and implementations exist.

## Current prototype

The Go frontend currently exposes `fairyc experimental parse <file.fy>`,
`fairyc experimental check-syntax <file.fy>`, and
`fairyc experimental check-semantics <file.fy>`. These are developer inspection commands whose
spelling and numeric exit codes may change. `check-semantics` resolves names and creates typed HIR
for the implemented single-file scalar `bool`/`i32` subset. It does not analyze Flow, resources,
borrows, regions, or effects, generate code, or execute programs. In particular, it does not
satisfy the planned `fairyc check` contract.

Human developers can run the examples shown in the repository `README.md`. Agent verification uses
the configured `fairyc_go_build`, `fairyc_go_vet`, and `fairyc_go_test` intents; the manual `go run`
examples do not grant agent command authority.

`targets` must distinguish `emittable`, `conformant`, and `supported`. The planned `x86_64-linux-gnu` profile cannot be reported as supported before its runtime, standard-library, ABI, toolchain, and clean-CI fixtures pass without skips.

## Profile principle

- Stage 0 development and release both use the C reference backend with different explicit optimization profiles; `check` uses no backend.
- A later fast or production backend changes this profile only after a measured adoption decision.
- ThinLTO, PGO, full debug information, sanitizers, and cross-target builds are explicit requests.

## Naming

The executable is `fairyc`. `fc` is not used. A `fyc` alias may be considered only after packaging and shell compatibility are specified.
