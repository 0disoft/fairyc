# ADR 0035: Observe compiler-owned drops in the native C differential lane

- Status: Accepted for the Stage 0 prototype
- Date: 2026-07-19
- MIR source: `docs/mir/mir-v0.md`
- Backend source: `docs/mir/c-backend-safety-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Fixture source: `spec/fixtures/c-emission/manifest-v0.json`
- Implementation source: `internal/cbackend/`, `internal/cruntime/`, and `internal/cbackendnative/`

## Context

The Linux differential lane already maps each canonical trap category to a distinct reserved process
status, so it can compare return values and trap identity. Its resource cases still proved only the
final process status. A generated program could skip, duplicate, reorder, or select the wrong
active-payload cleanup and still return the expected small integer because every admitted user drop
body is empty.

Adding output to production drop glue would pollute `experimental emit-c`, add an observer to normal
builds, and turn a conformance mechanism into an accidental runtime ABI. Comparing only static C
text would not prove which control-flow path actually executed.

## Decision

1. `Stage0Profile()` remains the only profile used by product `emit-c` and `build`. It emits no
   drop observer declaration, call, metadata, or output.
2. The internal closed `drop-trace-v0` instrumentation profile is accepted only by the native
   differential path. Arbitrary instrumentation names fail before MIR inspection or C emission.
3. Each possible observed resource leaf is identified by canonical resource type plus the original
   source path and half-open binding span carried by its MIR `drop` operation. Equal identities are
   deduplicated, sorted by type and span, and assigned dense `uint32_t` site IDs. Build-local symbol
   IDs, C local names, and absolute paths are excluded.
4. A direct resource drop records its site. A resource-bearing `Option` records only `Some`; a
   resource-bearing `Result` records only the active resource payload. Scalar variants record
   nothing. Moves never record a drop.
5. The instrumented translation unit calls the internal
   `fy_runtime_trace_drop(uint32_t site_id)` observer. The Linux-only conformance runtime writes one
   fixed four-byte little-endian record to stderr with bounded retry on `EINTR`; a write failure
   terminates with reserved status 254. The product scalar runtime does not contain this symbol.
6. The native runner captures stdout and stderr separately under one aggregate output limit,
   decodes only complete four-byte records, and requires exact sequence equality with the bounded
   MIR interpreter's source-linked drop trace. Unexpected stdout, malformed stderr, missing sites,
   extra drops, missing drops, or ordering drift fail the compiler/profile case.
7. All seventeen C-emission cases use the instrumented unit for the 68 planned Clang/GCC
   low/optimized executions, including cases whose correct trace is empty. The four product-build
   smokes continue to execute uninstrumented artifacts and require no output.
8. The native JSON report advances from `fairyc.native-c-report/v1` to
   `fairyc.native-c-report/v2` and records `dropTraceExecutions` separately from total fixture and
   product build executions. A configured lane or counter is not a hosted success receipt.
9. This evidence covers compiler-owned cleanup for the currently admitted direct all-`i32`
   resources and built-in resource-bearing tags with empty user drop bodies. It does not execute
   user drop-body effects, prove reverse cleanup of resource fields, define a public trace format,
   expose a runtime ABI, or establish target conformance.

## Consequences

- A wrong active variant or cleanup edge can no longer hide behind the correct process status.
- Production C and product-built executables remain byte-for-byte free of conformance observation.
- Site metadata is deterministic and source-linked, but it is derived from the same verified MIR as
  the interpreter. Independent language semantics still come from the versioned interpreter
  fixtures; native comparison detects backend and execution divergence rather than proving MIR
  lowering independently.
- Hosted Linux execution, nonempty user drops, resource fields that need drop, output comparison,
  wider return transport, sanitizers, layouts, and bidirectional ABI fixtures remain later gates.

## Rejected alternatives

- Emit observer calls in every product unit: adds runtime coupling and ordinary-build overhead for
  test-only evidence.
- Print source names or type names from native code: leaks unnecessary text, increases output and
  formatting surface, and makes the record protocol locale-sensitive.
- Assign IDs from MIR traversal order alone: duplicated cleanup blocks for one source binding would
  become indistinguishable from semantic ordering drift.
- Infer cleanup from generated C text: cannot prove the executed branch or active tag.
