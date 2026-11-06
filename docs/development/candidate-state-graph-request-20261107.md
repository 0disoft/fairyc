# Fairy graph request encoder, 2026-11-07

A third disjoint Sol high worker implements ADR 0531 in the new encoder and
its dedicated tests. The API is `snapshot_of`, `empty_work`, `encode`, `content`
and `digest`. A separate plain identity field avoids the frontend's still
unsupported direct fixed-array resource fields; digest storage is an owned
byte buffer. No language admission or public ABI is expanded to bypass this.

The worker's Windows low/optimized suite passes in **73.193s**. The independent
Go header writer supplies offsets/lengths and the complete expected bytes and
SHA256. Cases cover seven codecs, UTF8, opaque keys, ordering/duplicates,
layout, 64-bit policy values, read values larger than write policy, exact/
short/overdrawn structural and byte work, **65,535/65,536/65,537 entries** and
**16 MiB-minus-one/exact/one-over payload**. Private construction, content and
digest mutation are refused. Input storage is moved/owned and Packet never
retains caller views. No forced OOM is claimed.

Final current-source command, replay and lock evidence are recorded in the
transport integration receipt. Packet tests do not stand in for execution or
response/CAS admission. Linux execution is batched; full Go suite and hosted
CI are unexecuted. Public CLI/SDK/runtime ABI, DB, dependencies, existing wire
ceilings, repository and runners are unchanged. No push, dispatch, release or
deployment.
