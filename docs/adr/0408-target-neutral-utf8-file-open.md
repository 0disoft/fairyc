# ADR 0408: Target-neutral UTF-8 File opening

- Status: Implemented Windows and Linux source slice
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

`std::fs` adds `create_truncate_utf8(ref str)` and `open_read_utf8(ref str)`.
These explicit factories accept text already guaranteed valid UTF-8 by the
source type. They do not hide an owned `CString` allocation. The safe wrapper
passes pointer and byte length into a target-owned C bridge. Each bridge rejects
empty text, interior NUL, and lengths above 4,095 bytes before copying into a
4,096-byte stack array and adding one terminator. It then uses the existing
target File open path, preserving option validation, handle publication,
consuming close, cleanup, and typed `FileError` results.

Windows strictly converts the UTF-8 path to UTF-16 as admitted by ADR 0407.
Linux passes the same UTF-8 bytes to `open`; its filesystem may also contain
non-UTF-8 names through the retained `ref CStr` interface. This is a portable
*UTF-8 input* contract, not a promise of identical name normalization, case
matching, symlink handling, containment, or access rights across targets.

## Cost and compatibility

Calls to the new bridge occur only from the new factories. The existing File C
component also contains the bridge, so its emitted C and compile cost can grow
for programs that use only the older File operations; this requires separate
measurement. The bridge copies at most 4,095 bytes on the stack and adds no
heap allocation, runtime metadata, generic specialization, or changed behavior
for existing CStr callers. SDK
version advances to `0.0.0-stage0.69`; language `/19`, runtime ABI `/0`, MIR
version and public-interface schemas remain unchanged. The cumulative 8%
C-relative runtime and executable-size budget still requires representative
paired measurement.

## Verification

The Windows SDK quickstart calls both new factories with a Korean filename and
removes it after close. The Linux public File native smoke retains its old CStr
case and adds a second Korean path round trip under Clang and GCC profiles.
Component selection accepts the new C symbol on exactly those two targets.
The focused Go, native, and compatibility checks own the observed results;
full target promotion remains separate.
