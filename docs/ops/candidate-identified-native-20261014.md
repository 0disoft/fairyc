# Complete source-identified native build/run

`project_emit` collects the loaded package coordinates and calls the same
complete reporting compiler as `check`. Entry selection reads its verified IR;
the C bundle owns that IR together with every detached package interface.
Unsupported-signature reports survive build/run failures through the existing
diagnostic owner. No independent interface-only result admits an executable.

The authored `file_cli::run` dispatcher closure contains 439 source inputs. A
current field-aware Fairy emitter produces its 35,248,757-byte C packet without
a Go frontend fallback. Actual MSVC development and release images build and
pass 51 process invocations: 48 candidate/program executions and three explicit
Go oracle comparisons. The companion Go binary supplies native C transport;
it does not parse Fairy source or replace the Fairy command implementation.

Coverage includes actual build/run, aliased imported scalar generics, SDK Vec
storage and cleanup, unused u16/u64 specializations, identical unsupported
signature diagnostics across check/build/run and Go, unchanged prior images on
failure, missing/stale locks, invalid options/entries, missing companions,
publication-directory escapes, Korean paths, empty/Unicode/quote/backslash/8KiB
arguments, binary stdin and exact output, ordinary full-width Windows statuses,
recognized traps and bounded combined output. Scratch directories do not leak.
Root and locked source checks pass 450 files / 5822 MIR functions.

The private compiler images reserve 16MiB stacks. Product stack/deadline defaults
are unchanged. Vec run costs 74.95s in the development compiler and 7.36s in the
release compiler; the I/O case costs 69.52s and 7.06s. These are scoped process
measurements, not a general performance benchmark. Receipt:
[evidence](evidence/candidate-identified-native-20261014.json).

This private dispatcher entry forwards real OS argv and does not include the
original main's no-argument lexer framing route. Current-source complete
Stage1/Stage2 reproducibility, broader/multiple diagnostics, cache consumers,
product stack/deadline/performance integration, Linux/full suite and same-head
hosted CI remain open. Go remains the admitted bootstrap and reference.
Public CLI/SDK ABI, Go source, DB, CI and runner settings are unchanged; no push.
