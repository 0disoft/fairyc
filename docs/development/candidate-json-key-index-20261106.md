# Bounded decoded JSON key index, 2026-11-06

Owner: `examples/lexer_candidate/src/project_json_tree.fy`.
Native regressions: `internal/cli/candidate_json_keys_test.go` and the existing
`candidate_source_plan_test.go` Go/Fairy source-plan oracle.

The previous object checker compared each decoded key with every earlier key
in that object. A proposed unsafe-rich SDK map exhausted the shared 16,777,216
work limit during current-map joining. Its failure is recorded separately in
`.cache/atlas-unsafe-production-diagnostic-join-20261106.log`; it is not a pass.
The product work, input, depth, output and process limits remain unchanged.

Each active object's private key-index segment is now sorted. Binary search
checks the decoded value before insertion; integer token-index shifts retain
the segment order. Closing a nested object truncates only its segment, so equal
keys in different objects remain independent. Original token order, offsets,
decoded values, source bytes and container-next coordinates are not rewritten.
Duplicate errors retain code 4 and the later key's original byte span, including
escaped/Unicode/empty keys. Allocation and work failures still reject without
publishing a partial tree.

For K keys of at most L bytes, byte comparisons change from O(K^2 L) to
O(K log(K + 1) L). Index shifts remain O(K^2) worst-case integer operations;
they are explicitly charged. Space remains O(K) in the existing key vector.
This is not a claim that the complete parser is O(K log K) or a wall-time benchmark.

The 128-key ascending and descending fixtures measure total parsing work of
19,906 and 30,554 respectively in both Windows profiles. Previously, the key
uniqueness loop alone charged 170,688 units, before scanning/structural checks.
The native suite requires the gain and exercises nested reuse, insertion order,
decoded duplicates and one-short/exact/one-more/exhausted work limits.

`go test -mod=readonly -count=1 -failfast -timeout 10m -v ./internal/cli
-run '^TestCandidate(JSONKeys|SourcePlanParity)$'` passes in 61.225s. New key
checks pass in 4.92s (28 native executions); the unchanged Go/Fairy source-plan
corpus passes in 56.19s (1,698 native cases). Both low/optimized profiles pass
without selected skips. Log: `.cache/atlas-json-keys-native-input-20261106.log`.
The earlier empty-stdin probe wiring error is not a parser failure or a pass.

Authored check and check --locked pass with 674 files and 8,383 MIR functions
in `atlas-json-keys-check-20261106.log` and `atlas-json-keys-locked-20261106.log`.
Go formatting and diff checks pass. Native Linux, full suite, full authored-main
commands, self-build, distribution and hosted CI are separate. This changes no
stable CLI/grammar, runtime ABI, JSON/wire schema, dependency, DB, target, runner
or repository policy. No push, dispatch or deployment.
