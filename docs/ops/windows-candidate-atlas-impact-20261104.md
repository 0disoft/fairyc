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
