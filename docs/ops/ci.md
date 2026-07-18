# CI

- Status: Draft

## Current frontend gate

`.github/workflows/frontend.yml` runs on pushes and pull requests. It validates the Go frontend on
GitHub-hosted Ubuntu and Windows runners using the version declared by `go.mod`.

- Token permissions are limited to `contents: read`.
- Concurrent runs for the same workflow and ref cancel the older run.
- Go dependency and build caching is disabled because the current module has no external dependency
  graph and the project has not justified retained CI cache storage.
- `go vet -mod=readonly ./...` runs before `go test -mod=readonly -count=1 ./...`.
- The workflow uploads no build artifact and performs no deploy, release, publication, or write to
  repository state.
- Ordinary tests execute the fixed lexer and parser fuzz seeds. Timed fuzz campaigns remain
  unconfigured rather than pretending a short CI sample proves coverage.

The containing Mustflow workspace provides local parity through `fairyc_go_vet` and
`fairyc_go_test`. Those intents additionally isolate ignored Go caches and disable module network
access for local agent verification.

## Owners

- Primary owner: UNASSIGNED
- Backup owner: UNASSIGNED
- Escalation path: UNDECIDED

## Validation

- Required validation names: VALIDATION.md
- Release blocker status: UNDECIDED; branch protection is external repository state and is not
  claimed by this document.
- Remaining operational risk: remote workflow execution has not been observed yet, and no timed
  fuzz campaign or coverage threshold exists.
