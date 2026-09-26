# ADR 0404: Candidate for read-only SDK vector iteration

- Status: Candidate; compiler-private source and Flow proof, no stable source admission
- Date: 2026-09-24
- Admission source: [ADR 0218](0218-modern-language-concept-admission-boundary.md)
- Related boundaries: [byte-view iteration](0403-read-only-byte-view-iteration.md), [public vector](0390-public-windows-sdk-vector.md), [owner-bound storage borrows](0389-owner-bound-storage-borrows.md)

## Workload

The Fairy-written lexer candidate has three read-only `Vec<T>` loops that use
their index only to visit every initialized element in order:

- `scanner::report_work` visits `Vec<Token>` and holds `ref Token` for one body;
- `scanner::storage` visits `Vec<bytes::Buffer>` and `Vec<diagnostics::Entry>`
  to accumulate byte lengths.

These are three local uses in one application, not proof of demand across
applications. The diagnostic reporter keeps indices for stable ordering and is
not part of this candidate. The file-syntax parser also uses indices to treat
the last segment specially; changing it would hide required state.

Current code already expresses these loops with `v::len<T>` and `v::at<T>`.
The proposed benefit is removing repeated indexing without introducing a
callback, a collection copy or a runtime iterator. The following is an
illustration, **not accepted Fairy syntax**:

```fy
for (let ref Token token in ref tokens) {
    number(token.kind as usize);
}
```

The existing parser's `for (let type name in expression)` shape can retain this
source; semantic analysis currently admits only an immutable `ref u8[]` byte
view with a copied `u8` item. Parsing a shape is not language admission.

The first experimental implementation step prepares a separate AST block for
eligible shared-reference item loops. It finds the actual `std::vec` import
alias, captures a named borrowed view or explicit borrow of a path/field place
once, and sketches calls to that alias's `len<T>` and `at<T>`. Generic cloning
keeps the written loop and prepared block detached from the original source.
Only the authenticated SDK package graph now exposes those prepared calls to
bounded generic planning; an import spelling or unowned provider alone cannot
activate them. Stable workspace semantic analysis still rejects the vector
source form before lowering, so it emits no C for that form. Exact type and
borrow checking, Flow and verified-MIR execution remain future gates for stable
admission. A separate compiler-private proof lane now checks the exact
authenticated, specialized `std::vec::Vec` resource identity, the written
shared item and eligible view shape, then runs ordinary type checking and Flow
over the prepared loop. Its first empty-vector and named-borrow cases reach
verified MIR; wrong element type, mutable item, in-loop owner replacement and
returning an item borrowed from a local vector are rejected. Stable workspace
commands still use the closed source gate.

The private Windows AMD64 MSVC probe now runs both the candidate and explicit
index form under low and optimized profiles. Empty vectors execute no body;
four initialized scalar elements produce the same ordered `continue`/`break`
result. The C probe observes no live vector allocation after function return.
This proves these control and cleanup cases, not the representative lexer
workload or the cumulative C-relative cost gate.

The private native parity test also compares an early `return` from a scalar
vector loop and, separately, `?` propagation while a vector owns two
`bytes::Buffer` elements.
Both forms pass Flow and verified MIR, then run under low and optimized Windows
MSVC. The propagation probe observes at least three allocations (both buffers
and vector storage) and no live allocation after the caller handles the error.

A further native comparison runs both source forms with a Region inside the
vector loop and exits through `return` and `?`. A C observer checks that the
inner resource cleanup precedes the outer resource cleanup on both exits.
The allocator records the vector and Region allocations, then requires all of
them to be freed after the owning Arena drops. The Arena refuses to drop with
an active Region, so this also checks native Region reset under low and
optimized Windows MSVC. Exact interpreter Region/drop traces and cleanup of
resource-bearing vector elements across Region exits remain separate gates.

A second private native probe now inserts a shared `Vec<bytes::Buffer>` loop
into the existing nine-buffer ownership workload. The candidate and explicit
index forms both sum the nine five-byte views to 45, then complete the same
pop, move and resource-cleanup checks under low and optimized Windows AMD64
MSVC profiles. This covers resource elements and borrowed byte views without
admitting the syntax in stable workspace commands. It still does not compare
the actual lexer output or generated C size. The cumulative 8% C-relative
budget also remains unmeasured.

The real lexer source comparison loads the checked-in scanner package graph
with the authenticated private SDK. It tests the original explicit-index
source, each of the three loop rewrites separately, and all three together.
All five forms pass source analysis, Flow and verified MIR; the stable source
lane rejects every rewritten form. Native runs cover empty input and a
cooked-number plus unterminated-comment input. Exact console output matches
under low and optimized Windows AMD64 MSVC profiles. Storage counts match the
independent Go lexer observation, and the allocator probe finds no live SDK
allocation after the scanner calls and owned stream return. The C-relative
cost gate remains open.

A [one-run size comparison](../ops/evidence/vector-lexer-size-20260924.md)
found unchanged function, foreign-symbol and specialization counts and EXE
size for all three rewritten loops together. Generated C grew by 711 bytes;
optimized `unit.obj` grew by 24 bytes against the indexed Fairy source. This
does not establish check/build time, runtime or the independent-C budget.

An independently written checked C vector-read kernel and the indexed Fairy
SDK equivalent passed ten shared boundary executions and a
[five-pair optimized comparison](../ops/c-budget-vector-read-index-20260924.md).
The indexed Fairy kernel's paired runtime median was 1.38544 relative to C,
above the cumulative 1.08 limit, while its EXE ratio was 1.01824. The
candidate source form was not measured in that stable harness, so this result
cannot isolate the syntax cost. It keeps source admission open and points to
the existing vector access path as a performance investigation target.

The [private candidate C comparison](../ops/c-budget-vector-read-candidate-20260924.md)
now measures the proposed loop against the same independent checked C source.
Its five-pair runtime median is 1.37255 relative to C, still above 1.08;
EXE ratio is 1.01824. Ten boundary executions and the stable source rejection
preflight pass. The candidate and earlier indexed Fairy reports come from
separate runs, so their difference cannot yet be attributed to iteration
syntax. A directly paired index/candidate measurement remains open before
any source admission decision.

[Generated-access inspection](../ops/vector-read-access-inspection-20260924.md)
finds identical optimized disassembly for the indexed and candidate Fairy
units after excluding the dump-file path header. Both retain an inner-loop
call to the private vector slot lookup in `std::vec::at<T>`. That common access
path is the next optimization experiment; preserving safe bounds and the
owner-bound shared borrow is required. This evidence does not admit the source
form or explain the entire C-relative runtime gap.

[ADR 0405](0405-direct-address-shared-vector-access.md) now uses a checked
typed element address for shared `Vec.at<T>` in SDK `0.0.0-stage0.67` while
retaining the owner-bound borrow. A preliminary diagnostic measurement removes
the helper call and brings both indexed and candidate kernels close to, but
slightly above, the 1.08 paired runtime limit in that pre-commit diagnostic.
Stable source admission still requires the representative real-workload and
remaining cost evidence.

The [post-commit shared-access comparison](../ops/c-budget-vector-read-address-20260924.md)
observed both indexed and private-candidate four-element kernels below 1.08
on paired runtime medians and EXE size. Individual pairs cross the runtime
limit, and the representative lexer and unused-feature gates remain open.

A [prebuilt-image three-way comparison](../ops/vector-read-three-way-20260924.md)
rotates C, indexed Fairy and the private candidate through twelve measured
triads. Candidate/C is observed within 1.08 on paired and lane-median ratios,
and candidate/indexed is close to parity. Indexed/C is inconclusive because
its two median estimators straddle 1.08. This remains microkernel evidence;
the representative lexer, cleanup and unused-feature gates are open.

The [real-scanner execution comparison](../ops/vector-lexer-execution-20260924.md)
builds both optimized Fairy forms once and times the checked-in lexer's two
storage loops over 10,000 scans. Both EXEs are the same size, but the runtime
samples exceed the predeclared stability span; they support no speed verdict.
The token reporter runs outside the timed section, and the independent-C
full-scanner budget remains unmeasured.
A predeclared 25-times-longer window also exceeded the stability limit and
reversed the point-estimate ordering. Runtime cost remains inconclusive; more
repetitions under the same uncontrolled host are not an admission substitute.
A [single-image paired run](../ops/vector-lexer-execution-20260924.md#same-image-paired-follow-up)
also exceeded the stability limit despite adjacent calls in one process. The
private source form stays closed pending controlled runtime and independent-C
evidence; this combined image does not measure the token reporter's hot loop.
An [additional paired run](../ops/vector-lexer-cpu-time-20260924.md) records
QPC and current-thread CPU time for the same two real scanner forms in one
optimized image. This run meets the prior timing-stability heuristic and finds
the two timed storage loops close to parity. Its CPU readings are coarsely
quantized, and the difference from earlier unstable QPC runs cannot be
attributed to the added timer. The reporter, unused-feature overhead and
independent-C full-scanner budget remain unmeasured; stable source admission
is unchanged.

The private rejection matrix also rejects moving a borrowed vector owner
(`E2105`), treating a scalar as the vector owner (`E1006`), and using a
computed vector expression as the iterable. The latter cannot acquire an
implicit temporary owner or hide evaluation in this source form.

The indexed real scanner also serves as an unused-source-form control. A
compiler test analyzes the same checked-in package graph through the stable
and private candidate lanes, verifies both MIR programs, and compares their
canonical private MIR serialization, generated C bytes, function count,
foreign-symbol list and specialization count. Equality here is an executable
artifact check for this workload. A subsequent
[unused-source check-cost comparison](../ops/vector-scanner-unused-check-cost-20260925.md)
uses eight balanced pairs of the same indexed source. Its lane and paired-ratio
spans exceed the predeclared stability rule, so analysis overhead remains
inconclusive. A separate native unused-mode check compiles that exact indexed
source with the private parser path off and on. Generated C is byte-equal at
395,860 bytes, and both modes have equal PE sections and EXE sizes in low
(278,528 bytes) and optimized (201,728 bytes) MSVC profiles. This proves no
unused-mode native image growth for this workload; it does not measure object
file sizes, analysis overhead or a future integrated stable compiler path.

An independently authored C11 scanner reference now starts the full-scanner
comparison lane. Its first checked slice covers source token boundaries,
keywords, trivia, comments, punctuation, UTF-8 recovery and diagnostics under
low and optimized Windows MSVC, compared with the Go lexer oracle. A subsequent
integer-cooking step covers decimal, binary and hexadecimal values beyond 64
bits, invalid spellings and the representative cooked-number/unclosed-comment
input. A bounded decimal-float step now checks exponent normalization and
malformed spellings against the same oracle. The C reference now also checks
plain string, byte-string and C-string values, simple, hex and Unicode
escapes, raw UTF-8, invalid bytes, NUL diagnostics and unterminated strings.
The C reference now admits the 70,000-digit positive and negative exponent
cases. A compact length and full-payload FNV-1a digest probe compares them
with the Go lexer under low and optimized MSVC without exceeding the
native output cap. Expansions beyond the 16 MiB default Go work-unit ceiling
still return an unsupported status in C. This remains insufficient for an 8% C budget baseline:
remaining lexical forms,
limits, storage and output must be implemented and checked before timing or
comparing executable sizes.
This large-decimal check is against the Go lexer, not the Fairy-written
candidate scanner. The latter still returns candidate-size status 4 above
16,384 materialized decimal bytes, caps cumulative cooked bytes at 65,536,
and rejects source longer than 32,768 bytes. The C candidate budget images and
library mode now return candidate-size status 4 before scanning a longer source;
the standalone Go-oracle C build still returns bad-input status 8 for a frame
above its 32,768-byte input cap. Low/optimized MSVC probes accept exactly
32,768 bytes and reject 32,769 bytes in both C modes. The same boundary now
runs through the actual Fairy scanner with both explicit-index and private
vector-loop source: its accepted token report matches the C candidate report,
and the extra byte returns status 4 without a partial report. Both C budget
storage lanes enable the candidate cap. This proves one source-length boundary,
not full C/Fairy lexical parity. The C reference now admits larger decimal
payloads; the Go lexer's default source limit is 8 MiB and its 16 MiB work
budget also charges scanning.
The 70,000-digit digest check therefore cannot prove Fairy/C equivalence.
The standalone C Go-oracle build retains that wider range. The C budget image
now enables a candidate decimal-materialization limit of 16,384 bytes.
The checked-in 38-case lexical corpus now runs through both C budget storage
lanes and the explicit-index and private vector-loop Fairy scanners under low
and optimized Windows MSVC. Each lane matches the Go lexer's token and
diagnostic observations in its output encoding. The cases include BOM and
comments, number and string spellings, Unicode, invalid bytes, recovery and
unterminated input. Eleven added cases cover overlong UTF-8, surrogate and
out-of-range encodings, invalid leading bytes, truncated sequences, the
largest valid scalar, and malformed bytes inside strings. This establishes
parity for those inputs, not every limit,
allocation failure or source shape; it is an untimed correctness comparison.
Low and optimized MSVC runs check positive and negative exponent forms at
16,384 bytes (accepted) and 16,385 bytes (candidate-size status 4) against
the actual Fairy scanner. The accepted C output also matches the Go lexer.
The C budget image now also caps total cooked bytes at 65,536 and returns
candidate-size status 4 before publishing an over-limit number or string.
For strings, Fairy stores ASCII hex bytes as the cooked payload, so both Fairy
and the C budget image charge twice the decoded-byte length; C strings include
the terminating NUL before encoding. The prior claim that the limit used raw
decoded bytes was incorrect. A library-mode MSVC probe checks exact and
one-byte-over boundaries for plain, byte, C and Unicode-escaped strings under
low and optimized profiles. This does not establish the full-scanner cost gate.
Four 16,384-byte decimals succeed, while a fifth is rejected, in both C and
Fairy under low and optimized MSVC; the standalone C Go-oracle build remains
unrestricted by these candidate switches. The C budget image also charges
consumed source bytes plus prior and current decimal expansion before
materialization. At work limits 6 and 7, `1e3` respectively yields lossless
`E0010` exhaustion and a normal token in C and Go; the actual Fairy scanner
follows the same boundary. A default-budget large exponent also yields
`E0010` before allocating its expansion. The candidate C budget images now
charge string source bytes before consuming them, including prefixes, UTF-8
sequences and escape components. Thirteen low-budget cases compare the first
invalid token, remaining-input token, `E0010` position, EOF and accepted cooked
length with the actual Fairy scanner under low and optimized Windows MSVC.
The candidate C image also checks one work unit at every token start and
charges each whitespace or comment segment before consuming it. Seventeen
additional C/Fairy native cases cover spaces, CRLF, line and documentation
comments, and nested block comments, including two-byte segments rejected
without partial consumption. They compare the first invalid token, retained
tail, `E0010` position, EOF and successful trivia kind in both MSVC profiles.
Fifteen further C/Fairy cases cover identifier and keyword continuation,
one- through three-byte symbols, and valid or invalid UTF-8 rejected as a
source token. They check atomic multibyte work exhaustion, `E0010` or `E0001`
spans, retained tails and EOF under both profiles. Six more C/Fairy
native cases cover a three-byte BOM alone and followed by an identifier.
They check atomic BOM rejection below three work units, exact acceptance,
next-token exhaustion, retained input and EOF in both MSVC profiles.
Twenty C/Fairy native cases now cover numeric source-work boundaries for
decimal, base-prefixed and malformed integers, fraction and exponent scanning,
and the transition to decimal expansion work. They compare the first invalid
token, retained tail, `E0010` or `E0004` span, EOF and accepted token kind
under low and optimized MSVC. Seven further C/Fairy cases exercise competing
work and token limits on short source and the 256-diagnostic boundary on
257 invalid bytes. They check which terminal `E0010` position wins, token and
diagnostic counts, retained tail and EOF in both profiles. Other combined
limits and whole-scanner runtime remain unproven.
The candidate C budget images also stop after 256 ordinary diagnostics and
retain one terminal `E0010`.
Runs with 256, 257 and 258 invalid bytes match the Go lexer exactly in both
C storage modes under low and optimized MSVC. The actual Fairy scanner checks
the same boundary's token count, diagnostic count, terminal span and EOF.
This establishes the diagnostic-count boundary for those inputs, not all
multi-diagnostic lexical paths. Other low-budget recognition paths and combined
limits remain unproven, so this is not a complete C-relative baseline.
The C budget images also apply the candidate's 1,048,576-token default limit.
A separate probe sets limits 0 through 3 and compares both C storage modes
against the Go lexer under low and optimized MSVC. Empty input, exact-limit
success, BOM, invalid bytes, a cooked number and exhaustion with remaining
input are covered. The actual Fairy scanner checks the corresponding counts,
terminal `E0010` location, rejected token and EOF. The broad standalone C
Go-oracle build remains unlimited by this candidate switch. These small
boundary inputs do not prove every interaction between token, diagnostic and
work exhaustion or establish an 8% cost result.
The C budget images now copy each nonempty diagnostic token into its own
allocation and release it with the diagnostic record. Empty tokens retain no
heap allocation, matching `bytes::copy` and the underlying system-heap
zero-size behavior used by the Fairy scanner. A low/optimized MSVC probe
checks that changing the caller's token bytes cannot change the retained
diagnostic in either C storage mode. The optimized three-image scanner smoke
still matches output and 1,000-round checksums. The C budget image now copies
the payload before reserving a geometric diagnostic slot, matching Fairy's
`bytes::copy` then `v::push<Entry>` order. Fault-injected low/optimized MSVC
probes cover failure of each allocation and success: both failures return
storage status 3, publish no diagnostic and leave no allocation live after
caller cleanup. This covers one nonempty diagnostic path, not all allocation
failures or a new runtime/EXE measurement. Both C budget storage lanes now use
the declared Fairy `diagnostics::Entry` field shape: numeric code, two byte
positions, repair flag and an owned four-word Buffer descriptor. A C11 MSVC
probe asserts the 64-byte record and field offsets, and checks the retained
payload, ordering and Go-observed output under low and optimized profiles in
both storage lanes. The broad Go-oracle C build keeps its original record.
This target-layout proof is not a stable runtime-cost or full-failure verdict.
The measured short sample is inside all three supported ranges; a broader
budget comparison needs the same accepted input and limit behavior in both
implementations before using executable-size or runtime ratios as a gate.
An independent-C workload entry now copies the real scanner sample on every
round, scans it, releases token, cooked and diagnostic payload storage, and checks the exact
first-round observations plus source/cooked/diagnostic byte and item totals
against the Go lexer under both MSVC profiles. One- and two-round runs verify
repeatability. This is an untimed parity gate, not the whole-scanner runtime
or executable-size comparison.
The matching C benchmark entry runs that same source-copy, scan, storage and
cleanup loop between QPC calls, with its reporter flushed beforehand. Its
one-thousand-round smoke checks the independent lexical output, completed
rounds and checksum under low and optimized MSVC; it does not compare Fairy
runtime or executable size yet.
An opt-in three-image smoke now builds standalone optimized C, explicit-index
Fairy and private-candidate Fairy executables for that sample. It checks the
same completed-round count and checksum in all three, compares the two Fairy
reporters, and records QPC ticks and EXE bytes. One execution per image is not
a stable paired measurement, so these observations do not decide the 8% gate.
The follow-up three-image measurement keeps those builds and the sample fixed:
one warmup triplet, then twelve measured triplets with C, indexed Fairy and
candidate Fairy rotated through every execution position. Each lane performs
250,000 checked scanner rounds per triplet; QPC brackets only the scanner work.
Every run must retain its reporter, completed count and independent Go checksum.
All raw timings and image bytes are retained without retries or exclusions.
The existing 10% relative-full-span heuristic applies to lane and paired-ratio
series before interpreting an 8% result. The timed loop omits token reporting,
so this is a scanner-workload comparison, not a whole-application verdict.
The [measured triplets](../ops/vector-lexer-c-budget-20260924.md) preserve all
thirteen triplets. Every runtime series exceeds the predeclared stability
heuristic. The two Fairy images have the same EXE byte count, but both exceed
the restricted C reference's size by 13.24%. That C reference is not yet a
complete scanner, so this does not close the representative 8% gate or admit
the private vector source form.
The [current-image triplets](../ops/vector-lexer-c-budget-20260925.md) repeat
the fixed three-way workload after the intervening C scanner changes. Both
Fairy images again have equal EXE size, 11.67% above the still restricted C
image. The private-vector/index paired median is 0.98705 with an 8.88% full
span; the C lane and both C-relative ratios exceed the 10% stability limit.
This remains diagnostic evidence, not an 8% admission result.
The [four-image rotation](../ops/vector-lexer-four-image-c-budget-20260925.md)
then pairs both C storage modes with both Fairy forms in one run. For the fixed
sample, up-front C is both faster and smaller than geometric C with stable
lane and storage-ratio series. Against that C lane, the private vector form
has a stable 1.79340 runtime ratio and an 11.67% EXE excess; the explicit-index
form is similarly over budget. This common-input screen fails the 8% limit,
but the C reference's incomplete behavior still prevents a representative
whole-scanner admission verdict or any relaxation of the gate.
The [plain-decimal scanner fast path](../ops/decimal-fastpath-four-image-20260925.md)
preserves existing integer spellings while avoiding the digit accumulator for
validated ordinary decimal input. In a new fixed four-image run, the stable
private-vector/up-front-C paired median falls from 1.79340 to 1.57821; both
Fairy EXE byte counts stay at 205,824. Generated C grows by about 0.9%,
check/build cost is unmeasured, and the 8% C-relative gate remains open.
The [symbol dispatch comparison](../ops/symbol-dispatch-four-image-20260925.md)
replaces a full symbol-list scan on every punctuation token with first-byte
dispatch. The 26-case lexical corpus includes every listed symbol spelling.
On the same fixed scanner input, the stable private-vector/up-front-C median
falls to 1.29292, while both Fairy EXEs remain 205,824 bytes. Generated C
grows about 2.84% from the previous image. This still fails the cumulative
8% runtime and EXE screen; check/build time and general-workload cost are open.
The [keyword-length grouping comparison](../ops/keyword-length-four-image-20260925.md)
reduces the reserved-word candidates visited for each identifier and expands
the cross-language lexical corpus to 27 cases. Its fixed-sample timing spans
exceed the predeclared stability limit, so the lower point estimates do not
establish a runtime improvement or 8% admission. Both Fairy EXEs grow to
206,336 bytes and remain 11.94% above the up-front C image; generated C grows
another 2,486 bytes per Fairy image. The source form stays private.
The [PE section receipt](../ops/scanner-pe-section-budget-20260925.md) accounts
for the entire 22,016-byte optimized EXE difference against up-front C:
20,992 bytes in `.text` and 512 each in `.rdata` and `.pdata`. It narrows the
size investigation to emitted code and linked support but does not identify
individual functions or satisfy the cumulative 8% gate.
The [COFF object receipt](../ops/scanner-coff-code-budget-20260925.md) finds
identical 32,116-byte `unit.obj` code totals for indexed and private-vector
Fairy forms, plus 8,039 runtime-object code bytes in each. The up-front C
`probe.obj` has 13,653 code bytes. Object bytes are not final linked-image
contributions; function-level attribution remains necessary before a size fix.
The [function-span receipt](../ops/scanner-function-spans-20260925.md) records
81 function starts in each Fairy unit object and identical five largest spans for both
source forms. `fy_fn_29` has the largest raw section span at 5,297 bytes.
These COFF spans can include padding and do not show which functions survived
linking; source-name and link-map attribution remain open.
The [source-name map](../ops/scanner-function-source-map-20260925.md) now
identifies the five largest Fairy unit spans as `scanner::scan`,
`strings::scan`, `numbers::decimal`, `strings::escape` and
`scanner::report_work`. Their retained linked sizes are still unknown.
The [MSVC link-map receipt](../ops/scanner-link-map-20260925.md) observes
positive `unit.obj` symbol addresses for those five functions in both
Fairy images and retains the same four EXE byte counts. It does not
measure exact linked function lengths or account for CRT contributions.
The [final-image unwind receipt](../ops/scanner-unwind-sizes-20260925.md)
finds that validated `.pdata` function intervals explain 20,122 of the
20,592-byte Fairy/up-front-C `.text` virtual-size gap. The five largest
COFF spans do not retain those sizes after linking: `scanner::scan` is
5,297 raw object bytes but its matched final `.pdata` interval is 134 bytes.
Object and CRT ownership of the interval gap remains unassigned.
The [map-owner join](../ops/scanner-unwind-owners-20260925.md) attributes
only 7,775 bytes of the 20,122-byte unwind-interval gap to uniquely named
owners. The other 12,370-byte difference has no matching public symbol
start; optimizing `unit.obj` or the runtime on this partial map alone
would be unsupported.
An [optimized PDB layout comparison](../ops/scanner-debug-layout-20260925.md)
matches all 526 C and 589 Fairy release-image unwind intervals with their
respective symbol-bearing builds. It enables a private-symbol attribution
attempt, but that attempt finds no exact PDB symbol for the map-unmatched
intervals. Their owner and the 8% gate remain unresolved.
The [linker library-member comparison](../ops/scanner-library-members-20260925.md)
finds identical selected archive members for both Fairy images and just one
additional C member, `libucrt.lib(strlen.obj)`. It rules out extra Fairy
archive-member extraction as the size explanation, but does not measure
retained COMDAT or generated function bytes.
A [two-field work-budget experiment](../ops/scanner-remaining-work-20260925.md)
reduces both Fairy scanner EXEs by 1,536 bytes to 204,800 without changing
the checked lexical or work-limit observations. Its C-relative timing series
fail the predeclared stability rule, and the EXE ratio remains 1.11111; this
improvement does not admit the vector source form.
A [mutable work-budget update](../ops/scanner-inplace-work-20260925.md)
removes another 1,024 bytes from each Fairy scanner EXE while preserving the
checked lexical and limit behavior. The EXE ratio is still 1.10556 against
up-front C, and the one-thousand-round smoke is not a stable runtime result.
A [one-word work-budget experiment](../ops/scanner-one-word-work-20260925.md)
removes another 512 EXE bytes per Fairy source form, leaving a 1.10278
C-relative size ratio. Generated C grows by 2,353 bytes, so check/build cost
and runtime still require separate measurement before any admission claim.
A [numeric work-borrow experiment](../ops/scanner-numeric-work-borrow-20260925.md)
removes 512 more bytes from each Fairy EXE and 3,774 bytes from each generated
C source by leaving work state in the caller. The EXE ratio remains 1.10;
neither check/build time nor stable runtime has been established.
A [string state-borrow experiment](../ops/scanner-string-state-borrow-20260925.md)
removes another 512 bytes from each Fairy EXE while preserving checked
lexical and limit behavior. The C-relative EXE ratio remains 1.09722;
compile-time and stable runtime evidence remain open.
A [current COFF function comparison](../ops/scanner-current-function-source-map-20260925.md)
finds `unit.obj` raw code reduced by 4,470 bytes from the earlier scanner
image, with the five largest function spans now smaller or unchanged. This
does not attribute the remaining final-image gap to any one function.
A [unified payload reporter](../ops/scanner-unified-payload-reporter-20260925.md)
uses one immutable byte-view loop for source literal and cooked payload
output. Generated C shrinks slightly, but the optimized EXEs do not; this
does not reduce the remaining 3,584-byte size gap.
The [full COFF function-source map](../ops/scanner-full-function-source-map-20260925.md)
now covers all 81 Fairy unit-function starts and 59 runtime-function starts
per image. Both Fairy source forms have 21 `std::vec` spans totaling 1,654
raw object bytes; their unit code differs by only one raw byte. COFF spans
do not establish retained linked size or a stable runtime result, so this
does not close the 8% gate.
A [decimal Buffer-reuse experiment](../ops/scanner-decimal-buffer-reuse-20260925.md)
compacts validated mantissa digits into the existing normalization Buffer.
Checked scanner behavior passes the focused corpus, and both Fairy EXEs
shrink by 512 bytes to 201,728. The C-relative size ratio remains 1.09444,
still 3,072 bytes above the 8% ceiling; stable runtime and build cost are
unmeasured.
An [indexed/vector check-phase comparison](../ops/vector-scanner-check-cost-20260925.md)
times fresh source loading through verified MIR in eight alternating measured
pairs. Both lanes retain 21 specializations. Their check-time medians are
close, but lane and paired-ratio spans are too wide for a speed verdict.
A [paired build-phase comparison](../ops/vector-scanner-build-cost-20260925.md)
then includes C emission, support assembly and optimized MSVC build. Its five
measured indexed/vector pairs have a stable phased-total median ratio of
0.97548, with identical 201,728-byte EXEs. C emission and vector check phase
spans remain too wide for individual speed verdicts. This compares the two
Fairy forms, not independent-C build cost; unused-feature overhead remains
unmeasured.
The [configured C source-byte limit](../ops/scanner-configured-byte-limit-20260925.md)
now models Fairy callers that set `max_bytes` below the hard 32,768-byte
candidate cap. Thirteen boundary inputs pass against the Go lexer in both C
storage modes and both Fairy source forms. The optimized independent C EXE
grows by 512 bytes while the Fairy images stay at 201,728, leaving a 1.09141
size ratio and 2,110 bytes above the 8% ceiling. Other limits, failures and
stable runtime remain open.
Two additional combined-limit cases use `a@` with a source-byte limit of one
or two, a work limit of one and a token limit of zero. The first terminates at
source position zero before token or work exhaustion; the second admits the
source and terminates at position one. The C reference and both Fairy loop
forms agree on token counts, retained tail and terminal diagnostic under low
and optimized MSVC. This checks those competing-limit orders only; it does not
complete the remaining limit or runtime gates.
Five more low/optimized cases cover UTF-8 BOM handling when work, token and
caller byte limits compete. The independent C scanner and both Fairy loop
forms agree on the retained invalid tail, first token, EOF and terminal E0010
position. These cases do not establish every limit combination.
The C candidate now also accepts the previous-revision setting used by the
Fairy and Go lexers: `defer` is an identifier in that mode and a keyword in
the current revision. Low and optimized native probes pass for both C storage
modes and both Fairy loop forms. The current optimized four-image smoke keeps
the smaller up-front C EXE at 184,832 bytes and both Fairy EXEs at 201,728;
the 2,110-byte excess over the cumulative 8% ceiling is unchanged.
The [configured C diagnostic limit](../ops/scanner-configured-diagnostic-limit-20260925.md)
now models caller-selected limits below the default 256. Nine boundary inputs
match the Go lexer in both C storage modes and both Fairy source forms under
low and optimized MSVC. The optimized EXE sizes do not change: the smaller C
image remains 184,832 bytes and both Fairy images remain 201,728 bytes, still
2,110 bytes above the cumulative 8% ceiling. Other limit interactions and
later allocation failures remain open.

The [current four-image rotation](../ops/scanner-current-four-image-20260925.md)
measures that source after the limit changes. On the fixed input, explicit-index
Fairy has a stable 1.06282 paired runtime ratio to the smaller, faster C lane.
The private vector form's 1.07110 point estimate has a 12.34% paired span,
exceeding the 10% stability rule. Both Fairy EXEs remain 2,110 bytes above
the cumulative 8% size ceiling. This does not admit the private source form.

The bounded PE-associated PDB probe now resolves every validated unwind
interval to a DBI module index and privacy-reduced object or archive basename.
In the current optimized scanner debug-map builds, the C `probe.obj` owns
11,924 bytes of `.text` code contributions; Fairy `unit.obj` owns 24,845 and
`runtime.obj` owns 4,138. Across all modules, the contribution sums are
122,180 C bytes and 137,552 Fairy bytes. The private vector form changes the
Fairy sum by one byte. These are PDB contribution sizes, not exact retained
function sizes or a reconciliation of all EXE sections and file alignment.
They identify the Fairy unit and runtime as the next size-analysis targets;
the cumulative gate still fails.

A later bounded CodeView module-stream check joins procedure records to the
current optimized debug PE's `.text` contributions. Each selected module's
nonoverlapping procedure ranges partition its complete code contribution:
the independent C `probe.obj` has 19 linked functions totaling 11,984 bytes;
the explicit-index Fairy `unit.obj` has 37 totaling 24,845 bytes and
`runtime.obj` has 27 totaling 4,138 bytes. The private vector form changes
only one `unit.obj` byte. The largest Fairy unit records are `fy_fn_29`
(4,083 bytes), `fy_fn_28` (2,397 bytes) and `fy_fn_10` (2,313 bytes); the
largest C record is `fc_scan_string` (3,291 bytes). The current C `.text`
contribution sum is 122,240 bytes after the previous-revision mode change;
the earlier 122,180-byte observation above belongs to the preceding source.
These exact selected-module spans narrow source attribution but do not explain
PE file alignment, establish equivalent per-function work, or close the
2,110-byte EXE excess over the 8% ceiling. The current C emitter's ordered
function list joins these retained PDB records to source names: `fy_fn_29` is
`scanner::scan` (4,083 bytes), `fy_fn_28` is `scanner::report_work` (2,397
indexed / 2,398 vector bytes), and `fy_fn_10` is `numbers::decimal` (2,313
bytes). Their individual sizes do not establish that any one source rewrite
will reduce the final EXE by 2,110 bytes or preserve runtime and compile cost.

Three bounded `scanner::scan` source refactors failed to reduce the optimized
image: a shared cooked-value limit/storage helper, direct two-byte dispatch
tests in place of selected `matches` calls, and a shared diagnostic-limit
helper. The lexical parity check passed for each completed form, and the
four-image smoke kept both Fairy EXEs at 201,728 bytes against the 184,832-byte
smaller C image. All three source changes were reverted. Inspection of the
generated C found repeated checked arithmetic and bounds branches in the
retained scanner function; their source counts alone do not establish native
instruction cost. Further size work should measure a proof-preserving backend
change rather than assume another spelling or helper extraction saves space.

Two further source-only trials also left this `/O2` size gate unchanged.
Making the identical scanner and diagnostics numeric-output bodies share one
function, and replacing the string scanner's per-byte append of a validated
UTF-8 sequence with one `bytes::append`, each passed the configured 38-input
C/Fairy lexical-parity check. Each separate four-image smoke verified the
reporter and 1,000-round checksum; the smaller C image remained 184,320
bytes and both Fairy images remained 201,728 bytes. Both source edits were
reverted. These runs did not measure stable runtime, build cost, or every
allocation-failure point, and do not justify a general claim that source
deduplication has no value.

A compiler-private C-emission trial replaced dynamic bounds-trap category
assignment with fixed-category trap blocks only when the target was empty and
had one checked predecessor. It retained each bounds condition and passed a
Go build plus the four-image reporter/checksum smoke. All four optimized EXE
sizes stayed at 184,320, 185,344, 201,728 and 201,728 bytes respectively.
The emitter trial was reverted; it did not run the full backend fixture suite
or establish a runtime change.

A later `matches` experiment replaced per-byte `input[position + index]`
access with one checked subview and `candidate[index]`. The independent-C
lexical-parity test passed, but both optimized Fairy scanner images grew from
201,728 to 202,240 bytes while the smaller C image stayed at 184,320 bytes;
the source edit was reverted. A direct `memcmp` experiment did not reach native
execution: casting a byte view to `ptr void` is unsupported by the C backend,
while declaring `memcmp` with `ptr u8` conflicts with the standard C header's
pointer parameters under `/WX`. Both attempts were reverted. They neither
justify removing checked indexing nor admit a raw-pointer conversion. A future
safe byte-view equality operation would need its own SDK and C ABI contract,
with explicit empty-view and pointer-validity handling and a measured
incremental and C-relative cost.

Another compiler-private candidate adds a separate, symbol-selected C byte
matcher with explicit input length, position and word length. It checks bounds
before pointer arithmetic, accepts an empty word at the end, and rejects null
pointers for nonempty reads. A temporary `scanner::matches` implementation
called it through an explicit unsafe FFI boundary while the public SDK surface
stayed unchanged. The independent-C lexical-parity test and four-image smoke
passed; the indexed and private-vector Fairy EXEs remained 201,728 bytes.
After the paired timing probe failed its stability rule, the checked-in
scanner returned to its prior safe per-byte loop. The native helper remains
isolated for comparison but is no longer called by the default scanner. This
does not establish runtime superiority, Linux native behavior or the
cumulative 8% C-relative admission gate. The public `std::bytes` API remains
unadmitted.

The benchmark-only C reporter now emits the same numeric value sequence as
`scanner::report_work`; the broad C lexical oracle keeps its separate `T`/`D`
format. The optimized four-image smoke checks one C/Fairy reporter sequence
and 1,000-round checksums across both C storage modes and both Fairy loop
forms. The up-front C EXE shrank from 184,832 to 184,320 bytes, the geometric
C EXE is 185,344 bytes, and both Fairy EXEs remain 201,728 bytes. Thus the
smaller-C size ratio is now 1.09444, still above the cumulative 1.08 limit.
Earlier size and runtime receipts used the former benchmark reporter and remain
historical; no fresh paired runtime result or output-failure parity is claimed.

The benchmark-only reporter also passed a separate 38-input differential
corpus under low and optimized Windows MSVC. Each of the two C storage modes
and two Fairy loop forms was built once per profile; every reporter line was
compared with the independent Go lexer observation. Inputs cover empty source,
keywords, strings and escapes, cooked numbers, comments, BOM, malformed and
truncated UTF-8, and diagnostic recovery. The C harness uses binary stdout so
Windows newline translation cannot obscure byte-level differences. This check
does not exercise altered scanner limits, allocation failures, process timing,
or the cumulative 8% executable-size gate; those remain separate evidence.

A second reporter check covers 32 caller-limit cases: thirteen byte-limit
boundaries, nine diagnostic-limit boundaries, and ten work/token/byte-limit
interactions including BOM, decimal and string paths. Both C storage modes and
both Fairy loop forms produced the Go lexer's full numeric observation under
low and optimized Windows MSVC. The C probe sets the same per-case limits on
the benchmark scanner before using its reporter; it does not modify the timed
benchmark image. Allocation-failure behavior, every possible limit combination,
runtime, build cost and the cumulative 8% size gate remain open.

An [MSVC size-optimization probe](../ops/scanner-msvc-size-optimization-probe-20260925.md)
reduced the same four-image scanner's Fairy EXE to 193,536 bytes under
temporary `/O2 /Os` flags, within 8% of the 181,248-byte C image. Its
rotated runtime pairs failed the predeclared stability rule, and build cost
was not measured. The driver flags were reverted; this is not stable source
or build-profile admission.

A later [thread CPU-time probe](../ops/scanner-thread-cpu-time-20260925.md)
records QPC and `GetThreadTimes` observations for the same four scanner lanes
under the unchanged `/O2` profile. The fixed twelve measured rotations have
nonzero CPU samples, but all paired C-relative QPC and CPU ratio spans exceed
the predeclared 10% stability limit. The instrumented images cannot decide
the uninstrumented EXE budget; the QPC-only smoke remains above its 8% size
ceiling. The private vector source form remains unadmitted.

A same-image optimized native parity probe now links the independent C
scanner workload beside each Fairy scanner form. On the fixed cooked-number
and unterminated-comment sample, the C work result, Fairy work result and
independent Go checksum agree; the Fairy vector allocator has no live block
after return. This establishes coexistence for a future adjacent timing probe,
not full lexical equivalence, timing stability or an executable-size result.

The subsequent [same-image timing probe](../ops/scanner-same-image-c-pairs-20260925.md)
alternates adjacent C and Fairy scanner calls in each combined image and
retains twelve measured QPC and thread-CPU pairs after one warmup. Both Fairy
forms produce the checked Go-oracle work checksum, but every C-relative ratio
span exceeds the predeclared 10% stability limit. The combined-image sizes
cannot decide the standalone EXE gate; stable source admission remains open.

A later [paired byte matcher probe](../ops/scanner-byte-match-pairs-20260925.md)
puts the prior checked per-byte `matches` loop and the bounded native matcher
in the same optimized private-vector scanner image. Its twelve adjacent pairs
return the same Go-oracle checksum. The current/previous QPC median is 1.00283
with 38.39% relative span; thread CPU median is 1.01250 with 35.53% span.
Both exceed the 10% stability limit, so the native matcher has no established
runtime advantage. The combined image cannot decide the standalone 8% size
gate, and Linux native execution remains unverified.

A [pinned-thread same-image comparison](../ops/scanner-same-image-cycles-20260926.md)
records twelve adjacent C/Fairy pairs per source form with QPC and
`QueryThreadCycleTime`. Both images return the Go-oracle checksum, but all four
paired-ratio spans exceed the predeclared 10% stability limit (39.35% to
109.40%). The cycle medians do not establish elapsed-time parity, and the
instrumented images do not decide the standalone size gate. Further timing on
this host needs a new control or environment; the private vector form remains
unadmitted.

## Proposed first boundary

The iterable would have to be an explicit shared borrow of an exact SDK-owned
`Vec<T>` place, or a named immutable `ref Vec<T>` local or parameter. The item
would have exact type `ref T` and be valid only within one loop-body entry.
`T` would follow the already admitted SDK vector element set, including named
resource elements by shared reference. No element move, mutable element view,
user-defined iterator, other generic collection, implicit borrow, or hidden
allocation would be admitted. An effectful iterable expression or a vector
from an unauthenticated package would be rejected.

The vector length would be observed once before entry. At each index the
implementation must apply the same initialized-prefix and bounds checks as
`v::at<T>`, return a shared owner-bound element reference, and keep the vector
borrow live through loop exit. An empty vector executes no body. A safe move,
resize, push, pop, clear or exclusive borrow of that vector in the body must
remain an error. The hidden index is not source-visible. `continue` advances
it once; `break` skips the advance. `return`, `?`, `defer`, drops and Region
resets must take the same paths and order as the explicit indexed loop. A
borrowed item must not escape a local owner.

This first boundary must not broaden SDK generic arguments or expose the
private storage layout. A backend implementation that adds a per-element
runtime helper, copies `T`, duplicates a specialization or weakens checked
access is not equivalent to the current source loop.

## Evidence required before acceptance

1. Compare the three real loops with explicit-index Fairy source and the
   proposed form. Preserve exact scanner output, storage counts, failure
   behavior and ownership. Keep the index-dependent loops unchanged.
2. Prove source and Flow rejection for wrong item/owner types, forged vectors,
   mutation and move during the loop, item escape, effectful iterable and
   unsupported targets. Check empty, ordered, `continue`, `break`, early
   `return` and `?`, with resource and Region cleanup in verified MIR and
   native execution.
3. Compare normalized executable MIR, generated C, specialization and helper
   counts, object and EXE sizes, check/build time and process runtime against
   the explicit-index variant. Record this same-program difference as the
   incremental cost attributable to the source form. Separately compare an
   equivalent independent C workload under the optimized profile against the
   cumulative 8% runtime and EXE budget. That absolute C-relative gate remains
   required for stable admission; an unchanged Fairy EXE does not pass it when
   the indexed baseline already exceeds it. Conversely, the indexed baseline's
   excess must not be described as a cost introduced by vector iteration.
   A new source form cannot claim a zero-cost result from syntax or interpreter
   parity alone. Separately measure unused-feature overhead.
   A separate C budget variant matches the geometric capacity steps of
   `Vec<Token>` and `Vec<diagnostics::Entry>`; the up-front C storage lane
   remains available. Use the faster equivalent C lane for runtime and the
   smaller equivalent C image for size when applying the gate.
   The target C diagnostic record now mirrors the Fairy source shape, while
   other failure paths and remaining scanner limits still need parity.
   A low/optimized MSVC fault-injection probe now covers both decimal-cooking
   allocations and the first geometric token-slot allocation in the C lane.
   Each injected failure returns status 3 with no retained allocation or
   published token. The explicit-index and private-vector Fairy scanners also
   return status 3 with no output or retained allocation when their first
   allocation fails, under both MSVC profiles. This matches observable
   first-failure behavior, not the distinct allocation order or later failures.
   A full-scanner Fairy diagnostic probe now fails the second and third
   allocator calls on the unterminated `/*` input, after the first token
   allocation. Both source forms return status 3 with no report, exactly one
   injected failure and zero live allocations under low and optimized MSVC.
   The C reference's separate diagnostic-payload and diagnostic-storage
   failure probe passes in the same profiles. Their call numbers differ
   because the C probe supplies token storage while Fairy grows its vector;
   this checks the observable failure and cleanup boundary, not identical
   allocation order or all diagnostic paths.
   A second C token probe fails the 8-to-16-slot growth after eight symbol
   tokens. It requires status 3, the intact initialized prefix and zero live
   allocation after release under low and optimized MSVC; the successful path
   stores all nine symbols and EOF. This covers a later C token allocation,
   not the corresponding later Fairy allocation or every failure point.
   A matching generated-Fairy probe schedules failure on the second vector
   allocation while scanning nine symbols. Both explicit-index and private
   vector-loop forms return status 3 without a report under low and optimized
   MSVC; the one successful token block is released on exit. This covers token
   growth, not every later allocation or diagnostic failure in Fairy.
   The [paired storage A/B](../ops/c-scanner-storage-ab-20260925.md) measured
   a smaller geometric C image but did not stably identify a faster C lane.
4. Decide the exact source eligibility, diagnostic, cross-package identity,
   formatter and release pin only after the proof. A failed gate keeps this
   feature a candidate; it does not authorize relaxing checks.

No grammar, product spec, language revision, SDK, runtime ABI, MIR, cache or
target contract changes in this preparation. Other collection types, mutable
iteration and user-defined iterators remain outside this candidate.
