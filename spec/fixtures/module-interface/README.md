# Public-interface hash fixtures

`manifest-v4.json` compiles thirteen complete Fairy C source variants through syntax, semantic Flow,
typed HIR, public-interface extraction, and the v4 hash. The baseline freezes the full digest.
Private implementation, ordinary public bodies, declaration order, private resource
representation, private must-use state, private module constants, and private source-location use must preserve it. Public signatures, public struct field
order, public foreign link names, public must-use state, and public resource abilities must change it. V1 through v3 remain frozen
historical inputs rather than current runner contracts.

The fixture does not prove inline-body reuse because Stage 0 exposes no inline function contract. It
does not serialize HIR, layout, native ABI, object code, cache records, manifests, or lockfiles.
