# Retained generic growth evidence — 2026-09-16

Issue: [#27](https://github.com/0disoft/fairyc/issues/27).
Compiler source baseline: `f2ec982b32563cc2557082baeaf5584752295dee`.
Production compiler sources were unchanged. The stable-path test was added on top
of that baseline; its exact file SHA-256 is retained rather than claiming a clean
baseline test binary. Every receipt includes the executing test binary's digest,
Go/MSVC identities and workload source hashes. This is local evidence, not a new
same-commit hosted CI result or a standalone fairyc.exe measurement.

## Three distinct paths

| Receipt | Path and bounded workload | Result |
| --- | --- | --- |
| [Scalar planner](evidence/generic-growth-20260916-scalar.json) | Existing private compiler entry; 1/2/4/8 specializations, twice as many calls, 3 repetitions, low/optimized | 24 native samples return 42; duplicate calls share specializations; both 7-of-8 limit rejection cases publish no MIR |
| [Aggregate comparison](evidence/generic-growth-20260916-aggregate.json) | Existing private aggregate adapter; generic/manual at 1/3/5 types, 2 reversed-order repetitions, low/optimized | 24 native samples pass; interpreter return/drop assertions retained; no public generic-aggregate admission implied |
| [Stable workspace](evidence/generic-growth-20260916-stable.json) | New in-process ordinary `check`, `build --locked`, `run --locked` CLI test; two-package scalar source, generic/manual, 1/3/5 types, 2 reversed-order repetitions, development/release | 24 samples check/build successfully and run with exit 42, empty stdout/stderr; source/manifest hashes and executable sizes recorded |

No existing probe was replaced. The stable fixture uses bool/u8/i32/u32/usize and
two calls per type. Each sample starts in a fresh temporary workspace. Check timing
includes first lock creation; build timing includes the complete stable build call,
not only specialization. Run rebuilds and validates behavior separately; the size
belongs to the preceding build, not a claim that run executes those exact bytes.
There is no compiler-process startup measurement or production representative load.

## Limits versus observations

The scalar probe explicitly uses planner limits 8 total, 8 per declaration, depth 1;
its two rejection cases lower one count bound to 7. The aggregate probe uses 20/5/4.
The stable path uses [workspace compiler defaults](../../internal/workspacecompiler/check.go)
from [DefaultPlanLimits](../../internal/genericplan/planner.go): 4096/256/64 at this
baseline. Stable samples exercise at most five scalar specializations, not those
upper bounds. Scalar rejection proof is not a stable-CLI maximum-budget measurement.

Exact bounds are correctness assertions; timing values are observations only.
No timing threshold, p95 estimate, speedup claim, recursion admission or generic
language/API change is introduced. OS/toolchain caches are ambient; there are no
warmups and the first stable generic build took about 880ms, while the other stable
build samples were about 298–369ms. Two repetitions cannot establish causality or
a performance distribution. Sub-millisecond timers can report zero on this host.

For the stable samples, generic/manual executable sizes matched at each shape and
profile: development 108544 bytes for 1/3 types and 109056 for 5, release 107520
bytes for all shapes. Executable file size includes alignment/runtime overhead;
constant-return code may be optimized away. This does not prove zero specialization
cost. The private scalar receipt separately records C, object and code-section sizes.

## Reproduction and delivery status

Configured local checks used `fairyc_stable_generic_growth_measure` (27.887s),
`fairyc_generic_growth_measure` (6.858s), and `fairyc_aggregate_growth_measure`
(7.531s), serially with existing Go/MSVC caches. All passed. Formatting, retained
JSON/sample-matrix review and scoped strict checks accompany this evidence.
Raw command logs and private host paths are not copied into the receipts.

This fills the missing local public-path comparison and retained measurement
deliverable. No full suite or hosted replacement CI was rerun for these test/docs
changes. Push and remote evidence publication remain separate; #27 should not be
closed as published until the committed receipts are accessible to reviewers.
