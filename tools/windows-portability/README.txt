Fairy C Windows/AMD64 portability fixture

This is a test bundle, not the compiler preview download.
It includes fairyc.exe, two prebuilt native programs, their dependency and hash
manifest, this verifier, and the applicable licenses. LSP is separate.

Transfer the ZIP and its SHA-256 to a separate Windows/AMD64 machine with no
Go, Node, Bun, MSVC or Windows SDK installation. Verify the archive hash and
extract it. Then run the following in Windows PowerShell 5.1 or PowerShell 7:

  .\verify.ps1 -Receipt .\portability-receipt.json

The receipt path must not exist. The verifier checks all supplied file hashes,
creates a fresh starter, runs compiler check/test without a native toolchain,
and executes both prebuilt programs with an empty child PATH. Each program
allocates a byte buffer, reads and writes binary data, syncs/closes files and
writes a known console message. Temporary work is removed after verification.

The generated programs use MSVC /MT. The checked images directly import only
kernel32.dll; no UCRT or VCRuntime DLL is bundled. Static import inspection does
not enumerate dynamic LoadLibrary calls or prove all Windows versions work.

A successful run on a development PC is only a local result. The receipt never
asserts a clean machine from PATH alone. Retain separate evidence identifying
the clean host/image and transferred archive before claiming portability.
Building new native programs still requires MSVC and the Windows SDK.
