# ADR 0222: Retain explicit factory construction

- Status: Implemented
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Evidence scope: compiler-bundled SDK and checked-in examples at language revision `/6`
- Depends on: ADR 0001, ADR 0005, ADR 0218

## Context

C++ constructors, Java constructors and static factories, and Go composite literals solve related
but different problems. Importing constructor syntax into Fairy C would also invite overload sets,
implicit conversions, partial initialization rules, hidden allocation, special copy/move members,
and a second call-resolution path. The admission queue therefore required evidence that ordinary
functions returning `T` or `Result<T, E>` create material duplication before any shorthand could be
considered.

The audited non-fixture corpus contains 29 Fairy C files under `internal/sdkstdlib/` and `examples/`.
It has four public fallible factories that publish owned resources:

- `examples/owned_text/owned_text.fy`: `copy(ref str) -> Result<String, TextError>`;
- the same module: `cstring_from_text(ref str) -> Result<CString, TextError>`;
- `examples/windows_file/windows_file.fy`: `create_truncate(ref CStr) -> Result<File, FileError>`;
- the same module: `open_read(ref CStr) -> Result<File, FileError>`.

The first two distinguish copy and C-string conversion. The latter two distinguish destructive
creation from read-only opening and both carry `filesystem`. Their private implementations create
one inert empty owner, call an explicit unsafe transport, publish the owner only on success, and
consume or normalize it exactly once on failure. A constructor spelling would not remove that
rollback work. It would mostly replace useful policy names with `new`, hide failure/effect facts at
the call site, and make later overload or default-argument pressure more likely.

Direct record literals in the same corpus mostly build invariant-free transport structs or tiny
private demonstration resources. Consumers of `String`, `CString`, and `File` already use their
named factories instead of spelling private resource representation. No three independent owner
modules exhibit one identical factory body that a language rule could remove safely.

## Decision

1. Fairy C keeps ordinary explicit functions as its complete construction model. There is no
   constructor declaration, `new` expression, initializer list, implicit conversion constructor,
   copy or move constructor, destructor pairing rule, or overload-resolution exception.
2. An infallible factory returns `T`. A factory that validates, allocates, opens an external handle,
   performs I/O, or otherwise can fail returns `Result<T, E>`. Effects remain on the ordinary
   function declaration and are checked like every other call.
3. Factory names describe policy and origin: examples include `copy`, `from_text`, `open_read`, and
   `create_truncate`. A generic `new` or `create` is appropriate only when the module has exactly one
   unsurprising construction policy; it receives no language privilege.
4. A fallible resource factory owns its candidate locally until success. It must publish the fully
   initialized resource only through `.ok(value)` and must consume, drop, or restore the local
   candidate exactly once before returning `.err(error)`. Existing Resource and Result Flow, rather
   than a constructor-specific definite-initialization engine, enforce the ownership boundary.
5. A direct record literal is appropriate only when every required field is intentionally
   source-visible and the value has no hidden validity transition. Private resource representation
   is not made constructible merely because the resource type is public.
6. Factory calls already participate in mandatory Result/resource handling. Adding `@must_use`, a
   type-wide marker, or another constructor-only obligation is redundant.
7. This decision adds no syntax, diagnostic code, HIR or MIR form, ABI rule, runtime symbol, or
   language revision. The type phase applies existing visibility diagnostic `E0103` when an external
   module attempts a resource record literal, closing an implementation gap in the `/6` contract.
8. Reopening construction syntax requires all of the following evidence:
   - at least three independent owner modules with the same irreducible source pattern;
   - proof that a library helper or ordinary associated function cannot remove it;
   - exact initialization, failure, cleanup, effect, visibility, and evaluation-order semantics;
   - no implicit conversion or overload set;
   - parser, semantic, Flow, MIR, backend, compatibility, and build-cost evidence named in advance.

## Consequences

Construction stays visually ordinary and searchable: a factory call is a call, its result and
effects remain in the header, and policy remains in the name. Resource rollback code is still
explicit inside the owner module, which is deliberate because that code is the safety boundary,
not boilerplate to hide. Fairy C gives up C++-style terse construction and Java-style constructor
conventions in exchange for one call model and fewer invisible rules.

## Implementation evidence

The type checker permits public inert-struct literals and calls to public resource factories across
module boundaries, but rejects a cross-module resource literal with `E0103` and a related resource
declaration span. The 59-case semantic fixture corpus includes one positive factory case and one
negative representation-bypass case. Focused semantic unit tests also prove that same-module
resource literals retain their existing behavior.
