# ADR 0526: Restore graph Complete through the real disk cache

- Status: Accepted for the explicit private single-module graph family
- Date: 2026-11-07

`project_state_graph_disk_cache` composes pre-compilation routes, the existing
rooted byte store and key-bound current-source graph restoration. Its low-level
load/save APIs retain typed cache/storage failures. Its build API treats cache
failures as disposable misses and returns the original Core source diagnostic
if independent cold compilation fails.

Load scans one immutable bounded inventory newest first. A digest-valid file
still requires full current-source, coordinate, decoded graph, portable body,
public/source-interface and compatibility-key admission before Complete escapes.
Malformed/incompatible semantic candidates are skipped without deleting another
owner's bytes. Work/allocation failure stops the attempt without resetting a
ledger. The existing raw store separately skips corrupt content addresses.

Save accepts an actual Complete rather than caller-paired source/coordinate
arguments. Both its route and envelope derive from that same owner. Publication
uses the existing immutable no-replace generations, metadata-conditional corrupt
collision repair, staging cleanup and retention. There is no second graph file
implementation, global lock, stronger power-loss claim or new default policy.

Lookup, verification and encoding each retain their 16 MiB histories; raw I/O
and scan keep their existing 1 GiB/16 MiB histories. Cold source compilation
uses its independent caller-owned budget. Missing/semantically incompatible
cache generations allow cold storage; inventory/I/O/work failure disables writes
for that build attempt. Already exhausted save-side scan/I/O is refused before
publication. These are private counters/observations, not compiler diagnostics.

`stored` means successful publication and retention completion. As in the raw
store contract, a late filesystem/sweep error can leave a fully admitted valid
generation; it does not establish a global instantaneous bound across writers.
Failed source/Complete admission never enters save.

Explicit compatibility snapshots serve deterministic comparisons. Production
adapters consume the sealed current environment. Workspace ownership, normal
CLI selection and native state execution/persistence are still separate.
Windows evidence: `docs/development/candidate-state-graph-disk-cache-20261107.md`.
