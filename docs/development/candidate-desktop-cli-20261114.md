# Candidate desktop command parity — 2026-11-14

Work in progress, following completion of the Result CLI/cache item.

The first slice moves resource preparation, bounded Python helper execution and
package-receipt verification into `internal/desktopapp`. Existing Go CLI entry
points retain forwarding wrappers; stable output, failure codes, archive size
and containment/hash checks are unchanged. The shared package contains no Fairy
frontend and launches no desktop windows.

Validation: `go test -mod=readonly ./internal/desktopapp ./internal/cli -run
'^(TestStablePackage|TestDesktop)' -count=1 -timeout=3m` passes. No new tests were
needed for the mechanical move. Candidate new/package command wiring and real
generation/build/package CLI acceptance remain pending.
