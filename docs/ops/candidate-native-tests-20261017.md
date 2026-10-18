# Candidate native test CLI

The Fairy candidate dispatcher implements `test --native` for Windows AMD64
MSVC. Each selected private `fn void` test gets its own C entry wrapper,
executable, process, trap channel and deadline. A valid trap records a failed
test and execution continues with the next selected test. A driver failure,
invalid trap receipt, unclassified nonzero exit or timeout stops the command
without publishing a partial test report.

Selection supports `--exact`, `--file`, `--dir`, `--package` and `--workspace`.
Text and JSON reports retain canonical identity order, source locations,
lock state and the Go report schemas. `--release` selects the child program's
optimized build. `--json --capture-output` retains binary stdout and stderr
as base64; output without that flag is rejected. Each child retains the
existing shared 64 KiB output ceiling.

Input defaults to EOF. `--stdin` forwards caller input to exactly one selected
test. `--stdin-file` reads at most 64 KiB once from a regular workspace file,
then gives every child a fresh independent replay. The candidate's existing
source-file reader rejects reparse/symlink components and checks file identity;
this is stricter than Go's root-confined file reader. `--` forwards program
arguments including empty and UTF-8 arguments. `--timeout-seconds` remains
canonical 1 through 60, with a 60-second default for each child.

Workspace preparation and lock resolution finish before the test loop.
`module_project_complete_c::prepare_tests` lowers once into a `TestSession`
owning `owner::Lowered`. Within one invocation, `emit_prepared_test` calls
`c::emit_lowered_test` to reuse the same immutable IR, proven CFGs and SDK plan;
names are rebuilt for each selected test. There is no cross-invocation cache.
The existing source, C partition, stack and deadline limits are unchanged.

The compiler-host build helper inherits the seed PE's stack reservation when
`--stack-reserve-bytes` is omitted. This preserves the existing 16 MiB host
reservation; generated test programs keep their existing reservation. The first
rebuild accidentally used the linker's 1 MiB default and overflowed on the
development SDK test. Restoring only the host PE reservation fixed the same
case. The receipt proves all executable code and data bytes stayed identical;
only stack-reservation and checksum metadata could change.

Native execution is not compiler promotion. The candidate also has an
[initial scalar semantic test interpreter](candidate-semantic-tests-20261017.md).
Full interpreter coverage and Linux native command transport remain separate work.

Reproduce a fresh candidate build with `tools/build-candidate-native-probe.py`,
then run `tools/verify-candidate-native-tests-cli.py` with its `--build-receipt`,
a Go `--reference` image and a new `--output` directory under `.cache`.
The verifier checks source/image hashes, compares actual native outcomes and
rejects changes to locked fixture files. Every selected comparison, including
deadline diagnostics, requires exact exit status, stdout and stderr equality.
Existing test-list regression coverage uses
`tools/verify-candidate-test-list-cli.py` with the same build receipt.

## 2026-10-17 checkpoint

Final evidence is in
[`candidate-native-tests-cli-20261017.json`](evidence/candidate-native-tests-cli-20261017.json).
The release suite passes 29 comparisons (28 exact output/exit matches and one
deadline infrastructure comparison). Development SDK execution passes its
targeted EOF case. Both compiler profiles pass all 89 test-list comparisons,
178 exact matches in total. Go observations are reused only when the reference
image, arguments, input bytes and fixture contents match. At this checkpoint the
full development native suite, Linux execution and stage1/stage2 bootstrap had not
been rerun.

## Current verification (2026-10-18)

The reviewed Windows run in
[`candidate-cli-local-validation-20261018.json`](evidence/candidate-cli-local-validation-20261018.json)
compares `test --native` against the Go image rebuilt after the enum payload fix
(`4b2b1693`) on both candidate profiles: 30 of 30 comparisons exact in development and
30 of 30 in release, where the development suite ran in three isolated partitions after
reusing 19 completed observations. The same record passes all 89 `test --list`
comparisons in each profile, 178 exact matches, and 447 of 447 exact per profile across
the metadata, cli, semantic, native, example and list gates. This supersedes the
2026-10-17 checkpoint's open development-suite note for the native and list gates.
Linux execution and the stage1/stage2 bootstrap fixed point remain outside this record,
and the compiler self-source `lexer_candidate` example is still excluded.

## Full CLI C-part fixed point (2026-10-19)

The reviewed [fixed-point record](evidence/candidate-native-fixed-point-20261019.json)
passes with all 512 source hashes unchanged: both 41,798,801-byte C packets and
both 5,913,600-byte release PE images match exactly, with zero excluded bytes,
and the frozen `--version` output matches exactly. The Go-built C-packet bridge
and MSVC remain part of this bootstrap; full Go replacement remains false.
The 2026-10-18 behavioral record above is unchanged and reused, without rerunning
its comparison suites. Linux verification remains deferred.
