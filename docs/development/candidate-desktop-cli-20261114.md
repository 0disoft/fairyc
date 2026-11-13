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

The next slice adds private `--desktop new/prepare/validate/package` operations
to the existing C-only companion. It extracts the embedded Win32 starter and
formats verified local archive reports; it never invokes a Fairy frontend.
Native packet builds now prepare declared desktop resources before loading their
Windows build inputs. Ordinary console projects remain independent of Python.
Argument refusal before filesystem access, exact scaffold identity/output,
no-overwrite and the ordinary-project/no-Python path pass focused Go tests.
Existing companion mode/input tests also pass. Public candidate dispatch is not
yet wired in this slice.

Public candidate dispatch is now wired: `new` accepts console/win32 templates
in either flag order, `package` validates flags before filesystem work and asks
the candidate itself for release C before invoking archive tooling. Quiet builds
explicitly close their scratch directory. Go-owned CLI help/diagnostic metadata
was regenerated with its existing embed tool and passes exact `--check`.

Source checking passes (779 files / 9493 MIR functions) and the two isolated
native package-option tests pass. The initial low authored CLI passed the full
existing Win32 package verifier in 569.9s: 26 CLI cases, 12 save cases, 9 async
read cases, Unicode resources, JSON/plain reports and preflight/lock/output
preservation. No GUI/input/publication occurred. Its app is 192000 bytes and ZIP
100200 bytes; this is a candidate-backend measurement, not a Go-path regression.
Receipt: `.cache/desktop-cli-20261114/low-acceptance.json`, with the initial
source/image identity retained in `low/host.json` in that directory.

After that replay, empty `--template` validation was moved ahead of name/UID
checks to match Go's invocation precedence. The optimized host replay includes
six exact Go/candidate invocation comparisons before the same package verifier;
that final replay is still in progress at this commit.
