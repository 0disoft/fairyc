# Quoted-page C comparison (2026-09-24)

This is a Windows AMD64 MSVC `/O2` comparison of the real Fairy
`pages::run` path with an independently written C11 reference for **quoted
ASCII string input only**. The Fairy image uses its ordinary scanner, SDK,
page protocol and byte-view loop. The C program implements the same protocol,
quoted-string recognition, hexadecimal cooked text and 4,096-value observation
window for the admitted input class. It is not a C port of the full lexer.
The [native parity test](../../internal/cli/byte_view_quoted_page_c_test.go)
first checked exact Fairy, C and Go-lexer output for empty, short, previous-
revision and 10,000-byte strings in both low and optimized profiles.

The paired test builds each optimized image once and writes it once to a local
temporary path. It then runs three uncounted warmup pairs and twelve fixed
alternating C/Fairy pairs on one 10,002-byte quoted string. Each measured run
checks all 12,290 output bytes against the Go oracle. The timer covers process
startup, input parsing, scanning, decimal output and capture; it excludes
compilation and EXE materialization. Windows process CPU times are retained but
their 15.625 ms steps are too coarse to apportion the work. OS scheduling,
frequency and caches were uncontrolled. The existing summary rule requires
the relative full span of both lanes and paired ratios to stay at or below
10%, and requires median estimators and order groups to agree at the 1.08
budget. The rule was fixed before this run.

| Observation | C | Fairy | Fairy/C |
| --- | ---: | ---: | ---: |
| Median process wall time | 12.1715 ms | 37.4546 ms | 3.0772 ratio of medians |
| Median paired wall ratio | | | 3.0454 |
| EXE bytes | 146,944 | 142,848 | 0.9721 |
| Single build, observation only | 302.2782 ms | 2,485.3864 ms | Different build scopes |

The first measured C execution took 37.7848 ms versus 11.2907–12.9144 ms
for its other eleven executions. Fairy ranged from 35.1992 to 39.3543 ms.
Eleven of twelve paired ratios lie between 2.7733 and 3.4605; the first is
1.0162. The C, Fairy and paired relative full spans are 217.7%, 11.1% and
80.3%, respectively. **The predeclared stability rule therefore returns
`inconclusive`.** The repeated gap is a serious performance signal, but this
single noisy run does not certify either side of the 8% runtime gate. The
optimized EXE is smaller for this one restricted workload; it does not prove a
full-app size result. Single build times are not comparable because the Fairy
lane includes workspace loading, checking and SDK compilation while the C lane
compiles its already loaded unit and support source.

Source inspection suggests an actionable difference: `pages::number` calls
`std::io::write_usize` and `std::io::write` separately for every emitted
integer, which reaches Windows `WriteFile`; the C reference uses buffered
`printf`. This identifies a likely output-call cost, not a measured attribution
to those calls alone. A bounded next change is to batch page output using the
existing explicit `std::bytes::Buffer`, preserve the exact bytes and error
contract, then compare it with the committed [raw samples](evidence/quoted-page-c-budget-20260924.json)
without changing or discarding this baseline. The broader lexer C-relative
release gate remains open.
