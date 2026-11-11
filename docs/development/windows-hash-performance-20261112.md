# Fairy Hash SHA-256 performance, 2026-11-12

Scope: retain file/hash/UI policy in Fairy C. No native hashing library or C hash
implementation. Preserve SHA-256 reference output, progress and cancellation.

## Baseline and component measurement

Retained the 187,392-byte authored release as
`.cache/windows-hash-performance-20261112/baseline.exe`, SHA-256
`f4c94b3303ea54914c341c5e163a5f847d54a2d7aa736302340fd8d1d13bb8af`,
with its original source alongside it. Compiler remains
`.cache/windows-gui-fairy-cli-20261109/low/fairyc.exe`, SHA-256
`e124d7a921f455f4c3b90ef5e7460959f2338844c930405929fa7b7c5ef157ea`.

`profile_costs.py` copies source/config/native files into an owned temporary
workspace, substitutes `profiling/main.fy`, and links `profiling/clock.c` only
there. The separate entry point calls the production fs/SHA/UI primitives, with
the same 64 KiB chunk and 100ms progress cadence. QPC brackets separate read,
SHA update, message poll/layout, progress and finalization intervals. This is
instrumented wall time, not CPU sampling or the entire production app/dialog
workflow. The production EXE is unchanged by measurement. The pure-memory case
reuses the repeating file's first 64 KiB block. All three hashing cases check
Python's SHA-256; read-only checks total bytes. Fixtures and processes are owned
and removed on completion.

Baseline command, 64 MiB materialized cached local file:

```powershell
python -B examples/win32_hash/profile_costs.py --compiler .cache/windows-gui-fairy-cli-20261109/low/fairyc.exe --source .cache/windows-hash-performance-20261112/baseline-source --mib 64
```

| Mode | Total inner seconds | Read | SHA | Poll/layout | Progress |
|---|---:|---:|---:|---:|---:|
| Read only | 0.024927 | 0.024857 | 0 | 0 | 0 |
| Reused-memory SHA | 12.082316 | 0 | 12.082057 | 0 | 0 |
| File + SHA | 13.351874 | 0.078824 | 13.272756 | 0 | 0 |
| File + SHA + native UI | 14.178003 | 0.072125 | 13.307323 | 0.424990 | 0.372201 |

The file+SHA run spends about 99.4% of its measured loop in SHA update. Native
UI contributes about 5.6% of the GUI loop; 1,024 polls and 129 progress updates
were observed. SHA is the first optimization target. Read-only throughput is
cached-memory/filesystem performance, not storage bandwidth. Wall intervals
include scheduling and concurrent host work; separate runs are not subtracted
to infer costs. Each case used the same Python reference
`281e519df3077b557c6b03f5da83c4e8d397219259615dd7c3308f89cae8f2a6`.
Receipt: `.cache/windows-hash-performance-20261112/baseline-components.json`.
