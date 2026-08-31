# Contract-state conflict fixtures

`manifest-v1.json` owns the current deterministic schedules for overlapping
semantic-oracle cell and map transactions. `manifest-v0.json` remains the
historical three-schedule cell baseline. V1 fixes store-wide conflict across
distinct map keys, map snapshot selection, and exact physical storage keys.

The corpus is process-local executable evidence, not a scheduler, persistence
backend, isolation-level promise for external databases, or source contract.
