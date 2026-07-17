# MIR verifier mutation fixtures

`manifest-v0.json` defines verified named seeds, one fixture-local mutation per case, and the single expected `Vxxxx` rejection. The seed and entity labels are test vocabulary, not MIR block numbers, local numbers, source symbols, or a backend exchange format.

A runner must construct the named baseline, verify it successfully, confirm every target and optional supporting entity exists, clone it, apply exactly one mutation, and then require the complete verifier result to equal the one listed code. A baseline failure or extra verifier code is a fixture harness failure.

`schema-v0.json` rejects unknown fields and uses JSON Schema draft 2020-12. Arbitrary graph serialization, mutation cascades, fuzz-corpus encoding, and crash-bundle rendering remain separate contracts.

No MIR implementation or verifier runner exists yet, so the fixture data does not prove current compiler behavior.
