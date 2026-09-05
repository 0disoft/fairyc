# ADR 0383: Compile experimental source state transactions to native execution

- Status: Accepted
- Date: 2026-09-05

## Decision

Add `compiler.CheckExperimentalState` as an explicitly selected experimental entry point over an
ordinary parsed Fairy source file. Admit one package-owned `cell u64` and one non-public, safe,
parameterless `fn void transaction()`. Its body consists only of expression statements calling
`storage::write("field", decimal_u64)`, `storage::read("field")`, or
`storage::delete("field")`, followed by exactly one `storage::commit()`, `storage::reject()`, or
`storage::trap("category")`. Field names are literal strings naming the declared cell.

The adapter inspects AST structure, not source text. It rejects other declarations, control flow,
ordinary calls, dynamic operands, missing or early exits, and unsupported signatures. Normal
semantic analysis retains ownership of the state declaration, attributes, and storage effects.
Extracted operations enter the existing HIR side table before atomic ordinary/state MIR publication.
They are not emitted as unverified foreign runtime calls. The ordinary compiler entry point keeps
rejecting these experimental unresolved calls.

Native regression tests parse the checked-in example, compile its operations through this entry
point, prepare the verified compiler result, emit C, and run it under low and optimized MSVC.
Source variants cover writes and deletes with commit, reject, and trap; runtime probes observe
exact retained bytes, repeated execution, rejected malformed storage, and exact trap categories.

## Boundaries

This is an experimental source admission and does not change the released language revision or
stable CLI. No standard-library module named `storage` is added. `fairyc experimental run-state
FILE.fy --profile low|optimized` executes the source with a command-owned empty in-memory host
store and reports the native exit kind, trap category when present, and final cell bytes. Successful
command execution returns zero independently of the transaction exit kind. Paths use the existing
bounded source-root loader; other flags, profiles, and hosts fail closed. It is not a normal workspace
build/run command. Source reads
produce transaction observations, not ordinary expression values. Graphs, maps, helpers, dynamic
values, public ABI, persistence providers, and other native targets remain separate work.

## Evidence

- `internal/compiler/state_source.go`
- `internal/compiler/state_source_test.go`
- `internal/contractstatenative/source_test.go`
- `internal/contractstatenative/native_windows_test.go`
- `examples/contract_state_native/transaction.fy`
- `spec/fixtures/contract-state-native/manifest-v0.json`: three versioned source-to-C fixtures
  for commit, reject, and trap, executed by `internal/contractstatenative/fixture_test.go`.
- `internal/cli/state_run_test.go`: command-level low/optimized MSVC execution and exact receipts.
