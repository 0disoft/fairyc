# Constrained C emission fixtures

`manifest-v0.json` is the first source-to-generated-C stage contract. Every case must parse, resolve,
type-check, complete Flow, lower to MIR, pass the MIR verifier, produce the declared interpreter
outcome, and emit one deterministic C11 translation unit for the provisional
`x86_64-linux-gnu` profile.

The manifest freezes target and dialect identity, canonical Fairy C function names, supported
feature families, and the interpreter outcome. It deliberately does not freeze the complete C text,
private C identifiers, block labels, or formatting. The runner emits twice and requires byte-identical
output, LF-only text, and no absolute fixture path. It also creates a deterministic internal
`drop-trace-v0` unit, proves every interpreter-observed drop has a source/type/span site, and keeps
that unit separate from the production translation unit.

These 50 cases are emission evidence, not native-code evidence. They cover scalar control,
checked arithmetic, direct `i32` minimum emission as `INT32_MIN`, always-checked assertion control, `Option<i32>` Some and None, and `Result<i32, i32>` propagation success and
failure, copy-only `Result<usize, inert-struct>`, user-enum-bearing `Result` transport and
propagation, user-enum-bearing `Option` Some and None, and copy fixed-array `Option` Some/None plus
`Result` success/error propagation with indexed payload reads,
restricted inert user-enum construction,
named payload extraction, exhaustive switch, and acyclic inert enums nested inside user-enum
payloads with dependency-ordered complete definitions,
direct copy-scalar `Option` and `Result` fields inside private inert structs and user enums with
their private typedefs emitted before every by-value aggregate use,
plus acyclic aggregate-bearing `Option` and `Result` fields whose complete struct or user-enum
payload definitions precede the tag and whose tag precedes every containing aggregate,
plus copy-only tags whose direct payload is the already-fixed compact `Option<i32>` or
`Result<i32, i32>` record,
plus flat move-only resource construction over Fairy copy scalars or raw pointers, field access,
whole-root call move, empty-drop and compiler-private typed nonempty-drop
discharge, live reassignment, resource-bearing Option Some and None, resource Result Ok and Err, and
resource Result propagation success and error, plus positive scalar fixed-array construction,
contextual zero-copy immutable and mutable byte-view borrowing, internal value calls, exact canonical length
observation, checked reads and fixed-array or mutable-view indexed stores, zero-cost
byte-length reads, checked explicit
and omitted-bound end-exclusive immutable byte-view slicing, and internal
function transport for borrowed UTF-8 and byte-string views, plus exact explicit `i32 <-> c::int`
casts with target C-scalar assertions. Generated resource and resource-tagged structs use target-layout assertions, but this
lane does not ask a C compiler to evaluate them. No C parser, Clang, GCC, sysroot,
glibc, assembler, linker, executable, sanitizer, object metadata, or ABI probe runs in this lane.
Those boundaries do not become emission evidence even when a separate native command consumes the
same cases.

`internal/cbackendnative` is the separate Linux/amd64 lane. It consumes the interpreter-approved
instrumented units from this runner, requires glibc plus Clang and GCC, links the scalar runtime plus
a closed four-byte recorder, and plans 200 process-status and exact drop-sequence comparisons at low
and optimized profiles. Those executions include scalar fixed-array indexing plus explicit and
omitted-bound byte-view slicing. This observes compiler-owned empty resource cleanup and active built-in
payload selection plus pure user drop bodies. Effect-bearing drop remains deferred. The
emission runner itself remains
toolchain-free and does not inherit native evidence from the presence of that separate command.
