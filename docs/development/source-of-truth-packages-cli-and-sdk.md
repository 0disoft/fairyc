# Source of Truth - Packages, CLI, and Standard Library

Part 2 of 3 of the [source index](source-of-truth.md).

## Runnable C interoperability example

- Contract: `docs/abi/c-abi-v0.md` and `docs/adr/0490-ordinary-synchronous-callbacks.md`.
- Checked-in Fairy policy and C filter/header: `examples/c_interop_callback/`.
- Windows separate-object build/run harness: `tools/dev/fairyc-c-interop/`.
- Scope and reproduction: `examples/c_interop_callback/README.md`; Linux execution is queued in `docs/development/ordered-work-20261031.md`.

## Package graph, interface identity, and digests


- **Resolved package graph fixtures**
  - `spec/fixtures/package/manifest-v1.json` with historical manifest-v0.json coverage
- **Resolved package graph validation and lookup**
  - `internal/packagegraph/`
- **Resolved package graph fixture runner**
  - `internal/packagefixture/`
- **PackageInstanceId v1 fixtures**
  - `spec/fixtures/package-identity/manifest-v1.json`
- **PackageInstanceId v1 encoding and fixture runner**
  - `internal/packagegraph/identity.go`
  - `internal/packageidentityfixture/`
- **PackageCoordinateId v1 encoding and workspace mapping**
  - `internal/packagegraph/coordinate.go`
  - `internal/workspaceloader/semantic.go`
  - `docs/package/package-coordinate-id-v1.md`
- **Public-interface hash v5 fixtures**
  - `spec/fixtures/module-interface/manifest-v5.json` with historical manifest-v1.json through manifest-v4.json coverage
- **Public-interface extraction, hashing, and fixture runner**
  - `internal/moduleinterface/`
  - `internal/moduleinterfacefixture/`
- **Opt-in ordinary public-interface/effects review and metadata-only large-source notice**
  - `internal/interfacereview/`
  - `internal/sourcenotice/`
  - `tools/dev/fairyc-review/`
  - `docs/ops/interface-source-review-20261101.md`
- **Package-source digest v1 fixtures**
  - `spec/fixtures/package-source/manifest-v1.json`
- **Package-source digest and fixture runner**
  - `internal/packagecontent/`
  - `internal/packagecontentfixture/`
- **Source-file content and line-map identities**
  - `internal/packagecontent/file_identity.go`
  - `docs/package/source-file-identities-v1.md`
  - `docs/adr/0184-source-file-integrity-identities-v1.md`
- **Generated lockfile v1**
  - `internal/lockfile/`
  - `docs/package/lockfile-v1.md`

## Cache, artifacts, and compilation reuse


- **Whole-package cache-key v4 fixtures**
  - `spec/fixtures/package-cache/manifest-v8.json` with historical manifest-v1.json through manifest-v7.json coverage
- **Whole-package cache key, bounded memory store, and fixture runner**
  - `internal/packagecache/`
  - `internal/packagecachefixture/`
- **Exact running-compiler executable identity**
  - `internal/buildidentity/`
  - `docs/adr/0165-executable-digest-compiler-build-identity.md`
- **Experimental persistent workspace-cache CLI and schema**
  - `internal/cli/workspace_cache.go`
  - `spec/cli/experimental-workspace-cache-check-v0.schema.json`
  - `docs/adr/0166-experimental-persistent-workspace-cache-command.md`
- **Bounded immutable disk cache and shared opened-root durable publication**
  - `internal/packagecache/disk.go`
  - `internal/durablefile/`
  - `docs/adr/0163-bounded-immutable-disk-cache-store.md`
  - `docs/adr/0169-opened-root-cache-containment.md`
- **Compiler-private verified whole-program MIR artifact codec**
  - `internal/mircodec/`
  - `docs/adr/0149-verified-mir-artifact-codec.md`
- **Compiler-private coordinate-stable function identity**
  - `internal/packageartifact/`
  - `docs/adr/0174-coordinate-stable-linking-and-cache-key-v2.md`
- **Process-local package MIR artifact extraction, verification, and reassembly**
  - `internal/packageartifact/artifact.go`
  - `docs/package/process-local-mir-artifact-v0.md`
- **Stable package type-context artifact and identity**
  - `internal/packageartifact/context.go`
  - `docs/adr/0161-stable-type-context-artifact.md`
- **Stable cross-process package MIR bodies, package-scoped context projection/merge, codec, and checked rebind**
  - `internal/packageartifact/body.go`
  - `internal/packageartifact/context_projection.go`
  - `internal/packageartifact/stable_artifact.go`
  - `docs/adr/0171-package-scoped-stable-context.md`
- **Process-bound package MIR artifact codec**
  - `internal/packageartifact/codec.go`
  - `docs/adr/0154-process-bound-package-artifact-codec.md`
- **Bounded process-local package artifact cache**
  - `internal/packageartifact/cache.go`
  - `internal/packagecache/store.go`
  - `docs/adr/0155-bounded-process-package-artifact-cache.md`
- **Package-aware process-local MIR reuse coordinator**
  - `internal/workspacecompiler/`
  - `docs/adr/0156-all-hit-process-mir-reuse.md`
- **Explicit stable disk package-compilation reuse coordinator**
  - `internal/workspacecompiler/`
  - `internal/packageartifact/stable_cache.go`
  - `docs/adr/0164-verified-stable-disk-mir-reuse.md`
  - `docs/adr/0353-stable-compilation-cache-records.md`
  - `docs/adr/0354-combined-cached-state-reassembly.md`
- **Bounded workspace check-session CLI and protocol schema**
  - `internal/cli/workspace_session.go`
  - `spec/cli/experimental-workspace-check-session-v0.schema.json`
  - `docs/adr/0159-bounded-workspace-check-session.md`

## Build measurement, manifest, scaffold, and workspace loading


- **Build-measurement report schemas and configured Windows calibrations**
  - `internal/buildmeasure/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0150-no-cache-windows-build-calibration.md`
  - `docs/adr/0157-process-cache-measurement-receipt.md`
  - `docs/adr/0158-process-cache-edit-distributions.md`
  - `docs/adr/0160-editor-shaped-session-process-measurement.md`
  - `docs/adr/0167-paired-persistent-cache-process-measurement.md`
  - `docs/adr/0171-package-scoped-stable-context.md`
  - `docs/adr/0191-atlas-whole-package-rewrite-measurement.md`
  - `docs/adr/0227-atlas-call-process-measurement-control.md`
  - `docs/adr/0228-atlas-workspace-call-scaling-control.md`
- **Human project-manifest v1 parser and fixtures**
  - `internal/projectmanifest/`
  - `internal/projectmanifestfixture/`
  - `spec/fixtures/project-manifest/manifest-v1.json`
- **Stable exact starter-project creation**
  - `internal/projectscaffold/`
  - `internal/cli/project_new.go`
  - `docs/adr/0204-stable-starter-project-creation-v1.md`
  - `docs/adr/0206-starter-project-guide-v2.md`
  - `docs/adr/0208-starter-project-guide-v3.md`
  - `docs/adr/0266-starter-guide-native-test-v4.md`
- **Local workspace loader and fixtures**
  - `internal/workspaceloader/`
  - `internal/workspaceloaderfixture/`
  - `spec/fixtures/workspace-loader/manifest-v1.json`

## Stable CLI and workspace commands


- **Stable semantic/native JSON test execution reports, schema drift guards, exact exit-code preservation, binary native stream attachments, and standalone optimized MSVC evidence**
  - `internal/cli/workspace_tests.go`
  - `internal/cli/schema_test.go`
  - `spec/cli/stable-test-report-v1.schema.json`
  - `spec/cli/stable-test-report-v4.schema.json`
  - `tools/dev/fairyc-windows-host-smoke/main.go`
  - `docs/adr/0287-stable-json-test-execution-report.md`
  - `docs/adr/0314-stable-native-test-output-attachments.md`
- **Stable bounded native-test stdin-file replay**
  - `internal/cli/workspace_test_input.go`
  - `examples/native_test_stdin_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/native_test_stdin_proof.go`
  - `docs/adr/0315-stable-native-test-stdin-file-replay.md`
- **Stable bounded native-test process timeout**
  - `internal/cdriver/windows_host.go`
  - `internal/cli/workspace_tests.go`
  - `examples/native_test_timeout_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/native_test_timeout_proof.go`
  - `docs/adr/0316-stable-native-test-timeout.md`
- **Package-aware Windows workspace build/run CLI and native smoke**
  - `internal/cli/workspace_native.go`
  - `internal/cli/program_output.go`
  - `internal/cdriver/windows_host.go`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `examples/workspace_quickstart/`
  - `examples/std_io_quickstart/`
  - `examples/process_args_quickstart/`
  - `docs/adr/0148-package-aware-windows-build-run.md`
  - `docs/adr/0205-target-neutral-stage0-stdout-write.md`
  - `docs/adr/0207-stable-windows-workspace-run-v2.md`
  - `docs/adr/0212-stable-windows-workspace-run-v3.md`
  - `docs/adr/0261-stable-run-program-argument-forwarding.md`

## Standard library and SDK surface


- **Compiler-bundled Stage 0 standard-library source package, public safe Arena creation, fallible native Region placement and generated-Fairy allocation-failure execution
  byte Buffer with SDK-owned canonical scalar text append, owned text, allocation-free UTF-8 validation and immutable text-byte views, zero-copy Buffer-to-String conversion
  Windows MSVC File ownership, partial/exact byte I/O, one-call partial write, append, exclusive create, open-or-create, existing-file update open, cursor-preserving length change
  absolute positioning, shared size observation, exhaustive same-handle file-kind observation, bounded file-identity observation, file/directory path lifecycle and no-replace rename
  user-enum Result C transport and propagation, user-enum Option C transport, console I/O, and process-argument APIs**
  - `internal/sdkstdlib/`
  - `internal/cbackend/`
  - `internal/cstdlib/`
  - `internal/cstdlibcontract/`
  - `examples/std_alloc_quickstart/`
  - `examples/regions/`
  - `examples/std_bytes_quickstart/`
  - `examples/std_text_quickstart/`
  - `examples/std_text_from_buffer_quickstart/`
  - `examples/std_fs_quickstart/`
  - `examples/process_args_quickstart/`
  - `examples/std_io_scalar_quickstart/`
  - `docs/adr/0250-public-safe-arena-creation.md`
  - `docs/adr/0251-fallible-region-placement.md`
  - `docs/adr/0252-native-region-lifecycle-lowering.md`
  - `docs/adr/0255-generated-fairy-region-failure-execution.md`
  - `docs/adr/0256-generated-fairy-buffer-failure-execution.md`
  - `docs/adr/0257-generated-fairy-string-append-failure-execution.md`
  - `docs/adr/0258-entry-reachable-c-emission.md`
  - `docs/adr/0259-generated-fairy-cstring-failure-execution.md`
  - `docs/adr/0260-generated-fairy-string-copy-failure-execution.md`
  - `docs/adr/0261-stable-run-program-argument-forwarding.md`
  - `docs/adr/0262-scalar-and-bool-stdout-output.md`
  - `docs/adr/0263-generated-fairy-buffer-to-string-conversion.md`
  - `docs/adr/0302-public-allocation-free-text-byte-view.md`
  - `docs/adr/0264-public-windows-stage0-files.md`
  - `docs/adr/0269-public-windows-file-positioning.md`
  - `docs/adr/0270-public-windows-file-size.md`
  - `docs/adr/0271-public-windows-file-update-open.md`
  - `docs/adr/0272-public-windows-file-length-change.md`
  - `docs/adr/0274-public-windows-file-open-or-create.md`
  - `docs/adr/0275-public-windows-file-exclusive-create.md`
  - `docs/adr/0276-public-windows-file-append-open.md`
  - `docs/adr/0277-public-windows-file-kind.md`
  - `docs/adr/0278-public-windows-file-read-exact.md`
  - `docs/adr/0279-public-windows-file-identity.md`
  - `docs/adr/0280-result-user-enum-c-transport.md`
  - `docs/adr/0281-public-windows-file-kind-enum.md`
  - `docs/adr/0282-generated-user-enum-result-propagation.md`
  - `docs/adr/0283-user-enum-option-c-transport.md`
  - `docs/adr/0285-public-windows-file-partial-write.md`
  - `docs/adr/0288-public-windows-file-buffer-read-append.md`
  - `docs/adr/0290-public-windows-file-read-to-end.md`
  - `docs/adr/0291-public-windows-file-removal.md`
  - `docs/adr/0292-public-windows-file-no-replace-rename.md`
  - `docs/adr/0293-public-windows-directory-lifecycle.md`
  - `docs/adr/0294-public-windows-directory-no-replace-rename.md`
  - `docs/adr/0246-public-safe-stage0-byte-buffer.md`
  - `docs/adr/0247-public-safe-stage0-owned-text.md`
  - `docs/adr/0248-public-allocation-free-utf8-validation.md`
  - `docs/adr/0205-target-neutral-stage0-stdout-write.md`
  - `docs/adr/0209-target-neutral-stage0-stderr-write.md`
  - `docs/adr/0210-target-neutral-stage0-byte-output.md`
  - `docs/adr/0211-target-neutral-stage0-stdin-read.md`
  - `docs/adr/0215-allocation-free-stage0-i32-output.md`
  - `docs/adr/0373-sdk-buffer-scalar-formatting.md`
- **Public Windows directory lifecycle, no-replace rename, bounded recursive creation, dedicated raw probe ownership, and low/optimized generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file.c`
  - `internal/cstdlib/stage0_windows_file_probe.c`
  - `internal/cstdlib/stage0_windows_file_directory_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0293-public-windows-directory-lifecycle.md`
  - `docs/adr/0294-public-windows-directory-no-replace-rename.md`
  - `docs/adr/0296-public-windows-recursive-directory-creation.md`
- **Public PathKind file/directory predicates and SDK `0.0.0-stage0.46`**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `examples/std_fs_quickstart/src/path_ops.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0317-public-path-kind-predicates.md`
- **Allocation-free empty Buffer construction, zero-copy String-to-Buffer ownership transfer, and SDK `0.0.0-stage0.47`**
  - `internal/sdkstdlib/sources/std/bytes.fy`
  - `internal/sdkstdlib/sources/std/text.fy`
  - `internal/cstdlib/stage0_owned_text.c`
  - `internal/cstdlibcontract/`
  - `examples/std_text_from_buffer_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0318-zero-copy-string-to-buffer-transfer.md`
- **Dependency-ordered inert-enum struct fields, public sequential FileMetadata, and SDK `0.0.0-stage0.48`**
  - `internal/cbackend/aggregate_types.go`
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `spec/fixtures/c-emission/cases/pass/inert-enum-struct-field/`
  - `examples/std_fs_quickstart/src/metadata.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0319-public-file-metadata-snapshot.md`
- **Target-neutral public File open/read/write/sync/close core and SDK `0.0.0-stage0.49`**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_linux_file_portable.c`
  - `internal/cstdlib/stage0_windows_file_portable.c`
  - `internal/cstdlibcontract/`
  - `internal/cbackendnative/`
  - `docs/adr/0331-target-neutral-public-file-core.md`
- **Target-neutral public File position/absolute-seek and SDK `0.0.0-stage0.50`**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_linux_file_portable.c`
  - `internal/cstdlib/stage0_windows_file_portable.c`
  - `internal/cstdlibcontract/`
  - `internal/cbackendnative/`
  - `docs/adr/0332-target-neutral-public-file-positioning.md`
- **Target-neutral public File size observation and SDK `0.0.0-stage0.51`**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_linux_file_portable.c`
  - `internal/cstdlib/stage0_windows_file_portable.c`
  - `internal/cstdlibcontract/`
  - `internal/cbackendnative/`
  - `docs/adr/0333-target-neutral-public-file-size.md`
- **Target-neutral public File relative/end seek and SDK `0.0.0-stage0.52`**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_linux_file_portable.c`
  - `internal/cstdlib/stage0_windows_file_portable.c`
  - `internal/cstdlibcontract/`
  - `internal/cbackendnative/`
  - `docs/adr/0334-target-neutral-public-file-relative-end-seek.md`
- **Public bounded Windows File-to-Buffer append reads, cross-component dependency closure, initialized-prefix commit semantics
  and generated low/optimized MSVC allocation-failure evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_buffer.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/read_append_failure_probe.go`
  - `docs/adr/0288-public-windows-file-buffer-read-append.md`
- **Allocation-free scalar stderr output, target-neutral numeric symbols, shared stack formatting, and low/optimized generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/io.fy`
  - `internal/cstdlib/stage0_linux_console.c`
  - `internal/cstdlib/stage0_windows_console.c`
  - `internal/cstdlibcontract/`
  - `examples/std_io_scalar_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0303-allocation-free-scalar-stderr-output.md`
- **Bounded Windows File current/end seek, same-handle target symbols, dedicated raw probe, and low/optimized private/public generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file.c`
  - `internal/cstdlib/stage0_windows_file_seek_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/windows_file/`
  - `examples/std_fs_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0305-public-windows-relative-end-seek.md`
- **Same-handle Windows File read-only metadata, target-neutral bool mapping, dedicated implementation/probe owners, and low/optimized public generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file.c`
  - `internal/cstdlib/stage0_windows_file_metadata.c`
  - `internal/cstdlib/stage0_windows_file_metadata_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0306-public-windows-file-read-only-metadata.md`
- **Same-handle Windows File hard-link count, target-neutral usize mapping, real-link raw/public fixtures, and low/optimized generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_metadata.c`
  - `internal/cstdlib/stage0_windows_file_metadata_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0307-public-windows-file-hard-link-count.md`
- **No-replace Windows File hard-link creation, bounded pre-effect path validation, dedicated target/probe owners, shared-identity and alias-survival evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_link.c`
  - `internal/cstdlib/stage0_windows_file_link_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/src/link_ops.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0308-public-windows-file-hard-link-creation.md`
- **Same-handle Windows File allocated size, target-neutral usize mapping, cursor/size preservation, and low/optimized raw plus generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_metadata.c`
  - `internal/cstdlib/stage0_windows_file_metadata_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/src/metadata.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0309-public-windows-file-allocated-size.md`
- **Same-handle Windows File read-only mutation, unrelated-attribute preservation, exact access-denied failure, and low/optimized raw plus generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_metadata.c`
  - `internal/cstdlib/stage0_windows_file_metadata_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/src/metadata.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0310-public-windows-file-read-only-mutation.md`
- **Root-protected Windows recursive directory removal, reparse-leaf handling, exact partial progress, and low/optimized raw plus generated-Fairy evidence**
  - `internal/sdkstdlib/sources/std/fs.fy`
  - `internal/cstdlib/stage0_windows_file_remove_tree.c`
  - `internal/cstdlib/stage0_windows_file_remove_tree_probe.c`
  - `internal/cstdlibcontract/`
  - `examples/std_fs_quickstart/src/path_ops.fy`
  - `tools/dev/fairyc-windows-host-smoke/`
  - `docs/adr/0311-public-windows-recursive-directory-removal.md`

- Explicit inline export identity and cache successor: `docs/adr/0493-cross-package-inline-bodies-and-cache-v5.md`
- Current whole-package key: `docs/package/whole-package-cache-v5.md`
- Fairy candidate inline source/identity/artifact/cache/emission contract: `docs/architecture/candidate-inline-v1.md`.
- Historical Windows candidate inline execution and source-admission evidence: `docs/ops/windows-candidate-inline-20261102.md`.
- Full Windows inline CLI and cross-package fresh-process cache: `internal/cli/candidate_inline_cli_test.go`, `candidate_inline_cli_cache_test.go` and `docs/ops/windows-candidate-inline-cli-cache-20261103.md`.
- Generic callers of nongeneric cross-package inline exports: `internal/cli/candidate_generic_inline_cli_test.go`, `candidate_generic_inline_cli_cache_test.go` and `docs/ops/windows-candidate-generic-inline-cli-cache-20261103.md`.
- Inline arithmetic trap diagnostics, cached execution and native-test continuation: `internal/cli/candidate_inline_trap_fixture_test.go`, `candidate_inline_trap_cli_test.go` and `docs/ops/windows-candidate-inline-traps-20261103.md`; private MSVC parameter barriers live in `internal/cbackend/functions.go` and `examples/lexer_candidate/src/module_unit_c_calls.fy`.
- Zero-parameter inline arithmetic traps and successful continuation: `internal/cbackend/inline_zero_trap_test.go`, the same CLI trap fixtures and `docs/ops/windows-candidate-inline-zero-traps-20261103.md`; the Go-only private MSVC divisor-guard correction lives in `internal/cbackend/terminators.go`.
- Inline remainder and multiplication traps/controls: `internal/cbackend/inline_remainder_multiply_test.go`, `internal/cli/candidate_inline_remainder_multiply_test.go`, the shared CLI trap fixture/runner and `docs/ops/windows-candidate-inline-remainder-multiply-20261103.md`. The original and new matrices have separate test entry points.
- Inline subtraction and negation traps/boundaries: `internal/cbackend/inline_subtract_negate_test.go`, `assert_checked_negation_test.go`, `internal/cli/candidate_inline_subtract_negate_test.go` and `docs/ops/windows-candidate-inline-subtract-negate-20261103.md`; exact i32 negation facts are transferred in `internal/cbackend/assert_checked.go`.
- Nested inline first-failure ordering and exact columns: `internal/cbackend/inline_nested_trap_test.go`, `internal/cli/candidate_inline_nested_trap_test.go`, optional source-counted columns in the shared trap fixture and `docs/ops/windows-candidate-inline-nested-traps-20261103.md`. Checked-storage liveness is owned by `analysis_executable.go`, `analysis_locals.go` and `functions.go` in `internal/cbackend`.
- Go/Fairy candidate v5 encoder, complete no-inline key and source-boundary evidence: `docs/ops/windows-candidate-parity-20261102.md`.
- Candidate parity probes: `internal/cli/candidate_cache_key_parity_test.go`, `internal/cli/candidate_complete_keys_parity_test.go`, `internal/cli/candidate_inline_boundary_test.go`.


## Pinned experimental remote source preparation

- Decision: `docs/adr/0498-explicit-pinned-remote-package-sources.md`.
- Contract: `docs/package/remote-sources-v1.md`; CLI admission: `docs/cli/command-contract.md`.
- Implementation and focused acceptance: `internal/packagesource/`, `internal/cli/package_fetch.go`, `internal/cli/package_fetch_test.go`.
- Fairy plan parser and URL policy: `examples/lexer_candidate/src/project_remote_plan.fy`, `project_source_url.fy`; native Go comparison: `internal/cli/candidate_source_plan_test.go`.
- Fairy workspace binding: `examples/lexer_candidate/src/project_remote_bind.fy`; manifest/Go oracle and shared-budget controls: `internal/cli/candidate_remote_bind_test.go`.
- Fairy Windows HTTPS transport: `examples/lexer_candidate/src/project_remote_https.fy`, `internal/candidatehost/windows_https.c`; opt-in real TLS acceptance: `internal/cli/candidate_https_windows_test.go`.
- Fairy private offline restoration: `examples/lexer_candidate/src/project_remote_offline.fy`, `project_remote_fs.fy`, `project_remote_archive.fy`, `project_inflate.fy`; native oracles: `internal/cli/candidate_remote_offline_test.go`, `candidate_remote_archive_test.go`, `candidate_inflate_test.go`.
- Windows offline API receipt: `docs/ops/windows-candidate-remote-offline-20261103.md`. Retained directory primitives belong to `internal/candidatehost/windows_rooted_file.c` and `examples/lexer_candidate/src/windows_rooted_file.fy`; the generated runtime asset pin must match the C sources.
- Fairy online preparation/cache: `examples/lexer_candidate/src/project_remote_online.fy`, `project_remote_cache.fy`; real TLS command/full CLI comparisons: `internal/cli/candidate_fetch_online_windows_test.go`; receipt: `docs/ops/windows-candidate-remote-online-20261104.md`.
- Windows online/offline command: `examples/lexer_candidate/src/project_fetch_cli.fy`, dispatched by `file_cli.fy`; command-leaf/full CLI acceptance: `internal/cli/candidate_fetch_command_test.go`, `candidate_fetch_cli_test.go`; historical offline receipt and pending Linux scope: `docs/ops/windows-candidate-fetch-cli-20261104.md`.
- Linux retained package-directory prerequisite: `internal/candidatehost/linux_package_directory.c`, selected by `linux_paths.go`; independent Windows boundary substitutes and authored GCC/Clang replay: `linux_package_directory_test.go`. Exact evidence and still-unimplemented Fairy filesystem/cache/HTTPS/CLI scope: `docs/development/candidate-linux-package-fetch-20261106.md`.
- Fairy Linux directory lease: `examples/lexer_candidate/src/linux_package_directory.fy`, unsafe owned-handle adoption in `linux_rooted_file.fy`; both-target emission and independent Windows lifetime/error/phase probes: `internal/cli/linux_package_directory_test.go`, `examples/lexer_candidate/tests/linux_package_directory.fy`. The Linux fetch receipt separates this capability from unimplemented package/cache/transport/CLI integration and deferred Linux execution.
- Linux package-only single-link metadata and conditional cleanup: `internal/candidatehost/linux_package_file.c`, `examples/lexer_candidate/src/linux_package_file.fy`; ordinary and package scan admission: `linux_directory.fy`. Independent C/Fairy Windows probes, authored Linux replay and exact 8,192-row/work controls: `internal/candidatehost/linux_package_file_test.go`, `internal/cli/linux_package_file_test.go`, `linux_directory_test.go`; evidence remains in the Linux fetch receipt.
- Windows receipt and deferred Linux replay: `docs/ops/windows-remote-packages-20261101.md` and `docs/development/ordered-work-20261031.md`.

## Fairy candidate direct scalar generic transport

- Owner: `docs/adr/0494-broader-scalar-generic-transport.md` and `docs/architecture/candidate-generic-transport-v1.md`.
- Authored admission and scalar mapping: `examples/lexer_candidate/src/generic_transport_scalars.fy`, `module_types.fy`, `generic_concrete_types.fy`, `generic_signature_diagnostics.fy` and `module_unit_generics.fy` in the same directory.
- Native replay: `internal/cli/candidate_generic_transport_test.go`, `internal/cli/candidate_generic_transport_pipeline_test.go`, and `examples/lexer_candidate/tests/module_generic_transport.fy`.
- Historical source/artifact evidence: `docs/ops/windows-candidate-generics-20261102.md`.
- Full Windows CLI and cross-package fresh-process cache: `internal/cli/candidate_generic_cli_test.go`, `candidate_generic_cli_observer_test.go`, `candidate_generic_cli_cache_test.go` and `docs/ops/windows-candidate-generics-cli-cache-20261103.md`.
- Deferred native Linux replay and remaining interpreter/ABI scope: `docs/development/ordered-work-20261031.md` and `docs/architecture/candidate-generic-transport-v1.md`.

## Fairy candidate static operation interfaces

- Owner: `docs/adr/0495-explicit-static-operation-interfaces.md` and `docs/architecture/candidate-static-operations-v1.md`.
- Declaration, wrapper and digest owners: `examples/lexer_candidate/src/static_operation_declarations.fy`, `static_operation_syntax.fy`, `static_operation_identity.fy` and the generic source/module-unit consumers named by the architecture page.
- Native admission, cross-package Go identity oracle and restored-artifact C replay: `internal/cli/candidate_static_operation*_test.go` and `examples/lexer_candidate/tests/static_operation*.fy`.
- Historical source/artifact evidence: `docs/ops/windows-candidate-static-operations-20261102.md`.
- Full Windows CLI and cross-package fresh-process cache: `internal/cli/candidate_static_operation_cli_test.go`, `candidate_static_operation_cli_cache_test.go` and `docs/ops/windows-candidate-static-operations-cli-cache-20261103.md`; scalar packet observers reuse `internal/cli/candidate_generic_cli_observer_test.go`.
- Deferred native Linux replay: `docs/development/ordered-work-20261031.md`.

## Fairy candidate copied local closures

- Owner: `docs/adr/0496-explicit-copied-local-closures.md` and `docs/architecture/candidate-copied-closures-v1.md`.
- Syntax, admission, private lifting and ownership: `examples/lexer_candidate/src/body_copied_closure.fy`, `copied_closure_admission.fy`, `copied_closure_lower.fy`, `copied_closure_tables.fy` and `copied_closure_reach.fy`.
- Native admission, Go identity oracle and restored-artifact C replay: `internal/cli/candidate_copied_closure*_test.go` and `examples/lexer_candidate/tests/copied_closure*.fy`.
- Windows source/artifact evidence: `docs/ops/windows-candidate-copied-closures-20261102.md`.
- Full Windows CLI and fresh-process persistent cache: `internal/cli/candidate_copied_closure_cli_host_test.go`, `candidate_copied_closure_cli_test.go`, `candidate_copied_closure_persistent_test.go` and `docs/ops/windows-candidate-copied-closures-cli-cache-20261103.md`.
- Deferred native Linux replay: `docs/development/ordered-work-20261031.md`.
