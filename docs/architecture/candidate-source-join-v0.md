# Candidate source-set AST join

The implementation owner is `examples/lexer_candidate/src/module_ast_join.fy`.

Joins parser-accepted source owners into one exact AST arena. The published
source is the original bytes connected by one newline, and every declaration,
expression, statement, type, entry, field, attribute and argument is copied
from actual parser-owned syntax with its IDs and byte spans shifted by the
cumulative arena and source offsets. Names are never substituted, no input is
reparsed textually, and no syntax-derived spelling is synthesized.

Scanner token IDs are remapped by the exact joined position of the original
token start. A separator newline can merge neighbouring trivia into one joined
trivia token, so a file token that does not start at its shifted position maps
to the joined token whose span contains it; significant tokens always start at
the shifted position and never move.

Problem positions are original module-local byte offsets inside the failing
input; joined offsets exist only in the returned owner.
9 unsupported or inconsistent input, 10 limit, 11 allocation.

Always-present spans shift by the source offset. Optional spans and one-based
or zero-based arena references keep zero as absent, so only real values move.

Declaration holds only spans and arena references: expression/body/result/generic
are one-based IDs, entry_start is a zero-based slot, and field_start and
attribute_start are zero-based arena slots. cleanup_attribute_* is the cleanup
attribute range. name_start is zero only for static_assert and state declarations.

Kind 12/14/31 retain a block statement in left; kind 18 retains its arm body
statement in right; kind 20/23 retain a type root in right. Every other non-zero
left/right is an expression ID, and token is always a significant token index.

Statement references: expression and target are expression IDs, first is a
statement ID, entry is an entry ID and operator is an assignment token. Only
second depends on kind: block count for 1, statement IDs for 7/14/15, statement
header/body links, and the item type for 18.

Every type node keeps left/right as type roots or argument links.

Entry name and type spans are zero for return and inferred bindings; id keeps the
one-based slot numbering that entry_start indexes.

Payload fields precede their enum member in the shared field arena, so the
member index payload_start+payload_count shifts with the same field base.

Scalar function templates retain original source ownership until binding and
specialization. Generic aggregate/receiver declarations remain closed here.

Limits of the joined owner: 256 inputs, 1048576 joined bytes including separators,
512 declarations, 16384 expressions, 8192 statements, 1024 entries, 4096 types,
1024 fields and 4096 attributes and attribute arguments. The existing scanner
token, diagnostic and work budgets apply to the joined source.

## Parsed arena observations

Private declarations: 1 module, 2 import, 3 function, 4 struct, 6 resource, 7 const,
8 static_assert, 9 unsafe extern C function, 10 state, 11 user enum; problem 4 is
outside the admitted subset. These are not public compiler diagnostic IDs.
Function entries in source order: 1 parameter, 2 let, 3 var, 4 return.
Zero name/type spans mean unwritten; expression zero means no initializer/value.
One-based child IDs; zero means absent. Nodes are postorder, with byte spans
and the source token index retaining exact literal/name/operator spelling.
1 block, 2 binding, 3 return, 4 assignment, 6 expression, 7 list link.
Standalone block/if expressions use kind 6; legacy statement kind 5 is not emitted.
8 while (expression condition, first body), 9 break, 10 continue.
11 assert, 12 discard and 13 defer retain their operands in expression.
14 for: expression is optional condition, first body, second kind-15 header.
15 header: first optional binding/assignment initializer, second optional assignment.
16 burn retains its operand in expression; syntax does not consume resources.
17 region: target is a kind-17 name node, expression the arena, first the body.
18 byte for: target is the item name, expression the view, first the body, second the item type.

## SDK compaction ownership

The CLI compacts its source vector in place. It authenticates the complete
source set and computes all SDK replacements before moving any source owner.
Failure in validation, copying or reservation leaves the input vector intact.
Unchanged user ASTs and scanners move through reserved Vec storage in manifest
order. The original source vector retains capacity, so restoration performs no
allocation. SDK replacements are copied in reverse module order and popped in
forward order; all source bytes and scanner observations survive. The borrowed
compact API remains available for snapshot comparison.

## Manifest path observations

Exact, untrimmed UTF-8 relative path lines. LF/CRLF and no final newline;
no empty lines, BOM, control bytes, colon/ADS, or leading slash/backslash.
Relative traversal is allowed: this is a cwd resolver, not a sandbox.
1 invalid manifest, 3 allocation, 4 path bound, 5 module bound.

## Joined scanner cooked storage

`scanner::collect_joined` preserves the ordinary scanner's source, token kinds,
trivia, spans, canonical cooked spelling and diagnostics while sharing equal
nonempty cooked buffers. Canonical string cooked data is hex text; its storage
length is twice the decoded string byte count. Token kinds retain their meaning
when different literal spellings or kinds share equal cooked text.

A private 256-slot fixed index uses exact byte comparison after hashing. When
the index fills, a charged scan of the owned value table also finds values that
were never indexed. The 65536-byte pool cap counts distinct owned payloads.
Hashing, probing and byte comparisons consume the existing 16777216 scanner
work budget; exhaustion produces the existing lossless E0010 stream. Fixed
slots avoid extra Vec specializations, and mutable slot access uses the existing
scalar-array view path. Stream snapshots copy each owned value once and retain
every token's original value index.

Ordinary `scanner::collect` and the lexer protocol keep their existing storage
and work behavior. Joined module declarations own complete original files,
including trivia; declaration name spans still identify the original module.
