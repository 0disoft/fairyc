# ADR 0510: Bounded Windows compiler-host C headers

- Status: Accepted for the explicit private compiler-host build entry
- Date: 2026-11-07
- Extends the existing private Windows compilation-unit builder only.

The current full authored compiler emits a 2,099,985-byte C header: 2,833 bytes
above the ordinary project builder's 2 MiB limit. Its literal unit, runtime,
205 body units and total source all remain within their existing bounds. This
is measured input, not a reason to expand ordinary user-program admission or
to change compilation semantics. Changing partition size cannot shrink this
shared header.

`BuildWindowsMSVCCompilerHostPartsWithStackReserve` is a separate trusted-source
entry for already verified compiler-host C. Its header limit is 8 MiB. Existing
ordinary builders, including their explicit-stack and image-only variants,
retain 2 MiB. A stack reservation alone never selects the broader header role.
Literals/body units remain 2 MiB each; runtime 8 MiB, body count 4,096 and total
transport 96 MiB. Existing timeout, stack, compiler flags, toolchain, COFF/image
inspection, output and failure cleanup policies are unchanged.

The new entry inspects and releases every object before linking; its returned
artifact contains image bytes and no retained object set. Existing builders
preserve their previous object-retention behavior. No C packet decoder, ordinary
companion, source frontend, cache format or fallback compiler is widened.

The full authored CLI test host explicitly uses this entry. It retains exact
C/profile/image digest binding before reusing a built image; no Go frontend or
cached syntax may replace Fairy command execution. Cheap role/shared-budget
tests cover exact/one-over limits. The measured header size also compiles and
returns 42 under native Windows low/optimized while the ordinary entry refuses
the same input. Existing stack and ordinary compile/link-failure tests pass.

Full CLI evidence and remaining scope are tracked in
`docs/development/candidate-inline-locals-full-cli-20261107.md`. Native Linux,
consecutive self-build, distribution and same-commit hosted acceptance remain
separate. No stable CLI/SDK/runtime ABI, database/provider, dependency, default
project budget, CI/runner or repository policy changes.
