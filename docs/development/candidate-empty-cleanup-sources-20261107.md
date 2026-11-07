# Source retention for empty cleanup providers

The ordinary-AST and reference-type-index improvements move the normal authored
self-build beyond the earlier native_cli Values failure. At source `a5f7f7ed`,
the new optimized Go-seeded CLI passes its 14 independent process checks in
244.666s. The normal 646-source build then fails after 342.601s, exit 2, with no
C packet:

```text
project_source_modules.fy:13:62: project phase 2 stage 3 code 10
```

Stage 3 is the entire call-closure preparation stage. The reported source is
the imported cleanup provider, not necessarily the unit being checked.
The failure must not be described as a name-resolution failure or a timeout.

## Diagnosis

A cache-only compiler probe retains all 646 original source inputs and their
metadata, while selecting units for diagnosis. The project_source_modules unit
alone succeeds. A project-unit sweep identifies project_atlas_source (module
427), whose cleanup dependency in module 564 fails in module_unit_bodies::source.
A second probe distinguishes source-byte, cooked-byte and work limits: the
failure is work exhaustion inside scanner::copy's token loop.

The source snapshot SHA-256 is
`20e5f871423cd688e1d67ed5097182a45b28feec8b3311be1050cb9586258e3d`;
the complete manifest SHA-256 is
`4db50f43d720ccb14f3bbc09e232062b19a576b92a00cf4b88f39ca79545fdf1`.
Evidence is under `.cache/current-source-module-focus-20261107/`:
`run-single.json`, `run-batch.json`, `run-source-failure.json`, and `run-empty.json`.
The last two receipts correct an initial focus label that named the imported
provider instead of caller 427; the recorded execution output remains unchanged.

With a diagnostic empty-cleanup source snapshot, that identical input succeeds
in 2.069s, with 678214 Values work and 223 contexts. This is a selected-unit
experiment, not normal self-build acceptance, a fixed point, or a timing claim.
Both diagnostic receipts report no input drift. Failed diagnostic-harness
compilations and their corrected runs are retained separately.

## Implementation

Generic bodies retain the full provider stream and are collected first. On a
later cleanup-only source-cache miss, the collector checks all declarations in
that provider. Only providers whose cleanups all meet the existing unannotated,
nongeneric empty-cleanup AST conditions use a text-only snapshot. Any other
cleanup makes the provider retain its full stream, including when the empty
cleanup is encountered first.

Text-only snapshots own the complete original source bytes, so original spans,
source equality and implicit receiver spelling remain available after the
input parser owners are destroyed. Their token, cooked-buffer and diagnostic
arenas are empty. Empty cleanup ASTs contain no references into those arenas.
The stream getter remains available; such a stream is a source snapshot, not
a reparseable token stream. General scanner::copy behavior is unchanged.

For a cleanup-only provider with D declarations, T tokens and C cooked/error
rows, lexical-row copying changes from O(T+C) to a charged O(D) inspection and
zero retained lexical rows. Full source-byte copying remains O(B), independently
owned and charged. Generic, nonempty, annotated and mixed-provider paths keep
the existing full stream. Source, cooked, AST, work and timeout caps are unchanged.

## Validation status

The integrated source passes unlocked check and then locked check: 755 files,
9515 MIR functions, 1225 input hashes matching; 13.187s and 13.003s respectively.
The lock is copied only after both successful commands.
The permanent `TestCandidateEmptyCleanupSources` passes low/optimized in 49.14s,
four scenarios per profile. After original Parsed owners are explicitly removed,
it checks complete source bytes, implicit receiver and source identities. The
nonempty case checks the real `this.value` token and reference; the generic case
checks its parameter reference. Fallback streams match each original token's
kind, trivia, span, raw bytes and cooked bytes.

| Provider | Original / retained tokens | Retained cooked bytes |
| --- | --- | --- |
| Empty cleanup only | 1945 / 0 | 0 |
| Empty and nonempty cleanup | 1965 / 1965 | 10532 |
| Generic body and empty cleanup | 1966 / 1966 | 10532 |
| Annotated empty cleanup | 1950 / 1950 | 10532 |

Scanner SHA-256 is
`25da382fe851a2619e7c15ffe8b9157e01cb432937774864a73d3366738d8fc5`;
body collector SHA-256 is
`0532af4c9f0a5abfab727a9a64b03b4ec40549d680426c258f365366004c4ff5`.
The separate test worktree matches these main inputs before and after execution.
Its log is `.cache/parallel-fixed-point-receipts-20261107/.cache/empty-cleanup-sources-20261107/native.log`.
The fresh optimized whole-CLI regression passes in 185.451s, retaining all
14 independent CLI processes, four cold/warm trap pairs and six exact
Go-matching reports. A normal authored retry at clean `83452419` then fails
after 372.375s in owned IR generation, stage 6 code 1002, with no C packet.
The next capacity diagnosis is recorded in
`candidate-cleanup-recipe-fields-20261107.md`.
Normal authored self-build, fixed point and latest-source full parity remain
pending. Native Linux, clean-machine distribution and hosted CI are unexecuted.
No push, workflow dispatch, release or deployment is performed.
