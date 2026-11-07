# Vector specialization AST reduction, 2026-11-08

The ADR0551 unit call/value frontend exceeded the existing 1,048,576-node
materialization budget. A read-only sidecar measured HEAD `1ff68c24` at 1,045,123
nodes and its sampled working sources at 1,049,234 nodes. Added ordinary function
bodies account for 4,111 nodes; the specialization set was unchanged. The std::vec
diagnostic span identified the final task crossing the budget, not a new vector
specialization.

The optional SDK vector source now infers exactly three local types from their
already typed initializers. `empty<T>` retains its explicit `Vec<T>` constructor;
`at<T>` and `at_mut<T>` retain their typed storage-address intrinsic calls. Public
signatures, type arguments, checks, ownership and borrow operations are unchanged.

In the measured sample, `empty<T>` loses two nodes across 335 copies, `at<T>`
across 334 copies, and `at_mut<T>` across 115 copies: 1,568 fewer cloned nodes.
The reduced sample used 1,047,666 nodes, leaving 910. This is a bounded reduction,
not substantial headroom for future growth. No budget, task selection, warning or
timeout was changed.

Windows validation with `GOMAXPROCS=1`:

- `go test -mod=readonly ./internal/sdkstdlib -run '^TestVectorCandidateIsExplicitAndDetached$' -count=1`: PASS.
- Sidecar baseline with the same SDK reduction: full `check` PASS, unchanged
  9,684 MIR functions.
- Actual integrated candidate `go run -mod=readonly ../../cmd/fairyc check`: PASS,
  773 source files, 9,697 MIR functions; lock updated by the normal CLI.

Sidecar commands, sampled hashes and task counts are under
`.cache/materialization-sidecar-20261108/`; the integrated check log is
`.cache/closure-unit-values-main-check-20261108.log`. The measurements describe
the recorded sample, not a claim of a final integrated node count. Linux,
whole-compiler native/self-build and remote CI remain deferred. The stable SDK
bundle is separate from this optional vector source; no public API, dependency,
DB, workflow or repository-hygiene configuration changed.
