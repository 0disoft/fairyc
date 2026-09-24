# ADR 0407: Strict UTF-8 Windows file paths

- Status: Implemented Windows slice
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)

## Decision

The Windows `std::fs` C component accepts a nonempty, NUL-terminated path of at
most 4,095 UTF-8 bytes. ASCII retains its direct byte-to-UTF-16 conversion. On
the first non-ASCII byte, the component confirms that the terminator is within
the existing bound, then decodes the complete path with `MultiByteToWideChar`
using `CP_UTF8` and `MB_ERR_INVALID_CHARS`. Malformed UTF-8 fails with the
existing path-encoding status before any filesystem operation or ownership
transfer. Missing termination retains the path-too-long status.

All Windows file and directory operations share this path conversion. The
`ref CStr` source type still promises only NUL termination, not UTF-8 validity.
Linux File continues to pass bounded path bytes to its OS. This decision adds
neither a portable path type nor Unicode normalization or case-folding rules.

## Cost and compatibility

The ASCII path loop keeps the same direct conversion and adds no allocation,
runtime table, or generated per-call specialization. Non-ASCII paths pay one
bounded scan and one Windows conversion call. The Windows File component is
selected only when used. SDK version advances from `0.0.0-stage0.67` to
`0.0.0-stage0.68`; language `/19`, runtime ABI `/0`, C symbol names, and public
function signatures are unchanged. Current workspaces refresh their SDK pin and
lock. This does not establish the cumulative C-relative 8% runtime or image-size
budget for representative full programs.

## Verification

The separate-translation-unit Windows File probe retains malformed-byte
rejection and exercises a Korean UTF-8 filename through create, exact write,
close, reopen, exact read, close, and remove under low and optimized MSVC
profiles. Existing ASCII File probes continue to cover the previous behavior.
