# Public-interface hash fixtures

`manifest-v1.json` compiles eight complete Fairy C source variants through syntax, semantic Flow,
typed HIR, public-interface extraction, and the v1 hash. The baseline freezes the full digest.
Private implementation, ordinary public bodies, declaration order, and private resource
representation must preserve it. Public signatures, public struct field order, and public foreign
link names must change it.

The fixture does not prove inline-body reuse because Stage 0 exposes no inline function contract. It
does not serialize HIR, layout, native ABI, object code, cache records, manifests, or lockfiles.
