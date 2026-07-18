# MIR interpreter fixtures

`manifest-v0.json` is the machine-readable contract for the first source-to-interpreter slice. Every case must parse, resolve, type-check, complete Flow, lower to MIR, pass the MIR verifier, and then produce the exact return value or trap category.

The v0 slice accepts an explicitly named zero-argument entry returning `i32` and executes the fixture-backed scalar, tagged, whole-resource, and Region subset. Region observations record the lexical span, exit kind, placement count, and number of resource drops completed before reset on fallthrough, direct return, Result propagation, and nested exits without exposing addresses or capacity. User drop-body effects, Region `break` and `continue`, I/O intrinsics, process output, and arguments remain listed deferrals rather than accidental behavior.

`schema-v0.json` uses JSON Schema draft 2020-12 and rejects unknown fields. Source paths are case-root-relative and `/`-separated.
