# Digits-only i64 composition quickstart

This executable self-test shows an application helper for ASCII `[0-9]+` fields.
`numbers::parse` rejects a leading minus, then delegates to the SDK's
`std::num::parse_i64_decimal`. It introduces no public SDK function or error type.
The package starts at version `1.0.0` and uses SDK stage0.76, language `/20`.

The allowed values are 0 through 9223372036854775807, including leading zeros.
Empty input returns `Empty`. A leading minus (including `-0` and an overflowing
negative spelling) returns `InvalidDecimal` before signed parsing. All other
bytes follow the SDK's left-to-right syntax/range error order:
`9223372036854775808x` returns `OutOfRange`, whereas
`922337203685477580x8` returns `InvalidDecimal`.

`src/main.fy` contains 31 fixed inputs. Each checks direct and `?`-propagated
Results on a full view and a nonzero-offset subview, then checks the padded
owner's bytes. Invalid UTF-8, NUL, BOM and a Unicode digit are decimal syntax
errors here. The 128-byte padded array is only fixture storage; the helper adds
no input-length limit. UTF-8 validation, row/field/line diagnostics, aggregate
overflow and output policy belong to the application.

With the current compiler already built, the standalone commands are:

```text
fairyc check --path examples/digits_only_quickstart
fairyc build --path examples/digits_only_quickstart
fairyc run --path examples/digits_only_quickstart
```

The program takes no input, writes no output and returns 0 after all assertions.
A failed assertion must cause the run to fail. Repository agents use configured
mustflow intents for verification, rather than these manual CLI examples.

The retained [execution receipt](../../docs/ops/evidence/digits-only-example-20261002.json)
records source/SDK/compiler identities, MIR/interpreter verification and the
two Windows MSVC native profiles. Those profiles run the actual checked example
with raw argv/cwd/stdout/stderr/exit and EXE hashes. This is composition evidence,
not a new SDK API admission or a rewrite of the frozen round4 projects.
