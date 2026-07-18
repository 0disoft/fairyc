# Constrained C emission fixtures

`manifest-v0.json` is the first source-to-generated-C stage contract. Every case must parse, resolve,
type-check, complete Flow, lower to MIR, pass the MIR verifier, produce the declared interpreter
outcome, and emit one deterministic C11 translation unit for the provisional
`x86_64-linux-gnu` profile.

The manifest freezes target and dialect identity, canonical Fairy C function names, supported
feature families, and the interpreter outcome. It deliberately does not freeze the complete C text,
private C identifiers, block labels, or formatting. The runner emits twice and requires byte-identical
output, LF-only text, and no absolute fixture path.

These eight cases are emission evidence, not native-code evidence. They cover scalar control,
checked arithmetic, `Option<i32>` Some and None, and `Result<i32, i32>` propagation success and
failure. No C parser, Clang, GCC, sysroot,
glibc, assembler, linker, executable, sanitizer, object metadata, or ABI probe runs in this lane.
Those boundaries do not become emission evidence even when a separate native command consumes the
same cases.

`internal/cbackendnative` is the separate Linux/amd64 lane. It consumes the interpreter-approved
units from this runner, requires glibc plus Clang and GCC, links the shared embedded scalar runtime,
and compares process status at low and optimized profiles. The emission runner itself remains
toolchain-free and does not inherit native evidence from the presence of that separate command.
