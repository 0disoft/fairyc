# Whole-package cache fixtures

`manifest-v1.json` freezes one exact `fycache1-sha256` key and proves independent invalidation for
the compiler build, source graph, dependency interface, target, feature set, and profile flags.
The Go unit tests additionally cover canonical ordering, text parsing, bounded byte/entry storage,
LRU eviction, deep-copy isolation, and oversize rejection.

The fixture payload describes disposable compiler-private inputs. It is not a public manifest,
lockfile, persistent cache file, typed-HIR serialization, MIR exchange format, or remote-cache API.
