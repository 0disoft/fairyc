# Fairy local-state record codec, 2026-11-07

Status: Go-compatible canonical record and detached snapshot prerequisite
implemented in Fairy; actual local file storage/CAS remains open.

A disjoint high-reasoning worker owns strict base64, while the coordinator owns
record layout/checksum/entry validation and integration. The record codec uses
the existing admitted Fairy graph layout family, all seven codec IDs and
explicit full-state limits. It retains the existing Go file format; it adds no
alternate database or 8MiB JSON-parser ceiling.

The worker's base64 test passes both Windows profiles in **19.528s**. Each
profile compares 339 small inputs against Go standard base64, then full bytes
at 65,536/65,537/65,538 and 16,777,216 raw bytes. It verifies raw/encoded/decoded
hard-limit refusals, padding/unused-bit/alphabet checks, exact/one-short and
overdrawn byte histories, error-history retention and detached ownership.
Log: `%TEMP%/fairyc-state-local-base64-20261107.log`. Encode charges raw bytes
plus twice the encoded bytes; nonempty decode charges twice each input/output
plus four tail checks. Empty and malformed paths preserve actual charges.

The coordinator's **52 native record cases** pass in **7.082s**, using both
profiles. Actual Go-created empty and published files match the independent
Go JSON/SHA256 oracle and Fairy re-encoding byte-for-byte. Seeded values include
null, nonnull empty, NUL/0xff/CR/LF/0x1a and 64KiB binary data. Version 0,
UINT64_MAX-1 and UINT64_MAX, exact/one-short file/value/count limits, invalid
policies, checksum/prefix damage, unknown fields, whitespace, leading zero,
overflow and duplicate/unordered keys pass. The normal populated case also
checks exact/one-short shared structure and byte histories. Log:
`.cache/state-local-codec-final.log`.

Separate empty-layout and sealed-owner checks pass in **5.796s**, including the
Go null-fields/empty-array distinction in both profiles and two forbidden State
construction/mutation cases. All successful fixture results include snapshot
version/layout/key/value comparisons after destroying the received wire and
decoded State. Log: `.cache/state-local-codec-owner-final.log`.

Preparation failures are retained separately: attempt 1 is a test-only Go
layout-type mismatch; attempts 2/3 identify ambiguous dereference following an
if-expression; attempt 4 identifies unsupported direct fixed-array resource
fields and narrowing casts. The implementation uses a nested identity struct
and checked octet construction rather than changing compiler admission.
Attempt 5's actual empty-file native case passes before the final matrix.
These failures are not counted as native successes.

An isolated commit-scope full-source check restores the three unrelated pending
Windows lock/runtime source files from the parent commit before checking, so
the first commit's lock describes exactly its own source tree. The following
Windows lock commit retains its independently checked complete-tree lock.
Record: `.cache/state-local-codec-scope-check-latest.json`.
Both commands pass in **12.498s / 12.405s** at **745 source files / 9,345 MIR
functions**, empty stderr. All 1,215 authored files match that exact scoped
snapshot, with only the explicitly restored parallel files excluded from the
working-tree comparison. Scoped Go vet, gofmt and diff checks pass.

Commands:

```powershell
go test ./internal/cli -run '^TestCandidateStateLocalBase64$' -count=1 -v
go test -mod=readonly -count=1 -timeout 6m -v ./internal/cli -run '^TestCandidateStateLocalCodec$'
go test -mod=readonly -count=1 -timeout 4m -v ./internal/cli -run '^TestCandidateStateLocalCodec(EmptyLayout|SealedOwner)$'
```

Linux, exhaustive allocation fault injection, actual Fairy file publication,
one-attempt persistent execution, ordinary CLI selection and final acceptance
remain open. Public CLI/SDK ABI, DB/provider/dependencies, runner configuration
and repository policy are unchanged. No push, CI dispatch, release or deploy.
