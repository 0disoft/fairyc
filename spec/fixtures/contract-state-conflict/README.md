# Contract-state conflict fixtures

`manifest-v0.json` owns deterministic schedules for overlapping semantic-oracle
transactions. It fixes snapshot selection, optimistic commit conflict, and
no-partial-publication behavior without changing compiler IR publication.

The corpus is process-local executable evidence, not a scheduler, persistence
backend, isolation-level promise for external databases, or source contract.
