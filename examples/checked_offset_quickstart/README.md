# Checked offset ranges

This example uses SDK `0.0.0-stage0.76` decimal parsing and
`std::num::usize_from_i64` to validate
a half-open range `[OFFSET, OFFSET + COUNT)` within a logical length. It does
not allocate a file or a buffer of that length.

From this example directory, using a compiler with the bundled SDK stage0.76:

```powershell
fairyc check
fairyc run -- 10 3 4
fairyc run --release -- 5000000000 3000000000 2000000000
```

Expected stdout from the first run:

```text
length=10 offset=3 count=4 end=7
```

The CLI takes exactly `LENGTH OFFSET COUNT`. Each argument is at most 20 UTF-8
bytes and follows `-?[0-9]+`; plus signs, whitespace and decimal points are
rejected. Leading zeros and `-0` are accepted. `numbers::parse` delegates to
`std::num::parse_i64_decimal` and maps its errors to the existing caller errors.
The SDK accepts i64 minimum without negating it. Conversion then
rejects negative values. Accepted values are 0 through 9223372036854775807;
conversion is verified for the current x64 targets, not narrower usize targets.

The range check first requires `offset <= length`, then
`count <= length - offset`. Only after both checks may `offset + count` be
constructed. Zero-length ranges, including an empty range at the end, succeed.
Conversion alone does not establish valid bounds or a safe allocation size.

Successful validation prints one summary and exits 0. Invalid numbers or ranges
print a `checked-offset:` diagnostic to stderr, leave stdout empty and exit 1.
Wrong argument count exits 2; output or formatting failure exits 3. For example,
`fairyc run -- 10 10 1` reports `count exceeds remaining length`.

[ADR 0418](../../docs/adr/0418-sdk-checked-integer-conversions.md) owns the
SDK conversion contract; [ADR 0420](../../docs/adr/0420-sdk-checked-i64-decimal-parsing.md)
owns decimal parsing and first-failure ordering. The 20-byte argv limit belongs
to this example, not the SDK parser. `9223372036854775808x` reports range overflow;
`922337203685477580x8` reports syntax, preserving the earlier parser's order.
The [adoption receipt](../../docs/ops/evidence/checked-parser-adoption-offset-20261002.json)
records 37 argv cases for each Windows MSVC low and optimized profile: exact
argv/cwd, UTF-8 stdout/stderr, exit, source/SDK/compiler identities and EXE SHA256.
The [original receipt](../../docs/ops/evidence/checked-offset-example-20261002.json)
retains the earlier 35-case validation without changing its historical identity.
The owning test is `internal/cli/checked_offset_example_test.go`. Linux native
execution, physical file access and performance measurements are outside this
example's validation. Example package `1.0.1` replaces the local accumulator
without changing CLI behavior; SDK/language/runtime versions are unchanged by
this adoption.
