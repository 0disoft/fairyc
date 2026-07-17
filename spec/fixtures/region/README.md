# Region Fixtures v0

This directory accompanies `docs/language/0003-regions-and-arenas.md` and `docs/adr/0012-stage-0-region-and-result-fixture-contracts.md`.

`manifest-v1.json` is the current case index and `schema-v1.json` is its JSON Schema 2020-12 contract. v1 preserves the v0 expectations and adds an optional `packageGraph` reference for sources that require an explicit resolved package instance. `manifest-v0.json` and `schema-v0.json` remain historical inputs. Parsing and name resolution must succeed for every case. Pass cases complete through Flow; fail cases name `type` or `flow` as the phase that owns the rejection.

Pass assertions expose only the compiler-known operation, source span, hidden origin, optional last use, and return escape decision. Fail cases list every independent `E31xx` diagnostic with role-labelled byte spans. AST, HIR, Flow block, MIR local, raw address, and backend allocation details are not contracts.

For the lexical return escape, `E3101` is the single independent diagnostic. `E2102` is suppressed for that expression because it would restate the same rejected `LocalRegion(scratch)` origin.

Most cases use declared `Region` parameters. `fail.flow.local-region-escape` instead links `pass.package.stdlib-arena-region`, resolves the `app` root against its explicit standard-library dependency, and uses that dependency's opaque `std::alloc::Arena` resource. A runner must reject a missing case, a non-pass package graph, a missing root node, a mismatched `sourceRoot`, or a provider other than the declared standard-library instance as fixture setup failure.
