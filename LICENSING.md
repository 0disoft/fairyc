# Fairy C licensing

The project-owned portions of Fairy C are licensed under the Apache License,
Version 2.0 ([LICENSE](LICENSE)), as approved by the project owner on 2026-09-16.
SPDX identifier: Apache-2.0.

This covers project-owned compiler code, runtime implementations, SDK sources,
tools, tests, examples and documentation unless a file has a different explicit
license or is identified as third-party material. Existing copyright ownership
and attribution remain with their respective holders; this declaration does not
claim ownership of third-party or contributor material beyond rights available
to the project. Third-party terms take precedence for their own material.

## Third-party components

Preserve [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) and
[GO-LICENSE.txt](GO-LICENSE.txt) when distributing the compiler package.
The TOML parser remains MIT-licensed. The Go runtime/standard library retain
their original terms; Apache-2.0 does not replace those terms. Review additional
toolchain/component notices when changing the build or adding bundled software.

## Generated programs

Using Fairy C does not by itself license user-authored source under Apache-2.0.
Project-owned runtime/SDK code incorporated into a generated program remains
Apache-2.0 material. Distribute the license and preserve applicable notices for
those incorporated portions, including modification notices when required.
This is not a blanket exception for generated output or permission to strip
third-party notices. Compiler-only dependencies must not automatically be
described as dependencies of every generated program.

## Release status

License adoption is separate from release approval. Current local preview ZIPs
are packaging rehearsals, not project-approved public releases. Their LOCAL-ONLY
labels describe release status and do not add restrictions to the rights granted
by LICENSE or third-party licenses. Clean-build identity, supported-target tests,
provenance/notice completeness and release ownership still require verification.
