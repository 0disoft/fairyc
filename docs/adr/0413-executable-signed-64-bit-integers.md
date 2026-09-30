# ADR 0413: Executable signed 64-bit integers

- Status: implemented
- Motivation: the CSV aggregation workload needs signed totals beyond the i32 range.

The source type `i64` uses the range -9223372036854775808 through
9223372036854775807. Unsuffixed integer literals still default to `i32`;
an explicit i64 context selects the wider type. Unary minus admits the direct
minimum-magnitude literal without first constructing an invalid positive value.
Decimal, hexadecimal and binary literals share the same magnitude validation.

Compile-time operations use exact intermediate arithmetic and reject overflow,
division by zero, minimum divided or remaindered by -1, and shift counts outside
0 through 63. Runtime operations must preserve those checks in both native
optimization profiles. No wrapping arithmetic or implicit mixed-width operations
are introduced. Explicit widening from u8, i32 and u32 is value preserving.

HIR identity, source literals and constants, signed MIR immediates, interpreter
values and native C transport now support i64. Internal calls, scalar references,
fixed arrays, aggregate fields, and Option/Result payloads retain the full width.
Ordinary arithmetic, comparison, bitwise operators and checked shifts use the
same source contract in the interpreter and C backend. The C transport checks
bounds before arithmetic and uses unsigned bit patterns for signed bitwise and
shift results, avoiding signed C overflow and implementation-defined right shifts.

SDK 0.0.0-stage0.73 adds `std::bytes::append_i64`. It retains negative digits while
formatting the minimum value and publishes once through the existing Buffer
append. Language pin /20 and runtime ABI /0 remain unchanged: this additive
admission changes no previously admitted source or external runtime symbol.
Interpreter/codec tests and low/optimized MSVC execution cover normal containers,
decimal extrema, arithmetic overflow, division by zero, minimum/-1 and bad shifts.
The CSV workload is the separate application unit.

MIR version 1, stable function-body version 3 and stable package-MIR version 4
reject older exact-version cache payloads. Storage-transfer artifacts use
version 1 and compilation envelopes version 2. The new immediate field is omitted
for zero so unrelated operand records retain their existing field shape.
Source-interface hashes already encode canonical type names; adding i64 needs
no interface schema change. Old immutable identity fixtures remain historical.

Power-loss experiments remain deferred. This integer change makes no additional
storage durability claim.
