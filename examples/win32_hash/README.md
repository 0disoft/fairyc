# Fairy Hash

A small file SHA-256 utility. File reads, hashing, UTF-8 JSON formatting and
prepared report publication are written in Fairy C. The private SHA algorithm
is reused from `lexer_candidate`, with a SHA-256 length-overflow bound rather
than that compiler example's 96 MiB source limit. Reads use a reusable 64 KiB
buffer, independent of input size.

```powershell
fairyc build --release
.\build\win32-hash.exe --hash C:\data\input.bin C:\data\result.json
python -B verify_cli.py
fairyc test --locked --file tests/sha256.fy
```

The last path is optional. CLI stdout is the same UTF-8 JSON written to disk.
Failure returns exit 1 and `{"ok":false,"errorCode":N}`. Save writes/syncs/closes
an exclusively created sibling, then publishes with `fs::replace_file`; it
does not truncate the source, its hard-link aliases or existing temporary names.
This is not a power-loss durability or concurrent-file snapshot guarantee.
