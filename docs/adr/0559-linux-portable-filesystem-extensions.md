# ADR 0559: Linux portable filesystem extensions

- Status: Accepted for source/model prototype; native Linux evidence separate
- Date: 2026-11-14

Implement missing Linux bodies behind the existing target-neutral std::fs
methods. Some v0 link names retain their historical `fy_std_windows_*` spelling;
those names are compatibility ABI, not a promise to call a Windows API. Linux
component selection may now supply exactly the separately implemented subset.
The Windows component mapping and implementation remain unchanged.

Keep dependency closure explicit and minimal. File path operations select an
additional Linux component after the base File adapter. Other extensions are
admitted individually as their implementations land; recognizing one portable
method must not authorize unrelated Windows-only entry points.

File rename uses Linux `renameat2(RENAME_NOREPLACE)`: existing destinations are
never replaced and unsupported kernels/filesystems fail without an overwriting
fallback. Hard links use the OS no-replacement operation. Byte copy opens a
regular source, creates the destination exclusively, retries interrupted I/O,
handles partial writes and consumes both handles exactly once. Failure reports
bytes already written and can leave the new partial destination; it does not
remove or truncate an existing path or promise a source snapshot/durable save.

Windows-executed POSIX syscall models validate operation logic only. Tests that
compile and execute the actual Linux C source are distinct and require a real
Linux GNU AMD64 host. Source/model success never promotes target maturity or
substitutes for that native gate.
