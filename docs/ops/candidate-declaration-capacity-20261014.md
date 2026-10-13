# Original declaration catalog capacity

The 437-source native dispatcher probe rejects before C generation at
`std/fs.fy`, bytes 7254..7366, phase 1/code 10. Its seed catalog still has the
4096 original-declaration limit. The complete source-identity implementation
requires a larger original catalog, independently of executable functions.

The original-declaration bound is now 8192. Per-file declarations remain 128;
imports/bindings, types, executable callables, source bytes and work limits are
unchanged. Duplicate checks visit only the current module's contiguous rows;
names in other modules never conflict and no global duplicate scan is needed.

Two authored tests pass Go-native and candidate C development/release: 8192
original declarations are accepted, 8193 reject without a catalog, and duplicate
declarations/import aliases retain their original errors after a preceding
128-declaration module. Initial authored tests used an unsupported Option field
and a conditional resource move; both test constructions were corrected before
the successful run. Receipt: [evidence](evidence/candidate-declaration-capacity-20261014.json).

Root and locked source checks pass 448 files / 5821 MIR functions. This unit
does not yet prove full dispatcher C generation. A fresh emitter must carry this
catalog implementation before replaying the complete build/run integration.
Public CLI/SDK ABI, Go, DB, CI and product stack/deadlines are unchanged; no push.
