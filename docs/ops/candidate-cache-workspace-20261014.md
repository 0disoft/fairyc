# Checked workspace facts for cache inputs

Fairy `project_cache_workspace::collect` accepts immutable
`project_module_visibility::Resolved` and opaque
`project_compiler_core::Complete`. It rebuilds coordinates and compiler-input
provenance from the loaded graph, comparing every source size/hash, owner,
namespace/module and import target/alias/binding against completed provenance.
It returns detached package instance IDs, coordinates, package-source digests,
canonical features and sorted unique direct dependency owner indices. These
indices refer only to this graph; persistent keys must use their coordinates
and interfaces. `sdk_instance` is the selected graph SDK ID, empty without SDK,
not Go Session's mandatory environment `StandardLibrary` value.

The actual Go workspace loader supplies seven graphs with 41 package rows:
Base, Edited, Reordered, NoSDK, Vec, Version and Feature. They include shared
packages with two feature instances and duplicate dependency edges. Regeneration
matches every binary fixture byte. Four tests pass in Go-native and candidate C
development/release, covering oracle equality, detached ownership/work bounds,
source/coordinate rejection and checked IR with swapped import providers.

Private test images use 16MiB stacks. The initial default 1MiB Go-native program
exits `0xC00000FD`; its failure is retained. Root and locked checks pass 458 files /
5913 MIR functions. Product stack/deadlines and public CLI/SDK ABI are unchanged;
no push.

Next: execution-environment facts and complete-owner key composition, verified
artifact reuse, current full fixed point, product defaults/performance and
Linux/full-suite/same-head CI. These facts alone authorize no cache hit.

Receipt: [evidence](evidence/candidate-cache-workspace-20261014.json).
