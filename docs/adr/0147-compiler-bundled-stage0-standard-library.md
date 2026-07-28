# ADR 0147: Inject one exact compiler-bundled Stage 0 standard library

- Status: Accepted for prototype
- Date: 2026-07-21
- Product source: `docs/product/02-spec.md`
- Package source: `docs/package/package-graph-v0.md`
- Runtime source: `docs/runtime/runtime-stdlib-v0.md`
- Implementation: `internal/sdkstdlib/`, `internal/workspaceloader/`, `internal/lockfile/`

## Context

Package fixtures already require `std::*` to resolve through an explicit standard-library package
instance. The local workspace path previously stopped short of selecting that instance, so a real
`fairy.toml` project importing `std::alloc` could not reach package-aware semantic analysis without
declaring a fake local standard library. Searching PATH, installation directories, or a registry
would make the same compiler binary resolve different source and would create an installer contract
before Fairy C has one.

The first accepted Stage 0 source surface was intentionally smaller than the provisional target C
library and exported only opaque `std::alloc::Arena`. ADR 0205 later adds the first safe I/O source
surface without changing the package-discovery boundary established here.

## Decision

1. `internal/sdkstdlib` embeds the exact Stage 0 standard-library Fairy source bytes in the
   compiler. Loading performs no filesystem search, environment lookup, network access, or source
   generation.
2. The current SDK identity is `fairyc-sdk-stdlib/0`; it requires language
   `fairyc-language/0`, runtime ABI `fairyc-runtime/0`, package UID `fairyc.std`, and version
   `0.0.0-stage0.5` after ADR 0211.
3. The workspace loader injects one `standard-library` node only when a root-reachable package
   imports `std` or `std::*`. Each such package receives a direct dependency with the reserved
   binding `stdlib`. User manifests may not claim that binding or package UID, and local packages
   may not declare modules in the reserved `std` namespace.
4. The SDK source identity combines the exact SDK identity, runtime ABI, and package-source digest.
   PackageInstanceId v1 still hashes the ordinary package UID, source identity, version, and empty
   feature set; SDK location is never identity.
5. The embedded source exports opaque `std::alloc::Arena`, ADR 0205's
   `std::io::write(ref str) -> Result<usize, ConsoleError>`, ADR 0209's matching
   `write_stderr` operation, ADR 0210's matching immutable byte-view output operations, and ADR
   0211's `read_stdin(ref mut u8[]) -> Result<usize, ConsoleReadError>`. It does not expose `SystemHeap`, allocation, construction, capacity,
   layout, formatting, text or line input, text ownership, or
   collection APIs.
6. `fairy.lock` records `sourceKind: "sdk"`, empty `localPath`, exact `sdkIdentity`, exact
   `runtimeAbi`, source digest, composed source identity, package UID, version, role, and empty
   features. Strict reading rejects any contradictory SDK field. Local packages use
   `sourceKind: "local"` and must not contain SDK fields.
7. SDK injection feeds the same package graph, import-provider bindings, semantic analysis, MIR
   lowering, and MIR verification as a local dependency. It is not a compiler-created type or
   implicit prelude.
8. Source-package selection remains separate from target C standard-library linkage. Importing the
   opaque Arena declaration does not select a C object, settle public layout, or make package-aware
   native build/run complete.

## Consequences

- One compiler binary has one reproducible Stage 0 SDK source package on Windows and other hosts.
- A lock detects SDK source, ABI, identity, and version drift without recording an installation path.
- Unknown `std::*` imports still fail package resolution; injection does not manufacture modules.
- Adding any public standard-library source API requires an explicit contract, SDK version review,
  lock compatibility review, and tests through package-aware compilation.
- Package-aware native build/run and persistent package artifacts remain separate work.

## Verification

- SDK tests parse every lexically ordered embedded source and assert its composed identity;
- workspace tests prove conditional injection, reserved identity/binding/module-namespace
  rejection, missing-module rejection, exact provider lookup, host-path-free package metadata, and
  verified MIR consumption;
- lock tests round-trip the SDK record and reject source-kind, path, SDK-identity, and runtime-ABI
  contradictions;
- all configured Go format, test, build, and vet checks remain the validation boundary.
