# File search

This Fairy C command counts lines containing the case-sensitive byte sequence
`error`. It takes one UTF-8 file path and reads the file in 4 KiB chunks.
Repeated matches on one line count once. A nonempty line without a final
newline also counts.

From this directory with `fairyc` and the native toolchain available:

```text
fairyc check --locked
fairyc test --native --locked
fairyc build --locked
```

Run `./build/file-search.exe sample.txt` on Windows or
`./build/file-search sample.txt` on Linux. Both print `lines=4 matches=2`.
When using `fairyc run --locked -- FILE`, pass an absolute path so the
temporary executable can find the sample.
A file or output error exits with status 1. An invalid invocation or path
argument exits with status 2. Diagnostics go to stderr.
