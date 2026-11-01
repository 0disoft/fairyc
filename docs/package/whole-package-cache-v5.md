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
reuse is reported. A mismatch falls back to verified lowering. Stable canonical body
encoding retains its existing envelope and adds an optional omitted-when-empty inline
identity; old strict readers fail closed on a new marked body. Compiler build identity
and v5 keys isolate incompatible artifacts.

EncodeLegacyV4 is for frozen compatibility evidence only and rejects inline facts. The
original v4 golden digest remains tested; new v5 baselines are recorded independently.
No automatic cache enablement, shared remote cache, executable/native object cache or
trust boundary expansion is introduced.
