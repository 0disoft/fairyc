# Whole-package cache key v5

- Status: Experimental executable contract
- Decision: [ADR 0493](../adr/0493-cross-package-inline-bodies-and-cache-v5.md)
- Implementation: internal/packagecache, internal/inlinebody, internal/workspacecompiler
- Fixture: spec/fixtures/package-cache/manifest-v8.json (unchanged fixture JSON format)
- Historical contract: [v4](whole-package-cache-v4.md)

V5 preserves the v4 tuple and exact contract-state body handling. Its domain version byte
is 5 and the inline class encoding `fairyc.inline-bodies/v1` follows that byte. The owner's
32-byte inline class follows CurrentBody; each dependency's 32-byte inline class follows
its coordinate/interface pair. Zero means no inline exports. All dependency coordinates
remain sorted and unique. Other field order, validation and caller bounds remain intact.

The text form is `fycache5-sha256-` plus the full 64 lowercase hexadecimal digest.
ParseKey rejects v4 and earlier prefixes. Old disk records need no migration and cannot
be selected as v5 hits. Record/artifact checks and bounded retention continue unchanged.

Only explicit bounded @inline exports contribute to the independent inline class.
Changing a published inline body, adding/removing its marker, renaming the exported
function or changing its signature invalidates the owning package and its direct
whole-package dependents. Ordinary public/private function and cleanup body edits do
not enter that class. Normal source and contract-state members still invalidate owners.
Public-interface v5 remains body-independent. No selective per-call import list is claimed.

Ordinary process/disk and generic process/disk paths bind the same inline class into
their base inputs. Reassembled inline markers must agree with freshly checked HIR before
reuse is reported. A mismatch falls back to verified lowering. For disk caches, the
observed generations of the rejected reassembly are quarantined by exact record bytes
before repaired artifacts are published. Other concurrently published generations are
preserved; stale and repaired immutable payloads cannot cause a second conflict miss.
Stable canonical body
encoding retains its existing envelope and adds an optional omitted-when-empty inline
identity; old strict readers fail closed on a new marked body. Compiler build identity
and v5 keys isolate incompatible artifacts.

EncodeLegacyV4 is for frozen compatibility evidence only and rejects inline facts. The
original v4 golden digest remains tested; new v5 baselines are recorded independently.
No automatic cache enablement, shared remote cache, executable/native object cache or
trust boundary expansion is introduced.

## Fairy candidate encoder parity, 2026-11-02

The authored `project_cache_key.fy` now encodes the complete v5 tuple and accepts
only v5 key text. Its explicit owner/dependency inline members allow zero classes
and nonzero classes, exactly like the Go encoder; the generic envelope includes
the same base bytes. `TestCandidateCacheKeyV5Parity` compares all canonical bytes
and digest text against independent Go oracles for 24 ordinary and 11 generic
cases, including separate owner/dependency inline mutations. Both Windows native
profiles pass, together with malformed inputs, exact work budgets and v4 text
rejection. The old Go `EncodeLegacyV4` remains frozen; no legacy Fairy cache key
is used to authorize v5 reuse.

The initial encoder/no-inline baseline is now extended by
[candidate inline v1](../architecture/candidate-inline-v1.md). The complete
source factory admits the bounded public i32 arithmetic family, verifies the
detached marker against current source during restoration, and derives the
actual owner/dependency inline classes. The five-graph source-key comparison
includes inline body edits and equivalent source spelling at both Windows
profiles; [the receipt](../ops/windows-candidate-inline-20261102.md) also records
shared-header C execution and missing/extra-marker rejection. Zero remains the
exact class for packages without inline exports. The private candidate IR cache
revision advances to `/1`; public-interface v5 and cache-key v5 stay unchanged.
Linux native execution remains in the combined queue. No cache default or
target promotion changes.
