# Atlas Evidence Bundle v1

- Status: Implemented output-only composition with explicit CLI exposure
- Decision: `docs/adr/0386-generic-symbol-identity-preparation.md`
- CLI: `fairyc experimental atlas-evidence show --bundle-version 1`

`EncodeShowV1` consumes only an exact-current v2 query, its retained workspace, and a matching
semantic snapshot. It composes Context Pack v2 and two unchanged attributed-text v0 documents:
original declaration source and attached documentation. Original generic declarations appear once,
including unused templates. Concrete implementation facts remain separate inside the pack; text
retrieval does not imply that an unused template body has been checked.

The v0 encoder, default CLI, and explicit `--bundle-version 0` retain Context Pack v1 and existing
output. Cross-version query inputs are rejected, never converted silently. The version option is
canonical `0` or `1`, at most once; it does not admit `--pack-version` on this command.

Envelope and logical field order follow Evidence Bundle v0. Only the version, encoding, checksum
domain inputs, and facts child version change:

```text
bundle version:     fairyc.atlas-evidence/v1
encoding:           fairyc.atlas-evidence-json/v1
checksum algorithm: fairyc.atlas-evidence-checksum/v1-sha256
checksum prefix:    fyevidence1-sha256-
```

SHA-256 covers ASCII `fairyc.atlas-evidence-checksum`, bytes `00 01`, then the bundle version,
encoding, and canonical logical bundle JSON, each prefixed by its unsigned 64-bit big-endian byte
length. `ParseChecksumV1` rejects v0, uppercase, and malformed checksums. Integrity is not
authentication or permission. The bundle remains `none`, facts `compiler-fact`, and both text
children `untrusted-text`; hostile prose is inert data. Pack text arrays stay empty.

The same explicit selection, fragment, aggregate text, per-child output, and final-output limits
apply. A text omission is reported by its child; an oversized child or bundle fails without
returning partial bytes. Budgets for the two text children remain independent. Source/snapshot
mismatch fails closed. No source is reopened after compilation. Locked mode, output errors,
no-newline canonical JSON, no host paths, and no decoder/store/network/model behavior are unchanged.
