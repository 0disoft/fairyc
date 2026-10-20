# Dependency and Change Policy

- Status: Draft

## Contract

Dependency policy covers necessity, alternatives, license, maintenance health, vulnerabilities, runtime impact, bundle impact, major upgrade policy, and removal cost.

## Required Evidence

- Source of truth: `go.mod`, `go.sum`, `THIRD_PARTY_NOTICES.md`, and dependency-owning ADRs
- Owner: UNASSIGNED
- Merge-blocking validation: VALIDATION.md
- Related checklist: CHECKLIST.md

## Review Blockers

- A change bypasses the source of truth.
- A change weakens validation or hides skipped checks.
- A change lacks failure, recovery, security, performance, or test evidence where relevant.

## Current compiler dependencies

`github.com/BurntSushi/toml v1.6.0` is pinned solely behind `internal/projectmanifest` under ADR
0143. It is MIT licensed, requires Go 1.18 or newer, has no transitive module requirements, and
provides duplicate-key parsing, positioned errors, and undecoded-key metadata that a project-owned
partial TOML parser would otherwise have to reimplement. Upgrade or replacement requires complete
project-manifest fixture parity, dependency/license review, offline configured validation, and a
rollback path that preserves the semantic manifest model.
