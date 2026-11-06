# ADR 0516: Match Go portable state graph body identity

- Status: Accepted for detached single-transaction graph identity
- Date: 2026-11-07
- Uses the existing `fairyc.contract-state-package-body/v1` contract.

The Fairy graph identity factory emits the exact existing canonical preimage
and `fycsbody1-sha256-` digest for one verified root and its helper closure. It
preserves Go's graph-v1/helpers-v1/borrows-v1 selection, storage types, literal
contents, typed edges, exits, helper names and borrow/view facts. Capability
numbers, layout arena indexes and literal offsets remain producer-local and
are excluded, matching Go's stable representation.

A separate opaque Context owns and validates the raw package coordinate and
binary stable function descriptor, including canonical type grammar, ordered
effects and complete consumption. The root descriptor effects must match the
verified graph. Identity owns both its preimage and digest; neither resource
has an external constructor or mutable accessor. Parsing, serialization and
hashing retain the existing 16 MiB ledger and private 1001/1002/1003 errors.

These are canonical identity facts, not proof that current source owns a
supplied signature or body. No Complete, cache hit, execution authority or
transaction/persistence handle is produced. Zero/multiple transactions and
current package/source association remain separate required integration work.
Stable type grammar coverage does not expand executable language signatures.

Independent Go preimage/hash checks pass 65 HIR bodies, 33 additional valid
descriptor shapes and 29 invalid contexts in each Windows profile: 254 native
processes and four ownership refusals. Exact/one-short budgets, portable-ID
stability and semantic hash changes are covered. Full details are in
[the identity receipt](../development/candidate-state-graph-body-hash-20261107.md).
Linux and allocator failure injection remain unexecuted. Stable CLI/SDK/runtime
ABI, DB/provider, dependencies, ordinary budgets, repository rules and runners
are unchanged. No push or dispatch accompanies this slice.

Current-source/package association is now supplied by ADR 0518's source-derived
identity adapter and ADR 0520's explicit Complete graph factory. The detached
encoding API itself still grants no cache or execution authority.
