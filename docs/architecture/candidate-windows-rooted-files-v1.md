# Private candidate Windows handle-relative files v1

`windows_rooted_file.fy` owns Directory/File resources, UTF-8 admission, flat leaf
selection, chunked write loops, failure reports and consuming close. The closed
adapters in `internal/candidatehost/windows_rooted_file.c` marshal Windows calls.
They contain no cache key, artifact decoder, compiler frontend or reuse policy.
The public SDK and public CLI contracts are unchanged.

Directory open retains one non-reparse disk-directory handle. Every child open
uses that handle as `NtCreateFile`'s RootDirectory, with a single UTF-16 leaf and
`FILE_OPEN_REPARSE_POINT`. A renamed or replaced root pathname therefore does not
redirect later child operations. Directory/reparse child objects reject. Reads
deny concurrent writer sharing; creation is exclusive and never truncates an
existing name. The adapters independently reject empty/dot/dot-dot leaves,
separators, NUL, alternate streams, invalid modes and overlong leaves.

The caller supplies an existing root directory. Its path is resolved once on
open; this is not a portable path policy or an ancestor-symlink prohibition.
Leaves contain at most 255 UTF-16 words. Each read/write request is at most
64 KiB; the caller owns the aggregate payload limit. Zero-length reads and EOF
remain ordinary successful partial I/O. The cache layer must check its separate
artifact and work limits before allocating or publishing payload bytes.

An exclusively created File remains staged until successful handle-relative
rename. Abandoned staged files are marked for deletion through their own handle
and then closed, without reopening an attacker-replaceable temporary path.
Published files survive resource cleanup. Explicit close clears the owned
integer handle before the one native close attempt; cleanup never retries an
uncertain close. Handles use the same pointer-width integer transport as the
existing SDK File carrier.

The durable caller sequence is write-all, flush, rename within the retained
Directory, flush the same published File, and consuming close. Native rename
uses `NtSetInformationFile(FileRenameInformation)` and the retained root handle;
the Win32 rename attempt rejected that relative-root request with error 87 in
the recorded local experiment. No absolute-path retry is used. A failed rename
preserves the destination; a failure after successful rename can leave the new
complete file. Cache readers must revalidate it and may treat any failure as a
miss. These operations do not establish physical power-loss retention.

The maintained Fairy tests cover actual root rename/replacement, 131073 binary
bytes across multiple chunks, existing-name collisions, directory obstruction,
replacement, abandoned staging cleanup, repeated handle counts, real junctions
and file symlink rejection, and raw-ABI escape attempts. The fixture generator
creates the native reparse points under an isolated repository `.cache` path.

All four authored tests pass through Go-generated native C and candidate C
development/release. Two real junctions and one real file symlink are present;
the external marker retains its bytes and modification time. All test-created
ordinary directories are removed, and root/locked source checks pass 511 files
and 6307 MIR functions. The private transport selector tests pass. Existing SDK
directory remove/rename now use their already authenticated rules 38/40 in the
candidate C emitter; a current-source 425-input emitter was rebuilt without Go
frontend fallback to execute those paths. [Receipt](../ops/evidence/candidate-rooted-file-20261014.json).

This is the cache I/O prerequisite. Bounded cache namespace/read/publication
policy, verified cache reuse and CLI selection remain separate work.

## Retained-root enumeration and retention (2026-10-14)

A private Directory scan enumerates children of the retained non-reparse
Directory in `FILE_ID_BOTH_DIR_INFO` batches of at most 64 KiB read through
`GetFileInformationByHandleEx`, so continuation resolves against the same root
handle instead of a reopened pathname. One scan visits at most 4096 entries and
draws on a separate 16 MiB work budget from payload I/O. The entire result fails
closed: an oversize or malformed batch, an unreadable record, or an exhausted
entry or work bound discards the whole scan rather than returning a partial
listing.

Enumeration is not an atomic snapshot; it observes whatever the retained handle
resolves at each call. Metadata (file ID, timestamps, size, attributes) is a
retention hint only and never authority. `erase_if` reopens the same relative
leaf with DELETE access, denies write and delete sharing, and compares the
current metadata against the recorded hint; it marks the handle for deletion
only when they match, so a replaced or modified file is left in place, and it
refuses junction metadata outright. Metadata equality does not prove content
equality, so future cache consumers must fully verify restored bytes rather than
trust an ID or timestamp match.

All seven authored tests pass Go-native and candidate-generated C
development/release, extending the four existing rooted-file regressions with
601-entry multi-batch enumeration of Unicode names after root pathname
replacement, exact/exhausted 16 MiB work and maximum rows, changed-file
retention with current-metadata delete, malformed batch rejection, and real
junction/file-symlink flags on scanned entries. Root and locked source checks
pass 511 files / 6319 MIR functions. Actual disk cache storage and reuse are
still pending, and the public SDK and public CLI contracts are unchanged.
[Inventory receipt](../ops/evidence/candidate-rooted-inventory-20261014.json).

Windows ABI references: [NtCreateFile RootDirectory and reparse options](https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntcreatefile),
[relative rename information](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/ns-ntifs-_file_rename_information),
[NtSetInformationFile classes](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntsetinformationfile),
[FILE_ID_BOTH_DIR_INFO](https://learn.microsoft.com/en-us/windows/win32/api/winbase/ns-winbase-file_id_both_dir_info),
and [GetFileInformationByHandleEx](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-getfileinformationbyhandleex).
