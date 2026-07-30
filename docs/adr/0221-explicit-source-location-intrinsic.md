# ADR 0221: Add one explicit source-location intrinsic

- Status: Accepted, not implemented
- Date: 2026-07-30
- Product source: `docs/product/02-spec.md`
- Admission source: `docs/adr/0218-modern-language-concept-admission-boundary.md`
- Depends on: ADR 0136, ADR 0184, ADR 0219, ADR 0220

## Context

Diagnostics, assertions, logging helpers, and test utilities sometimes need to report the source
site that invoked them. Hand-written file and line strings drift during edits. C preprocessor
macros and C++ `std::source_location` solve parts of this problem, but Fairy C has no preprocessor,
general default arguments, runtime stack inspection, or stable aggregate ABI that should be opened
for one convenience feature.

The narrow useful operation is lexical: materialize the exact location of an explicit expression
in the current source file. It must remain independent from inlining and optimization, expose no
absolute host path, allocate nothing, and lower through existing static UTF-8 string transport.

## Decision

1. Fairy C admits exactly one contextual compiler intrinsic:

   ```fy
   source_location()
   ```

   `source_location` is not a lexer keyword. The exact zero-argument call is recognized by semantic
   analysis. The name is reserved in module and local value namespaces; a user declaration or
   binding with that name is rejected with `E4401` rather than changing the intrinsic by shadowing.
   A non-call path does not produce a location value. Any argument uses the existing call-arity
   diagnostic `E1010`.
2. The expression has exact type `ref str`, static lifetime, and no effects. Its cooked UTF-8 bytes
   use this canonical ASCII format:

   ```text
   <source-root-relative-path>:<line>:<column>@<start-byte>
   ```

   The path is the compiler input's canonical forward-slash `.fy` path and never an absolute path.
   `line` and `column` are one-based decimal numbers. Column counts original UTF-8 bytes from the
   current line start, not Unicode scalars, graphemes, or terminal cells. `start-byte` is the
   zero-based byte offset of the `source_location` identifier. Decimal fields have no leading zeroes.
3. Line geometry is derived from the exact immutable input bytes. LF begins the next line; in CRLF,
   CR remains a byte on the preceding line and LF begins the next line. The intrinsic span must be
   inside the retained source geometry and the path must satisfy the existing canonical source-path
   contract. Missing or inconsistent geometry fails with `E4402`; the compiler never substitutes
   an empty path, current working directory, generated placeholder, or host path.
4. Expansion happens while producing typed HIR. The HIR expression is an ordinary static UTF-8
   literal retaining the intrinsic's source span. Existing MIR string-literal lowering, interpreter
   behavior, static-storage deduplication, and constrained C11 emission are reused unchanged. No
   source-location operation, record, symbol, runtime function, reflection table, or debug dependency
   reaches MIR or generated C.
5. The value always names the lexical intrinsic invocation. A helper receives it only when the
   caller passes it explicitly:

   ```fy
   log_error(source_location(), "open failed");
   ```

   The compiler does not inject hidden arguments, rewrite omitted parameters, capture the outer
   caller through a callee, inspect a runtime stack, or change the value after inlining.
6. The intrinsic is forbidden inside ADR 0220 constant initializers and `@const_eval` bodies. It is
   already a compile-time expansion, but admitting it into scalar constant evaluation would widen
   that evaluator to borrowed strings for no useful gain.
7. Implementation advances the exact language pin from `fairyc-language/5` to
   `fairyc-language/6`. Existing `/5` syntax remains parse-compatible, but the reserved name and new
   call meaning are semantic changes. The previous-revision migrator preserves source bytes and
   requires a manual manifest pin update. Public-interface v3, MIR v0, runtime ABI, target schemas,
   C11 transport, SDK version, and Atlas formats remain unchanged. Whole-package cache vectors gain
   a new manifest version because the exact language component changes.

## Required evidence

- semantic pass coverage for exact path, LF and CRLF line geometry, a multibyte UTF-8 prefix, static
  borrow origin, and explicit argument transport;
- semantic failures for reserved declarations/bindings, wrong arity, non-call use, and unavailable
  or inconsistent source geometry;
- HIR evidence that the intrinsic becomes one ordinary static string literal with the invocation
  span and no callable symbol;
- interpreter and constrained-C backend evidence that a program observes the exact bytes while no
  intrinsic spelling or runtime marker survives;
- previous-revision migration identity, public-interface stability, package-cache invalidation, and
  deterministic repeated-build evidence; and
- documentation that distinguishes byte column, line, and start-byte coordinates from Atlas
  content and line-map identities.

## Consequences

Callers gain a cheap, reproducible location token without a preprocessor or hidden call contract.
The value is intentionally a display-and-log string rather than a structured object: consumers that
need machine joins must continue using the versioned source-file and line-map identities owned by
Atlas. The deliberate cost is explicit plumbing through helper parameters and a fixed textual form;
that cost prevents one convenience feature from importing default arguments, reflection, debug-info
dependence, or a new public aggregate ABI.
