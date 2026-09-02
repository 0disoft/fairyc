# Contract-state caller-visible retry fixtures

`manifest-v0.json` proves that the kernel never retries an `Execute` call internally. The fixture
runner owns the attempt loop, opens a fresh snapshot on each call, counts every attempt, and reports
budget exhaustion as caller policy rather than a new transaction-kernel error.

Each injected conflict publishes one complete competing overlay. A failed attempt publishes none of
its own overlay, and success after conflict reads the newly committed value from a fresh snapshot.
