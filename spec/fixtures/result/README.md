# Result Fixtures v0

This directory accompanies `docs/language/0004-results-traps-and-control-flow.md` and `docs/adr/0012-stage-0-region-and-result-fixture-contracts.md`.

`manifest-v0.json` is the case index and `schema-v0.json` is its JSON Schema 2020-12 contract. Parsing and name resolution must succeed for every case; fail cases name `type` or `flow` as the owning phase. Pass assertions classify one typed `Result` expression as returned, explicitly discarded, or propagated with `?`. They record canonical payload types and exactly-once evaluation without exposing tag layout or MIR blocks.

The initial suite accepts `Result` values as parameters. It does not invent `Ok` or `Err` constructor syntax before that surface is accepted. Active resource-payload moves, cleanup traces, and representation remain later MIR and runtime fixtures.
