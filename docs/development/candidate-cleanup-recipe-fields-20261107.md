# Unit cleanup recipe field capacity

At clean source `83452419`, the fresh optimized Go-seeded full CLI passes its
14-process trap/cache/test regression in 185.451s. A normal 646-source authored
self-build then exits 2 after 372.375s without a C packet:

```text
optional/std/vec.fy:4:1: project phase 2 stage 6 code 1002 (bytes 78..600)
```

This is an owned-IR capacity failure, not a timeout or completed self-build.
The original source snapshot, invocation and zero-output hash are retained in
`.cache/current-selfbuild-empty-cleanup-20261107/`.

## Cause and bounded policy

A selected-unit probe keeps all 646 source inputs and their order. It identifies
caller project_atlas_source (427) and separates cleanup lowering from owned-body
and layout-name work. A one-charge diagnostic then distinguishes field capacity
from work exhaustion: the next field is rejected with **1024 fields already
retained and 647221 work before the successful charge**. The resource type ID is
2182; the diagnostic deliberately carries that ID in its module-number slot.

The owning unit contract previously limited generated cleanup recipe fields to
1024, even though its imported interface can already contain 2048 fields. This
change explicitly admits 2048 generated recipe fields per unit. The source AST's
1024-field limit and linked-project IR's separate implemented 8192-field bound
are not reused as the unit policy. Type/frame/operand/drop/work/recursion limits,
source admission forms, default timeouts and public SDK ABI remain unchanged.

Each recipe field still retains its exact owner, source ordinal and canonical
type. No fields are omitted or shared between different owners. On amd64, the
three-usize field payload is bounded by 49152 bytes instead of 24576; this is a
payload bound, not a measured allocator or process-memory result.

With the 2048 bound, that same selected unit produces **223 owned IR frames**.
All original source bytes and their order match. Since main now contains the
bound change, one source path is redirected to its authenticated original
`83452419` blob; the frozen-path override is explicit in the diagnostic receipt.
This is a diagnostic comparison, not normal self-build acceptance.

Evidence: `.cache/current-owned-ir-focus-20261107/run-cleaning.json`,
`run-fields-1024.json` and `run-fields-2048.json`. Original snapshot SHA-256 is
`3f81f5e6fbf84316fa95d65fd3bc92ec29b5a22ed38d6c00a7fcf4417358212a`.
The original manifest is
`9db7c28d75481761ffb229bb46077bd97136d4c3455c7f0c56982c9ff01c2c5a`;
the one-path frozen manifest is
`36b2538fc144674b507cd23f757ca892afdd15bd7e2b9433314665ec03c320e6`.

## Current validation

Unlocked and locked source checks pass with 755 files, 9516 MIR functions and
1225 source hashes matching (20.673s / 16.118s). The lock is copied only after
success. A fresh optimized build-only seed and help smoke pass in 134.874s;
this does not repeat or relabel the preceding 14-process CLI result.

The permanent `TestCandidateCleanupRecipeFields` passes Windows low/optimized
in 106.25s: **1023, 1024, 1025, 2047 and 2048 fields** are accepted, and **2049**
is rejected at the imported-interface gate (stage 3, code 10). It uses real
source, signatures and compile::range, then checks every generated cleanup
field's owner, ordinal and type and every glue's range. Input hashes stay fixed.
Each provider has at most 512 written source fields, retaining the source AST
limit. The final 2048-field case uses 33244 owned-IR work in both profiles.
The earlier parser/context-type fixture failures are retained separately and
are not counted as successful checks.

The tested lowerer SHA-256 is
`c0525246c3152cfae09b7a3ad802281395e787d3ede6f0a18ed145bbcf5b6920`.
Evidence is
`.cache/parallel-cleanup-fields-20261107/.cache/cleanup-recipe-fields/windows-native.log`.
Scoped vet, formatting and diff checks pass. No allocation-failure injection
or process-memory measurement is claimed.

The normal full authored retry, consecutive fixed points and latest-source
full parity remain pending. Native Linux, clean-machine distribution and
hosted CI remain unexecuted. No push, dispatch, release or deployment is made.
