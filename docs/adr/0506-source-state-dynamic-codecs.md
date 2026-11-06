# ADR 0506: Explicit source state bytes and text codecs

- Status: Accepted for the explicit experimental source-state graph path
- Date: 2026-11-07
- Extends ADRs 0503, 0504 and 0505; ordinary source admission is unchanged.

The explicit state-graph compiler admits `bytes` and `text` in state cells and
map key/value positions, alongside the five existing source codecs. It uses
the existing `contractstate.Bytes` and `contractstate.Text` identities and
encoding; this adds no storage format or runtime codec. Ordinary semantic
analysis and the original single-cell source path retain their five-codec
vocabulary. An explicit semantic entry point carries the opt-in through both
layout checking and the helper call-only projection. It grants no Complete,
artifact, cache, ordinary CLI or target-support authority.

## Source and runtime rules

`bytes` literals require byte strings, while `text` literals require valid
UTF-8 strings. Neither scalar literals nor the other string kind convert
implicitly. Typed read aliases, helper storage actions, dynamic map keys and
copied root borrows retain the exact codec identity. Source ASTs stay detached.
Bytes may include NUL, 0xff, CRLF and 0x1a. Empty present values and absent
values have the same zero-length encoding and different presence bits.

Text preserves exact valid UTF-8, including NUL, without Unicode normalization.
Canonically equivalent Unicode spellings therefore remain distinct map keys.
View offsets and lengths count bytes, not characters. A copied text fragment
keeps the text codec and must be valid UTF-8 when used as a write or map key;
splitting a multibyte scalar fails without publishing any earlier writes.
Existing zero-length-view, ownership, scope and mutation restrictions remain.

Existing policy limits apply to encoded byte lengths, including the explicit
developer command's 1,024-byte map keys and 64KiB values. A value exactly at a
limit is accepted; one byte over fails. No default limit, retry or truncation
is added. Failed execution performs one snapshot and no CAS and leaves the
durable state unchanged. Commit performs at most one existing layout/version
CAS; reject and trap do not publish.

## Evidence and continuation

`state_experimental_test.go` checks isolated repeated ordinary/explicit
analysis, retained schema diagnostics, cancellation and immutable source.
`state_graph_dynamic_test.go` checks exact source bytes, presence, helper and
nested borrow copies, UTF-8 fragments, typed refusals and distinct text keys.
The authored example combines both codecs, binary/text map keys, helper
writes, branch control and copying before replacement, with independent
expectations of seven reads, six final writes and cost 25.

`graph_source_dynamic_windows_test.go` runs actual generated C with Windows
MSVC low/optimized and independent Go state-MIR comparisons. Its 54 attempts
cover commit/reject/trap, close/reopen/repeat, absent/present-empty, binary
keys/values, UTF-8 views and fragments, exact Unicode key identity, and
1,023/1,024/1,025-byte keys plus 65,535/65,536/65,537-byte values for both
dynamic codecs. Every attempt checks snapshot/CAS counts, retained unrelated
state and reopened reference equality. The developer command additionally
checks init/execute/snapshot for this authored example under both profiles.
Commands and actual results are recorded in
[source-state-graph-v1](../architecture/source-state-graph-v1.md).

Complete/current-source provenance, artifact/cache invalidation, ordinary
CLI selection, native Linux and all final acceptance gates remain required.
No stable language/CLI/SDK/runtime ABI, dependency, DB/provider, timeout,
repository-policy or CI/runner change accompanies this private extension.
