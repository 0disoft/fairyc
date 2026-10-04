# Direct integer array element references

Counts UTF-8 bytes using a fixed `usize[256]` histogram. Each increment passes
`ref mut bins[byte]` directly to a Fairy function; output reads `ref bins[code]`.
No intermediate named view or second histogram array is needed.

From this directory run `fairyc check`, `fairyc build --locked` and
`fairyc run --locked -- ababa`. Build/run also accept `--release`.

Expected stdout (byte value, tab, count):

```text
97	3
98	2
```

With no argument the input is `Fairy C`. One UTF-8 argument of up to 4096 bytes
is accepted. Empty input prints no rows; more than one argument or an argument
that does not fit returns code 2. Output is sorted by byte value and counts
bytes, not Unicode characters. The existing public argument/output SDK is used.

Requires ADR 0455. Windows dev/release execution is verified; Linux execution
is deferred to the owner's batch.
