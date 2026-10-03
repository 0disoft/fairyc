# Stable CLI contract v0

## Status

Accepted pre-1 command contract for the implemented stable `fairyc` surface. Experimental commands remain unstable but are owned and classified here so they cannot silently acquire stable semantics.

## Ownership and inventory

`internal/cli` owns command parsing, flags, stdin ownership, stdout/stderr separation, process status classification, workspace configuration loading, lock behavior, and stable JSON selection for the `fairyc` executable.

The exact command and flag inventory is the compiler-owned `usage` value in `internal/cli/run.go`. A command or flag not present there is not admitted. `docs/cli/command-contract.md` owns human semantics and `docs/cli/output-and-exit-codes.md` owns byte/status semantics. Experimental subcommands are inventory-visible but do not gain compatibility merely by appearing in help.

The stable top-level commands are `new`, `check`, `build`, `run`, `test`, `explain`, and `targets`, with source-independent `--help` and `--version` inspection. The removed `emit` spelling is a stable rejection/migration surface. Every command below `experimental` is owned by `internal/cli` but remains explicitly unstable.

## Exit taxonomy

The compiler failure namespace is fixed to these non-overlapping values:

| Code | Compiler meaning |
| ---: | --- |
| 0 | success |
| 2 | invalid invocation, rejected selector, or explicitly unsupported command capability |
| 3 | source diagnostics |
| 4 | compiler invariant, internal processing, or output transport failure |
| 5 | recognized Fairy language trap during compiler-owned execution |
| 6 | native toolchain, target build/link, artifact inspection, or execution infrastructure failure |
| 7 | workspace, manifest, package graph, source-set, or lock failure |

`run` is the deliberate boundary: before program launch the values above are compiler statuses; after successful ordinary launch, the program's complete 32-bit status is forwarded unchanged. A recognized Fairy trap is intercepted and mapped to compiler status 5, so scripts must use the command contract rather than assume every post-launch small integer is a compiler failure.

No future compiler failure class may reuse an ordinary executed-program status by post-processing an already completed ordinary exit. New compiler classes require an explicit contract revision.

## stdout and stderr

Stable success payloads go to stdout. Invocation, source, package, toolchain, target, and internal diagnostics go to stderr unless a command's versioned machine report explicitly represents a completed language-test outcome. Compilation or admission failures must not emit a partial success report. External writer failures can retain a prefix already accepted by the operating system and return code 4; the CLI does not claim stream rollback.

Help is deterministic, ANSI-free compiler-owned text. Stable machine reports contain no absolute host paths or source/file contents unless an explicitly versioned schema and opt-in field contract says otherwise.

For ordinary direct function call type mismatches, E1006 retains the expected and actual canonical types internally. stderr shows both names quoted and module-qualified, preserving build-local package qualifiers when the names differ only by package instance. Exit code 3, the primary argument span, and the expected-type related span are unchanged. No semantic-diagnostic JSON report is added: stable --json test modes report compilation failure on stderr without a partial stdout report, and success schemas are unchanged. Other mismatch sites, method and foreign calls, grammar, and SDK version are unchanged.

For ordinary direct calls, E1006 adds expected-type-declaration and actual-type-declaration related spans only when the canonical type is itself a named struct, with no fabricated declaration span for scalar, ref, or container types. Related spans retain the existing role-based ordering and expected-type parameter span; code 3, the primary span, and the JSON contract are unchanged.

## Machine-readable envelopes

Only commands that explicitly advertise `--json` or an experimental versioned JSON command may emit machine-readable success documents. Stable test reports use the existing `fairyc.test-report/v1`, `/v2`, `/v3`, and `/v4` families and stable test listing uses `fairyc.test-list/v1`, `/v2`, and `/v3`. Schema selection is driven only by the documented command mode, workspace/package scope, locations flag, and native capture mode.

Experimental JSON identities such as `fairyc.experimental.parse/v0`, target-spec, Atlas, cache, and session documents remain experimental even though their bytes are deterministic within their current schema generation. They may not be treated as a stable top-level CLI envelope.

A stable JSON generation may add no hidden source text, arbitrary file content, credential material, environment snapshot, or absolute path. Any future field that can expose user-authored bytes requires a new schema generation and explicit documentation.

## Configuration precedence

Stable workspace commands use exactly the `fairy.toml` located at the supplied process source root/current workspace root. There is no parent-directory search, environment-variable override, user config, system config, implicit profile file, network config, or prompt.

Manifest values are the configuration authority. Command-line flags only select behavior explicitly owned by that command and do not mutate manifest defaults. Ordinary workspace operation may refresh canonical `fairy.lock`; `--locked` verifies without writing. Test source overlays are not production lock inputs.

This precedence is fixed for the stable commands. Experimental commands may require extra explicit flags but cannot alter stable configuration discovery.

## Host, runtime, and toolchain floor

`new`, `check`, semantic `test`, `explain`, and `targets` require no native compiler or linker. Their supported hosted correctness lanes are Linux amd64 and Windows amd64 under the repository's Go frontend CI. The distributed `fairyc` executable itself embeds its Go runtime; users do not need a separate Go installation to run it.

Stable `build`, `run` and `test --native` select `x86_64-windows-msvc` on Windows amd64 and `x86_64-linux-gnu` on Linux GNU amd64. Windows requires x64 MSVC and its SDK; Linux requires glibc and GCC or Clang on PATH (GCC is preferred). Unsupported hosts fail closed. Linux CLI integration and save-failure tests are in the existing Ubuntu test job; their native results have not yet been observed for this change.

Stable Linux command admission does not promote the target's independent conformance or distribution status. See [ADR 0425](../adr/0425-stable-linux-build-and-run.md) and [ADR 0426](../adr/0426-linux-native-workspace-tests.md).

The experimental `build-workspace --target x86_64-linux-gnu` path retains its explicit `--cc gcc` or `--cc clang`, Linux amd64 glibc host, and output-under-`build/` contract. It shares the private Linux storage transport with stable build/run.

Native workspace compilation can reuse only the runtime/support object in the
bounded OS user cache (`fairyc/native-runtime-v1`): 64 generations and 64 MiB
encoded bytes. Source checks, generated-unit compilation, linking and image
inspection stay fresh. Header preprocessing and compiler/profile/input identity
gate reuse; cache failures fall back to compilation. `--locked` verifies the lock
and does not disable this private cache. CLI arguments, stdout and exit codes
are unchanged. See [ADR 0428](../adr/0428-native-runtime-object-reuse.md) for
invalidation, recovery, exact bounds and the unverified Linux execution boundary.

The experimental Linux `run-workspace` path uses the same explicit toolchain and private vector linkage, executes a temporary ELF, and reports the eight-bit process status. A dedicated one-byte trap channel distinguishes a trap from an ordinary return with the same status. It retains its output rejection; stable `run` forwards program output, stdin and arguments.

## Compatibility rule

Language /14 admits explicit scalar generic functions in stable workspace check/build/run/test
through one coordinate-bound compiler entry point. ADR 0385 owns the accepted types and remaining
generic restrictions. This changes source admission, not flags, output schemas, or exit taxonomy.

Changing a stable command's accepted flags, configuration discovery, success bytes, JSON schema selection, compiler exit classification, stdin ownership, or pre-launch/post-launch status boundary requires a deliberate contract revision with golden tests.

Adding or removing experimental commands does not by itself revise this stable contract, but help text remains the exact inventory and must change atomically with parser behavior.

## Validation ownership

`internal/cli` tests own invocation, help, stdout/stderr separation, JSON schemas, workspace failures, locks, and stable native behavior. `cmd/fairyc-windows-host-smoke` owns the hosted Windows native path. Repository CI runs Go vet/tests on Linux and Windows and the Windows native smoke before native CLI claims are accepted.
