# Contract-state layout publication fixtures

`manifest-v0.json` is the exact executable corpus for append-only layout publication and fail-closed
breaking changes. Each case starts from the same committed bytes, keeps one base-layout transaction
open, publishes or rejects one candidate layout, and proves the resulting version, layout, bytes,
and old-transaction behavior.

The historical contract-state and conflict manifests remain separate and unchanged.
