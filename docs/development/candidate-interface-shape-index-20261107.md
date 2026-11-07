# Interfaces shape-index regression, 2026-11-07

This sidecar adds only a Go regression, its Fairy testdata fixture, and this
document. The parent owns the production `module_unit_interfaces.fy` index change,
lock, full source check and self-build. No production FY or lock was edited by
this worker; no commit was made.

## Actual builder and public lookup

`internal/cli/candidate_interface_shape_index_test.go` uses the existing
`candidateAtlasProbeSize` with `module_unit_interfaces` and the real authored
import closure. `internal/cli/testdata/interface_shape_index/main.fy` calls:

`scanner::collect -> syntax::parse -> signatures::build -> binding::bind(0,1)
-> interfaces::build -> interfaces::field_measured`.

There is no extracted private index test, synthetic `Interfaces`, or modified
production fixture copy. Two imported providers each define 80 ordinary structs;
a resource and enum complete 162 retained shapes. The root has 80 functions with
shared/exclusive references to both providers. Each module stays below the
independent 128-declaration policy; graph and build work stay below the existing
4096-type and 1048576-work limits. An initial single-provider fixture exceeded
the existing per-module declaration policy and was split without changing caps.

A test-only linear oracle uses public shape/field records and the pre-index
lookup algorithm, including its first matching shape and ordered field search.
975 real public lookup calls must match its result or error code:

- Every ordinary/resource shape: first and last field IDs, field order, missing
  field code 18, and the same names through both shared and exclusive references.
- Ordinary fields remain accessible from the caller module. Resource access from
  the wrong module returns 6201, including a missing name and borrowed owners.
- Enum field access returns 18. Zero and past-end type IDs return 18 before any
  probe; a valid scalar without a shape returns 18 after one index probe.
- Probe accumulators start at 17 to check additive accounting. Expected lookup
  costs are exactly 7 for `first`, 11 for `last`, and 17 for `missing`: one index
  probe plus the unchanged `name.len + 1` per candidate field. Permission and enum
  rejection cost one probe, independently of shape position.

The fixture covers all retained shape positions and first-match semantics on the
builder's canonical records. It does not forge duplicate shape rows or inject
allocation failures. Generic/vector field behavior remains covered separately by
the existing aggregate-interface tests, which this sidecar does not rerun.

## Validation

```powershell
bun tools/run-go.ts test -mod=readonly ./internal/cli -run '^TestCandidateInterfaceShapeIndex$' -count=1 -timeout 6m -v
gofmt -l internal/cli/candidate_interface_shape_index_test.go
git diff --check
```

The corrected production bytes tested are SHA-256
`45df26b072fed2ec69d16414b7b9f34d25ec76286705af0e5620903e610f45e2`.
This includes the parent's Fairy grammar fix that places the probe increment
after a `let`, rather than after an `if` across a newline. The Go test logs that
hash and refuses to execute an image if the source changes during its build.

Both low and optimized native execution passed with identical results:

```text
shapes=162 types=487 work=10611 indexed=11 linear=172 queries=975
ok
```

`TestCandidateInterfaceShapeIndex`: **PASS**, Go package 45.744s (test 45.56s),
low subtest 6.51s, optimized subtest 30.74s. These subtest durations include native
compilation. Formatting and `git diff --check` also pass. Verification HEAD was
`48eda648c745d148570d0b79d5016bc43edf857e` with the parent's dirty production/lock
patch; these results verify the explicit production hash above, not HEAD alone.

This is deterministic probe accounting, not a wall-clock speedup claim. Native
compilation shares the laptop with the parent's self-build. Full source/locked
checks and self-build belong to the parent; Linux and policy-cap boundary tests
are outside this sidecar. No production limit, public signature, permission code,
runner, dependency or global timeout was changed by this worker.

The final low/optimized log and exact file/PE hashes are retained under
`.cache/interface-shape-index-20261107/`; failed fixture attempts are retained
separately and are not passes. Successful images are temporary Go test outputs.

| File | SHA-256 |
| --- | --- |
| Go regression | `72bf0e109a43eba25f97ba4021b3e8e6f0c15ec0767a4abcfb8d9046e874f619` |
| FY testdata fixture | `089151ad0f5571fe23a1459dbd1a1ef485ac8fcff5acc3e05ca579632c80c61b` |
| Low PE | `ecdb74044c560591e3eb8294cb9d4f60d2692223c45d7499b7703e7eca50ae11` |
| Optimized PE | `a775967168b26703fa17289d5dd9c7de3162c64a4b6b3b1b614e09643280fd6a` |

`worker-freeze.json` records the final three owned file hashes and
`windows-native.log` hash. Production and lock changes belong to the parent and
are deliberately excluded from the owned files.
