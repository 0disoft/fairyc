# Source-authenticated state-graph Complete publication, 2026-11-07

The explicit graph mode now uses the common `project_compiler_core::Complete`
publication boundary. It owns the real verified state graph, exact original
source, selected package coordinate, portable body identity and both public/source
interface records. Ordinary factories retain their signatures and mode. No new
command, cache hit, native execution or persistence result is implied.

The graph family remains the existing one-module/one-root/up-to-32-helper source
profile, seven codecs, 64 blocks per graph, 1,024 combined operations and 64 KiB
source ceiling. Every body is independently graph-verified. Original private
void functions admit no public, generic or inline declarations; their public
projection therefore contains the actual normalized module and zero exported
items, templates or specializations. Inline hash is the existing empty value.
These are checked source facts, not defaults for wider source families.

`project_compiler_state_graph` composes cold source or exact current-source
restoration with fresh coordinate/body/interface derivation. `public_interfaces`
returns the same records API in both Complete modes. `project_cache_body_identity`
returns the exact transaction-bearing identity through its existing signature;
wrong owners fail. All getters borrow immutable owners and constructors remain
private. The ordinary program getter requires ordinary mode, and existing
consuming entry points check that mode before reaching it.

## Windows evidence

Final parent command after checking body identity from the independently decoded
program as well as the cold/source-restored owners:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphComplete(SealedOwner)?$'
```

PASS **71.717s**: Complete **36.33s**, seven sealed-owner refusals **35.23s**.
Complete uses 71 cases per profile / **142 native processes**. Expected
preimages, body digests, public-interface bytes/hashes and source-interface
hashes come from actual Go source checking and production identity APIs. Cold,
source-restored and independently decoded programs produce the same canonical
body identity. Cold/restored Complete and copied body identity enforce exact/
one-short work, overdrawn meters, wrong owners, source comment drift and caller-
buffer destruction. Invalid sources/coordinates emit only their error status.
Locals, all codecs, borrows, helper ordering/effects, 32-helper and 64-block bounds
retain their source oracle controls. Source identity's earlier 92 native cases
and seven identity-owner refusals are documented separately in the prerequisite
package receipt; they are not added to this final command's coverage.

Final expanded guard/owner command:

```powershell
go test -mod=readonly -count=1 -timeout 5m -v ./internal/cli -run '^TestCandidateStateGraphComplete(Guards|SealedOwner)$'
```

PASS **95.011s**: four native processes (ordinary consumers and interpreter,
low/optimized) check six C entry points, three ordinary cache entry points and
three Atlas admission gates, plus the interpreter's exit/diagnostic. That command also passed six sealed-owner
refusals. They are a subset of the final seven above and are not added twice
to the coverage totals. The final owner suite additionally checks the immutable
verified-program getter.
Guard preparation uses a real checked/locked workspace, not forged loader facts.
The worker's earlier narrower guard run passed and is superseded by this final
run; neither it nor repeated ordinary runs is added again to coverage totals.

Existing `TestCandidateOwnerQualifiedComplete` passes unchanged assertions in
**110.162s**, both profiles, with independently checked source acceptance and
ordinary Complete output. This is representative ordinary cold publication,
with aggregate unit-cache publication additionally checked by
`TestCandidateGenericAggregateUnitCache`: **344.008s**, low **198.17s** and
optimized **133.91s**. Both profiles retain exact cold/warm reuse, invalidation,
corruption repair and failed-Complete cache preservation assertions. Ordinary
unit cache publication now occurs only after common Complete construction
succeeds; a constructor failure cannot publish an earlier unit prefix. This is
not a full aggregate/whole-project cache-restoration regression. The unchanged
seven-codec interface prerequisite passed 42 native cases in the previous commit; its
original-layout heavy replay remains explicitly gated.

Fresh current Go bootstrap complete-example `check` and `check --locked` pass
**718 source files / 8,975 MIR functions**, empty stderr. All **1,188 `.fy` files**
match the checked isolated copy; only its successful lock is copied back.
Records: `.cache/state-complete-committed-check-f6a5f754/check-results.json`.
Scoped vet, gofmt, UTF-8/LF/final-newline and diff checks pass. Primary logs are
`.cache/candidate-state-graph-complete-decoded-final-20261107.log`,
`candidate-state-graph-complete-guards-final-20261107.log`,
`candidate-state-graph-ordinary-regression-20261107.log` and
`candidate-state-graph-cache-finalization-20261107.log`. Ignored outputs record
execution, not source authority.

## Failed preparation and remaining work

The initial complete-example check failed E4501 because a new getter shadowed
its imported source alias. Renaming the alias fixed it. The next failed E1104
because a short-circuit expression conditionally moved two independent resource
arguments; two sequential checked moves fixed the ownership paths. These failed
commands are not passes. The first native Complete run then passed in 64.065s;
the final command above supersedes it after the entry-scan metering correction.

Native Linux stays batched. Ordinary workspace/key cache publication, Atlas
package production and graph execution/persistence are intentionally still
closed; their future graph adapters need actual source/workspace binding and
native lowering. Direct Atlas package, ordinary cache key/envelope execution,
allocator-failure injection, the full Go suite, complete self-build, distribution
and same-commit hosted acceptance were not run here. The full implementation
queue remains active. Public CLI/SDK/runtime ABI, DB, dependencies, product
limits, repository policy and runners are unchanged; Complete's private storage
and explicit compiler API are expanded. No push, dispatch, release or deployment.
