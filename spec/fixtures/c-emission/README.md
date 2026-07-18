# Constrained C emission fixtures

`manifest-v0.json` is the first source-to-generated-C stage contract. Every case must parse, resolve,
type-check, complete Flow, lower to MIR, pass the MIR verifier, produce the declared interpreter
outcome, and emit one deterministic C11 translation unit for the provisional
`x86_64-linux-gnu` profile.

The manifest freezes target and dialect identity, canonical Fairy C function names, supported
feature families, and the interpreter outcome. It deliberately does not freeze the complete C text,
private C identifiers, block labels, or formatting. The runner emits twice and requires byte-identical
output, LF-only text, and no absolute fixture path.

These four cases are emission evidence, not native-code evidence. No C parser, Clang, GCC, sysroot,
glibc, assembler, linker, executable, sanitizer, object metadata, or ABI probe runs in this lane.
Those boundaries remain listed in `deferred` until a configured Linux toolchain intent exists.
