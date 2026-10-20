# Validation

- Status: Draft

## Validation Source of Truth

This document defines the stable validation names for this repository.

## Standard Validation Names

- format
- lint
- typecheck
- test
- contract
- migration-check
- smoke
- docs
- check

## Required Final Report

Final responses must list executed validations, passed validations, skipped validations, skip reasons, and remaining risk.

## Runner Policy

Task runner files are optional. Runner `none` means no executable task runner is generated.
If a runner is generated, runner command names must match this document.
Unconfigured runner commands must fail rather than report a fake success.

## Hygiene Validation

Changes to repository hygiene files must be checked for line-ending churn, binary diff
pollution, tracked secret files, ignored build/cache artifacts, and generated-output drift.

## Scope

General validation routes must stay stack-neutral unless a runner file explicitly defines a command.

## Frontend CI routing

`frontend.yml` first runs a lightweight changed-path check. Root Markdown, documentation and
agent-guide Markdown, example/editor/fixture READMEs, example/editor guides, and images under
`assets/brand/`, `docs/`, or `diagrams/` can skip compiler jobs when they are the only changes.
Mermaid files under `diagrams/` also qualify. Other paths, including fixture JSON, grammar,
source, dependency files, and workflows, retain the full compiler checks.

The check reads the complete push diff or PR merge-base diff, including removed paths. Empty
diffs, new branches, tags, missing history, and failed path classification run compiler checks.
Manual Frontend runs also execute the full checks. The scheduled/manual Measurements workflow
keeps its existing triggers.

The Windows core job allows 20 minutes, with a 15-minute Go test timeout per package, because
it includes native compile/run regressions. Linux and the separate native/parser jobs retain
their 10-minute limits. The test selection and runner count are unchanged.

Local routing checks: `python -B -m unittest discover -s .github/scripts -p 'test_ci_changes.py'`.
Workflow syntax: `actionlint .github/workflows/frontend.yml .github/workflows/measurements.yml`.

## Repository Shape

cli-tool and monorepo validation must stay focused on repository shape and must not imply generated application source code.
