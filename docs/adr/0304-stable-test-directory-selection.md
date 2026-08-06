# ADR 0304: Stable exact test-directory selection

- Status: Accepted
- Date: 2026-08-06
- CLI source: `docs/cli/command-contract.md`
- Depends on: ADR 0267, ADR 0268, ADR 0295, ADR 0300, ADR 0301

## Context

Stable `fairyc test` can select one or more exact test identities or files. Editors and CI jobs still
have to enumerate every file when they want one logical test subtree. Adding host path discovery,
glob syntax, or directory walking at selection time would weaken the exact portable path contract
and could diverge from the already admitted in-memory test overlay.

## Decision

1. Advance stable workspace test behavior to v10 with repeatable
   `--dir <portable-test-directory>` in execution and `--list` modes.
2. Derive the selectable directory catalog only from admitted portable test-source paths. For each
   file, include its parent and ancestors down to that package's exact `tests` root, never above it.
   Selection performs no new filesystem access.
3. Root directories use spellings such as `tests/unit`; workspace member directories retain their
   prefix, such as `libs/answer/tests/unit`. Matching is byte-for-byte, case-sensitive, and `/`-only.
   Absolute, host-separated, dot-segment, trailing-slash, missing, and uncataloged spellings fail as
   code 2 rather than being normalized.
4. A selected directory recursively includes tests whose admitted file path begins with the exact
   directory plus one `/` boundary. Repeated distinct directories form a union without changing
   canonical report order.
5. A directory containing admitted files but zero `@test` declarations is valid and produces the
   existing zero-test report. An empty filesystem directory has no admitted source identity and is
   therefore not selectable.
6. Keep `--dir` mutually exclusive with `--file` and `--exact`. Selection happens after declaration
   validation and limits only listing or execution; syntax, semantic, Flow, MIR, and verifier
   admission still cover the complete selected package or workspace overlay.
7. Preserve engines, trap continuation, locking, exit codes, text bytes, and JSON schemas. No report
   field changes.
8. Keep language `fairyc-language/10`, SDK `0.0.0-stage0.38`, runtime ABI `fairyc-runtime/0`, MIR,
   target, manifest, lockfile, package, interface, and generated-code versions unchanged.

## Evidence

- Go tests cover root, nested, recursive union, member/workspace prefix, located JSON, zero-test,
  unknown, duplicate, alternate separator, case, dot segment, trailing slash, and selector-conflict
  behavior.
- An invalid source outside the selected directory still fails complete-overlay compiler admission.
- The standalone compiler runs exact directory selection through isolated low and optimized Windows
  MSVC images while another admitted directory contains an excluded failing test.

## Consequences

Editors and CI can select one exact test subtree without host-specific paths, globs, directory
walking, or identity reconstruction. The selector is intentionally not a general filesystem path;
tags, fuzzy names, globs, changed-file discovery, and empty-directory selection remain separate.
