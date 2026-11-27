# Bounded Atlas impact, MCP and structural edit v1

Owner: ADR 0499. These are separately tested Go Stage 0 experimental commands, not a new map
schema, full semantic diff, arbitrary refactoring engine or Fairy-authored candidate promotion.

## Impact

`fairyc experimental atlas-impact --package-uid <uid> --symbol <visible-or-full-id>
--max-packages <n> --max-calls <n> --max-depth <0..64> --max-symbols <1..4096>
--max-output-bytes <1..4194304> [--locked]` always verifies an existing `fairy.lock`.
Run ordinary `fairyc check` first. The query never updates that lock or fetches dependencies.

Rebuild logical v3 maps from verified current source, derive independent load expectations, and
join all root-reachable package partitions before traversal. Missing/stale/incompatible maps or
an incomplete join fail; no old cached map is silently substituted. Ambiguous feature-instance
UIDs reject. Package/call construction ceilings are explicit and at most 65,536.

The JSON `fairyc.atlas-impact/v1` lists the subject at depth 0, deduplicated reverse-call reachable
symbols, deterministic depths, and exact-current package contexts. The Go and Fairy hosts
cover all checked implementations of the selected declaration. Subsequent reverse-call steps
follow matching concrete implementation IDs. A declaration reached through multiple implementations
appears once at its shortest depth. The symbol limit counts distinct declarations, and the call
construction limit bounds the implementation work. The report includes source/compiler/
interface identities and original completeness facts, not source text or host paths. Output is
buffered and size-checked before writing. No partial document is emitted on budget failure.

In the Go and Fairy CLI and MCP hosts, append `#<specialization-id>` to a visible or full
SymbolId to start traversal at one verified implementation, for example
`<symbol-id>#g0_<64-lowercase-hex-digits>`. The implementation must belong to
the selected declaration in the exact-current partition. Saved catalogs from
older source revisions may contain IDs that no longer resolve. The report
includes `subjectSpecialization` for this form. A bare SymbolId continues to
select the declaration as a whole.

`state` is bounded or partial, never exact safe-change evidence. `traversalComplete` only describes
the admitted resolved-call graph. Unknown/external coverage, partial maps and explicit traversal
cutoffs remain visible. Non-call/type/effects dependencies, outside-workspace callers, dynamic
targets and callbacks are not inferred. Adding/removing a declaration does not imply a rename.
The existing interface/effects review remains the separate comparison tool for its documented
ordinary-interface subset; none of its uncertainty becomes an unchanged declaration here.

## Semantic diff

`fairyc experimental atlas-diff --baseline <workspace-directory> --package-uid <uid>
--max-declarations <n> --max-output-bytes <n>` compares the baseline with the current
workspace. Both workspaces need valid, current `fairy.lock` files. A relative
baseline is resolved from the current workspace. This Go Stage 0 command reads
the two workspaces without updating locks, fetching dependencies, or writing maps.

The command selects exactly one reachable package instance by UID in each
workspace. It compares v3 partitions with the same source-independent package
coordinate, language, compiler and target context. Changes sort by module, kind
and name and show additions, removals and modifications. Modification categories
are declaration shape, observed call sequence, and observations such as trap
categories, borrow kinds and counts of unsafe, move, drop, allocation and
error-return sites. Call targets use package coordinates to ignore source-only
instance changes. Byte offsets and line maps are evidence metadata and do not
affect declaration or call equality.

Ordinary function bodies and nonempty resource cleanups also compare their
verified executable MIR graphs.
The comparison normalizes block, local and diagnostic entity identities, and
uses stable function and foreign declaration identities for calls. A changed
literal or operation can therefore report `executable-body` even when the
declaration and call sequence match. Removing or adding a nonempty cleanup on a
matched resource reports `executable-body` on the resource row. If a stable body projection is unavailable,
the report is partial and records `executable-body-comparison-incomplete`.
Neither executable graphs nor their fingerprints appear in the output.

Concrete generic implementations are matched by original declaration and
ordered scalar type arguments. Their rows include `typeArguments` and sort
after the corresponding declaration row. Changes include added and removed
instantiations, concrete aggregate shape, observed calls and resource cleanup,
and the verified MIR for specialized functions and nonempty cleanups. The
comparison normalizes concrete call targets and nominal types to the original
package coordinate, declaration and arguments. A reissued specialization ID
alone does not change the report.

Matched generic functions and resource cleanups also compare the token spelling
of their original bodies, ignoring comments, whitespace, and positions. A token
change appears as `template-body-syntax` on the declaration row, including when
no specialization exists. Verified executable changes still appear on their
concrete implementation rows. Template syntax comparison retains the
`generic-template-bodies` semantic coverage limitation and exposes no body text
or fingerprints. If a required syntax fingerprint is unavailable, the report
records `template-body-syntax-comparison-incomplete`.

Output is `fairyc.atlas-diff/v1` JSON. Its two contexts carry the exact source
identities and completeness ledgers. `bounded` covers the compared facts.
`partial` records incomplete map facts, opaque or unknown calls, or unavailable
body projections. Renames, full behavior equivalence, and source changes that
leave these projections unchanged are outside its scope.
The declaration limit covers both selected partitions together. Either limit
failure leaves stdout empty.

To guard one function in the selected package, add
`--symbol <module::function> --deny-new-effect filesystem`. Use its full
source-qualified name, including an owner segment for a method. The function
must have a unique compiler-declared entry in both snapshots. The guard rejects
a new `filesystem` declaration, including one required by a direct call or
implicit resource cleanup. A pre-existing `filesystem` declaration, source
movement that preserves the qualified name, and changes to other effects or
functions are allowed. It compares the declared effect set, so authority
already declared in the baseline is outside the guard's scope.
A newly denied effect or unavailable target exits with status 2, a diagnostic
on stderr, and empty stdout. A pass emits the usual semantic diff JSON. The two
flags must be used together and require single-package mode.

For the whole root-reachable graph, use
`fairyc experimental atlas-diff --baseline <workspace-directory> --workspace
[--affected] --max-packages <n> --max-declarations <n> --max-output-bytes <n>`.
This mode returns `fairyc.atlas-workspace-diff/v1`. The package and declaration
limits count records across both snapshots, including unchanged packages.
Every package must have an independent current expectation and its direct
dependencies must be present. Comparison pairs packages by source-independent
coordinate, which includes UID, version and features. Package rows sort by
coordinate and carry exact contexts and sorted direct dependency coordinates
for each side. An absent package has a null context and an empty dependency list.
Added and removed packages list their declarations. Matched packages report
changes in the same declaration and executable facts as the single-package
mode, as well as changed direct dependencies. An `unchanged` row reports
equality within that comparison's coverage. Any partial package makes the
whole report partial. A limit failure leaves stdout empty.

`--affected` adds an `affected` array to this report. Each package coordinate appears
at most once, in coordinate order, with `before` and `after` booleans indicating the
snapshot graphs where it directly or transitively consumes a changed package. Modified
packages start the search in both graphs, removed packages in the baseline and added
packages in the current graph. The array is empty when there are no such consumers.
The `packages` rows keep their original actions and the output byte limit also applies
to `affected`. Use this report to review impact. It does not establish which tests can be skipped.

## Fairy reader and impact core

The private Fairy implementation reads canonical v1/v3 envelopes, validates
metadata and rederived SymbolIds, and admits a partition only after comparing an
independent current compiler/source/interface expectation. `project_atlas_current`
does not derive that expectation from the candidate map. `project_atlas_join`
requires one schema, unique instances and a complete direct-dependency closure;
reference UID/coordinate pairs and exact public cross-package targets must agree.
Only resolved calls consume the explicit package/call join ceilings.

`project_atlas_impact` performs the same bounded reverse-call traversal, visible/full
selector normalization, canonical selection and cutoff reporting as the Go core.
It builds transient incoming lists for declarations and concrete implementations
from the admitted edges and visits only the corresponding callers. Lists retain
the join order used by bounded selection. The index shares the query's work
budget and is released with the traversal.
`project_atlas_impact_json` buffers the complete existing impact JSON document and
checks its output ceiling before returning bytes. Shared work bounds apply to the
reader, identity checks, join, traversal and output. No source bodies or host paths
are included, and neither an exact-current map nor a complete traversal proves a
change safe.

The ordinary semantic/current-source reader still rejects generic declarations
and specialization records; it never silently discards them. The explicit
`project_atlas_validate::validate_generics` v3 entry separately checks declaration
binders/identities, canonical implementation IDs, local ownership, scalar argument
bindings, function/aggregate/cleanup compatibility, recursive aggregate substitution,
contained cleanup spans, exact local call implementations and conservative generic
completeness. It retains separate specialization records and counts all body
relations. Cross-package implementation existence is checked by the complete workspace
join. Semantic consistency is not current-source/compiler authorship.
V1 refuses generic fields and v2 remains unsupported. `project_atlas_current::read_generics` requires an independently derived v3
current expectation, including the exact generic source-interface hash. The shared
join preserves source/target implementation IDs, cleanup ownership and call spans,
resolves the remote ID against its exact original declaration, and retains template
uncertainty in impact output. Ordinary `read` remains closed to generic maps.
Actual Fairy generic source production and command integration use the
Complete-bound paths below rather than obtaining authority from these ports.
Original generic declaration headers now project retained AST binders, types,
coordinates and spans. `prepare_declarations` is source-only; the normal `prepare`
continues to require exact Complete provenance. Generic function headers pass the
complete Fairy factory. User generic aggregates now have a separate private
Complete instantiation/layout/cleanup, restoration/cache and Atlas producer path.
The actual impact command now selects that factory from exact current retained
source through `project_atlas_source_mode`; map/cache headers cannot upgrade the
mode. Ordinary/function-only source keeps its original path. Source-only aggregate
headers do not establish executable or CLI support.
See [the generic receipt](../ops/windows-candidate-atlas-generics-20261105.md).
The Fairy CLI now dispatches `experimental atlas-impact`
with the selectors and explicit limits above. Both with and without `--locked`,
it freshly resolves the workspace and requires an existing current lock, completes
Fairy semantic/body/Flow/IR and interface verification, hashes its own executable,
and derives source/compiler/interface expectations independently of the produced map.
It then encodes fresh v3 partitions in memory and re-enters the same reader/current
checks, join and traversal. No persisted Atlas map, cached header, Go child or build
companion supplies current facts. Neither locks nor caches are written by the query.

The command admits root-reachable local packages with ordinary functions, C
foreign functions, structs, resources and enums. Declaration signatures preserve
checked builtin/C scalar, nominal, borrow/slice, pointer, fixed-array and
Option/Result types. Representation, declared effects, cleanup effects and C link
names come from the complete verified source catalog, including unused foreign
declarations. Constants and static assertions are checked but are not Atlas
declarations. Module and file tables retain their independently canonical order.
Direct local/imported calls, recursion and cycles are collected from verified IR;
same-package foreign calls are external/opaque with their exact C ABI/link name.
The seven ordinary authenticated SDK modules (alloc, bytes, fs, io, num,
process and text) may participate in source production and current queries.
Authenticated scalar SDK Vec declarations now participate in impact and read-only
MCP queries. The existing single-operation interface family in ADR 0495 now has
Windows production, impact-command and read-only MCP evidence: original generic headers,
exact scalar implementations and ordinary provider call targets are retained.
The existing bounded copied-local i32 closure family now has a Complete-bound
source projection. Private helper declarations preserve written closure spans,
copied capture/parameter types and exact checked call endpoints. Ordinary owners
precede g0-sorted concrete generic owners in module-wide helper numbering;
unused generic templates do not acquire helpers. One checked pure helper body
may supply distinct private Atlas identities for its concrete owners without
changing executable IR. See the [closure source and acceptance record](../development/candidate-atlas-copied-closures-20261105.md).
Complete-bound source production now also preserves nongeneric struct/resource
associated functions and shared, exclusive or consuming methods. Written qualified
names, nominal owner coordinates, receiver parameters and exact call endpoints
remain distinct through canonical v3 maps and current-source admission; typed,
alias/grouped and imported receiver sources have independent Windows evidence.
Actual impact, read-only MCP and ordinary root-body edits have Windows command
evidence; the changed full-image gates are tracked in the
[owner-qualified source and acceptance record](../development/candidate-atlas-owner-qualified-20261105.md).
Enum/generic owners, broader interfaces and first-class closures remain separate.
Every package must contain an admitted Atlas
declaration. These candidate limits do not narrow the broader Go command or
language contracts.

The private source-production/current/join APIs now also retain authenticated
SDK scalar Vec templates, concrete fields, cleanup implementations and source
call endpoints. Checked cleanup intrinsic observations retain inlined
`take_slot<T>` implementations without inventing callable frames. The
[SDK vector plan](../development/candidate-atlas-sdk-vector-20261105.md) and
[Windows receipt](../ops/windows-candidate-atlas-sdk-vector-20261105.md) record
producer and impact/MCP command acceptance. SDK Vec root-body preview/apply also
passes the Windows command leaf, including new bool element specializations.

The compiler-private mixed producer now also retains scalar SDK Vec and scalar
user struct/resource templates in the same Complete source graph. SDK
constructors keep authenticated kind-13 storage; a client struct named Vec keeps
its ordinary aggregate identity. Five native producer/current/join families pass
the independent Go v3 oracle and bounded-failure controls in both Windows
profiles. The actual Windows impact command also accepts a source combining SDK
Vec, a same-name client Vec and a client resource with nonempty cleanup. SDK
visible and client full selectors, exact/one-short call admission, package
cutoffs and stale/refreshed contexts match Go with no workspace writes. See the
[impact receipt](../ops/windows-atlas-sdk-aggregate-impact-20261105.md).
The corresponding read-only Windows MCP session also accepts that mixed source,
preserving exact Go contexts, stale-source error and same-session recovery,
protocol-only stdout and workspace immutability. See the
[MCP receipt](../ops/windows-atlas-sdk-aggregate-mcp-20261105.md).
Mixed root-body editing also passes the Windows command leaf as described below.
The refreshed optimized authored-main replay passes all twenty current groups;
low replay remains separate and is tracked in the
[full-CLI receipt](../ops/windows-atlas-full-cli-replay-20261105.md);
see the [composition plan](../development/candidate-atlas-sdk-aggregates-20261105.md)
and [producer receipt](../ops/windows-candidate-atlas-sdk-aggregates-20261105.md).

Call reachability is exact within the admitted resolved graph. Foreign calls make
the `calls` completeness partial with reason `external-ffi`; only resolved calls
consume `--max-calls`. Non-call summaries are not
collected by this producer: its completeness/header omissions explicitly use
`candidate-atlas-not-collected`. Generic template bodies retain the existing
unavailable reason. Output remains partial, carries the independent current
identities, and never implies exact safe-change evidence. The producer does not
claim byte parity with Go's richer completeness coverage. Common declaration and
type ports are now connected to the producer and current-map reader, not only to
private prebuilt-map probes.

Syntax/selector/budget failures return 2, workspace/lock/source or unsupported
candidate-source failures 7, and preparation/output I/O failures 4. Help succeeds
without reading a workspace. Windows Fairy read-only MCP uses the same current
query producer. Structural editing retains its own ordinary-source selection
restrictions below. SDK/user-aggregate combinations and Linux command wiring
remain separate. Windows native evidence is recorded in
[the source-family receipt](../ops/windows-candidate-atlas-families-20261105.md)
and [the original Fairy receipt](../ops/windows-candidate-atlas-impact-20261104.md).

SDK queries retain the 16 MiB source/projection work allowance and use one
separate 16 MiB allowance for current-map admission, the complete join, impact
exploration and rendering. That second allowance is shared across all packages;
it is never reset per map, symbol or request component. Its larger combined
allowance explicitly covers re-reading the SDK's full declaration/call inventory.
Workspaces without SDK retain their existing single shared 16 MiB allowance.
Existing source, package, call, traversal and output ceilings still apply.
Compile/link work has its existing independent bounds. This is compiler-private
SDK admission policy; no CLI flag, wire format or default build timeout changes.

## Read-only MCP over stdio

The private Fairy `project_mcp_protocol` core now implements the same bounded
JSON-RPC lifecycle, explicit query selection and response envelopes. Its pure
`step` returns either a response, no response for a notification, or an owned
pending selector; `finish` accepts one validated host document or an unavailable
result. It opens no workspace and owns no stdin/stdout. The protocol-only Windows
oracle is recorded in [the candidate MCP receipt](../ops/windows-candidate-mcp-20261105.md).
The Fairy command now connects explicit invocation policy, bounded streaming
stdin/stdout and a fresh locked-source query for each tool call. Windows focused
command and full CLI acceptance pass in both profiles, including stale-source
refusal/recovery and output failure. The Go host retains the prepared-session
optimization described below; the Fairy host re-verifies the source on each call.
The same authenticated scalar SDK Vec query admission applies to MCP. Broader
source-family combinations, plus native Linux dispatch/execution, remain separate.

```sh
fairyc experimental atlas-mcp --allow-impact --max-packages 32 --max-calls 8192 --max-depth 8 --max-symbols 256 --max-output-bytes 262144 --max-requests 100
```

Launch from the selected workspace with a current lock. The invocation grants the capability and
every limit. The client cannot widen the root, change budgets or enable another tool. Only
`atlas_impact` is advertised, with exact `packageUid` and visible/full `symbol` strings. Results
contain structured evidence and fixed explanatory text, not authored source/doc bodies or host paths.

The MCP 2025-11-25 stdio profile implements newline-delimited JSON-RPC, initialize/version
negotiation, initialized notification, ping, tools/list and tools/call. An unsupported client
version receives the supported server version; the client decides whether to continue. Tool calls
require the ready phase. There is no shell, network listener, sampling, prompts, client-root
request, credentials, tasks or write tool. Metadata cannot grant authority.

Limits: 1-1,024 messages (notifications count), 64 KiB/JSON frame payload (excluding LF or CRLF),
nesting 16, 512 encoded bytes/ID, and 2 KiB-4 MiB/response payload. Duplicate keys, batches and
invalid framing/IDs reject. Stdout contains only protocol frames. EOF ends the session. Every query revalidates locked current
source; unavailable/stale input becomes `isError` without raw diagnostics. Session policy failures
return 2, output failure 4; tool failure can leave the session usable.

The output bound also applies to fallback errors. If HTML escaping expands an admitted
request ID so that even the fallback cannot fit, the bounded `-32603` response uses
`id:null`. An ID that fits is preserved; no over-limit response is emitted.

The Go MCP host retains one prepared workspace and one bounded reverse-call index
within the sequential session. Every call compares all local catalog manifests,
source names and exact bytes, and lock presence/bytes against the retained inputs;
it also verifies that the current lock still matches that workspace. Bundled SDK
sources and compiler identity belong to the same running executable. File metadata
alone cannot enable reuse. A changed or unreadable input discards the preparation
and follows ordinary locked preparation, including its failures. Package/symbol
selectors, join limits, traversal limits and output limits remain per-query checks;
no cached response bypasses them. Nothing is retained after the session ends.

References: [stdio](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports),
[lifecycle](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle),
[tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools).
This is the named profile, not every optional MCP feature or a client-product certification.

## Single-function structural body edits

The private Fairy `project_edit_body` and `project_edit_preview` owners now supply
AST-bounded byte replacement and fresh current-source preview for the candidate's
existing ordinary-source Atlas subset. The complete edited overlay passes the
generic-aware semantic/Flow/MIR/link/interface factory before a verified report
is constructed. Windows low/optimized reports and rejection controls match the Go
owner; see [the body-edit receipt](../ops/windows-candidate-edit-20261105.md).
The Windows Fairy body-file command is connected through retained-root reads
and validates outside-production-root paths. Its application owner acquires the
exclusive edit lock, revalidates and stages before one opened-directory rename.
Focused command-leaf and full candidate acceptance are separate evidence. The
Go reader also enforces this existing body-path restriction.
Both Windows full candidate profiles now pass impact/MCP/edit and post-edit lock
recovery; see [the current edit receipt](../ops/windows-candidate-edit-20261105.md).
Each Fairy preview now owns the exact manifest/production-source/lock fingerprint.
Its fresh recheck rebuilds the complete edited overlay from the owned replacement
and compares the fingerprint, path and complete before/after bytes. This supplies
application preconditions; the separate `project_edit_apply` owner acquires the
lock and publishes the staged source. The MCP owner never calls that entry.

```sh
fairyc experimental atlas-edit replace-body --package-uid <root-uid> --symbol <symbol-id> --body-file edits/body.txt --expected-source-sha256 <raw-file-hash>
# Add --apply to authorize the current validated replacement.
```

The body file holds one complete braced body, at most 64 KiB, in a portable regular workspace
file outside production roots. Exact current IDs select root-owned, private, safe, non-entry
functions without owners or attributes. Both the Go command and Fairy candidate accept a generic
function's logical-v3 declaration ID when the locked workspace has at least one checked concrete
specialization. Preview and apply require the edited workspace to verify every previously observed
type binding for the selected template. The candidate joins the selected declaration's verified
frames to the corresponding source plan and compares the ordered argument keys across the edit.
Coverage is limited to the concrete bindings observed in the current workspace. A Go batch that
removes the last use of any required binding fails.

`fairyc experimental atlas-edit preview-batch --package-uid <root-uid> --edit-file edits/batch.json`
checks 1 to 16 replacements together. The request is one JSON object with an
`edits` array. Each entry supplies `symbol`, `bodyFile`, and
`expectedSourceSha256`. The request and body files use the same portable,
bounded, regular-file input rules as single-body edits. All selectors and
hashes refer to the same locked source snapshot.

Batches may target several distinct functions in the same file or in
different root source files. Duplicate or overlapping bodies fail. Each
changed file must parse with every replacement at its expected body extent.
The complete workspace is then verified with all replacements in one overlay.
The bounded `fairyc.structural-body-edit-batch/v1` report lists entries in
request order, with original byte ranges and before/after whole-file hashes.
Entries for the same file share the combined after hash.

`fairyc experimental atlas-edit apply-batch --package-uid <root-uid> --edit-file edits/batch.json`
validates the request again and acquires the same exclusive workspace edit lock
as single-file apply. It stages and syncs the original and replacement bytes for
each file, records their paths and SHA-256 values in a recovery journal, and
rechecks the workspace. It publishes files in path order using one rename per
file. Each rename verifies the staged replacement and the current original
against their expected identities and bytes. Successful reports mark every
entry as applied. The production graph lock still requires an ordinary check
after applying changes.

On a publication failure, the command attempts to restore earlier files in
reverse order. Each restore checks that the destination still contains the
published replacement and that the original backup passes verification. If
restoration fails, the error identifies the journal in `.fairyc-edit.lock`
and preserves the remaining original backups. After interruption, inspect the
journal and compare current file hashes before manually restoring a backup or
removing the edit lock. Application uses separate file renames. Readers can
observe intermediate states, and process or power loss can require manual
recovery.

The Fairy editor now accepts local workspaces containing generic struct/resource
templates and concrete aggregate function signatures. Its selector derives all
root headers through the verified Complete-bound Atlas schema. Original and
replacement overlays independently select the aggregate factory from retained
source, including newly requested scalar specializations in a replacement body.
This does not relax editable-target restrictions or publication/freshness rules.
Windows command-leaf evidence is in
`docs/ops/windows-candidate-atlas-aggregates-20261105.md`. The ordinary SDK
user-body-edit command leaf is tracked in the SDK receipt. Authenticated scalar
SDK Vec root-body editing also passes exact Go preview/apply comparisons and
fresh bool specialization controls in the SDK vector receipt. Mixed SDK/client
aggregate body editing also passes the Windows command leaf: both the SDK Vec
and same-name client Vec acquire fresh bool instances, nonempty client resource
cleanup is retained, and exact Go reports/publication plus refusal/freshness
checks pass. See the [mixed editor receipt](../ops/windows-atlas-sdk-aggregate-edit-20261105.md).
SDK declarations remain noneditable. This is checked source publication, not
execution of the edited client or cache-hit proof. The refreshed optimized
authored-main evidence covers all twenty original groups plus the three added
static-operation groups on the same current image. Complete low authored-CLI and
native Linux acceptance remain separate.

Workspaces containing the existing single-operation interface family also admit
ordinary root-body edits. Windows low/optimized preview/apply matches Go for an
overlay that replaces i64 calls with newly requested bool operation/algorithm
implementations. Dependency declaration writes, invalid return types and stale
lock/source hashes still reject; provider source and the lock remain unchanged.
See the [static-operation receipt](../ops/windows-candidate-atlas-static-operations-20261105.md).
This does not make interface/provider/public generic declarations editable.

Require the exact raw source SHA-256. Replace the original AST body span and preserve all outside
bytes. Parsing must find the same function/body start and exactly the replacement extent; early
closing to inject siblings fails. Recompile the complete production workspace through an immutable
overlay and require verified generic-aware semantic/MIR output. This validates accepted source,
not behavioral equivalence or safe-change proof.
Go overlay validation derives private SDK storage admission from the freshly
loaded workspace through `workspacecompiler.CheckGeneric`. Only the exact SDK
bundle's package owns those intrinsics; importing Vec grants no client authority.

Preview returns bounded `fairyc.structural-body-edit/v1`: portable path, module/function, original
body byte range, before/after raw source hashes and verified/applied states. It writes no source or
lock. Reports over 8 KiB fail before application. Apply prepares the current body file independently,
not a serialized old plan; keep that input fixed between review and application.

Apply acquires `.fairyc-edit.lock`, repeats local manifest/production-source/graph-lock fingerprints
and candidate validation, writes/syncs a same-directory temporary, rechecks the closure and original
file, then performs one opened-root rename. No delete-and-retry fallback is used. Ordinary staging
errors clean temporaries/locks. Crash leftovers require inspection; hostile concurrent writers and
physical-media durability are not guaranteed. Output or lock-cleanup errors can follow publication;
inspect hashes before retrying. Keep `fairy.lock` unchanged and run ordinary `check` before locked
build/run/test. Language/runtime/SDK/map schemas do not change. MCP never exposes apply.
Neither write authority nor certainty follows from an impact result.

Go application records the exclusive temporary file's opened identity. Immediately
before rename it reopens a regular file, checks that identity and exact length,
then streams SHA-256 with a 32 KiB buffer against the verified overlay bytes.
Same-size modifications and replaced temporaries reject without changing the
original source. Failed staging cleanup removes only the observed owned file;
successful rename releases ownership of the old temporary name. `Apply()` keeps
its existing signature; `ApplyWithOptions` optionally supplies a context and
temporary-creation/write/chmod/sync/close/rename operations with host defaults.
The default context is Background. Cancellation propagates through workspace
loading and checking and is observed around staging I/O. Cancellation after
rename can return an error with `Applied=true`, preserving the committed source.
These observations do not atomically exclude hostile writers or interrupt a
blocked filesystem call. The Fairy-authored edit path is unchanged.
Each fingerprint attempt shares directory spelling listings only within that
attempt. Fresh attempts reread the directories; every selected path segment is
still checked for symlinks and every opened input for regular-file bounds and
identity. Enumeration and retained names share the loader's 100,000-entry
workspace ceiling. Failed reads or closes never publish a cached listing, and
the fingerprint bytes and apply/lock checks are unchanged.
All tools use explicit package/symbol selectors and treat attributed text or client strings as
untrusted data, never instructions or authority to run a command. A tool result cannot grant
write permission or widen the root selected by the process invocation.
