# Windows Fairy Atlas reader and impact, 2026-11-04

## Transport slice

`project_atlas_wire` reads the existing canonical `.fymap/v1` and `.fymap/v3`
JSON envelopes without a new transport schema. This first private slice rejects
specialization/type-parameter records rather than dropping them. The optional
ordinary v3 generic-interface header pair is retained. It validates exact field
names/order/types, canonical Go JSON spelling, no whitespace/trailing values,
duplicate decoded keys, declared unsigned integer forms, the version pair and
SHA-256 over the domain-separated length-prefixed canonical map document.

The returned `Document` is a **transport-only** view. Semantic validity and
current-source authority require the next reader layer. A checksum is neither
source freshness nor publisher authentication. Input is at most 4 MiB, JSON
structure depth at most 160, arrays at most 65,536 entries, with the existing
shared 16,777,216-unit work meter. Failure returns no usable document.

Executed: `go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlasWire$'`.
Windows MSVC low/optimized pass (8.506 seconds overall). Four real Go-generated
partition envelopes (two source packages, v1 and v3) are admitted. Sixteen
whitespace/trailing/truncation/duplicate/unknown/checksum/version/escape mutations
and a tiny byte-budget rejection execute in each profile. Go supplies fixture
bytes only; the native Fairy process owns decoding, canonical checks and hashing.

Linux execution is deferred. No CLI/MCP/edit dispatch, map production, source
freshness claim, public API/SDK ABI, dependency, DB, repository or runner setting
change, push or hosted CI accompanies this first slice.

## Semantic and current-source gates

`project_atlas_model`, `project_atlas_symbols` and `project_atlas_validate` check
header identity algorithms/nonzero digests, exact namespace references, canonical
file/module/declaration/symbol/function tables, spans, type forms, declared record
counts, call provenance/targets and completeness/omission consistency. SymbolId
signatures and full/visible hashes are rederived in Fairy, including owners,
parameters, effects, fields and variants. Neither a valid checksum nor a matching
stored visible/full pair can bypass declaration identity verification.

The bounded reader admits v1 and nongeneric v3 records. Generic declarations,
concrete specialization records, nominal type arguments and v2 remain rejected.
Rows are capped at 4,096 (call arrays at 65,536), references at 33, type depth at
64 and metadata strings at 4,096 decoded bytes. Comparisons and signature work
charge the same shared meter; no unbounded identifier search is introduced.

`project_atlas_current::read` additionally requires a detached independent current
expectation with compiler, ordinary/generic interface, package/source/namespace,
features/dependencies/references and every file digest/line-map/kind. This private
argument carries the same facts as Go `LoadExpectation`; it is not a new `.fymap`
version or stable file format. The caller must derive it from verified current
retained source, not copy it out of the map being validated. The gate performs no
filesystem scan, lock update or source compilation. A ready map is evidence only
relative to that caller-supplied current expectation.

Executed: `go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateAtlas(Validation|Current)$'`.
MSVC low/optimized pass, 24.485 seconds overall. Checks use real v1/v3 source maps,
record/enum/foreign-call/resource fixtures, Go-rejected rechecksummed metadata,
counts, file identity, span, declaration SymbolId and provenance mutations.
Current expectations are independently serialized from the verified Go compiler
result; changes to ten identity fields, files, dependencies and unknown fields
reject. These are native Fairy readers, without a Go child or native C parser.
Current-source ingestion and graph joining remain separate from these APIs.
