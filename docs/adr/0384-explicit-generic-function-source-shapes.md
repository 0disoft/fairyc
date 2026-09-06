# ADR 0384: Retain explicit generic function source shapes

- Status: Accepted frontend preparation and compiler-private scalar specialization
- Date: 2026-09-05

## Decision

Use C-shaped function declarations with explicit ordered type parameters:

```text
fn T identity<T>(T value) { return value; }
fn i32 main() { return identity<i32>(42); }
```

The parser retains parameter names in a distinct type-parameter node and recognizes a type-argument
list only after a path when its closing angle is followed by a call. Bounded lookahead of at most
256 significant tokens separates this shape from ordinary comparisons. The AST owns ordered
`Function.TypeParameters` and `Expression.TypeArguments`; type arguments are never ordinary value
arguments and must never be erased before specialization.

## Admission sequence

This frontend commit deliberately rejects generic functions in semantic analysis and retains
unsupported specialization expressions. Even an unused type parameter on an otherwise concrete
function cannot bypass the gate. It does not release a new language/SDK version or claim runnable
generics.

Source admission must connect these AST facts to ADR 0339's bounded planner, clone and
substitute concrete types before ordinary HIR publication, check all admitted instantiations, and
reject implicit inference and unsupported ability or recursive expansion. Package-owned identity,
public-interface/cache transport, and native evidence are required before closing issues 26 and 27.

## Compiler-private source execution

`compiler.CheckGenericFunctions` now connects one source module to the existing planner through
`internal/genericsource`. It admits private, unqualified, module-local generic functions with
explicit builtin scalar arguments. Type references are cloned and structurally substituted through
parameters, results, bindings, casts, compound types, blocks, and loop regions. Generic-to-generic
calls become planner dependencies; concrete requests are deduplicated before function generation.
Every generated function then passes the ordinary semantic, typed-HIR, Flow, MIR lowering and
verification pipeline. No runtime dispatch or type-variable MIR representation is introduced.

The planner retains its catalog, arity, cycle, depth, per-declaration and total-expansion limits.
Source declarations carry complete AST digests; dependent specialization IDs change transitively.
The original entry point uses build-local package ownership. The package-bound paths described
below add coordinate ownership without claiming cache compatibility. Nominal type arguments,
source abilities, inference, and stable workspace/interface transport remain unadmitted.
Bindings cannot shadow generic functions in this initial private path, and reserved generated names
cannot be supplied by source. Template removal must not hide module declaration collisions.

The ordinary compiler entry points still reject generic source. This private preparation changes
no released language, SDK, runtime ABI, or CLI contract. Stable admission still requires versioned
compile fixtures, diagnostic integration, public-interface/cache invalidation evidence and a
language-version decision. The private fixture coverage below does not complete issues 26 or 27.

The compiler tests execute the same source through the MIR interpreter and low/optimized Windows
MSVC C transport, expecting 42. They also cover repeated instantiations, transitive calls, input AST
preservation, package-owner separation, template-body invalidation, negative admission and budgets.

`spec/fixtures/generic-source/manifest-v0.json` versions the private compile-pass/fail corpus
separately from stable language fixtures. Pass cases execute in the interpreter and both MSVC
profiles. Planner failures assert exact kernel codes, while the semantic failure asserts the
diagnostic code, failed phase, primary source text, and related expected-type location in the
original template. All cases must parse successfully, so an unrelated parser failure cannot satisfy
a compile-fail expectation. The ordinary compiler must still reject every case.

## Compiler-private resolved package execution

`CheckGenericFunctionsForPackage` binds a module to an exact nonzero `PackageCoordinateId`.
`CheckGenericPackages` extends this to an already-resolved `semantic.PackageInput` and its matching
coordinate map. Neither function discovers packages or accepts a coordinate as cache evidence.
Missing coordinates and duplicate coordinates assigned to different semantic packages are rejected.
The full coordinate digest is encoded in a versioned planner-safe namespace; disposable package
numbers and file/import ordering do not affect specialization identity.

`genericsource.ExpandPackages` collects all modules before planning, resolves alias-qualified calls
through exact source-path/provider bindings, and allows public scalar generic functions across
module boundaries. Private templates cannot be imported, including from another module in the same
package. Generated functions remain in their defining modules with the original visibility,
imports and source spans. One catalog and plan enforce global expansion limits and detect
cross-package cycles. Exact provider selection and reachable template changes invalidate transitive
specializations. Source ASTs and the caller's package input remain unchanged.

Tests cover a three-package transitive call returning 42 in the interpreter and low/optimized
Windows MSVC execution, equal module paths with distinct exact providers, renumbering/reordering,
private access, missing/conflicting providers, alias collisions, cycles and global budgets.
This is still an internal source execution path, not stable CLI or workspace-cache admission.
The internal source-interface path below retains exported templates separately from typed HIR.
Package-cache/artifact versioning, stable diagnostics and language-version promotion remain
required before issues 26 and 27 close.

## Internal generic source-interface identity

`AnalyzeGenericPackages` preserves a source-interface hash for every successfully analyzed package
before MIR lowering. `genericsource.PreparePackages` retains each public template's canonical
reachable generic declaration closure, including private helpers and uninstantiated dependencies.
Closure traversal is finite even for unused recursive definitions; actual requests still face the
planner's cycle and expansion gates. Unreachable private template changes do not enter the closure.

`moduleinterface.ComputeGeneric` combines the ordinary v5 descriptor with these sorted template
closures under `fairyc.generic-public-interface/v0`. Generated concrete specializations are removed
from the ordinary descriptor, so current type-argument requests do not alter the source interface.
The resulting `GenericHash` uses `fygif0-sha256-` and is deliberately incompatible with current v5
interface hashes and cache records. Failed semantic analysis publishes no interface hashes.

This private encoding retains the existing exact AST-based definition digests and may therefore
invalidate on source-span changes. It does not serialize template source or claim stable ABI,
whitespace-insensitive compatibility, cache admission, or a released interface-version change.

## Internal specialization-aware cache key

`GenericAnalysis` also retains the ordinary source-interface projection and sorted concrete
specialization IDs for each defining package. Ownership comes from the validated declaration
catalog, not from parsing generated function names. Packages with no emitted instantiations receive
an empty set; semantic rejection publishes none of these cache inputs.

`packagecache.ComputeGeneric` composes the complete v4 environment/source/ordinary-interface tuple,
the current generic source interface, every direct dependency's generic interface, and the exact
package-local emitted specialization set. Generic dependency coordinates must match the ordinary
dependency list exactly. Missing hashes, malformed specialization IDs, duplicates and noncanonical
order are rejected. This closes the key-level case where a consumer adds `bool` alongside `i32`
without changing the producer source or its public generic interface.

The key has a distinct `GenericKey` type and `fycacheg0-sha256-` spelling. Current v4 stores and
artifact records do not accept it. Persistent transport and stable CLI promotion remain separate
gates; the isolated in-memory MIR lane is described below.

## Bounded process-local generic artifact cache

`packageartifact.GenericCache` keeps generic keys and interfaces in a separate typed LRU with
mandatory entry, logical-byte and artifact-byte limits. Logical bytes count the serialized MIR
payload and its two full digests, not allocator/container overhead. The existing process codec
verifies artifacts before insertion and after decoding, and rejects foreign-process payloads.
Interface contradictions and malformed payloads are quarantined; quarantine compares the retained
record generation so an intervening replacement survives. Oversize writes preserve prior entries.

Tests store all three packages of the generic import chain under computed generic keys, load every
artifact through verified hits, reassemble the whole MIR program, and execute it in the interpreter
and low/optimized Windows MSVC profiles. Mutation isolation, LRU eviction, missing bounds, invalid
MIR, foreign/corrupt payloads and replacement-safe quarantine have direct regression coverage.

This cache transports ordinary package MIR only. A hit is not whole-compilation reuse: the caller
must still reassemble and verify against current compilation inputs. Contract-state compilation
bundles and stable CLI integration remain unadmitted. Persistent MIR transport is described below.

## Compiler-private generic workspace session

`workspacecompiler.GenericSession` explicitly connects current workspace analysis to one bounded
memory or disk cache. Every check performs generic expansion and semantic analysis before constructing
keys. Memory hits require exact current package IDs, instances and coordinate maps; disk hits require
stable instance/coordinate ownership and fresh ID rebinding. Only complete verified hits may skip
MIR lowering, and the collected artifacts must pass whole-program reassembly. Misses, mapping
changes and failed reassembly fall back to lowering and verified storage. Oversize records are not
stored; invalid current source never publishes a cache hit.

Memory and persistent cache policies cannot be combined. Contract-state transactions take the uncached compiler path
because ordinary MIR artifacts do not carry their complete compilation bundle. The ordinary session
and stable CLI retain their existing source admission and cache behavior.

Tests count lowering calls across cold and warm checks, invalidate both packages when a consumer
requests another specialization, reject invalid source after warm reuse, and exercise bounded-store
fallback. Warm workspace MIR executes as 42 through the interpreter and low/optimized MSVC profiles.
This internal activation does not change language, SDK, runtime ABI or stable CLI versions.

## Exact-version generic disk MIR transport

`packageartifact.GenericStableCache` projects verified package MIR to the existing stable MIR codec
and stores it beneath a separate `generic-v0` directory. The opened-root disk store continues to own
immutable durable publication, bounded reads, retention and generation-conditional quarantine.
Entry/byte limits apply to this separate store, not jointly to ordinary and generic caches.

The adapter uses domain-separated storage addresses derived from generic keys and interfaces; these
addresses are not v4 compiler compatibility identities. Its canonical `fairyc.generic-stable-mir-cache/v0`
envelope also retains the original complete generic key and interface plus exact-version stable MIR.
Read admission checks both identities, canonical encoding, byte bounds, stable MIR validity, package
instance and coordinate. A mismatch quarantines only the observed generation. Oversize writes do
not disturb a prior valid generation. General v4 cache APIs still do not accept generic key types.

A three-package generic import-chain test closes the writer and starts a fresh test process. The
consumer reads only stable artifacts, rebinds different package/function/local IDs without lowering,
and executes the result as 42 through the interpreter and low/optimized Windows MSVC transport.
The fixture's fixed storage keys isolate this transport proof from existing source-invalidation tests.

`GenericSession` selects this adapter only through explicit persistent configuration, including an
optional opened filesystem root and mandatory disk/artifact limits. CompilerBuild must have the
canonical nonzero executable SHA-256 identity format; the caller is responsible for supplying the
current executable's identity. The session binds it together with the generic storage family and
exact stable MIR/disk record versions, rather than its process-local codec identity.

Workspace tests fill the disk cache from current generic source and create a fresh process using
the same actual executable digest. Its lowering hook fails if called; complete verified hits execute
as 42 in the interpreter and MSVC low/optimized profiles. New specialization requests and a changed
compiler digest miss, invalid current source cannot hit, and tiny artifact bounds skip storage while
returning the verified uncached result. Disk publication failures likewise return the compiler result
with StoreFailed counts; they never claim a stored artifact.

No CLI or default storage path is exposed, and contract-state compilation bundles remain uncached.

## Evidence

- `internal/syntax/generic.go`
- `internal/ast/generic_test.go`
- `internal/compiler/generic_syntax_test.go`
- `internal/compiler/generic_source_test.go`
- `internal/compiler/generic_native_test.go`
- `internal/compiler/generic_fixture_test.go`
- `spec/fixtures/generic-source/manifest-v0.json`
- `spec/grammar/stage0.ebnf`
