# Signed 64-bit CSV aggregator

This Fairy C example selects a named CSV column and prints its count, sum,
minimum and maximum. Values and every running total must fit signed `i64`
(-9223372036854775808 through 9223372036854775807). Decimal integers may have
a sign and surrounding ASCII spaces or tabs; floats and exponents are rejected.

The original practical project is preserved. This copy widens numeric parsing,
aggregation and decimal output using SDK `std::bytes::append_i64` and
`std::io::write_bytes`. Its parser and aggregation logic are entirely Fairy C.

From this directory, build with a compiler bundling SDK stage0.73 or newer:

```powershell
fairyc build
fairyc build --release
fairyc run -- samples/wide-total.csv value
```

Expected output:

```text
count=2 sum=4294967294 min=2147483647 max=2147483647
```

CSV quoting, escaped quotes, embedded newlines, LF/CRLF, a leading UTF-8 BOM,
UTF-8 validation and named header selection retain the original implementation.
Limits remain 1048576 input bytes, 65536 raw record bytes, 16384 decoded field
bytes and 64 columns. The whole bounded input is read before aggregation.
Parse or running-sum errors return 1, report record/line on stderr and emit no
summary. Argument errors return 2. Allocation or console failures are reported;
console writes are not transactional and may leave partial output.

The overflow guard uses explicit i64 bindings because a large literal on the
left of a binary expression does not yet inherit the right operand's type.
An intermediate sum overflow is rejected even if later rows would cancel it.

The regression cases and native-image hashes are described in
[REPORT.ko.md](REPORT.ko.md). The source contract is
[ADR 0413](../../docs/adr/0413-executable-signed-64-bit-integers.md).
