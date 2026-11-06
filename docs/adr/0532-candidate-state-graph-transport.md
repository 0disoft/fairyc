# ADR 0532: Complete-bound Fairy graph framing and requests

- Status: Accepted for explicit private Windows graph transport preparation
- Date: 2026-11-07

`module_state_graph_transport::prepare` consumes the actual Core Complete.
Prepared retains that source/program owner, an audited native Artifact, copied
framing facts and its own C/program bytes. Ordinary Complete, detached caller
facts and every target except `x86_64-windows-msvc` are refused. Entry emission
and framing have separate caller-owned 16 MiB histories; neither is reset to
hide work already performed by the previous phase.

The exact transport hash domain binds audited entry identity, big-endian
expanded work/literal bounds and the complete pinned C frame. C includes the
layout/body/transport identities, native resource constants, sorted known-cell
codec table and the unchanged real framing kernel. Sticky bounded emission
publishes no partial Prepared. Trap count is obtained from the Artifact's
canonical owned JSON, avoiding a second native slot/trap-plan construction.

`module_state_graph_transport_request` separately creates immutable Input from
Prepared-derived hashes and known cells plus a caller snapshot/policy. A
structural Packet alone cannot forge Input or Prepared. The copied request and
its complete SHA256 remain independent of caller storage. Complete is retained
for the future checked CFG/call-return response admission, not reconstructed
from C or a request receipt.

Windows tests compare complete C, transport identity and request/digest bytes
against current Go, then execute the actual Fairy C with the actual Fairy
request. Go's existing response decoder and independent MIR/storage oracle
remain test drivers. This does not claim Fairy-authored response admission,
compiler/build/run orchestration, persistent CAS or ordinary CLI selection;
those remain required. Evidence:
`docs/development/candidate-state-graph-transport-20261107.md`.
