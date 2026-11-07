# Explicit Windows compiler-host bootstrap admission, 2026-11-07

The private Go `fairyc-candidate-build` bridge accepts `--compiler-host` for
bootstrap compiler images. It selects `DecodeCompilerHost` and the existing
image-only `BuildWindowsMSVCCompilerHostPartsWithStackReserve` route together.
Only the shared header limit changes from 2 MiB to 8 MiB. Literal/body limits
remain 2 MiB, total packet size 96 MiB, unit count 4096, and frame limit 16384.
This transport is not a sandbox or public language/API feature.

Ordinary `Decode` and target builds keep their limits. A 16 MiB stack
reservation alone does not enable compiler-host admission. Linux packet input
rejects this Windows-only option before opening input. The Fairy-authored
runtime companion retains its ordinary 2 MiB packet-part policy; the Go bridge
is still the private bootstrap native builder for full compiler images.

## Executed checks

```text
go test -mod=readonly ./internal/candidatebuild ./tools/candidate/fairyc-candidate-build -run 'Test(CompilerHost|LinuxRejectsWindowsCompilerHost|PacketRejects|PacketFrame|PacketDecodeOracle|WorkspaceSelection)' -count=1 -timeout 4m
go vet -mod=readonly ./internal/candidatebuild ./tools/candidate/fairyc-candidate-build
git diff --check
```

All passed. Fifteen packet cases exercise both decoders (30 admissions),
including immediately below/at/above 2 MiB and at/above 8 MiB for headers,
literals and bodies. A real Windows MSVC test verifies that an oversized
ordinary header preserves a previous destination, then explicitly opts in,
builds both profiles, and executes both images successfully.

Before authored-source limit changes, the clean `b1ce7d5f` whole-CLI low
baseline also passed `TestCandidateInlineLocalsFullCLI/low` in 252.092 seconds
(205 C units, 14 independent CLI processes). Its immutable evidence is
`.cache/current-whole-cli-low.log` and `.cache/current-whole-cli-20261107/low/`;
`host.json.Source` is
`8e0b83e9b2e004dcdfc6081839859fb1a20b2b9fd83627ebdba2160e353e20eb`,
and the image SHA-256 is
`56022dfb1f5e4a027a03e9764037d0cad1f0aa0d86826700b5a6addb9c46c390`.
That image must not be reused as a current-source seed after source changes.

Current-source native self-build/fixed-point acceptance is a subsequent step;
these bridge checks do not establish it. Native Linux remains deferred.
No public SDK/ABI, DB, dependency, runner or CI policy changed. No remote
operation was performed.
