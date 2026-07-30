# Public-interface hash fixtures

`manifest-v3.json` compiles twelve complete Fairy C source variants through syntax, semantic Flow,
typed HIR, public-interface extraction, and the v3 hash. The baseline freezes the full digest.
Private implementation, ordinary public bodies, declaration order, private resource
representation, private must-use state, private module constants, and private source-location use must preserve it. Public signatures, public struct field
order, public foreign link names, and public must-use state must change it. V1 and v2 remain frozen
historical inputs rather than current runner contracts.

The fixture does not prove inline-body reuse because Stage 0 exposes no inline function contract. It
does not serialize HIR, layout, native ABI, object code, cache records, manifests, or lockfiles.
