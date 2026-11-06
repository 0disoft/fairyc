# Fairy native graph structural preparation, 2026-11-07

The parallel Sol high slot/trap worker supplies the two-file ADR 0528 unit.
An independent Go model derives slot maps and encoding/json expectations;
candidate-generated packets are never used as expected results.

Windows low/optimized each pass **17 native cases**, with **five** sealed-owner
and lifetime checks: **39 cases total**, **22.052s**. The final log is
`.cache/graph-native-plan-windows.log`. Both profiles cover multiple bodies,
helper/branch/borrow slots, sparse u32 identities, canonical trap order and
deduplication, exact/one-short/overdrawn work, unknown lookups and the
4095/4096/4097 aggregate-slot boundary. No selected failures remain.

Only the new native plan and its focused test are changed in this step.
Complete source/lock checks follow with final operation-emission integration.
This is detached planning, not C execution or persistent publication. Native
Linux, allocation-failure injection, full Go suite and hosted CI are unexecuted.
Public APIs/SDK ABI, DB, dependencies, default limits, repository and runners
are unchanged. No push, dispatch, release or deployment.
