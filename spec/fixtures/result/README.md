# Result Fixtures v0

This directory accompanies `docs/language/0004-results-traps-and-control-flow.md` and `docs/adr/0012-stage-0-region-and-result-fixture-contracts.md`.

`manifest-v0.json` is the case index and `schema-v0.json` is its JSON Schema 2020-12 contract. Parsing and name resolution must succeed for every case; fail cases name `type` or `flow` as the owning phase. Pass assertions classify one typed `Result` expression as returned, explicitly discarded, or propagated with `?`. They record canonical payload types and exactly-once evaluation without exposing tag layout or MIR blocks.

The initial parameter-driven cases still isolate handling from construction. Later v0 cases use the
qualified `Result::Ok(expression)` and `Result::Err(expression)` forms accepted by ADR 0017 and add
shallow matching assertions without changing this Flow schema. Active resource-payload moves and
cleanup traces are proved by the MIR and interpreter manifests; backend representation remains a
later code-generation contract.

The expanded v0 set treats Result locals and parameters as pending obligations. It proves transfer
into locals and callees, all-branch handling, branch-produced values, rejection at scope and early
return exits, rejection before overwrite, and repeated-loop overwrite detection. These observations
remain source spans and terminal handling facts; internal Flow block or state identities are not
serialized.
