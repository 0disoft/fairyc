# Signed 64-bit formatting

This example uses SDK 0.0.0-stage0.73 `std::bytes::append_i64` and public console
output. Expected stdout:

```text
-9223372036854775808,9223372036854775807,0,-5000000000
```

Values are formatted in Fairy C with a bounded stack digit array. The minimum
value never needs a positive magnitude. [ADR 0413](../../docs/adr/0413-executable-signed-64-bit-integers.md)
owns the arithmetic, literal and compatibility contract.
