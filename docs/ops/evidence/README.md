# Evidence

- Status: Draft

Raw machine-readable receipts for the measurements and probes described in `docs/ops/*.md`. A
receipt records what one run observed; it is evidence, and it cannot override a source-of-truth
document.

- Operational contract: `docs/ops/00-operational-contract.md`

## Where output goes

- Scratch, exploratory, and re-run receipts go to `.cache/evidence/` (git-ignored). Nothing is
  committed from there.
- New root-level JSON in `docs/ops/evidence/` is ignored by default through
  `/docs/ops/evidence/*.json`, so a fresh run does not show up as an untracked file.
- A receipt is committed only when it is explicitly reviewed and promoted. Tracked historical
  receipts keep working because the ignore rule does not affect files already in the index.
- Receipts that fail review are parked in `.cache/evidence/unreviewed/` rather than deleted.

## Promoting a receipt

```text
git add -f -- docs/ops/evidence/<name>.json
```

Promote a receipt only after its note records the command, toolchain and version, source identity
(commit or digest), outcome, limits of the claim, and hashes that make the run reproducible.
Receipts must not contain secrets, tokens, credentials, or private values, and a receipt without a
naming note is not a receipt this repository accepts.

## Historical receipts

Tracked historical receipts are preserved verbatim. Do not rewrite their timestamps, hashes, paths,
or toolchain fields to make an old run look fresh, and do not present a historical receipt as
reproduction of a current build.

## Locks

Workspace lockfiles stay tracked and are not ignored. A stale lock is refreshed through the official
`internal/lockfile` path (`FromWorkspace` plus `Marshal`) and confirmed with its workspace comparison,
with the previous lock kept as a backup under `.cache/evidence/unreviewed/` before the refresh.
