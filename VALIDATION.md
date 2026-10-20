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

## Repository Shape

cli-tool and monorepo validation must stay focused on repository shape and must not imply generated application source code.
