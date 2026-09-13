# Private parser: SDK observations, pointers and generic declarations

The four increments preserve syntax only. The SDK sources, Go compiler, language
grammar, fixture manifest and stable compiler/SDK/runtime versions are unchanged.
No generic aggregate semantic admission, specialization, pointer execution or
name resolution is introduced. No push was performed.

## Contract

- Pointer types use private kind 11 (`ptr`) or 12 (`ptr mut`). Their required
  element is `left`, with zero `right`; original half-open byte spans and prefix
  binding distinguish pointers to arrays from arrays of grouped pointers.
- Functions, structs and resources retain generic parameter lists separately from
  type argument applications. Declaration header spans cover `<...>`; kind 13
  stores a name and kind 14 stores a backward ordered link. Neither is a value
  type. Names, order, duplicate spelling and trailing commas are preserved.
- Parameter records share the 4096-node type arena, charging two nodes per name.
  Failed declarations roll back these nodes, ordinary types, fields and other
  arenas together. The 32-level type limit and 32768-byte source limit are unchanged.
- The SDK observation lane reads only three explicitly selected files, at most
  32768 bytes each, without editing or importing their source as candidate modules.
  Exhaustion can use at most 128 original complete top-level prefixes per file to
  identify the first blocker. A partial or exhausted result is not full admission.
- Final retained SDK declarations must be an ordered subsequence of Go CST
  declarations with exact kind, complete span and name/alias span equality. Both
  profiles must produce identical full records. Complete parses additionally
  require whole-list equality. Integrity checks run on incomplete results too.

## SDK observations

Counts below are retained declarations / syntax problems, not executed functions
or semantic conformance. Initial results precede pointer and generic support.

| Unchanged SDK input | Bytes | Initial | Final | First blocker, initial to final |
| --- | ---: | ---: | ---: | --- |
| `optional/std/vec.fy` | 4859 | 1 / 20 | 15 / 6 | `<` at 94..95 to `null` at 418..422 |
| `std/bytes.fy` | 8839 | 16 / 17 | 32 / 1 | `ptr` at 129..132 to `null` at 1362..1366 |
| `std/text.fy` | 7606 | 17 / 21 | 36 / 2 | `ptr` at 84..87 to `null` at 2668..2672 |

All three full outcomes have status 0 with nonzero problems; none is fully admitted.
Pointer support raised the vector count to 8 / 13; function generics raised it to
15 / 6. Aggregate generics then moved its first blocker into the resource body
without changing those counts. This distinction matters: a later failure can still
roll back an otherwise recognized declaration. `null` is only the first remaining
blocker; this run does not claim it explains every remaining problem.

## Null literal increment (2026-09-13)

The candidate now retains `null` as an ordinary literal expression (kind 1),
with keyword token kind 4, zero children and exact half-open byte spans. Both
raw and cooked views preserve `null`, matching the existing scanner contract for
`true` and `false`. No SDK source, Go compiler, normative grammar, stable version,
pointer execution or type-checking behavior changes.

| Unchanged SDK input | Retained declarations | Syntax problems | First remaining blocker |
| --- | ---: | ---: | --- |
| `optional/std/vec.fy` | 20 | 1 | `>` at 1790..1791 in `Vec<T> { ... }` |
| `std/bytes.fy` | 33 | 0 | None |
| `std/text.fy` | 38 | 0 | None |

Both Windows native profiles produce identical records. The complete bytes/text
declaration lists match Go by kind, full span and name span. Vector remains a
partial syntax observation, not full SDK admission; generic record construction
is deferred. Complete syntax does not establish expression-tree equivalence or
runtime/semantic conformance. These files are not yet mandatory-complete gates.

`fairyc_go_test_parser_literal_items` passed in 57.030s, covering keyword spelling,
spans, child absence, pointer initialization/comparison, record fields, calls,
grouping, identifier boundaries and the existing literal/postfix/assertion cases.
Its first run failed because the new test incorrectly expected an empty cooked
keyword value; correcting that expectation to the existing spelling contract
resolved the failure without changing the scanner. `fairyc_go_test_parser_sdk`
passed in 46.595s, including pointers and generic declarations in both profiles.
Full compiler/release suites remain out of scope; the child docs validator is
unconfigured. Only exact-path commit authority changes in the parent runner.

## Generic record increment (2026-09-13)

Generic record construction now uses the existing kind-10 record and field list.
Its constructor child may be a path or kind-20 type application, which retains
the path and ordered type arguments. Read-only lookahead accepts `(` or `{` after
a balanced type-argument list; its 256-significant-token bound is unchanged.
The existing field parser is shared by ordinary and generic records. The tree
verifier rejects literal constructors and non-link type-argument roots.

All three unchanged SDK inputs now parse completely in both native profiles:
`optional/std/vec.fy` has 21 declarations, `std/bytes.fy` 33, and `std/text.fy` 38,
each with zero syntax problems. Full kind/name/span declaration records match Go.
These are still observations, not mandatory-complete SDK regression gates or
proof of generic semantics, executable SDK behavior or expression equivalence.

The current 16 candidate source files also parse completely with exact Go
declaration comparisons in both profiles. `bodies.fy` is 31732 bytes; the 32768-byte
source cap, 4096-node arenas and recursion limits remain unchanged. No module
split, arena expansion, normative grammar, SDK source or stable version change
was needed.

The record tests cover empty/qualified/nested generic construction, pointer and
array arguments, trivia, trailing/duplicate fields, postfix chains, exact node
links and byte spans, verifier mutations, malformed-input rollback and the
existing expression-arena boundary. The old generic-record rejection fixtures
now use malformed initializers. Go checks both accepted and rejected examples.

Initial verification failures exposed two incorrect new test assumptions:
`R<T>>{}` is accepted by Go as comparison/shift syntax, and an unclosed record
may consume the following declaration during recovery. The tests now preserve
that valid expression and check that retained declarations have no leaked type
nodes. Temporary failure-location logging required a console effect annotation;
both logging and annotation were removed before final validation.

`fairyc_go_test_parser_records` passed in 23.234s in both native profiles. The
combined `fairyc_go_test_parser_generic_records` lane passed in 46.536s and covers records, generic
calls, SDK observations and self-source comparisons with a 180-second timeout,
closed stdin, no network and exclusively locked cache writes. The parent command
contract also scopes the exact seven-file implementation commit. Full compiler,
benchmark and release suites remain out of scope; no child docs validator is
configured. Public API, DB and repository hygiene policies are unchanged.

## Required SDK completeness gate (2026-09-13)

The three previously observed complete files are now mandatory regression cases.
`requiredParserSDKFiles` independently fixes their disk/name identities. Before
native execution the inventory rejects missing required entries, duplicate disks,
duplicate source names and remapped required names. File read failures remain
fatal, and each native profile must finish every required case.

Required outcomes must have status 0 and zero syntax problems before any prefix
fallback. Complete outcomes retain the existing exact whole Go declaration-list
comparison and cross-profile equality. A required file cannot regress to a
partial or exhausted observation and still pass. Future observation-only inputs
remain separate from this requirement; no additional SDK files were read in this
increment.

`fairyc_go_test_parser_sdk_gate` passed in 7.618s. Native low/optimized results
remain 21/33/38 declarations and zero problems for vec/bytes/text. Platform-neutral
controls reject omission of each required file, partial/exhausted/truncated
outcomes, duplicate disk/name identities and required mapping changes. Controls
also accept a valid additional observation identity without making its partial
result mandatory-complete. That synthetic identity control is not an SDK coverage
claim. Go formatting and scoped strict checks cover the changed test/contract.

The parser implementation, input/node limits, SDK sources, normative grammar,
public API, DB, stable versions and repository hygiene policies are unchanged.
Only the corpus test, candidate docs and exact verification/commit authority are
updated. Full compiler/release suites are omitted because no compiler behavior
changed; the child documentation validator remains unconfigured. The gate proves
bounded syntax and declaration parity, not executable SDK semantics.

## Allocation and process SDK observations (2026-09-13)

The inventory now also reads unchanged `std/alloc.fy` (1453 bytes) and
`std/process.fy` (1456 bytes) as observation-only data. Both complete successfully:
alloc has 10 declarations and process has 8, with status 0 and zero problems.
Their full kind/name/span declaration records match Go and are identical in both
native profiles. No parser implementation or SDK source changes were necessary.

`fairyc_go_test_parser_sdk_gate` passed in 7.698s, including the five-file native
corpus and existing inventory/outcome rejection controls. Logs now label each
file with `required=true` or `required=false`; vec/bytes/text remain the only
three required-complete cases. Alloc/process promotion is a separate increment,
so their current complete results are not yet mandatory regression guarantees.
Neither `std/io.fy` nor `std/fs.fy` was added to the native corpus.

This is source parsing and declaration comparison, not execution of allocation,
process-argument access, foreign functions or SDK effects. The parser budgets,
normative grammar, public API, DB, stable versions and repository hygiene policies
are unchanged. Candidate README/evidence and exact commit authority are updated.
Go formatting and scoped strict checks cover the test/contract changes; full
compiler/release suites remain unnecessary for this test-only increment and the
child documentation validator remains unconfigured. No push is included.

## Allocation and process promotion (2026-09-13)

Alloc/process are now required-complete cases alongside vec/bytes/text. The
independent required inventory and contract controls cover all five files;
omission, partial syntax, exhaustion and truncated outcomes are explicitly tested
for each newly promoted file. No new SDK sources or parser behavior are introduced.

`fairyc_go_test_parser_sdk_gate` passed in 6.813s. Both native profiles report
`files=5 required=5`, zero problems and exact whole Go declaration records for
21/33/38/10/8 declarations respectively. Required cases fail before prefix fallback
and must complete in each profile. Observation-only support remains available for
future inputs, but no current native input is observation-only.

The corpus test and candidate README/evidence were updated; parent commit authority
is restricted to those exact paths. Formatting and scoped strict checks accompany
the gate. Parser/SDK sources, normative specifications, public API, DB, stable
versions, resource limits and repository hygiene policies remain unchanged.
Full compiler/release suites are omitted for this test-only promotion; the child
docs validator is unconfigured. IO/fs expansion and SDK execution remain outside
this increment, and no push is included.

## Console SDK observation (2026-09-13)

Unchanged `std/io.fy` (6518 bytes) is now an observation-only corpus input. It
parses completely with 29 declarations, status 0 and zero problems in both native
profiles. Whole declaration kind/name/span records match Go, including the
attributed external-function and console-wrapper declarations. No parser or SDK
source modification was required.

`fairyc_go_test_parser_sdk_gate` passed in 6.603s with `files=6 required=5` and
the existing required-inventory/outcome rejection controls. IO logs explicitly
retain `required=false`; promotion remains a separate increment. The native
corpus does not run IO console functions or their declared effects. FS is not
part of this comparison.

Only the corpus inventory, candidate README/evidence and exact commit authority
change. Formatting and scoped strict checks accompany the focused SDK gate.
Normative specifications, public API, DB, stable versions, repository hygiene,
parser limits and runtime behavior are unchanged. Full compiler/release suites
are omitted for this test-only increment; the child docs validator is unconfigured.
These observations establish bounded syntax and declaration parity, not semantic
or console execution conformance. No push is included.

## Console SDK promotion (2026-09-13)

IO is now the sixth required-complete SDK input. Its inventory omission and
partial/exhausted/truncated outcomes are explicit rejection controls. The
observation-only policy control now uses a synthetic identity with no file read,
so it cannot conflict with promotion of an actual SDK module.

`fairyc_go_test_parser_sdk_gate` passed in 7.489s. Both native profiles report
`files=6 required=6`, with zero problems and exact Go declaration records for all
six sources; IO retains 29 declarations. Partial parsing or exhaustion of IO now
fails before any prefix fallback. FS has not been added to the corpus.

Only the corpus test, candidate README/evidence and exact parent commit authority
change. Formatting and scoped strict checks accompany the focused gate. Parser
and SDK source, normative specifications, public API, DB, stable versions,
resource limits and repository hygiene policies are unchanged. Full compiler and
release suites are omitted for this test-only promotion; the child documentation
validator is unconfigured. This is syntax/declaration evidence, not console
execution or semantic conformance. No push is included.

## Earlier validation and limits (2026-09-12)

- `fairyc_go_test_parser_sdk` passed in both Windows MSVC profiles: final 51.582s.
  It covers Go type-tree shape/span comparisons, pointer depth 31/32, generic list
  name/header spans, empty/malformed lists, rollback, disallowed foreign/state
  generic headers, 4096-node budget boundaries, missing-child verifier mutation,
  and SDK full-record equality plus retained-declaration comparisons.
- `fairyc_go_test_parser_syntax` passed after pointer support (173.285s), including
  the updated old pointer-rejection expectation and existing type/attribute/literal/
  record/visibility regressions. Later generic changes use the narrower SDK lane
  and final surface/evidence checks rather than repeating this expensive suite.
- The final surface run passed diagnostics, postfix and all 16 own-source cases
  (complete syntax and Go declaration equality in both profiles), but initially
  failed the old struct generic-rejection assertion. The failure was classified
  as a stale test contract. Struct/resource/visibility rejection cases now use
  malformed generic lists; a formerly rejected pointer type argument is also a
  positive generic-call case. `fairyc_go_test_parser_sdk_compatibility` reran all
  four affected checks successfully in 48.505s. The failed surface run is not
  reported as an overall pass; its unaffected successful checks remain evidence.
- `fairyc_go_test_parser_evidence` passed in 14.070s: the unchanged 22 fixtures
  still give 12 supported cases and 10 expected rejections, with identical records
  in both profiles; the integrity probes also passed.
- Go formatting and scoped strict checks are used for the changed files and exact
  command contracts. The new SDK lane is offline, closed-stdin, bounded at 180s,
  and declares cache-only writes with the existing exclusive cache lock.
- Full compiler suites, benchmarks and release checks are not selected: no public
  compiler/runtime/performance contract changed. No child documentation validator
  is configured. Repository hygiene policy, DB and deployment surfaces are unchanged.

These are bounded parser and declaration-shape results, not full language or SDK
support, diagnostic-ID parity, full expression equivalence, semantic validity,
cross-arena acyclicity, unique ownership, fuzz coverage or bootstrap promotion.
The Go implementation remains authoritative. Only the candidate README and this
private evidence report document the new boundary; normative specifications stay
unchanged.
