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

Windows ABI references: [NtCreateFile RootDirectory and reparse options](https://learn.microsoft.com/en-us/windows/win32/api/winternl/nf-winternl-ntcreatefile),
[relative rename information](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/ns-ntifs-_file_rename_information),
and [NtSetInformationFile classes](https://learn.microsoft.com/en-us/windows-hardware/drivers/ddi/ntifs/nf-ntifs-ntsetinformationfile).
