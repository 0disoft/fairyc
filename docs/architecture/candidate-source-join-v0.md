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
