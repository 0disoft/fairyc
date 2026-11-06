# ADR 0528: Bounded native graph slots and trap identities

- Status: Accepted for private verified graph preparation
- Date: 2026-11-07

Fairy `module_state_graph_native_plan` prepares detached structural data only
from the sealed `module_state_graph_program::Verified`. Root precedes helpers;
each body's values follow block parameters, operation result and found result
in exact Go `graphSlots` order. Borrow results have a separate slot namespace.
Sparse logical IDs never become allocation lengths.

A preflight counts all value and borrow slots before slot-array allocation and
refuses a sum above 4096. Trap categories are copied, sorted and deduplicated
across all bodies. Their one-based IDs and JSON bytes, including empty `null`,
match Go. Accessors retain an immutable owner and independently charge lookup
work. The existing 16 MiB ledger and private errors remain unchanged.

Plan confers no current-source, Complete, cache, execution or storage authority.
Native operation C emission must additionally require Complete and pinned
runtime assets. Windows evidence and exclusions:
`docs/development/candidate-state-graph-native-plan-20261107.md`.
