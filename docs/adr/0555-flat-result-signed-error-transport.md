# ADR 0555: Flat Result transport with signed i32 errors

- Status: Accepted for bounded prototype
- Date: 2026-11-14
- Extends: [ADR 0550](0550-flat-result-success-generic-transport.md)

Admit root `Result<S,i32>` alongside `Result<S,u32>` for the seven existing
extended transport scalars. Direct `T` success and error `E` substitutions may
resolve to these same concrete forms. Signed errors use the existing i32
semantics, including negative errors; copy obligations and exactly-once `?`
evaluation remain unchanged.

This does not admit wider/error-side extended scalars, nesting, composite type
arguments, references, pointers, deferred/storage capture or new Result foreign
ABIs. Those remain separate families in the ordered implementation work.
No new scalar operations, syntax, serialization tags, allocation or boxing are
introduced. Old admitted type/artifact identities remain unchanged; the new
form uses existing concrete type vocabulary and existing verified transport.

Source and restored native observers must cover all seven success leaves,
floating signed zero and negative error propagation. Candidate source/artifact,
cache and CLI evidence are recorded independently of Go acceptance before
claiming this family complete.
