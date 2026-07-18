# MIR fixtures

`manifest-v0.json` is the machine-readable Stage 0 contract for source-to-MIR normalization facts. Its cases must parse, resolve, type-check, complete Flow analysis, lower to MIR, and pass the MIR verifier before observations are compared.

The manifest freezes source-linked semantic events rather than textual MIR, block numbers, local numbers, temporary names, or cleanup-block layout. It covers ordered calls, ownership transfer, explicit cleanup order, `Result` propagation, non-unwinding calls, and division or checked-arithmetic trap paths that skip cleanup.

`schema-v0.json` uses JSON Schema draft 2020-12 and rejects unknown fields. Source paths are case-root-relative, `/`-separated, and paired with half-open UTF-8 byte spans.

`internal/mirfixture` executes all ten cases through parser, resolver, typer, Flow, mutable-local MIR lowering, and the MIR verifier. The Region cases prove inert placement, inner-resource cleanup before normal reset, direct return and Result-propagation resets, and inner-to-outer nested reset order. The arithmetic case proves static overflow categories for negation, addition, subtraction, and multiplication plus invalid signed-right-shift amounts. Passing this runner proves only the source-linked v0 subset above; verifier mutations and interpreter outcomes are owned by their separate manifests, and no backend is proven.
