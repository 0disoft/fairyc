# Token index struct boundary experiment

Date: 2026-09-17 (KST). Status: measurement only; no language or production scanner change.

Existing structs reject a byte offset passed as a token index, but this workload does
not meet the proposed requirement of unchanged generated-artifact size. This is a
reason to investigate an erased nominal scalar design, not evidence that such a
design would be free. ADR 0254 remains a candidate decision, not an accepted syntax.

## Workload and semantic result

The test copies the real `examples/lexer_candidate/src` scanner and its five imports
into temporary workspaces. The baseline keeps `token_info(ref Stream, usize)`.
The variant introduces two ordinary structs:

```text
pub struct TokenIndex { usize raw; }
pub struct ByteOffset { usize raw; }
pub fn TokenInfo token_info(ref Stream stream, TokenIndex index) {
    let ref Token token = v::at<Token>(ref stream.tokens, index.raw);
    // Existing body follows unchanged.
}
```

The client scans `module alpha;`, requests token 2, and asserts kind 3, byte range
7..12 and total token count 5. The variant constructs both domain wrappers explicitly.
Replacing the call argument with `byte_offset` or the raw literal `2` succeeds at
the baseline type boundary and fails with E1006 at the wrapped boundary. Valid calls
pass in both variants. The diagnostic provides incompatible/expected-type spans;
it does not spell out the two domain names in its message.

This is a comparison of the pair of wrappers at one real call boundary, not a
measurement of a single wrapper's marginal cost. Parser callers are excluded.
`TokenInfo.start/end` remain `usize`; other untyped boundaries remain. Public `.raw`
allows explicit bypass, and neither wrapper enforces ranges or stream provenance.

## Size observations

Windows AMD64, Go 1.26.4, MSVC 14.51.36231; two repetitions per variant/profile.
All eight native executions returned zero with empty stdout/stderr.
Sizes below are bytes; section sizes are PE VirtualSize, not file allocation sizes.

| Artifact | Baseline | Structs | Difference |
| --- | ---: | ---: | ---: |
| Generated C, either profile | 344257 | 345271 | +1014 |
| Low EXE | 203776 | 204288 | +512 |
| Low `.text` | 150528 | 150624 | +96 |
| Low `unit.obj` | 108671 | 108767 | +96 |
| Optimized EXE | 138240 | 138240 | 0 |
| Optimized `.text` | 85296 | 85328 | +32 |
| Optimized `.data` | 5552 | 5520 | -32 |
| Optimized `unit.obj` | 69017–69021 | 69200 | +179–183 |

Generated C hashes were stable within each variant. Some object sizes varied by
four bytes even for the common runtime object; total object size is not an isolated
instruction count. No ABI equivalence or runtime-performance equivalence is claimed.
No PDB was requested: the recorded zero PDB bytes mean absence, not free debug data.

## Timing observations and limitations

Ranges across only two samples, milliseconds:

| Profile / phase | Baseline | Structs |
| --- | ---: | ---: |
| Low check | 287.8–306.7 | 320.8–328.8 |
| Low compile | 855.3–1288.7 | 851.8–865.0 |
| Optimized check | 339.6–353.9 | 309.7–312.5 |
| Optimized compile | 1758.9–1790.3 | 1707.1–1769.6 |

Each sample uses a fresh workspace. Variant order reverses in repetition two.
Check runs in process and includes lock creation. Storage-aware C emission occurs
before the measured shared stable build compiler, which excludes writing the final
EXE. Ambient filesystem/toolchain caches are uncontrolled. These are bounded
observations, not cold CLI benchmarks or evidence of compilation speed equivalence
or improvement. Runtime duration, Linux, CI, cache footprint and debug output were
not measured. The full compiler suite was not rerun for this isolated test addition.

## Evidence and reproduction

- Test: `internal/cli/token_index_experiment_test.go`.
- [Raw samples and source hashes](evidence/token-index-struct-20260917.json).
- Compiler source baseline: `5e171171a5ab0319e4376eff2973596be0f72929`.
- Evidence includes test-source hash and the test executable's compiler identity.
- Workspace command: `mf run fairyc_token_index_experiment --repo projects/hobby/opensource/fairyc`.
- Passing receipt: `.mustflow/state/runs/run-2026-09-16T16-45-51-576Z-14896-3062e5ac4b21/receipt.json`
  in the parent workspace (local, not shipped with this repository).
- `TestTokenIndexStructBoundary` and `TestTokenIndexStructCostMSVC` passed;
  package duration was 19.173 seconds. Scoped Mustflow strict checks also passed.

The command exports evidence only after both tests pass. Recorded compiler baseline
is the source revision used for this experiment; future measurements must review
that provenance and the guarded scanner replacement before replacing the snapshot.
The parent root command file exceeds Mustflow's parser size limit, so root manifest
synchronization used the configured script argv directly; scoped project runs used
`mf run`. No root-run receipt is claimed for that synchronization.

## Decision

Do not migrate the production parser to wrappers under a strict no-growth promise.
The experiment establishes a concrete type-confusion boundary and a useful existing
language alternative, while documenting its cost. The next bounded design exercise
is a nominal scalar checked before lowering and erased to its underlying scalar,
with explicit conversions and no new generic specialization policy. Compare that
prototype against this same workload before deciding adoption. Type identity,
interface/cache metadata, debug size and compilation cost still require separate
checks; erasing the runtime representation alone does not settle them.

No grammar, language revision, SDK version, public API or production code changes
are part of this experiment.
