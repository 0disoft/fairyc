# Region Fixtures v0

This directory accompanies `docs/language/0003-regions-and-arenas.md` and `docs/adr/0012-stage-0-region-and-result-fixture-contracts.md`.

`manifest-v0.json` is the case index and `schema-v0.json` is its JSON Schema 2020-12 contract. Parsing and name resolution must succeed for every case. Pass cases complete through Flow; fail cases name `type` or `flow` as the phase that owns the rejection.

Pass assertions expose only the compiler-known operation, source span, hidden origin, optional last use, and return escape decision. Fail cases list every independent `E31xx` diagnostic with role-labelled byte spans. AST, HIR, Flow block, MIR local, raw address, and backend allocation details are not contracts.

The initial suite uses declared `Region` parameters. `spec/fixtures/package/manifest-v0.json` now provides explicit standard-library package identity and injection, but local lexical-region escape still needs a bounded fixture-owned `std::alloc::Arena` API. Do not replace that missing API with a user-declared type merely named `Arena`.
