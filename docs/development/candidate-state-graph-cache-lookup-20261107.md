# Pre-compilation graph disk lookup, 2026-11-07

ADR 0525 adds separate graph lookup preimages and Complete-derived save routes.
The disjoint Sol high worker changed only the new source and test; the parent
reviewed/integrated them and owns source checks, lock, docs and commit.

```powershell
go test ./internal/cli -run '^TestCandidateStateGraphCacheLookup$' -count=1 -v
```

The final run PASSes **108.805s**, **67 cases/profile / 134 Windows native
processes**, no selected skips. Independent Go length framing and SHA-256
construct expected canonical/route bytes. Actual source, coordinate and
environment changes invalidate routes; numeric owner rebinding preserves them.

Controls cover all eight compatibility fields, empty/invalid UTF-8 values,
Unicode and adjacent-field framing; standard-library and coordinate digest
shape, namespace selected-owner/duplicate/span/missing/size/unrelated-row scope;
empty/binary untrusted source and 65,536/65,537-byte bounds. Exact/one-short,
overdrawn and shared-work controls cover raw compute and environment derivation.
The actual sealed environment adapter is executed, not simulated by labels.
Complete save routing agrees after input source/table destruction and rejects
ordinary Complete. Returned detached route bytes outlive all input owners.

The first attempt fails before native execution on two probe integer inference
errors (E1006); explicit `usize` annotations fix them. The successful second
command supersedes it. An initial hygiene wrapper mistakes new-file diff exit
1 for an error; corrected checks report no whitespace diagnostics. Logs are
`.cache/graph-cache-lookup-native-attempt1.log`,
`graph-cache-lookup-native-attempt2.log`, `graph-cache-lookup-hygiene.log` and
`graph-cache-lookup-summary.log`.

Fresh complete-source `check` and `check --locked` pass **722 source files /
9,023 MIR functions**, empty stderr. All **1,192 authored files** match the
checked copy, whose successful lock is copied back. Snapshot locator:
`.cache/state-graph-cache-lookup-check-latest.json`. Scoped vet, format, diff
and UTF-8/LF/final-newline checks pass.

This is routing, not graph disk reuse, workspace/CLI selection or execution.
Full Go suite, Linux native execution, allocator-failure injection and all final
gates remain unexecuted. Public CLI/SDK/runtime ABI, DB, dependencies, default
limits, repository and runners are unchanged. No push or dispatch.
