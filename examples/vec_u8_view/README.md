# Shared and exclusive byte vector views

```fy
let ref u8[] bytes = vec::view_u8(ref values);
for (let u8 byte in bytes) { total += byte as i32; }

let ref mut u8[] writable = vec::view_mut_u8(ref mut values);
writable[0] = 255;
```

Both functions borrow the initialized `Vec<u8>` prefix without allocating or
copying elements. They expose length, checked indexing, half-open subranges and
named-view for-in. Empty views are valid, including before the first allocation.
A mutable descriptor binding is not iterable; use `let` or a parameter.

A retained shared view blocks vector mutation/replacement/move. An exclusive
view also blocks other owner reads and borrows. Child views freeze their parent
and retained siblings under ADR 0415. Short shared readers are permitted when
ordinary reborrow rules allow them. Release the old view before growing the
vector and acquire a fresh view afterward. Capacity beyond length is not exposed.
These functions do not reinterpret other Vec element types or expose a raw-pointer API.

Run `fairyc check`, `fairyc build`, `fairyc run --locked`, and
`fairyc run --release --locked` in this directory. The example checks boundary
bytes, shared summation, subrange reversal, returned/tagged views, shared readers,
exclusive iteration with a future-element update, growth and fresh views,
pop/truncate/clear, replacement and move after release.

`TestVecU8ViewNative` in `internal/cli` uses the existing Go native harness for
normal/release Windows execution and the later Linux batch. It also checks bounds,
overflow/underflow and the existing i64 vector example. Linux execution is currently
deferred. No additional PowerShell runner is required.
