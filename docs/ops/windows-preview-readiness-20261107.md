# Public Windows preview ZIP readiness, 2026-11-07

Status: local Windows AMD64 verification passed. No publication or clean-machine
claim. The archive contains the public Go Stage 0 compiler and LSP, separate from
the private Fairy-authored CLI and its whole-source/bootstrap acceptance.

## Exact clean snapshot

The existing release command ran first, before verifier/document edits:

```text
node tools/starter-smoke.mjs --package-local
```

Both executable build records identify clean main commit
`b1ce7d5f1168d94aae9660bef713a04ef8f2036c`, `vcs.modified=false`, Go 1.27.1,
Windows AMD64, CGO disabled, and `-trimpath -ldflags="-s -w"`.
All sixteen packaging/starter steps passed, including LSP version/protocol and
development/release native tests and runs. Existing Go license and notice
checks passed without changes. No new packager was introduced.

The candidate root is `.cache/starter-smoke-Y1GkZ4/`. Its `receipt.json` SHA-256
is `ef5fe20498e77dfced01d13e00a5cd49242a6cbe5524eb264594914351d1b777`.
Later parent, verifier or documentation commits do not change this artifact's
source identity. No equivalence with a later final HEAD is claimed; do not
rebuild and reuse this receipt.

| Retained artifact relative to candidate root | Bytes | SHA-256 |
| --- | ---: | --- |
| `fairyc-windows-amd64-preview-candidate.zip` | 9,781,514 | `8b5926b117ff66055265e75171af66ba13a63e65be860a4fb5f01f2de123c323` |
| `fairyc-windows-amd64-preview-candidate.zip.sha256` | 109 | `f9b71c06b6593b285c96c8ec1c24f503933320926a2b1e1116a976a86b791fef` |
| `extracted/fairyc.exe` | 17,947,648 | `510bea855facb60128bc9c07f3f11e9216f83b310a46b990b4e89f30ab49d3b1` |
| `extracted/fairyc-lsp.exe` | 8,647,680 | `b77ded9040d0330a7860e4dc1d9aea96a021efbc23c0cad5ea5324b02f8e34eb` |
| `extracted/SHA256SUMS.txt` | 558 | `eeab11bd9d83c8cd5903ea564910d749418b20bf7f071c977138471c9cc2d5d0` |

The exact eight-entry archive manifest is `fairyc.exe`, `fairyc-lsp.exe`,
`README.txt`, `SHA256SUMS.txt`, `LICENSE`, `LICENSING.md`,
`THIRD_PARTY_NOTICES.md`, and `GO-LICENSE.txt`. The seven payload hashes are
pinned by `SHA256SUMS.txt`; its own hash is pinned by the packaging receipt.
No VSIX, private CLI or source checkout is included. The historical
`windows-editor-preview-assets.json` remains a 2026-10-28 draft inventory,
not the inventory for this newly verified candidate.

## Reproducible external execution

The previous `.cache/release-readiness-20261104/external-smoke.mjs` and its
successful receipt were inspected. The original packaged candidate's retained
verification also still passes at `9bc77723bd0a02714173f50883ac348c8caa0a52`.
The cached verifier has a fixed dated output path and cannot be reproduced from
tracked source alone. Its workflow is now retained in
`tools/windows-preview/external-smoke.mjs`, using the existing `verifyCandidate`
gate and actual ZIP extraction; compiler packaging contracts are unchanged.

```text
node tools/windows-preview/verify.mjs --receipt .cache/starter-smoke-Y1GkZ4/receipt.json --revision b1ce7d5f1168d94aae9660bef713a04ef8f2036c
node tools/windows-preview/external-smoke.mjs --receipt .cache/starter-smoke-Y1GkZ4/receipt.json --revision b1ce7d5f1168d94aae9660bef713a04ef8f2036c
node --test tools/windows-preview/verify.test.mjs tools/windows-preview/external-smoke.test.mjs
```

All passed. The Node regression command reports 25 passes, no failures/skips.
New checks cover environment filtering, unsafe/wrong-profile build receipts,
expanded payload/notice/checksum tampering, unexpected files/non-files, real
nonzero exits, stderr, timeout and expired-deadline rejection.

The permanent verifier accepts only the public compiler-and-LSP v2 ZIP, rechecks
all eight expanded files before and after execution, and rechecks retained
candidate identity after execution. It creates a unique receipt per attempt,
retains actual command results on failures, limits each process to 90 seconds
and the process workflow to 210 seconds, and preserves lock/source hashes after
locked operations. Receipts record the verifier hash and input receipt hash.
It does not rebuild the compiler or use compiler sources/caches as executable
inputs.

Actual external root:
`C:/Users/cherr/AppData/Local/Temp/fairyc ZIP 검증-7tMoQS/`.
The exact ZIP was expanded to `preview/`; a fresh starter was created below
`projects/`. Child PATH was only `C:\WINDOWS\System32;C:\WINDOWS`.
An allowlist retained Windows discovery/profile roots and removed inherited Go,
Fairy, compiler/SDK overrides and developer-shell variables. Go, Node and Bun
were not on child PATH. Node orchestrated verification by its host executable;
installed MSVC/Windows SDK remained accessible through normal compiler discovery.

All fifteen real external steps passed: extraction, LSP version/protocol,
`new`, `check`, `check --locked`, semantic `test --locked`, and each profile's
`build`, direct generated-EXE execution, `run`, and `test --native`. Both native
test outputs equal the semantic test output: one passed, zero failed. Both
direct images and `run` exited zero with empty stdout/stderr.
Lock SHA-256: `8677be7ef366cfb28984a028581742c7dbb33968067210d3692fbf86a41e275d`.
The two image hashes in the receipt describe the bytes observed before direct
execution; the release build subsequently replaces the same public build path.

External receipt SHA-256:
`1139c5c34b662e77a2a13f3b0cee39f1d4e4437bf5a67482531b5c547fa1a1d6`.
It remains at the external root's `external-zip-receipt.json`; an identical copy,
the raw command log, retained-artifact recheck and artifact identity list are in
`.cache/release-readiness-20261107/public-preview/`.
Executed verifier SHA-256:
`99e7edc7757c6aa294517d46412febcc27cc7aa40256f3078f5576b3a5d8ca13`.

## Scope and remaining gates

This is same-host execution outside the checkout with minimal PATH, not a clean
machine, VM or second-PC proof. Native Linux remains deferred by the maintainer.
Full Go/compiler suites, the private CLI/whole-source/fixed-point acceptance,
other Windows toolchains, editor/VSIX and hosted CI were not run in this slice;
they are outside this public ZIP verification scope. Release identity, exact
candidate hosted CI, security route refresh and any separate-host evidence stay
subject to the existing [release contract](release.md).

No compiler/Go/Fairy source, public API, DB/schema, dependency/version, CI job,
workflow, package script, repository hygiene rule, source-of-truth contract,
queue/index or ADR changed. The only additions are the permanent verifier, its
Node tests and this record. No commit, push, dispatch, release or publication
was performed in this worker; parent owns integration and commits.
