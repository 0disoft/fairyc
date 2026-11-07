# Candidate source artifact bounds, 2026-11-07

Base and observed verification HEAD: `0f3f1f2c5c9ce62f1ac18f533744f706e180008f`.
This is a dirty-worktree, same-host Windows verification, including the concurrent
parent/Mencius source-cardinality changes. It is not verification of that commit
alone or a clean-machine claim. No commit, lock update, bootstrap, or full source
check was performed by this worker.

## Policy and implementation

- Source count in provenance metadata admission and artifact model/extract/decode
  now follows `candidate_limits::source_files()` (currently 1024).
- Rebind source-index `mapped` and `seen` arrays become a bounded `Vec<usize>` and
  byte `Buffer`. Allocation follows the already checked source counts; allocation
  errors retain code 11. Shape/current checks still precede these allocations.
- Current-artifact module mapping owns a source-count-sized `Vec<usize>`. Every
  module-index consumer reads that vector. Its separate owner mapping still has
  exactly 512 slots. A small `Owners` value record wraps that array because the
  current compiler subset rejects a direct fixed-array resource field. Owner IDs
  remain below 512; neither this wrapper nor source storage expands owner policy.
- Detached/artifact import model/extract/decode retain **32768**, independently of
  active `bindings::build` admission at shared `project_imports()` (8192, owned by
  Mencius). No serialized import policy was tightened. The 1 MiB names limit and
  existing work budgets can refuse inputs before the detached import count cap.
- `FYPROV00` marker, field widths, ordering, payload and error codes are unchanged.
  Existing <=512-source bytes keep the same encoding. No public-descriptor,
  package/table, joined declaration, or cleanup policy was expanded.

## Narrow Windows evidence

Cache root: `.cache/source-artifact-bounds-20261107/`.

```powershell
python .cache/source-artifact-bounds-20261107/prepare.py
bun tools/run-go.ts test -mod=readonly '-overlay=.cache/source-artifact-bounds-20261107/overlay.json' ./internal/cli -run '^TestWorkerSourceArtifactBounds$' -count=1 -timeout 6m -v
git diff --check
```

The cache-only Go overlay copies the real authored import closure, reuses the five
existing `tests/module_source_artifact.fy` tests with only module/export/test-entry
adaptation, and adds a private-map wrapper only in the temporary fixture copy.
The permanent production map remains private. `probe.fy`, `worker_test.go`, and
`prepare.py` retain the reproducer; no permanent test/helper file was added outside
the assigned paths. Native child execution has a test-local 180-second context;
no product/global timeout or process-wide cleanup was changed.

`windows-low.log`: **PASS**, Go package 25.333s, native regression 13.3444354s,
21 C partitions, Windows AMD64 MSVC low profile, 16 MiB stack reserve. Host load
included other workers; these durations are observations, not performance claims.

- Existing independent 998-byte golden, encode/decode detachment, exact work and
  exhaustion, source/owner reorder, stale source/provider/coordinate refusal,
  every truncated prefix, forged version/plumbing/spans and trailing-byte refusal:
  PASS.
- Real parsed source sets at 512, 513 and 1024: bindings/extract/encode/decode PASS.
  At 1024, real source metadata/provenance authentication, reverse-order rebind and
  the actual private current-artifact mapping all PASS, including source slot 1023.
- 1025: active bindings and artifact model refuse; independent wire fixture
  decode returns 1002. Existing owner 512 wire refusal returns 1001.
- Independent valid detached fixtures with 8192 **and 8193** imports both decode
  successfully (`64/8192`, `64/8193`). This confirms that active 8192 admission
  was not accidentally imposed on serialized artifacts. Count 32769 returns 1002.
- `git diff --check`: PASS. No lock or source-snapshot diff was introduced.

The initial unquoted PowerShell overlay invocation did not run the probe; it is
not a pass. Two subsequent compile attempts exposed mutable-resource and direct
fixed-array-resource limitations; their failure logs are retained as
`attempt1-resource-mutation.log` and `attempt2-resource-fixed-array.log`. The final
implementation builds owner/vector locals and transfers them into the resource.

Optimized/Linux execution and the full current-artifact planner/selection runtime
fixture were skipped: this bounded probe executes the actual mapping function,
not an end-to-end 1024-source artifact planner. The copied import closure was
checked/lowered by the Go frontend. Full isolated source check, locked acceptance,
fresh optimized seed and integration belong to the parent. No allocation-failure
injection was performed. Public API, DB, runner and dependency files are unchanged.

## Frozen hashes

SHA-256; source paths are relative to `examples/lexer_candidate/src/`.

| File | SHA-256 |
| --- | --- |
| `module_source_provenance.fy` | `0a89ef8d12fee173bf1ca6bfe841d371a4aa5d11e7d8d4b4f1bd495e07bcfea5` |
| `module_source_artifact_model.fy` | `665c199b7facca05eee4abdb64a0085f4f41bb1cefae9560e261d4eb52821783` |
| `module_source_artifact_decode.fy` | `82cbebd361e9caa436456be76a9b4cb716135cf7b89dc0b7fa8dbda136787f60` |
| `module_source_artifact_rebind.fy` | `5a0a4bd60b842d61a36525ba2438256eccd70697cd73c6e811624bfaeb59ee08` |
| `module_project_artifact_current.fy` | `ff62dd0bd7fb270d5b6bf36485fdc693882babef6ec4c23109926c32ec43f724` |

Cache artifact hashes:

| File | SHA-256 |
| --- | --- |
| `windows-low.log` | `a182e39523f7f8c5d3e3abfcea2e2da48d9a6b4295b96e3288afd694124da227` |
| `source-artifact-low.exe` | `60f8982ee4ea81504850956c61bcba9466f8af02f5f45fbecf4ee7de5bc5e5bc` |
| `legacy-golden.bin` | `b2056ad173df9f45483038c4b78aeeef12c959ac227aa43d905b8e9434913f70` |
| `imports-8192.bin` | `77d67076f3d009768f1c824a656b1d53ed058081652928ba8fe408297d14ea4f` |
| `imports-8193.bin` | `813ac230c994c487fdab4692c09c573c345dec2d6a24c1d0918ab65e4870121a` |

`owned-files.json` and `worker-freeze.json` in the cache root record all six owned
file hashes, reproducer hashes and the observed Git revision. Integration may
change HEAD later; these hashes identify the exact worker result.
