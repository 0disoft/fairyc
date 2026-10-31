# Windows mixed-stack, callback and trap CI wiring - 2026-11-01

Status: Windows locally verified; hosted same-commit execution pending.

Ordered item 9 connects the existing native smoke's mixed-stack matrix to a source-addressed
CI receipt. It does not add a runner or repeat the native probe in that job. Native trace and
receipt validation originate in `68f6d8e5`; the following wiring commit contains this record.

## Native observations

Both MSVC profiles use independent generated-Fairy and C companion objects. Each execution
records the Fairy source, generated C, companion/runtime source, both objects and final PE
SHA-256 plus toolchain and inspected runtime-function count. These are identities of transient
artifacts, not retained build products or PDB lookup evidence.

| Case, in low and optimized | Exact result | Recorded boundary |
| --- | --- | --- |
| walk-and-c-boundary-catch | exit 42, no trap | C inspect, Fairy callback, C outer and Fairy main frames in order; live handler/data, caller PC/SP and establisher frame; callback returns 41; C outer adds one |
| callback-trap-no-unwind | exit 134, assertion-failed category 6 | Callback entry with value 13 and null data; private trap channel; no return, live walk or C finally event |
| reject-handler-address | exit 134, category 6 | Actual handler and deliberately shifted expected address differ by 1 |
| reject-handler-data | exit 134, category 6 | Actual handler data and deliberately shifted expectation differ by 4 |
| reject-caller-stack | exit 134, category 6 | Actual restored caller SP and deliberately shifted expectation differ by 8 |

For each mismatch the walk stops at the C outer frame, the C-contained exception is still caught,
abnormal inner finally executes once, and the callback returns -99 before outer adds one and
Fairy asserts. Catch/abnormal/normal outer-return counters must each be one. Frame order and
increasing stack addresses are validated. Trap bytes are preserved as hex separately from native
stdout; unexpected stderr and any unrecognized trace line fail. No exception is transferred
through a Fairy frame. Observations use fixture-only instrumentation, not product output.

## Source and CI identity

The existing `windows-native` job removes a stale receipt before invoking the smoke, passes its
verified `source_sha` to the writer, and runs `.github/scripts/windows_mixed_stack_evidence.py`
even when the smoke fails. `fairyc.windows-mixed-stack/v1` contains all ten observations and
the source revision. The checker requires that revision, current HEAD and workflow step's
`SOURCE_SHA` to agree; a nonempty requested candidate must also match. Workflow revision and
numeric run/attempt are required separately, allowing an exact-candidate run to use a newer
workflow revision without pretending that workflow code belongs to the candidate.

Missing or duplicate cases, wrong profiles, missing hashes, skipped/failed smoke, malformed
trace, wrong callback result, handler/data/caller mismatches outside the named controls, wrong
trap mapping or revision mismatch prevent success. The resulting
`fairyc.windows-mixed-stack-ci/v1` JSON in the job log contains all observations, source and
workflow identity, run/attempt and the SHA-256 of the exact raw receipt bytes. The step summary
retains the compact identity and digest. The enclosing `fairyc.windows-native-ci/v2` receipt
requires both smoke and mixed-stack checker outcomes to succeed.

Job logs follow the existing Actions retention policy; no upload, new permission, cache or
binary/PDB retention is added. There is no hosted run URL for this change yet. A later authorized
push or Windows-only exact-candidate dispatch must execute this wiring at the desired SHA before
item 9 can claim hosted completion. A local receipt does not establish that condition.

## Windows validation

Passed:

```powershell
go test -mod=readonly ./tools/dev/fairyc-windows-host-smoke -run '^TestWindowsMixedStackBoundaries$' -count=1 -v
go test -mod=readonly ./internal/windowsunwindprobe -count=1
go vet -mod=readonly ./internal/windowsunwindprobe ./tools/dev/fairyc-windows-host-smoke
python -B -m unittest discover -s .github/scripts -p 'test_*.py'
actionlint .github/workflows/frontend.yml
git diff --check
```

Ten native executions pass: two normal exit-42 and eight expected assertion traps. The Python
suite checks actual PowerShell revision/receipt blocks and failure paths plus synthetic mutation
cases; synthetic run IDs in tests are not hosted evidence. The live native receipt also passes
the Python row checker locally. Complete host-smoke and full `go test ./...` are not rerun for
this focused change; the native test and smoke use the same probe/writer.

Linux/WSL remains deferred by request. No push, workflow dispatch, release or deployment ran.
Public API/SDK ABI, DB, dependencies, production compiler flags, repository hygiene rules,
runner count/image/deadlines and Linux workflow paths are unchanged. Windows workflow steps
and private evidence schemas changed; owning validation, CI, ABI evidence and ordered-work
documents are updated. General callbacks, retained/asynchronous delivery, foreign unwinding,
full target conformance and PE/PDB retention remain separate work.
