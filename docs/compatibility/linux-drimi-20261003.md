# External Linux native evidence, 2026-10-03

The Drimi result archive is an external execution receipt, not a current-HEAD
full-suite result. Its baseline is `d866070a3001e63e6ff4c48e9b6c9a220e4bc3c4`,
SDK stage0.79, language/20 and runtime/0. Host: Debian 13, Linux amd64, glibc 2.41,
Go 1.26.7, GCC 14.2 and Clang 19.1.7. The source archive had no Git metadata;
the first build failed VCS stamping and the recorded successful builds used
`-buildvcs=false`. No source/dependency changes were made for that workaround.

The returned `fairyc-drimi-linux-d866070a-result-20261003.zip` SHA-256 is
`90fe77ee16e93f022f4d2bf02249c70cbe94d6edd1c5999e0aa99d7e84fe3dab`.
All 1,553 listed evidence files were independently hash-verified on receipt.
Drimi separately reported verifying 2,740 original source files.

| Test scope | Observed result | Revision boundary |
| --- | --- | --- |
| Original nine selected top-level tests | 7 pass / 2 fail / 0 skip | Original baseline |
| Native test execution, trap continuation, selection, streams, stdin/argv and timeout | Pass | Original baseline |
| Normal/trap status distinction and UTF-8 I/O, task-manager CRUD/restart | Pass | Original baseline, ordinary CLI selects GCC |
| Prepared publication | GCC/Clang, low/optimized pass | Original baseline |
| Prepared save fault injection and bounded child kills | 40 pass, 4 images, failed=false | Baseline plus reviewed two-file i64 helper patch |
| Independent 32-source i64 native/oracle matrix | 128 pass / 0 skip | Same patched snapshot |
| Remaining `TestStableLinuxBuildRun` | Fail: expected 3, actual stale-lock 7 | Both baseline and patched rerun |

The i64 compiler fix is adopted in
[ADR 0429](../adr/0429-demand-driven-i64-c-helpers.md). The remaining test
expectation is corrected by checking two separate failures: a changed source
with `--locked` must return package failure 7 and preserve both lock and image;
the same invalid source without `--locked` must return E0101/source failure 3
and preserve the image. Compiler lock policy and diagnostics are unchanged.
A shared Windows/Linux native regression exercises both states locally on the
available host; the repaired Linux-specific test still needs a Linux rerun.

The external result predates [ADR 0428](../adr/0428-native-runtime-object-reuse.md).
Its executions cannot prove that adapter's GCC/Clang cache-hit, invalidation or
recovery behavior. A follow-up source snapshot must exercise those cases and
the repaired Linux test; the already successful broad suite and 40/128 matrices
need not be repeated solely to restate the older evidence. New compiler defects
must retain their minimal reproduction and exact snapshot/tool identity.

The returned compiler images are SHA-256
`b7f58aa329d61ba92cb75690ba4b00249356fdefdd7248a22e8200f3b1e18559`
(original) and
`9b88a160309f3d9e3432b29613273f6286c1c22399f96a9352d8b6bde1d699b9`
(patched). Full raw argv/cwd/stdout/stderr/exit, save receipts, native images and
digests remain in the user-supplied return archive; build artifacts are not
committed into the repository. POSIX failures were test injection, not physical
disk/OS failure or power-loss durability. Ordinary Clang fallback, current-HEAD
Linux execution and build-time performance remain unverified.
