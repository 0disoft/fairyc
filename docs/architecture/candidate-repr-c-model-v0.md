# Candidate repr(C) observations

Private lane notes for `examples/lexer_candidate/src/repr_c_declarations.fy`.

Bounded lane: a source-declared, inert repr(C) struct shape only.
read decides the whole declaration. It approves only a kind-4 (struct)
declaration whose sole attribute is exactly an @repr(C) marker, whose tag and
field names are C identifiers, and whose fields all sit in the admitted value
lane. layout re-checks the tag/field lane alone against the parsed arena and
owned source bytes; the caller pairs it with read so a forged marker flag
cannot approve a lane the verifier cannot re-derive from source text.
Code contract (declaration-local, never a public compiler diagnostic ID):
  0    not an inert repr(C) struct, or a fully approved one.
  5106 malformed shape: a bad or duplicated @repr, a non-C spelling, any
       other struct attribute such as @effects, a non-C tag or field name
       (C keywords, runtime namespaces fy_/FY_/INT/UINT, libc names, and
       main, all named by ffi_symbols::valid), a duplicate field name, or a
       generic struct.
  5102 well-formed @repr(C) whose value lane is not admitted here: an empty
       body, a field type outside fixed scalar / closed C scalar / raw pointer
       to scalar, void or an inert struct, or a non-value field kind.
read reports the exact @repr(C) attribute span when it approves, the
offending attribute/argument/name/type span when it rejects a field or name,
and otherwise the declaration span. layout is the marker-independent lane
predicate: a plain struct with admissible fields is true even though read
reports repr_c=false, code=0 because it carries no repr(C) marker.
ADR0415 whole-owner rule: inputs stay shared (ref) and results are owned, so
no arena slot and no simultaneous mutable sibling view is created here.
Fixed scalar i32/i64/u8/u32/usize/bool, the closed C scalar set, or an
nullable raw pointer to a scalar, void or struct element. Nested by-value repr(C)
structs, arrays, borrows, resources, and tagged/generic types wait for later
units and are rejected here.
Tag and field-name lane plus value types. 5106 is a name/structure violation,
5102 a value-lane violation, 0 an admissible inert layout.
