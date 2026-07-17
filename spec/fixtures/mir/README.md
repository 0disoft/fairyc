# MIR fixtures

`manifest-v0.json` is the machine-readable Stage 0 contract for source-to-MIR normalization facts. Its cases must parse, resolve, type-check, complete Flow analysis, lower to MIR, and pass the MIR verifier before observations are compared.

The manifest freezes source-linked semantic events rather than textual MIR, block numbers, local numbers, temporary names, or cleanup-block layout. It covers ordered calls, ownership transfer, explicit cleanup order, `Result` propagation, non-unwinding calls, and trap paths that skip cleanup.

`schema-v0.json` uses JSON Schema draft 2020-12 and rejects unknown fields. Source paths are case-root-relative, `/`-separated, and paired with half-open UTF-8 byte spans.

No compiler or MIR runner exists yet, so these files are executable contract inputs rather than evidence that an implementation passes.
