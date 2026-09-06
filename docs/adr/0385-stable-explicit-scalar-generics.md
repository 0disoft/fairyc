# ADR 0385: Stable explicit scalar generic functions

- Status: Accepted
- Date: 2026-09-06

## Decision

Language `fairyc-language/14` admits explicit builtin-scalar generic functions through stable
workspace `check`, `build`, `run`, and `test`. SDK `0.0.0-stage0.57` carries the same language
identity; runtime ABI remains `fairyc-runtime/0`. Existing source must update its manifest pin and
refresh its lock. The starter and current fixture/example manifests use the new revision.

```text
pub fn T identity<T>(T value) { return value; }
fn i32 main() { return identity<i32>(42); }
```

The commands share `workspacecompiler.Check`: both semantic input and package coordinates come
from the same resolved workspace. ADR 0384's expansion runs before ordinary semantic/Flow and MIR
verification. Cross-module calls use explicit import aliases, public template visibility, and exact
package ownership. Tests use the loaded test overlay. Repeated calls remain executable calls even
when their static specialization dependency is deduplicated.

Type arguments must be explicit builtin scalars or directly forwarded type parameters. In generic
workspaces, generated names beginning `_fy_generic_` are reserved. Inference, nominal/composite type arguments, generic
methods, user-defined capabilities/interfaces, dynamic dispatch, recursive specialization and
unbounded expansion remain rejected. Planner defaults bound depth and total/per-declaration
specializations. Workspaces without generic declarations retain ordinary analysis and diagnostics
without generic preparation.

Source-owned generic rejection uses E4501-E4503 and the existing name/access codes; source
diagnostics retain exit 3 and original locations. Native execution preserves existing target,
toolchain, stream, lock, and program-exit rules. No new CLI flags are introduced.

## Cache and experimental boundaries

Stable commands do not read or write compilation caches. The existing private generic memory/disk
lane retains separate identities and current-source reanalysis. Experimental cache and session
commands now select that lane through `SourceSession` when the current workspace contains generic
declarations; ordinary workspaces retain their previous cache family. Atlas source-symbol support
is not promoted by this decision. Generic interface
and cache records are not public ABI or portable distribution formats. Generic nominal types and
static interface source syntax are still separate work; this slice does not claim their completion.

SourceSession holds one active session, closes it before a source-family transition, and discards
the old memory LRU. Counters restart for the new family. Every request still reloads current source;
invalid source cannot become a verified hit. Restoring valid source can reuse the same family's
retained compatible artifacts. Configuration is copied, while an opened filesystem root remains
borrowed from the command's enclosing lifetime. Check and Close are sequential operations.

Disk formats remain separate: ordinary records use the existing directory and generic records use
`generic-v0`. Entry/byte bounds and reported retention counters apply to the active format namespace,
not their combined on-disk footprint; switching formats does not delete historical cache data.
Generic contract-state compilation still falls back to verified uncached lowering.

## Explicit-file experimental commands

`check-semantics`, verified `check`, interpreter `run`, C emission, and the native commands sharing
the explicit-file lowering path now use AnalyzeExplicitFiles/CheckExplicitFiles. A source set is
one build-local default package; public generic helpers may be imported from another explicitly
supplied module. No package coordinate is synthesized. The owner `explicit_files_v0` is only an
internal specialization namespace, not a manifest identity or a cache compatibility promise.

Original file/module identity maps source-owned generic failures to existing diagnostics. Absent
imports and duplicate modules retain ordinary resolve diagnostics. Source files remain immutable,
and the existing file-count, aggregate-byte, path, entry, target and native-execution bounds remain
unchanged. This path discovers no implicit files, loads no manifest or SDK, and publishes no public
interface or cached artifacts. Ordinary files without generic declarations retain their previous
analysis. Direct-file acceptance does not imply Atlas can describe original generic declarations.

CLI tests execute a two-module generic helper chain as 42 in the interpreter and low/optimized
MSVC, cover typed-HIR/MIR checks and C emission, and verify source diagnostics without partial output.

## Evidence

Resolved-package and explicit-file analysis now retain an immutable `GenericProvenance` sidecar:
all original generic declarations, including unused private declarations, have exact current
package/module/file spans and ordered parameter names. Each emitted specialization references its
original declaration ID, generated function name and concrete parameter bindings. The mapping is
constructed from the declaration catalog rather than reverse-parsing generated names.

Successful lowering publishes this sidecar; failed analysis/lowering does not. Generic memory and
disk hits attach the current analysis sidecar after verified reassembly, never artifact-supplied
provenance. Getters detach nested parameter/binding slices. This is ephemeral source provenance,
not a serialized cache format, public SymbolId, signature encoding, or proof that an unused template
body has passed semantic analysis. The older private module-only test adapter is not an Atlas input.
Atlas snapshot and original generic SymbolId integration still remain required.

Tests cover unused declarations, exact name spans, emitted-function joins, getter mutation
isolation, source rejection, memory cold/warm reuse and fresh-process persistent hits.

`internal/cli/workspace_generic_test.go` checks ordinary/locked stable check and semantic test,
source diagnostic classification, and Windows low/release build, run, and native test. The shared
workspace and compiler fixtures retain interpreter, original AST, ownership, cache invalidation,
and low/optimized MSVC evidence.
