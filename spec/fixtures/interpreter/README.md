# MIR interpreter fixtures

`manifest-v0.json` is the machine-readable contract for the first source-to-interpreter slice. Every case must parse, resolve, type-check, complete Flow, lower to MIR, pass the MIR verifier, and then produce the exact return value or trap category.

The v0 slice intentionally covers only zero-argument `main::main() -> i32`, scalar literals, ordinary scalar calls, checked arithmetic, and canonical trap identity. Tagged values, resources, drop traces, regions, I/O intrinsics, process output, arguments, and loops remain listed deferrals rather than accidental behavior.

`schema-v0.json` uses JSON Schema draft 2020-12 and rejects unknown fields. Source paths are case-root-relative and `/`-separated.
