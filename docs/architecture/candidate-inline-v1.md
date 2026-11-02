# Fairy candidate inline bodies v1

- Status: Implemented bounded prototype with Windows full-CLI and persistent-cache
  evidence for generic/nongeneric callers under both profiles; native Linux deferred.
- Owner: [ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md),
  [cache v5](../package/whole-package-cache-v5.md), and the existing Stage 0 grammar.
- Evidence: [source/artifact and native-C receipt](../ops/windows-candidate-inline-20261102.md),
  [nongeneric CLI/cache receipt](../ops/windows-candidate-inline-cli-cache-20261103.md),
  [generic-caller CLI/cache receipt](../ops/windows-candidate-generic-inline-cli-cache-20261103.md),
  and [MSVC inline-trap receipt](../ops/windows-candidate-inline-traps-20261103.md).

## Source and identity

The authored candidate admits one argument-free `@inline` on a public, safe,
nongeneric ordinary function returning i32, with zero through four immutable i32
parameters and no effects or owner. Its returned expression contains at most 64
nodes: integer literals, parameter paths, grouping, unary +/- and binary
+,-,*,/,%. Calls, locals, mutation, control flow, unsafe/effectful bodies,
aggregates, resources, generic functions and cleanup annotations remain closed.
The current source grammar uses an explicit `return` in a function statement
block. No function-tail grammar extension is introduced by this implementation.

`module_inline_declarations` validates annotations, including declarations not
selected for ordinary signatures. Its canonical JSON and SHA-256 agree with
Go's `hir.InlineFunctionID`. Parameter positions replace names; equivalent
integer spelling and parentheses do not change the identity. The i32 minimum
normalizes to Go's single signed literal rather than a unary operation over an
out-of-range positive literal. Operator trees, values and arity participate.
Normal name/type/ownership/Flow checks still precede Complete publication.

## Detached bodies, source admission and keys

Each checked body retains its full 256-bit inline marker. Lowering, project
linking, duplicate-body comparison and artifact rebasing preserve it. The body
verifier reconstructs the pure expression identity from typed detached IR and
rejects a conflicting marker or unsupported body. An absent marker denotes an
ordinary body; source admission separately compares it against the freshly
checked declaration, rejecting missing and extra markers as well as changes.

`FYBODY00` retains byte-identical no-inline encoding. Inline bodies append eight
big-endian u64 fields carrying the eight u32 SHA words after the existing text
field. A zero, malformed, partial or extra trailer rejects; old strict readers
reject the extension. The private IR profile advances from
`fairyc.candidate-owned-body-ir/0` to `/1`, isolating incompatible cache inputs.
This is private candidate transport, independent of the Go MIR artifact codec.

After Complete source/IR validation, `module_inline_packages` hashes sorted
module/function names, their function identities and the stable package
coordinate using Go's `fairyc.inline-bodies/v1` JSON tuple. A package with no
inline exports has the exact zero class. Detached public-interface records
retain the class separately; public-interface v5 itself remains body-independent.
The ordinary v5 key binds owner and direct-dependency classes, and the generic
key includes the same base bytes. Cache lookup/admission does not skip current
source checking or authorize executable bytes solely through this digest.

## C emission and transport

Both single-unit and partitioned signatures use private `static inline`
linkage. Partitioned emission places the definition once in the shared header
and omits it from body units. Existing CFG evaluation and checked operations
remain responsible for argument ordering, temporaries and arithmetic. C compiler
inlining is optional; no forced substitution or performance guarantee is made.

For an MSVC build, a parameterized inline body first reads each private i32 parameter through
`volatile int32_t fy_inline_arg_N` and back before normal MIR emission. The preprocessor removes
this implementation barrier on other compilers. It keeps constant actuals from making an existing
checked trap appear unconditional to MSVC, which could otherwise reject a valid caller continuation.
The public signature, MIR, language semantics, ABI, cache wire format and C linkage remain unchanged;
the compiler may still inline the body. No warning is suppressed and no performance result is
claimed.

The Go and Fairy `FYCPARTS1` readers recognize ordered inline header definitions,
reject duplicates between header/body units and reject inline definitions in
body units. A program whose only function is inline may still have its separate
entry wrapper. Literal, frame, packet and entry bounds remain in force.
No public C export, retained callback, new backend, target promotion or cache
default is introduced.

## Generic callers

Generic callers may call the nongeneric inline exports. In the acceptance source,
`fn i32 evaluate<T>(T marker)` discards `marker`, executes the ordered side-effecting
four-argument call, checks i32 minimum and division, and returns `add(40)`. Both
`evaluate<i32>(1)` and `evaluate<bool>(true)` return 42; `main` asserts equality and
returns the first result. The test entry executes both specializations. The exports
remain public, safe, nongeneric and pure i32 arithmetic. This proves caller use of the
existing family and does not admit generic inline exports or another source form.

## Windows command acceptance

Both compiler profiles pass the full authored `check/build/run/test` path with
nongeneric and generic cross-package inline callers. Development/release products return
42, and interpreter/native tests cover ordered once-only arguments, i32 minimum
and division. Unmodified companion packets place four reachable inline definitions
in the shared header and none in body units.

Both generic and nongeneric callers pass all 26 fresh-process cache commands
under both profiles. Coverage includes cold/warm restoration, selected/unused inline edits,
private/equivalent-spelling mixed reuse, annotation changes, corrupt generations
and rejection of a private unselected annotation. Whole-project class invalidation
is separate from [current-source rechecked unit reuse](candidate-package-unit-cache-v0.md):
valid caller IR can be reused while fresh C emission uses the changed owner's
current inline body. A class edit rejects complete and all-unit hits, while cold
or mixed rebuilding is valid.

See the [nongeneric receipt](../ops/windows-candidate-inline-cli-cache-20261103.md)
and [generic receipt](../ops/windows-candidate-generic-inline-cli-cache-20261103.md)
for results and reproduction. Windows `test` retains its non-cache entry.
The [inline-trap receipt](../ops/windows-candidate-inline-traps-20261103.md) covers
division-by-zero, signed-division-overflow and addition-overflow through nongeneric and generic
bool-wrapper callers. Zero-parameter traps, the full operator matrix, C17, native Linux,
full-suite/hosted CI and distribution remain outside this evidence.
