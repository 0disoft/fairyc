# Interpreter value model and local allocation measurements

The Go interpreter stores aggregate fields in declaration-order slices. Its value
header carries one 64-bit scalar slot and one closed interface containing an
existing state pointer. Scalar values do not use the interface slot. Copies detach
inert aggregate storage while references and linear resource owners keep their
existing identities.

Measured on Windows/amd64 with Go 1.27.1 and an AMD Ryzen 5 7430U, using the same
`value_benchmark_test.go` harness before and after each change:

| Model | Value size | Scalar copy B/op | Scalar copy allocs/op | Eight-field struct copy B/op | Struct copy allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: |
| Original maps and separate payload fields | 88 B | 0 | 0 | 960 | 3 |
| Declaration-order field slices | 88 B | 0 | 0 | 800 | 2 |
| Field slices and compact value header | 40 B | 0 | 0 | 352 | 2 |

Command (one local sample per model):

```text
go test ./internal/interpreter -run '^$' -bench 'Benchmark(Scalar|Struct)ValueCopy$' -benchmem -benchtime=100ms -count=1
```

The struct benchmark copies eight i32 fields into a fresh detached value. These
are allocation and header-size measurements for that operation; whole-program
latency requires a representative workload measurement.

The same refactor adds caller-controlled step/depth budgets and context cancellation
to i32 and void entry points. Defaults remain 1,000,000 block/operation steps and
256 call frames. The existing value-shape cycle guard is independent of those
execution budgets. See [the MIR contract](../mir/mir-v0.md) for execution outcomes.
