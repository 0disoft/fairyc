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
