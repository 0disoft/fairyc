# i64 console output

Uses SDK stage0.86 `std::io::write_i64` and `write_stderr_i64` to print
15 decimal values on each stream, including zero, negative values, the i32
boundaries and the i64 minimum/maximum. Every numeric write checks its returned
byte count. Newlines are separate explicit writes.

From this directory run `fairyc check`, `fairyc build --locked`, and
`fairyc run --locked`. Build/run accept `--release`.

Both stdout and stderr contain the same lines:

```text
0
1
-1
9
-9
10
-10
2147483647
-2147483648
2147483648
-2147483649
4000000000
-4000000000
9223372036854775807
-9223372036854775808
```

Conversion uses a 20-byte local array in Fairy and the existing byte writer.
It performs no heap allocation and preserves `ConsoleError.status` and the
completed byte prefix on output failure. Windows dev/release is verified;
Linux C emission is verified and actual Linux execution is deferred.
