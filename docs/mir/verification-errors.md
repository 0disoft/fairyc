# MIR verification errors

Verifier codes classify compiler invariant failures, separately from source diagnostics. `Detail` is explanatory context, not a machine-readable discriminator. Results are sorted by `(Code, Entity, Detail)` and only exact triples are deduplicated. Multiple failures on one entity survive, including different details under the same code.

| Code | Invariant |
| --- | --- |
| `V1001` | Reachable control flow has valid entry, successor, terminator, and switch shapes. |
| `V1002` | Every use is dominated by initialization on all incoming paths. |
| `V1003` | Compiler admission capabilities must be valid. |
| `V1010` | Aggregate definitions have unique concrete identities and cannot collide across kinds. |
| `V1011` | Aggregate fields and variants have valid unique names and admitted field types. |
| `V1012` | Resource drop declarations, signatures, ownership obligations, and effect sets agree with their glue. |
| `V1020` | Foreign declarations have valid C ABI signatures and compatible link-name bindings. |
| `V1021` | Internal function symbols are present, unique, and distinct from foreign symbols. |
| `V1022` | Internal calls name an existing function and match its parameter count and result destination. |
| `V1023` | Return values agree with the enclosing function result shape. |
| `V1030` | Each operation has a known executable MIR kind. |
| `V1031` | Operation operands, destinations, operators, and metadata have the required shape. |
| `V1032` | Operation and signature types have concrete admitted identities. |
| `V1033` | Aggregate operations agree with their declared owners, fields, and variants. |
| `V1034` | Operands have known kinds, valid immediate ranges, and exact static or pointer types. |
| `V1035` | Compiler-scheduled cleanup stays within the function effect authority. |
| `V1040` | Private storage operations require explicit compiler admission. |
| `V1041` | Private storage operations have valid shapes and admitted element types. |
| `V1042` | Private storage and indexed-view operations use exact actual operand types. |
| `V1043` | Private storage transfers carry matching source and unsafe-scope provenance. |
| `V1044` | Returned storage views preserve their actual verified type. |

The existing `V11xx` ownership, `V12xx` drop-flag, `V13xx` borrow, `V14xx` region, `V15xx` trap/unwind, `V16xx` normalization and `V17xx` unsafe-provenance families retain their meanings. See [the MIR contract](mir-v0.md) and [the diagnostic catalog](../diagnostics/catalog-v0.md).

A private storage operation may report admission, shape, and provenance failures independently. Rejection never grants authority to execute an operation. Error descriptions and codes are declared explicitly at each verifier site; the verifier does not parse `Detail` to select a code.
