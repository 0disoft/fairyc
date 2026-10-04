# ADR 0456: Fairy i64 console output

- Status: implemented; Windows native verified, Linux execution deferred
- Date: 2026-10-04
- Related: ADRs 0303, 0410, 0430

## Decision

SDK `0.0.0-stage0.86` adds `std::io::write_i64(i64 value)` and
`std::io::write_stderr_i64(i64 value)`, both returning
`Result<usize, ConsoleError>` and declaring `@effects(console)`.
They emit decimal ASCII without a newline or locale-dependent formatting.
Success returns the completed byte count; failure preserves the existing
byte writer's raw status and completed prefix.

```fairy
import std::io as io;
@effects(console)
fn i32 main() {
    return switch (io::write_i64(-9223372036854775808)) {
        case .ok(written): { if (written == 20) { 0 } else { 2 } }
        case .err(_): { 1 }
    };
}
```

## Implementation

Conversion lives in `internal/sdkstdlib/sources/std/io.fy`. A private helper
fills a 20-byte local fixed array from back to front. Signed division and
remainder extract digits without negating the minimum i64 value. A shared
suffix view is passed to the existing stdout/stderr byte writer. No heap
allocation, new C runtime symbol, formatter ABI or language feature is added.
The two private decimal helpers are module-local; callers cannot provide
an undersized scratch view.

SDK source digest, public interface, package instance and dependent lock/cache
identities refresh. Language `/20`, runtime `/0`, grammar, MIR/wire schemas,
foreign ABI and target maturity stay unchanged.

## Verification and limits

`examples/io_i64_quickstart` checks every numeric write's byte count and writes
newlines explicitly. Windows dev/release executions compare stdout and stderr
against independently formatted values: zero, signs, one/two-digit transitions,
i32 boundaries, values beyond i32, and both i64 endpoints. Native Go tests use
Go's integer formatter as the output oracle. Public CLI check, locked build and
locked run exercise the same example with a separate compiler executable.

Four additional dev/release executions redirect each stream separately to a
real read-only OS file handle. Each checks nonzero error status, zero completed
bytes and unchanged file bytes. These are actual OS failures, not injected
application failures. Partial nonzero-prefix failures are inherited from the
existing write-all implementation and are not newly reproduced in this unit.

Windows MSVC and Linux C emission require exactly the existing stdout/stderr
write-all foreign symbols. Source tests reject missing console effects (E7102)
and wrong argument types (E1006). Related SDK, loader, console and component
regressions plus Go vet pass. Raw executions and compiler/source/SDK/image
hashes are recorded in `docs/ops/evidence/io-i64-20261004.json` and its CLI
receipt.

Linux actual execution remains deferred to the owner's batch. No benchmark,
installation, CI change, push or release is included.
