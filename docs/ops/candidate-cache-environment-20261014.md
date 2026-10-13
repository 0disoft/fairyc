# Current candidate cache environment

`module_unit_body_ir_model` owns `fairyc.candidate-owned-body-ir/0` compatibility
for its checked body/statements/cleanup/layout model, separate from the Go MIR/2
wire codec and a future artifact envelope. `project_cache_profile` owns the
`verified-candidate-owned-body-ir` backend and explicit
`not-applicable:verified-candidate-owned-body-ir` semantic toolchain, with the
current Windows ABI and target facts.

The default SDK instance is derived by authenticating the seven embedded source
files and using the existing identity encoder. It matches Go `sdkstdlib.Load`,
independently of the selected workspace SDK. `project_cache_environment::current`
owns the exact running-image identity and profile; `key_environment` returns
independent compatibility bytes to the v4 encoder without caller labels/hashes.

Two profile tests pass Go-native and C development/release. One environment test
passes both C profiles under normal and false argv[0], comparing complete PE
hashes through Python and Go-owned SDK/target fields. Detached copy and exact/
exhausted/invalid work are covered. The existing v4 detached/budget regression
also passes all three profiles, all at default stacks. The initial static-borrow
helper failed E2103; boolean comparisons fix the test without changing semantics.
Root and locked checks pass 462 files / 5939 MIR functions. Go-native environment
execution is skipped for private Windows linkage; Linux/full-suite/CI are unrun.

Complete-owner package keys and verified artifact reuse remain next. Public
CLI/SDK ABI, product stacks/deadlines and versions are unchanged; no push.
Receipt: [evidence](evidence/candidate-cache-environment-20261014.json).
