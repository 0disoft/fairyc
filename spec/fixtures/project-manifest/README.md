# Project manifest fixtures

`manifest-v1.json` executes eight complete `fairy.toml` files through the strict local-exact
manifest parser. The two passing cases cover a single package and explicit workspace/path
dependencies. Rejections freeze unknown and duplicate fields, missing explicit feature sets, path
escape, noncanonical feature ordering, and schema mismatch.

These fixtures own the human manifest parser boundary only. They do not read workspace members,
resolve symlinks, inject the standard library, solve versions, write `fairy.lock`, or authorize
network source acquisition.
