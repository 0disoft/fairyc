# File search

This Fairy C command counts lines containing the case-sensitive byte sequence
`error`. It reads files in 4 KiB chunks. Repeated matches on one line count
once. Each file's nonempty last line counts even without a final newline.

From this directory with `fairyc` and the native toolchain available:

```text
fairyc check --locked
fairyc test --native --locked
fairyc build --locked
```

Run `./build/file-search.exe sample.txt` on Windows or
`./build/file-search sample.txt` on Linux. Both print `lines=4 matches=2`.

Pass `--dir DIR` to scan the regular files immediately inside a directory
and print their combined line and match counts. Subdirectories, symbolic
links, reparse entries, and other non-regular entries are skipped. An empty
directory prints `lines=0 matches=0`.

File and directory paths must be UTF-8. Pass an absolute path when using
`fairyc run --locked -- FILE` or `fairyc run --locked -- --dir DIR`.
A file, directory, or output error exits with status 1. An invalid invocation
or path argument exits with status 2. Diagnostics go to stderr.
