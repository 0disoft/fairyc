# Build Footprint Benchmark

- Status: Draft

`fairyc-bench` measures the files a native build keeps after each command and the disk growth
caused by a sequence of source edits. It runs the same small workload in Fairy C, Zig, Rust, and C
inside separate experiment directories. It supports local runs and manually dispatched CI runs for
inspecting build workflow cost; these synthetic workloads do not establish a universal language ranking.

## Latest snapshot

[English](latest/report.html) · [中文](latest/report.zh.html) · [Español](latest/report.es.html) · [Français](latest/report.fr.html) · [हिन्दी](latest/report.hi.html)

- [Development summary SVG](latest/summary.svg) and [PNG](latest/summary.png): `dev` and Rust
  `small-dev` for `many-modules`
- [Full footprint SVG](latest/footprint.svg) and [PNG](latest/footprint.png)
- [Growth SVG](latest/growth.svg) and [PNG](latest/growth.png)
- [Raw receipt](latest/receipt.json)
- [Download HTML report](latest/report.html)

Download the HTML report and open it locally; GitHub's file view does not run it. The report is
self-contained and needs no hosting. The receipt records the snapshot's exact commit, dirty flag,
host, toolchain versions, settings, case statuses, and commands.

## Quick start

Run from the repository root. The full run uses the default matrix and an explicit 10-minute limit
for each child command:

```text
go run ./tools/dev/fairyc-bench run --output .cache/bench-YYYY-MM-DD --zig /path/to/zig --timeout 10m
```

For a short smoke run before a full snapshot:

```text
go run ./tools/dev/fairyc-bench run --output .cache/bench-smoke --workloads tokenizer --edits 1 --profiles dev --languages fairy,rust,c
```

A run does not render the report. It writes `receipt.json` and the case workspaces, plus the built
checkout compiler under `tooling` when `--fairyc` is omitted. Rendering is a separate command:

```text
go run ./tools/dev/fairyc-bench render --input .cache/bench-YYYY-MM-DD/receipt.json --output docs/benchmarks/latest
```

Rendering produces all five languages by default. Use `--locale en`, `zh` (Simplified Chinese),
`es`, `fr`, or `hi` for one language. Localized files use a suffix, such as `summary.hi.png`
or `report.fr.html`; English keeps the original filenames. Keep the HTML files together to use
the language selector. Each HTML file embeds its own charts and works offline.

The `run` output directory must not already exist. The parent directory must exist, and a failed or
unavailable case makes the command exit non-zero after the receipt has been updated.

## Manual CI

The [Build footprint workflow](../../.github/workflows/build-footprint.yml) runs only through
`workflow_dispatch`, using GitHub Actions' **Run workflow** control. Pushes, pull requests, and
schedules do not trigger it. Its inputs default to `languages=fairy,zig,rust,c`, `profiles=dev`,
`workloads=tokenizer`, `edits=5`, `modules=16`, and `repetitions=1`.

This small default matrix keeps manual runner use down. To request the full 18-case matrix, set
`profiles=dev,release,small-dev`, `workloads=tokenizer,many-modules`, and `edits=20`, leaving the other
inputs at their defaults. These match the CLI defaults used for a full local snapshot.

The job uses a Windows hosted runner with a 90-minute timeout, Go from `go.mod`, and preinstalled
Rust, MSVC, and Windows SDK tools. It downloads the latest stable Zig from the official index and
verifies the archive's SHA-256. Each cold Zig build can take several minutes; charged Windows runner
usage depends on the account's plan.

Run outputs stay under `runner.temp`. Artifacts contain only `receipt.json` and HTML, SVG, and PNG
reports, with seven-day retention; caches and binaries are not uploaded. The workflow uses
`contents: read`, publishes no site, and makes no README commits. Its syntax has been checked
locally, but it has not been dispatched or verified on a GitHub runner.

## Defaults

- Languages: `fairy,zig,rust,c`
- Profiles: `dev,release,small-dev`; `small-dev` applies only to Rust
- Workloads: `tokenizer,many-modules`
- Edit count: `20`; every fifth edit changes a public function signature and the others change a
  private helper
- Modules: `16` for `many-modules`; accepted range `2..64`
- Repetitions: `1` independent cold-to-clean sequence
- Timeout: `10m` per command; override it with `--timeout`
- Zig executable: `zig` on `PATH`; override it with `--zig`
- C compiler: MSVC on Windows and `cc` elsewhere; override it with `--cc`

Unless `--fairyc` names a prebuilt compiler, the runner builds the current checkout's Fairy
compiler with `go build -mod=readonly` before measurement. The compiler build and its output are
outside the measured project footprint.

## Profiles

| Language | `dev` | `release` | `small-dev` |
| --- | --- | --- | --- |
| Fairy C | `fairyc build` | `fairyc build --release` | Not applicable |
| Zig | `-Doptimize=Debug` | `-Doptimize=ReleaseFast` | Not applicable |
| Rust | Cargo development defaults | `cargo build --release` | Cargo development defaults with `debug = "line-tables-only"` and `incremental = false` |
| C | Unoptimized with debug information (`-O0 -g`, or MSVC `/Od /Zi`) | `-O2`, or MSVC `/O2` | Not applicable |

Rust uses its standard library, Zig links libc for `getchar` and `printf`, C uses libc directly,
and Fairy C uses its SDK. The C adapter invokes the compiler directly and recompiles every source
file on every build, including the no-op build. Its timings must not be read as incremental compiler
timings. Development debug information and incremental behavior also differ between toolchains.

## Workloads

`tokenizer` reads a fixed input from stdin, counts token transitions, and prints a token count and a
rolling checksum. `many-modules` does the same work and calls the configured number of helper
modules. The fixtures are deliberately small and use no third-party packages.

After every build, the runner executes the new binary and compares its normalized stdout with a Go
reference implementation of the same token and checksum rules. A case stops at the first build or
verification failure.

## What is measured

Each case starts with an empty project output tree and isolated experiment state. The case contains
the generated source, `input.txt`, project outputs, a Cargo home, shared user and Zig cache roots,
and a temporary directory. A cold sample means those experiment-owned caches are empty; the
operating system, installed toolchains, and their global files remain in place.

The footprint is the sum of regular file lengths found after a command in the known project output
roots and case state:

- binary
- separate PDB, DWARF companion, or similar debug files
- incremental files
- other intermediate outputs
- local build caches, including Zig's project `.zig-cache`
- shared experiment caches, including the isolated Cargo home and Zig global cache
- temporary files that are still retained after the command

Zig 0.17's `--global-cache-dir` argument is no longer used. The adapter sets
`ZIG_LOCAL_CACHE_DIR` and `ZIG_GLOBAL_CACHE_DIR`, which also isolates the maker and configurer
caches.

These are logical byte counts, not allocated disk usage. The per-case footprint excludes deleted
transient files, peak disk use, peak memory, operating system caches, downloaded dependencies,
runtime speed, and toolchain installation files, which are inventoried separately below. Sparse
files, filesystem compression, hard links, allocation units, and files outside the selected roots
can make physical use differ. Symlinks inside a measured root cause the sample to fail.

The clean sample records what remains after removing project build outputs. Rust uses
`cargo clean --offline`; Fairy C and C remove their project `build` directory; Zig removes
`zig-out` and its project `.zig-cache`. Each case keeps its isolated shared caches and can still
retain experiment temporary files. The tool does not remove a user-supplied output directory or a
toolchain's shared installation.

## Installed toolchains

The receipt and HTML report include a separate inventory of installed toolchain files. These logical
bytes stay outside all per-case footprint charts. The inventory covers only these explicit roots:

- Fairy C's standalone compiler file, including its embedded SDK; the Go build environment is excluded
- Zig's distribution directory, including bundled libraries and documentation
- Rust's active sysroot, including installed libraries, documentation, and tools
- On Windows, the active MSVC version tree and the active SDK version's `Include`, `Lib`, and `bin`
  subtrees

Shared native resources are counted once, with `usedBy` listing the languages that use them. Files
outside these roots are not counted. Symlinks are skipped and mark the inventory `incomplete`, as do
read errors. Linux system C compiler and libc installations are `unmeasured` because they have no
bounded installation root. These scopes describe the installed files found, rather than a minimal
installation requirement for each language.

## Reports and receipt data

`render` validates the receipt and writes a self-contained, static `report.html` with its charts,
styles, and table filters embedded. The top menu switches between summary, footprint, growth,
measurements, toolchains, and notes. The selected menu stays active when changing language.
It also writes `footprint.svg`, `footprint.png`, `growth.svg`,
`growth.png`, `summary.svg`, `summary.png`, and a copy of `receipt.json`. The compact summary selects
`dev` and Rust `small-dev` for `many-modules` when that workload is present. SVG and PNG use bundled
font subsets and Go text shaping, including Hindi conjuncts. Rendering needs no browser, installed
fonts, or network connection. See [font sources and licenses](../../tools/dev/fairyc-bench/fonts/README.md).

The footprint bars average retained bytes across repetitions, so their stacked components add up to
the mean total. The growth chart uses the median at each edit. The HTML table reports median
private-edit and public-edit build times for each case. The raw receipt keeps every sample from every
repetition. Failed or unavailable cases remain in the table and raw data and are excluded from the
charts. The HTML table separates **Project outputs/cache** from **Shared cache**. For Rust, the
project column measures `target/` directly. **End total** includes both columns plus retained
temporary files, so it should not be read as the size of `target/` alone.

The receipt records the UTC start and finish, Git commit and dirty flag, OS, architecture, CPU, CPU
count, runner Go version, selected settings, toolchain versions and backends, and every case sample.
Samples include source hashes and lengths, expected and actual output, footprint buckets, and each
command with its environment overrides, exit code, wall time, log, and timeout flag. Run-local paths
and the user's home directory are shortened before commands are stored. The toolchain executable
size and SHA-256 are identity metadata; installation sizes are recorded in the separate inventory.
See [the receipt schema](receipt-v1.schema.json) for the JSON structure.

Snapshots live under `docs/benchmarks/latest`; the receipt is the source of exact commands and raw
observations. Reproduce the selected matrix with `run`, then regenerate the reports with `render`.

## Limits

The runner accepts Windows and Linux. The Windows path has been exercised; the Linux adapter has
not yet been tested. A cold Zig 0.17 maker build can take several minutes, which is why the default
command timeout is 10 minutes. Use the receipt's case statuses and verification records to assess a
snapshot's coverage.

Use these results to compare repeated configurations under the same host, fixture, and profile.
Differences in compiler behavior, runtime libraries, debug information, and cleanup policy make
cross-language rankings unreliable. The benchmark does not establish equivalent debugging features.
Local and CI runs are both started manually.
