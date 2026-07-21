# Package-source digest fixtures

`manifest-v1.json` freezes one exact package-source digest and four ordering or invalidation
relations. File input order is irrelevant. Canonical relative path bytes, file contents, comments,
and line endings are exact and therefore invalidate the digest when changed.

Fixture contents use lowercase hexadecimal so arbitrary bytes can be tested without JSON or editor
normalization. This manifest is not a public package manifest, source archive, cache record, or
permission to read files outside an already validated package source root.
